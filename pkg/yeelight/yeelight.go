package yeelight

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"
)

const maxId = 1 << 10
const respWaitTimeout = 5 * time.Second

type Props struct {
	Power        string `json:"power"`
	Bright       int    `json:"bright"`
	Ct           int    `json:"ct"`
	RGB          int    `json:"rgb"`
	Hue          int    `json:"hue"`
	Sat          int    `json:"sat"`
	ColorMode    int    `json:"color_mode"`
	Flowing      int    `json:"flowing"`
	DelayOff     int    `json:"delayoff"`
	FlowParams   string `json:"flow_params"`
	MusicOn      int    `json:"music_on"`
	Name         string `json:"name"`
	BgPower      string `json:"bg_power"`
	BgFlowing    int    `json:"bg_flowing"`
	BgFlowParams string `json:"bg_flow_params"`
	BgCt         int    `json:"bg_ct"`
	BgLMode      int    `json:"bg_lmode"`
	BgBright     int    `json:"bg_bright"`
	BgRGB        int    `json:"bg_rgb"`
	BgSat        int    `json:"bg_sat"`
	NLBr         int    `json:"nl_br"`
	ActiveMode   int    `json:"active_mode"`
}
type Info struct {
	IP      string          `json:"ip"`
	Model   string          `json:"model"`
	ID      string          `json:"id"`
	Methods map[Method]bool `json:"methods"`
	Props   Props           `json:"props"`
}
type Yeelight struct {
	*Info
	con                 net.Conn
	notificationHandler *notificationHandler
	logger              *slog.Logger
	props               map[Property]interface{}
}

// New establishes a TCP connection to the Yeelight device at the specified IP address and initializes a Yeelight instance. It also starts a goroutine to listen for responses from the device.
func New(ctx context.Context, props *Info) (*Yeelight, error) {
	con, err := net.Dial("tcp", props.IP+":55443")
	if err != nil {
		return nil, err
	}
	d := &Yeelight{
		Info:                props,
		con:                 con,
		notificationHandler: newNotifier(),
		logger:              slog.Default(),
	}
	go d.receiveResponse(ctx)
	return d, nil
}

// receiveResponse continuously listens for responses from the Yeelight device. It reads data from the TCP connection, parses it as JSON, and pushes the responses to the notification handler. The function runs until the context is canceled, at which point it closes the connection and stops listening for responses.
func (d *Yeelight) receiveResponse(ctx context.Context) {
	defer d.con.Close()
	reader := bufio.NewReader(d.con)
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Stopping response receiver")
			return
		default:
			data, err := reader.ReadString('\n')
			if err != nil {
				slog.ErrorContext(ctx, "Error reading response", "error", err)
				continue
			}
			fmt.Println(data)
			var resp Response
			err = json.Unmarshal([]byte(data), &resp)
			if err != nil {
				slog.ErrorContext(ctx, "Error parsing response", "error", err)
				continue
			}
			slog.InfoContext(ctx, "Received response", "response", resp)
			d.notificationHandler.push(resp)
		}
	}
}

func (d *Yeelight) FetchProps(ctx context.Context) error {
	resp, err := d.SendCommand(ctx, Command{
		Method: "get_prop",
		Params: AllProperties,
	})
	if err != nil {
		return err
	}
	for i, prop := range AllProperties {
		value := resp.Result[i]
		if value == "" {
			continue
		}
		switch prop {
		case Bright, Ct, RGB, Hue, Sat, ColorMode, Flowing, DelayOff, MusicOn, BgFlowing, BgCt, BgLMode, BgBright, BgRGB, BgSat, NlBr, ActiveMode:
			intValue, err := strconv.ParseInt(fmt.Sprint(resp.Result[i]), 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing property %s: %w", prop, err)
			}
			switch prop {
			case Bright:
				d.Props.Bright = int(intValue)
			case Ct:
				d.Props.Ct = int(intValue)
			case RGB:
				d.Props.RGB = int(intValue)
			case Hue:
				d.Props.Hue = int(intValue)
			case Sat:
				d.Props.Sat = int(intValue)
			case ColorMode:
				d.Props.ColorMode = int(intValue)
			case Flowing:
				d.Props.Flowing = int(intValue)
			case DelayOff:
				d.Props.DelayOff = int(intValue)
			case MusicOn:
				d.Props.MusicOn = int(intValue)
			case BgFlowing:
				d.Props.BgFlowing = int(intValue)
			case BgCt:
				d.Props.BgCt = int(intValue)
			case BgLMode:
				d.Props.BgLMode = int(intValue)
			case BgBright:
				d.Props.BgBright = int(intValue)
			case BgRGB:
				d.Props.BgRGB = int(intValue)
			case BgSat:
				d.Props.BgSat = int(intValue)
			case NlBr:
				d.Props.NLBr = int(intValue)
			case ActiveMode:
				d.Props.ActiveMode = int(intValue)
			}
		default:
			strValue, ok := resp.Result[i].(string)
			if !ok {
				return fmt.Errorf("error parsing property %s: expected string, got %T", prop, resp.Result[i])
			}
			switch prop {
			case Power:
				d.Props.Power = strValue
			case FlowParams:
				d.Props.FlowParams = strValue
			case Name:
				d.Props.Name = strValue
			case BgPower:
				d.Props.BgPower = strValue
			case BgFlowParams:
				d.Props.BgFlowParams = strValue
			}
		}
	}
	d.logger.Info("props", "props", d.Props)
	return nil
}

// SendCommand sends a command to the Yeelight device and waits for the corresponding response. It generates a unique command ID if one is not provided, marshals the command into JSON format, and writes it to the TCP connection. The function then waits for a response with the same command ID using the notification handler. If a response is received within the timeout period, it returns the response; otherwise, it returns an error indicating a timeout.
func (d *Yeelight) SendCommand(ctx context.Context, cmd Command) (*Response, error) {
	//if ok := d.Methods[cmd.Method]; !ok {
	//	return nil, fmt.Errorf("unsupported method: %s", cmd.Method)
	//}
	if cmd.ID == 0 {
		cmd.ID = d.notificationHandler.genID()
	}
	if len(cmd.Params) == 0 {
		cmd.Params = []any{}
	}
	jsonData, err := json.Marshal(cmd)
	d.logger.Info("Send commmand", "command", jsonData)
	if err != nil {
		return nil, err
	}
	jsonData = append(jsonData, '\r', '\n')
	_, err = d.con.Write(jsonData)
	if err != nil {
		return nil, err
	}
	resp, ok := d.notificationHandler.wait(ctx, cmd.ID)
	if !ok {
		return nil, fmt.Errorf("response timeout for command ID %d", cmd.ID)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp, nil
}
