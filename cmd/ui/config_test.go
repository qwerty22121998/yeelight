package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"yeelight/pkg/yeelight"
)

func TestSettingSaveLoadRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	orig := &Setting{
		DiscoverConfig: &yeelight.DiscoverConfig{
			SSDPAddress: "239.255.255.250:1982",
			Timeout:     3 * time.Second,
			ListenPort:  19820,
		},
		Effect:         yeelight.EffectSudden,
		EffectDuration: 750,
	}
	orig.Save()

	if _, err := os.Stat(filepath.Join(tmp, configDirName, configFileName)); err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	loaded := &Setting{DiscoverConfig: &yeelight.DiscoverConfig{}}
	loaded.Load()

	if loaded.Effect != orig.Effect {
		t.Errorf("Effect = %q, want %q", loaded.Effect, orig.Effect)
	}
	if loaded.EffectDuration != orig.EffectDuration {
		t.Errorf("EffectDuration = %d, want %d", loaded.EffectDuration, orig.EffectDuration)
	}
	if loaded.DiscoverConfig.SSDPAddress != orig.DiscoverConfig.SSDPAddress {
		t.Errorf("SSDPAddress = %q, want %q", loaded.DiscoverConfig.SSDPAddress, orig.DiscoverConfig.SSDPAddress)
	}
	if loaded.DiscoverConfig.Timeout != orig.DiscoverConfig.Timeout {
		t.Errorf("Timeout = %v, want %v", loaded.DiscoverConfig.Timeout, orig.DiscoverConfig.Timeout)
	}
	if loaded.DiscoverConfig.ListenPort != orig.DiscoverConfig.ListenPort {
		t.Errorf("ListenPort = %d, want %d", loaded.DiscoverConfig.ListenPort, orig.DiscoverConfig.ListenPort)
	}
}

func TestSettingLoadMissingFileKeepsDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	s := &Setting{
		DiscoverConfig: &yeelight.DiscoverConfig{},
		Effect:         yeelight.EffectSmooth,
		EffectDuration: 500,
	}
	s.DiscoverConfig.Sanitize()
	s.Load() // no file on disk

	if s.Effect != yeelight.EffectSmooth || s.EffectDuration != 500 {
		t.Errorf("defaults clobbered: effect=%q dur=%d", s.Effect, s.EffectDuration)
	}
	if s.DiscoverConfig.ListenPort != yeelight.DefaultListenPort {
		t.Errorf("ListenPort = %d, want default %d", s.DiscoverConfig.ListenPort, yeelight.DefaultListenPort)
	}
}
