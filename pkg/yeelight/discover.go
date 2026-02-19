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
const DefaultTimeout = 5 * time.Second

const discoverMsg = "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1982\r\nMAN: \"ssdp:discover\"\r\nST: wifi_bulb"

type DiscoverConfig struct {
	SSDPAddress string
	Timeout     time.Duration
}

func (d *DiscoverConfig) sanitize() {
	if d.SSDPAddress == "" {
		d.SSDPAddress = DefaultSPSDAddress
	}
	if d.Timeout == 0 {
		d.Timeout = DefaultTimeout
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
		}
	}
	return prop, nil
}

func Discover(ctx context.Context, conf *DiscoverConfig) ([]*Yeelight, error) {
	if conf == nil {
		conf = &DiscoverConfig{}
	}
	conf.sanitize()
	ssdpAddr, err := net.ResolveUDPAddr("udp", conf.SSDPAddress)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", nil)
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
	var devices []*Yeelight
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
