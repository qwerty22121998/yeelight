//go:build linux

package audio

import (
	"context"
	"io"
	"os/exec"
)

// Capture starts recording the default monitor and returns a channel of Ticks.
// The channel closes when ctx is cancelled or the recorder exits. Killing the
// recorder and draining are handled internally.
//
// Linux only: it shells out to parec on the default sink's monitor
// (PulseAudio/PipeWire), which is independent of the display server, so it works
// under both Wayland and X11.
func Capture(ctx context.Context) (<-chan Tick, error) {
	cmd := exec.CommandContext(ctx, "parec",
		"--format=s16le", "--rate=22050", "--channels=1", "-d", "@DEFAULT_MONITOR@")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	out := make(chan Tick)
	go func() {
		defer close(out)
		defer cmd.Wait()
		buf := make([]byte, chunk*2) // 2 bytes per s16 sample
		peak := 1e-4
		for {
			if _, err := io.ReadFull(stdout, buf); err != nil {
				return
			}
			var t Tick
			t, peak = analyze(buf, peak)
			select {
			case out <- t:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
