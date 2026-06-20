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

	Data Data     `json:"data"`
	con  net.Conn // TCP connection
	done chan struct{}

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

// Done is closed when the device is closed; watchers should select on it to exit.
func (d *Device) Done() <-chan struct{} {
	return d.done
}

func (d *Device) Close() error {
	close(d.done)
	return d.con.Close()
}

// SendCommand sends a command to the Yeelight device and waits for the corresponding response. It generates a unique command ID if one is not provided, marshals the command into JSON format, and writes it to the TCP connection. The function then waits for a response with the same command ID using the notification handler. If a response is received within the timeout period, it returns the response; otherwise, it returns an error indicating a timeout.
func (d *Device) SendCommand(ctx context.Context, cmd Command) (*Response, error) {
	if ok := d.Methods[cmd.Method]; !ok {
		return nil, fmt.Errorf("unsupported method: %s", cmd.Method)
	}
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
	mergeData(&d.Data, &data)

	select {
	case d.updatedChan <- struct{}{}:
	default:
		return
	}

}

func (d *Device) FetchProps(ctx context.Context) error {
	resp, err := d.SendCommand(ctx, Command{
		Method: "get_prop",
		Params: AllProperties,
	})
	if err != nil {
		return err
	}
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
				d.Data.Bright = Ptr(intValue)
			case Ct:
				d.Data.Ct = Ptr(intValue)
			case RGB:
				d.Data.RGB = Ptr(intValue)
			case Hue:
				d.Data.Hue = Ptr(intValue)
			case Sat:
				d.Data.Sat = Ptr(intValue)
			case ColorMode:
				d.Data.ColorMode = Ptr(intValue)
			case Flowing:
				d.Data.Flowing = Ptr(intValue)
			case DelayOff:
				d.Data.DelayOff = Ptr(intValue)
			case MusicOn:
				d.Data.MusicOn = Ptr(intValue)
			case BgFlowing:
				d.Data.BgFlowing = Ptr(intValue)
			case BgCt:
				d.Data.BgCt = Ptr(intValue)
			case BgLMode:
				d.Data.BgLMode = Ptr(intValue)
			case BgBright:
				d.Data.BgBright = Ptr(intValue)
			case BgRGB:
				d.Data.BgRGB = Ptr(intValue)
			case BgSat:
				d.Data.BgSat = Ptr(intValue)
			case NlBr:
				d.Data.NLBr = Ptr(intValue)
			case ActiveMode:
				d.Data.ActiveMode = Ptr(intValue)
			case BGProact:
				d.Data.BGProact = Ptr(intValue)
			}
		default:
			value = resp.Result[i]
			switch prop {
			case Power:
				d.Data.Power = Ptr(value)
			case MainPower:
				d.Data.MainPower = Ptr(value)
			case FlowParams:
				d.Data.FlowParams = Ptr(value)
			case Name:
				d.Data.Name = Ptr(value)
			case BgPower:
				d.Data.BgPower = Ptr(value)
			case BgFlowParams:
				d.Data.BgFlowParams = Ptr(value)
			}
		}
	}
	return nil
}
