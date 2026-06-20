package yeelight

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"
)

const maxId = 1 << 10
const respWaitTimeout = 5 * time.Second

type Device struct {
	*Info

	data   Data         // device props; guarded by dataMu
	dataMu sync.RWMutex // guards data: written by listen() goroutine, read by callers via Snapshot
	con    net.Conn     // TCP connection
	done   chan struct{}

	mu          sync.RWMutex
	baseID      int
	pending     map[int]chan Response
	updatedChan chan struct{}
}

// New establishes a TCP connection to the Yeelight device at the specified IP address and initializes a Yeelight instance. It also starts a goroutine to listen for responses from the device.
func New(ctx context.Context, info *Info) (*Device, error) {
	con, err := net.Dial("tcp", info.IP+":55443")
	if err != nil {
		return nil, err
	}
	d := &Device{
		Info:        info,
		con:         con,
		done:        make(chan struct{}),
		mu:          sync.RWMutex{},
		baseID:      0,
		pending:     make(map[int]chan Response),
		updatedChan: make(chan struct{}),
	}
	go d.listen(ctx)
	return d, nil
}

func (d *Device) Updated() <-chan struct{} {
	return d.updatedChan
}

// Snapshot returns a copy of the device's current props, safe to read from any
// goroutine. Data's fields are pointers that mergeValue only ever replaces
// (never mutates in place), so a shallow struct copy is a consistent snapshot.
func (d *Device) Snapshot() Data {
	d.dataMu.RLock()
	defer d.dataMu.RUnlock()
	return d.data
}

// Done is closed when the device is closed; watchers should select on it to exit.
func (d *Device) Done() <-chan struct{} {
	return d.done
}

func (d *Device) Close() error {
	close(d.done)
	return d.con.Close()
}

