package main

import (
	"testing"
	"yeelight/pkg/audio"
)

func rgb(c int) (r, g, b int) { return (c >> 16) & 0xff, (c >> 8) & 0xff, c & 0xff }

func baseSetting() *Setting {
	return &Setting{
		MusicScheme:      defaultMusicScheme,
		MusicFloor:       0,
		MusicMode:        defaultMusicMode,
		MusicSensitivity: defaultMusicSensitivity,
		MusicSaturation:  defaultMusicSaturation,
	}
}

func TestMusicColorStrobe(t *testing.T) {
	s := baseSetting()
	s.MusicMode = "Strobe"
	s.MusicFloor = 0

	// Loud beat -> white.
	if r, g, b := rgb(musicColor(s, audio.Tick{Level: 1}, 0)); r != 255 || g != 255 || b != 255 {
		t.Fatalf("loud strobe = (%d,%d,%d), want white", r, g, b)
	}
	// Quiet -> off (floor 0).
	if r, g, b := rgb(musicColor(s, audio.Tick{Level: 0.1}, 0)); r != 0 || g != 0 || b != 0 {
		t.Fatalf("quiet strobe = (%d,%d,%d), want off", r, g, b)
	}
}

func TestMusicColorSteadyIgnoresTone(t *testing.T) {
	s := baseSetting()
	s.MusicMode = "Steady"
	a := musicColor(s, audio.Tick{Level: 0.5, Tone: 0}, 0)
	b := musicColor(s, audio.Tick{Level: 0.5, Tone: 1}, 0)
	if a != b {
		t.Fatalf("steady color changed with tone: %06x vs %06x", a, b)
	}
}

func TestMusicColorSaturationZeroIsGrey(t *testing.T) {
	s := baseSetting()
	s.MusicSaturation = 0
	r, g, b := rgb(musicColor(s, audio.Tick{Level: 1, Tone: 0.7}, 0))
	if r != g || g != b {
		t.Fatalf("sat 0 not greyscale: (%d,%d,%d)", r, g, b)
	}
}

func TestMusicColorSensitivityBrightens(t *testing.T) {
	s := baseSetting()
	s.MusicMode = "Steady"
	tick := audio.Tick{Level: 0.3}

	sum := func(c int) int { r, g, b := rgb(c); return r + g + b }
	s.MusicSensitivity = 1
	low := sum(musicColor(s, tick, 0))
	s.MusicSensitivity = 2
	hi := sum(musicColor(s, tick, 0))
	if hi <= low {
		t.Fatalf("sensitivity 2x not brighter: %d <= %d", hi, low)
	}
}

func TestMusicColorBeatPulseCyclesHue(t *testing.T) {
	s := baseSetting()
	s.MusicMode = "Beat Pulse"
	s.MusicScheme = "Rainbow" // full wheel so phase clearly moves the hue
	tick := audio.Tick{Level: 1}
	start := musicColor(s, tick, 0)
	mid := musicColor(s, tick, pulseCycleTicks/2)
	if start == mid {
		t.Fatalf("beat pulse hue did not advance over time: %06x", start)
	}
}
