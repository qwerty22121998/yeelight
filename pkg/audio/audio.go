// Package audio captures the system's audio output and reduces it to a stream
// of loudness/tone ticks for driving light effects. Linux + PulseAudio/PipeWire
// only: it shells out to parec on the default sink's monitor, mirroring
// pkg/screen's subprocess approach. There is no FFT — loudness is RMS and
// "tone" is the zero-crossing rate (a cheap bass-vs-treble proxy).
package audio

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"os/exec"
)

const (
	sampleRate = 22050
	chunk      = 2048 // samples per tick (~93ms => ~11 ticks/sec)
)

// Tick is one analysis window of audio.
type Tick struct {
	Level float64 // 0..1 loudness, AGC-normalized (drives brightness)
	Tone  float64 // 0..1 zero-crossing rate, low=bass high=treble (drives hue)
}

// Capture starts recording the default monitor and returns a channel of Ticks.
// The channel closes when ctx is cancelled or the recorder exits. Killing the
// recorder and draining are handled internally.
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

// analyze reduces one s16le mono buffer to a Tick. peak is a decaying running
// maximum carried between calls so loud and quiet tracks both span 0..1 (a
// crude auto-gain). It returns the updated peak.
//
// ponytail: zero-crossing rate is a rough tone proxy, not real spectral
// analysis; swap in an FFT band-split if the hue mapping feels wrong.
func analyze(buf []byte, peak float64) (Tick, float64) {
	n := len(buf) / 2
	if n == 0 {
		return Tick{}, peak
	}
	var sumsq float64
	var crossings int
	prevPos := true
	for i := range n {
		s := int16(binary.LittleEndian.Uint16(buf[i*2:]))
		f := float64(s) / 32768
		sumsq += f * f
		pos := s >= 0
		if i > 0 && pos != prevPos {
			crossings++
		}
		prevPos = pos
	}
	rms := math.Sqrt(sumsq / float64(n))

	peak *= 0.995 // decay so the gain re-adapts after a loud passage
	if rms > peak {
		peak = rms
	}
	if peak < 1e-4 {
		peak = 1e-4
	}

	level := math.Min(rms/peak, 1)
	// ZCR in real music sits well below 0.5; scale up so treble reaches the
	// top of the hue range, then clamp.
	tone := math.Min(float64(crossings)/float64(n)*4, 1)
	return Tick{Level: level, Tone: tone}, peak
}
