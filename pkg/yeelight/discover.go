package yeelight

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
)

const DefaultSPSDAddress = "239.255.255.250:1982"
const DefaultTimeout = 2 * time.Second

// DefaultListenPort is the fixed local UDP port bound for discovery so SSDP
// unicast replies arrive on a stable port that can be allowed in the firewall.
const DefaultListenPort = 19820

type DiscoverConfig struct {
	SSDPAddress string
	Timeout     time.Duration
	ListenPort  int
}

func (d *DiscoverConfig) Sanitize() {
	if d.SSDPAddress == "" {
		d.SSDPAddress = DefaultSPSDAddress
	}
	if d.Timeout == 0 {
		d.Timeout = DefaultTimeout
	}
	if d.ListenPort == 0 {
		d.ListenPort = DefaultListenPort
	}
}

func (d *DiscoverConfig) message() string {
	return fmt.Sprintf("M-SEARCH * HTTP/1.1\r\nHOST: %s\r\nMAN: \"ssdp:discover\"\r\nST: wifi_bulb", d.SSDPAddress)
}

func parseResponse(ip string, data []byte) (*Info, error) {
	prop := &Info{
		IP: ip,
	}
	lines := strings.Split(string(data), "\r\n")
	for _, line := range lines {
		key, value, _ := strings.Cut(line, ": ")
		switch key {
		case "id":
			prop.ID = value
		case "model":
			prop.Model = value
		case "support":
			methods := strings.Split(value, " ")
			prop.Methods = make(map[Method]bool, len(methods))
			for _, method := range methods {
				prop.Methods[Method(method)] = true
			}
		case "fw_ver":
			prop.FWVersion = value
		case "name":
			prop.Name = value
		}
	}
	return prop, nil
}

func Discover(ctx context.Context, conf *DiscoverConfig) ([]*Device, error) {
	if conf == nil {
		conf = &DiscoverConfig{}
	}
	conf.Sanitize()
	ssdpAddr, err := net.ResolveUDPAddr("udp", conf.SSDPAddress)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: conf.ListenPort})
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(conf.Timeout)); err != nil {
		return nil, err
	}
	if err := conn.SetReadDeadline(time.Now().Add(conf.Timeout)); err != nil {
		return nil, err
	}
	_, err = conn.WriteToUDP([]byte(conf.message()), ssdpAddr)
	if err != nil {
		return nil, err
	}
	buffer := make([]byte, 2048)
	propMp := make(map[string]*Info)
	for {
		_, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			var opErr *net.OpError
			if errors.As(err, &opErr) {
				if opErr.Timeout() {
					break
				}
				return nil, err
			}
			return nil, err
		}
		slog.InfoContext(ctx, "response received", "ip", addr.IP.String(), "data", string(buffer))
		props, err := parseResponse(addr.IP.String(), buffer)
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse response", "ip", addr.IP.String(), "error", err)
			continue
		}
		if _, ok := propMp[props.IP]; ok {
			continue
		}
		propMp[props.IP] = props
		slog.Info("discovered", "ip", props.IP, "model", props.Model)
	}
	var devices []*Device
	for _, prop := range propMp {
		device, err := New(ctx, prop)
		if err != nil {
			slog.ErrorContext(ctx, "failed to create device", "ip", prop.IP, "error", err)
			continue
		}
		devices = append(devices, device)
	}
	slog.InfoContext(ctx, "discovery completed", "count", len(devices))
	return devices, nil
}
