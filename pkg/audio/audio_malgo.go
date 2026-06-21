//go:build windows || darwin

package audio

import (
	"context"
	"fmt"
	"runtime"

	"github.com/gen2brain/malgo"
)

// Capture records the system audio output and returns a channel of Ticks via
// the miniaudio (malgo) backend. The channel closes when ctx is cancelled.
//
//   - Windows: WASAPI loopback captures the default output device directly — no
//     setup needed.
//   - macOS: CoreAudio has no loopback, so this captures the default *input*
//     device. To feed it real system audio, install the BlackHole virtual
//     device and route output to it (a multi-output device keeps your speakers
//     live). Without that it captures the mic (or silence).
func Capture(ctx context.Context) (<-chan Tick, error) {
	mctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("audio init context: %w", err)
	}

	devType := malgo.Loopback // Windows: grab system output directly
	if runtime.GOOS == "darwin" {
		devType = malgo.Capture // no CoreAudio loopback; needs BlackHole input
	}
	cfg := malgo.DefaultDeviceConfig(devType)
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = 1
	cfg.SampleRate = sampleRate

	// raw carries copied capture buffers off miniaudio's realtime audio thread.
	// The callback must never block, so a full channel drops samples rather than
	// stall audio — analysis falling behind is better than glitching playback.
	raw := make(chan []byte, 8)
	onData := func(_, in []byte, _ uint32) {
		// ponytail: alloc+copy per callback (in is reused after we return). ~100
		// small allocs/sec — trivial GC pressure, Go STW is sub-ms. Pool only if
		// profiling ever shows it matters.
		b := make([]byte, len(in))
		copy(b, in)
		select {
		case raw <- b:
		default:
		}
	}

	device, err := malgo.InitDevice(mctx.Context, cfg, malgo.DeviceCallbacks{Data: onData})
	if err != nil {
		mctx.Uninit()
		mctx.Free()
		return nil, fmt.Errorf("audio init device: %w", err)
	}
	if err := device.Start(); err != nil {
		device.Uninit()
		mctx.Uninit()
		mctx.Free()
		return nil, fmt.Errorf("audio start: %w", err)
	}

	out := make(chan Tick)
	go func() {
		defer close(out)
		defer mctx.Free()
		defer mctx.Uninit()
		defer device.Uninit() // stops the device too
		buf := make([]byte, 0, chunk*2)
		peak := 1e-4
		for {
			select {
			case <-ctx.Done():
				return
			case b := <-raw:
				buf = append(buf, b...)
				for len(buf) >= chunk*2 { // emit one Tick per full window
					var t Tick
					t, peak = analyze(buf[:chunk*2], peak)
					buf = buf[chunk*2:]
					select {
					case out <- t:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out, nil
}
