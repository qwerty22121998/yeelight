package audio

import (
	"encoding/binary"
	"math"
	"testing"
)

// s16le builds a mono s16le buffer from float samples in [-1,1].
func s16le(samples []float64) []byte {
	buf := make([]byte, len(samples)*2)
	for i, f := range samples {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(int16(f*32767)))
	}
	return buf
}

func TestAnalyzeSilenceIsDark(t *testing.T) {
	tick, _ := analyze(s16le(make([]float64, 1024)), 1e-4)
	if tick.Level != 0 {
		t.Fatalf("silence Level: want 0, got %v", tick.Level)
	}
}

func TestAnalyzeLoudIsBright(t *testing.T) {
	// full-scale square wave => rms ~= 1, should peg Level near 1.
	sq := make([]float64, 1024)
	for i := range sq {
		if i%2 == 0 {
			sq[i] = 1
		} else {
			sq[i] = -1
		}
	}
	tick, _ := analyze(s16le(sq), 1e-4)
	if tick.Level < 0.99 {
		t.Fatalf("loud Level: want ~1, got %v", tick.Level)
	}
}

func TestAnalyzeToneTracksFrequency(t *testing.T) {
	// A fast-alternating signal (high ZCR) must read as more treble than a
	// slow sine (low ZCR).
	fast := make([]float64, 1024)
	for i := range fast {
		if i%2 == 0 {
			fast[i] = 0.5
		} else {
			fast[i] = -0.5
		}
	}
	slow := make([]float64, 1024)
	for i := range slow {
		slow[i] = 0.5 * math.Sin(float64(i)/64) // ~8 crossings over the window
	}
	hi, _ := analyze(s16le(fast), 1e-4)
	lo, _ := analyze(s16le(slow), 1e-4)
	if hi.Tone <= lo.Tone {
		t.Fatalf("tone: fast (%v) should exceed slow (%v)", hi.Tone, lo.Tone)
	}
}
