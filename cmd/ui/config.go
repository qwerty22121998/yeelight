package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"yeelight/pkg/yeelight"

	"github.com/BurntSushi/toml"
)

const configDirName = "yeelight"
const configFileName = "config.toml"

// persistedSetting is the on-disk TOML shape. It is kept separate from Setting
// so durations serialize as plain millisecond ints rather than nanoseconds.
type persistedSetting struct {
	Effect         string              `toml:"effect"`
	EffectDuration int                 `toml:"effect_duration_ms"`
	Discover       persistDiscover     `toml:"discover"`
	DeviceSync     []persistDeviceSync `toml:"device_sync"`
	Style          string              `toml:"style"`
	Theme          persistTheme        `toml:"theme"`
	MusicScheme    string              `toml:"music_scheme"`
	MusicFloor     *float64            `toml:"music_floor"` // pointer: nil = absent (keep default); 0 is a valid floor
}

type persistTheme struct {
	Background string `toml:"background"`
	Surface    string `toml:"surface"`
	Text       string `toml:"text"`
	Accent     string `toml:"accent"`
	Border     string `toml:"border"`
}

type persistDeviceSync struct {
	ID          string `toml:"id"`
	Enabled     bool   `toml:"enabled"`
	ScreenIndex int    `toml:"screen_index"`
	IntervalMS  int    `toml:"interval_ms"`
}

type persistDiscover struct {
	SSDPAddress string `toml:"ssdp_address"`
	TimeoutMS   int    `toml:"timeout_ms"`
	ListenPort  int    `toml:"listen_port"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configDirName, configFileName), nil
}

func (s *Setting) toPersisted() persistedSetting {
	sync := make([]persistDeviceSync, 0, len(s.Sync))
	for id, c := range s.Sync {
		sync = append(sync, persistDeviceSync{
			ID:          id,
			Enabled:     c.Enabled,
			ScreenIndex: c.ScreenIndex,
			IntervalMS:  c.Interval,
		})
	}
	return persistedSetting{
		Effect:         string(s.Effect),
		EffectDuration: s.EffectDuration,
		DeviceSync:     sync,
		Discover: persistDiscover{
			SSDPAddress: s.DiscoverConfig.SSDPAddress,
			TimeoutMS:   int(s.DiscoverConfig.Timeout / time.Millisecond),
			ListenPort:  s.DiscoverConfig.ListenPort,
		},
		Style:       s.Style,
		Theme:       persistTheme(s.Theme),
		MusicScheme: s.MusicScheme,
		MusicFloor:  &s.MusicFloor,
	}
}

func (s *Setting) applyPersisted(p persistedSetting) {
	if p.Effect != "" {
		s.Effect = yeelight.Effect(p.Effect)
	}
	if p.EffectDuration != 0 {
		s.EffectDuration = p.EffectDuration
	}
	s.Sync = make(map[string]*DeviceSync, len(p.DeviceSync))
	for _, c := range p.DeviceSync {
		interval := c.IntervalMS
		if interval == 0 {
			interval = defaultSyncInterval
		}
		s.Sync[c.ID] = &DeviceSync{
			Enabled:     c.Enabled,
			ScreenIndex: c.ScreenIndex,
			Interval:    interval,
		}
	}
	if p.Discover.SSDPAddress != "" {
		s.DiscoverConfig.SSDPAddress = p.Discover.SSDPAddress
	}
	if p.Discover.TimeoutMS != 0 {
		s.DiscoverConfig.Timeout = time.Duration(p.Discover.TimeoutMS) * time.Millisecond
	}
	if p.Discover.ListenPort != 0 {
		s.DiscoverConfig.ListenPort = p.Discover.ListenPort
	}
	s.DiscoverConfig.Sanitize()

	if p.Style != "" {
		s.Style = p.Style
	}

	if p.MusicScheme != "" {
		s.MusicScheme = p.MusicScheme
	}

	if p.MusicFloor != nil {
		s.MusicFloor = *p.MusicFloor
	}

	// Overlay only the colors actually present on disk; s.Theme starts at
	// darkTheme() so any field omitted from an old/partial config keeps its
	// default rather than going empty.
	for _, f := range []struct {
		src string
		dst *string
	}{
		{p.Theme.Background, &s.Theme.Background},
		{p.Theme.Surface, &s.Theme.Surface},
		{p.Theme.Text, &s.Theme.Text},
		{p.Theme.Accent, &s.Theme.Accent},
		{p.Theme.Border, &s.Theme.Border},
	} {
		if f.src != "" {
			*f.dst = f.src
		}
	}
}

// Load overlays persisted config from disk onto s. A missing file is not an error.
func (s *Setting) Load() {
	path, err := configPath()
	if err != nil {
		slog.Warn("config path unavailable", "error", err)
		return
	}
	var p persistedSetting
	if _, err := toml.DecodeFile(path, &p); err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("failed to read config", "path", path, "error", err)
		}
		return
	}
	s.applyPersisted(p)
}

// Save writes s to disk as TOML. Best-effort: errors are logged, not returned.
func (s *Setting) Save() {
	path, err := configPath()
	if err != nil {
		slog.Warn("config path unavailable", "error", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		slog.Warn("failed to create config dir", "error", err)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		slog.Warn("failed to write config", "path", path, "error", err)
		return
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(s.toPersisted()); err != nil {
		slog.Warn("failed to encode config", "error", err)
	}
}
