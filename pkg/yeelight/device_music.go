package yeelight

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Music is an active Yeelight "music mode" session. In music mode the bulb
// opens a TCP connection back to us and accepts commands over it WITHOUT the
// normal ~60-commands-per-minute rate limit — required for music/screen
// visualization where we push dozens of updates per second. Send writes
// fire-and-forget (the bulb does not reply on the music channel).
type Music struct {
	dev  *Device
	ln   net.Listener
	conn net.Conn
}

// StartMusic puts the device into music mode: it opens a local TCP listener,
// tells the bulb (via the control connection) to connect back to it, and
// returns once the bulb has connected. The advertised support list is not
// consulted; a bulb that genuinely lacks music mode returns an IsUnsupported
// error from the set_music call.
func (d *Device) StartMusic(ctx context.Context) (*Music, error) {
	// The local IP the bulb can reach us on = the local side of our existing
	// connection to the bulb (same subnet, correct interface).
	host, _, err := net.SplitHostPort(d.con.LocalAddr().String())
	if err != nil {
		return nil, fmt.Errorf("resolve local address: %w", err)
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return nil, err
	}
	port := ln.Addr().(*net.TCPAddr).Port

	// Tell the bulb to connect back to host:port (action 1 = on). SendCommand
	// does not trust the advertised support list, so this reaches bulbs that
	// support music mode but omit set_music from it; bulbs that truly lack it
	// reply with an IsUnsupported error.
	if _, err := d.SendCommand(ctx, C(SetMusic, 1, host, port)); err != nil {
		ln.Close()
		return nil, err
	}

	ln.(*net.TCPListener).SetDeadline(time.Now().Add(respWaitTimeout))
	conn, err := ln.Accept()
	if err != nil {
		ln.Close()
		return nil, fmt.Errorf("bulb did not connect for music mode: %w", err)
	}
	return &Music{dev: d, ln: ln, conn: conn}, nil
}

// Send writes one command over the music connection. It does not wait for a
// reply (music mode is one-way), so it is safe to call at a high rate.
func (m *Music) Send(cmd Command) error {
	cmd.ID = m.dev.genID()
	if cmd.Params == nil {
		cmd.Params = []any{}
	}
	b, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	b = append(b, '\r', '\n')
	_, err = m.conn.Write(b)
	return err
}

// Stop closes the music connection and tells the bulb to leave music mode.
func (m *Music) Stop(ctx context.Context) error {
	m.conn.Close()
	m.ln.Close()
	_, err := m.dev.SendCommand(ctx, C(SetMusic, 0))
	return err
}