// SendCommand sends a command to the device and waits for the matching
// response (correlated by id, with a 5s timeout). The device's advertised
// support list is treated as a hint, NOT a gate: some firmware supports
// methods it does not advertise (notably set_music), so the command is always
// sent and the device itself is the authority. A genuinely unsupported method
// comes back as an *Error for which IsUnsupported reports true.
func (d *Device) SendCommand(ctx context.Context, cmd Command) (*Response, error) {
	if cmd.ID == 0 {
		cmd.ID = d.genID()
	}
	if cmd.Params == nil {
		cmd.Params = []any{}
	}
	jsonData, err := json.Marshal(cmd)
	slog.InfoContext(ctx, "Sending command", "command", string(jsonData))
	if err != nil {
		return nil, err
	}
	jsonData = append(jsonData, '\r', '\n')
	_, err = d.con.Write(jsonData)
	if err != nil {
		return nil, err
	}
	resp, ok := d.waitResponse(ctx, cmd.ID)
	if !ok {
		return nil, fmt.Errorf("response timeout for command ID %d", cmd.ID)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp, nil
}

// -- INTERNAL METHODS --

func (d *Device) pushResponse(ctx context.Context, resp Response) {
	// id = 0 means it's a notification
	if resp.ID == 0 {
		if resp.Method == Props {
			d.updateData(ctx, resp.Params)
		}
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.pending[resp.ID] == nil {
		d.pending[resp.ID] = make(chan Response)
	}

	select {
	case d.pending[resp.ID] <- resp:
	default:
		close(d.pending[resp.ID])
		d.pending[resp.ID] = make(chan Response)
		d.pending[resp.ID] <- resp
	}
}

func (d *Device) waitResponse(ctx context.Context, id int) (*Response, bool) {
	ctx, cancel := context.WithTimeout(ctx, respWaitTimeout)
	defer cancel()

	d.mu.RLock()
	ch, exists := d.pending[id]
	if !exists {
		ch = make(chan Response)
		d.pending[id] = ch
	}
	d.mu.RUnlock()

	select {
	case resp := <-ch:
		return &resp, true
	case <-ctx.Done():
		return nil, false
	}
}

func (d *Device) genID() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.baseID %= maxId
	d.baseID++
	return d.baseID
}

// listen continuously listens for responses from the Yeelight device. It reads data from the TCP connection, parses it as JSON, and pushes the responses to the notification handler. The function runs until the context is canceled, at which point it closes the connection and stops listening for responses.
func (d *Device) listen(ctx context.Context) {
	defer d.con.Close()
	reader := bufio.NewReader(d.con)
	for {
		select {
		case _, ok := <-d.done:
			if !ok {
				return
			}
			return
		case <-ctx.Done():
			return
		default:
			data, err := reader.ReadString('\n')
			slog.InfoContext(ctx, "Received data", "data", data)
			if err != nil {
				return
			}
			var resp Response
			err = json.Unmarshal([]byte(data), &resp)
			if err != nil {
				continue
			}
			d.pushResponse(ctx, resp)
		}
	}
}

func (d *Device) updateData(ctx context.Context, data Data) {
	// main_power and power are the same thing (the main light); firmware emits
	// one or the other in different props messages. Mirror within this message
	// so readers can rely on Power alone and always see the latest value.
	if data.Power == nil {
		data.Power = data.MainPower
	} else if data.MainPower == nil {
		data.MainPower = data.Power
	}

	d.dataMu.Lock()
	mergeData(&d.data, &data)
	d.dataMu.Unlock()

	select {
	case d.updatedChan <- struct{}{}:
	default:
		return
	}

}

// ApplyLocal optimistically merges caller-known state (only non-nil fields)
// into the snapshot and pulses watchers, for state the bulb won't echo back —
// e.g. the power we just set on firmware that omits power from async props.
// Without it, a later props notification would re-apply the stale value and
// fight the user's toggle.
func (d *Device) ApplyLocal(data Data) {
	d.updateData(context.Background(), data)
}

func (d *Device) FetchProps(ctx context.Context) error {
	resp, err := d.SendCommand(ctx, Command{
		Method: "get_prop",
		Params: AllProperties,
	})
	if err != nil {
		return err
	}
	d.dataMu.Lock()
	defer d.dataMu.Unlock()
	for i, prop := range AllProperties {
		if i >= len(resp.Result) {
			break
		}
		value := resp.Result[i]
		if value == "" {
			continue
		}
		switch prop {
		case Bright, Ct, RGB, Hue, Sat, ColorMode, Flowing, DelayOff, MusicOn, BgFlowing, BgCt, BgLMode, BgBright, BgRGB, BgSat, NlBr, ActiveMode:
			int64Value, err := strconv.ParseInt(resp.Result[i], 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing property %s: %w", prop, err)
			}
			intValue := int(int64Value)
			switch prop {
			case Bright:
				d.data.Bright = Ptr(intValue)
			case Ct:
				d.data.Ct = Ptr(intValue)
			case RGB:
				d.data.RGB = Ptr(intValue)
			case Hue:
				d.data.Hue = Ptr(intValue)
			case Sat:
				d.data.Sat = Ptr(intValue)
			case ColorMode:
				d.data.ColorMode = Ptr(intValue)
			case Flowing:
				d.data.Flowing = Ptr(intValue)
			case DelayOff:
				d.data.DelayOff = Ptr(intValue)
			case MusicOn:
				d.data.MusicOn = Ptr(intValue)
			case BgFlowing:
				d.data.BgFlowing = Ptr(intValue)
			case BgCt:
				d.data.BgCt = Ptr(intValue)
			case BgLMode:
				d.data.BgLMode = Ptr(intValue)
			case BgBright:
				d.data.BgBright = Ptr(intValue)
			case BgRGB:
				d.data.BgRGB = Ptr(intValue)
			case BgSat:
				d.data.BgSat = Ptr(intValue)
			case NlBr:
				d.data.NLBr = Ptr(intValue)
			case ActiveMode:
				d.data.ActiveMode = Ptr(intValue)
			case BGProact:
				d.data.BGProact = Ptr(intValue)
			}
		default:
			value = resp.Result[i]
			switch prop {
			case Power:
				d.data.Power = Ptr(value)
			case MainPower:
				d.data.MainPower = Ptr(value)
			case FlowParams:
				d.data.FlowParams = Ptr(value)
			case Name:
				d.data.Name = Ptr(value)
			case BgPower:
				d.data.BgPower = Ptr(value)
			case BgFlowParams:
				d.data.BgFlowParams = Ptr(value)
			}
		}
	}
	return nil
}
