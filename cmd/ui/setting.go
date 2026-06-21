package main

import (
	"strings"
	"time"
	"yeelight/pkg/yeelight"

	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

const defaultSyncInterval = 1000

type Setting struct {
	DiscoverConfig *yeelight.DiscoverConfig
	Effect         yeelight.Effect
	EffectDuration int
	Sync           map[string]*DeviceSync // per-device screen-sync config, keyed by device ID
	Style          string                 // Qt widget style key (e.g. "Fusion"); empty = platform default
	Theme          Theme                  // UI color palette
	MusicScheme    string                 // music-sync tone→hue scheme name (see musicSchemes)
	MusicFloor     float64                // music-sync brightness floor (0..1); quiet-passage minimum
}

// DeviceSync is one device's screen-sync config, persisted per device.
type DeviceSync struct {
	Enabled     bool
	ScreenIndex int
	Interval    int // poll interval, ms
}

// SyncFor returns the screen-sync config for device id, creating a default if absent.
func (s *Setting) SyncFor(id string) *DeviceSync {
	if s.Sync == nil {
		s.Sync = map[string]*DeviceSync{}
	}
	cfg := s.Sync[id]
	if cfg == nil {
		cfg = &DeviceSync{Interval: defaultSyncInterval}
		s.Sync[id] = cfg
	}
	return cfg
}

type SettingUI struct {
	*widgets.QTabWidget
	setting *Setting
}

func NewSettingUI(setting *Setting) *SettingUI {
	ui := &SettingUI{
		setting: setting,
	}
	ui.QTabWidget = widgets.NewQTabWidget(nil)
	ui.initDiscover()
	ui.initEffect()
	ui.initAppearance()
	return ui
}

// initAppearance builds the Appearance tab: a Qt-style picker ("Theme"), a
// color-scheme preset picker (Dark/Light), and per-color swatches that the
// preset fills and the user can fine-tune. Every change saves config and
// re-applies the look live.
func (ui *SettingUI) initAppearance() {
	w := widgets.NewQScrollArea(nil)
	w.SetWidgetResizable(true)
	form := widgets.NewQFormLayout(nil)
	w.SetLayout(form)

	// Theme = Qt widget style (Fusion, Windows, …). Empty Style means platform
	// default; the combo shows whichever style is actually active.
	// "Default" (first item) means no style override — Style stays empty and
	// applyAppearance falls back to the platform default.
	styleCombo := widgets.NewQComboBox(nil)
	styleCombo.AddItems(append([]string{"Default"}, availableStyles()...))
	if ui.setting.Style == "" {
		styleCombo.SetCurrentText("Default")
	} else {
		styleCombo.SetCurrentText(ui.setting.Style)
	}
	styleCombo.ConnectCurrentTextChanged(func(name string) {
		if name == "Default" {
			ui.setting.Style = ""
		} else {
			ui.setting.Style = name
		}
		ui.setting.Save()
		applyAppearance(ui.setting)
	})
	form.AddRow3("Theme", styleCombo)

	var paints []func() // repaint every swatch, e.g. after a preset is chosen

	// Color scheme presets. Picking one fills the palette below; editing any
	// single color switches the combo to "Custom".
	schemeCombo := widgets.NewQComboBox(nil)
	schemeCombo.AddItems([]string{"Dark", "Light", "Custom"})
	syncScheme := func() { schemeCombo.SetCurrentText(schemeName(ui.setting.Theme)) }
	form.AddRow3("Color Scheme", schemeCombo)

	addRow := func(label string, ptr *string) {
		btn := widgets.NewQPushButton2("", nil)
		paint := func() { btn.SetText(*ptr); btn.SetStyleSheet("background-color:" + *ptr + "; color:#fff;") }
		paint()
		btn.ConnectClicked(func(bool) {
			if hex, ok := pickColor(*ptr); ok {
				*ptr = hex
				paint()
				syncScheme() // edited color may no longer match a preset
				ui.setting.Save()
				applyAppearance(ui.setting)
			}
		})
		form.AddRow3(label, btn)
		paints = append(paints, paint)
	}

	t := &ui.setting.Theme
	addRow("Background", &t.Background)
	addRow("Surface", &t.Surface)
	addRow("Text", &t.Text)
	addRow("Accent", &t.Accent)
	addRow("Border", &t.Border)

	// ConnectActivated fires only on user selection (not programmatic
	// SetCurrentText), so syncScheme() above never re-triggers this.
	schemeCombo.ConnectActivated(func(index int) {
		switch index {
		case 0:
			ui.setting.Theme = darkTheme()
		case 1:
			ui.setting.Theme = lightTheme()
		default:
			return // "Custom" — leave the current colors as-is
		}
		for _, p := range paints {
			p()
		}
		ui.setting.Save()
		applyAppearance(ui.setting)
	})

	syncScheme() // set the combo to match the loaded palette
	ui.QTabWidget.AddTab(w, "Appearance")
}

// availableStyles lists the Qt styles installed on this system. therecipe's
// QStyleFactory_Keys() binding is broken (panics on a []interface{} → []string
// assertion), so we probe a candidate set with QStyleFactory_Create — a style
// that fails to create comes back as a null QStyle. Fusion is built into Qt and
// always present, so the list is never empty.
func availableStyles() []string {
	candidates := []string{"Fusion", "Windows", "WindowsVista", "macOS", "Macintosh", "Breeze", "Oxygen", "gtk2", "GTK+"}
	var out []string
	for _, c := range candidates {
		if s := widgets.QStyleFactory_Create(c); s != nil && s.Pointer() != nil {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		out = []string{"Fusion"}
	}
	return out
}

// currentStyleKey returns the style key to preselect: the saved one if set,
// else the active style matched case-insensitively against the available keys.
func currentStyleKey(saved string, keys []string) string {
	if saved != "" {
		return saved
	}
	cur := ""
	if qApp != nil && qApp.Style() != nil {
		cur = qApp.Style().ObjectName()
	}
	for _, k := range keys {
		if strings.EqualFold(k, cur) {
			return k
		}
	}
	if len(keys) > 0 {
		return keys[0]
	}
	return ""
}

// schemeName names the preset matching t, or "Custom".
func schemeName(t Theme) string {
	switch t {
	case darkTheme():
		return "Dark"
	case lightTheme():
		return "Light"
	default:
		return "Custom"
	}
}

// pickColor opens a modal color dialog seeded with initialHex and returns the
// chosen "#rrggbb", or ok=false if cancelled.
func pickColor(initialHex string) (hex string, ok bool) {
	c := gui.NewQColor()
	c.SetNamedColor(initialHex)
	res := widgets.QColorDialog_GetColor(c, nil, "Pick color", 0)
	if !res.IsValid() {
		return "", false
	}
	return res.Name(), true
}

func (ui *SettingUI) initDiscover() {
	dConfWidget := widgets.NewQScrollArea(nil)
	dConfWidget.SetWidgetResizable(true)
	dConfLayout := widgets.NewQFormLayout(nil)
	dConfWidget.SetLayout(dConfLayout)

	ssdpAddr := widgets.NewQLineEdit2(ui.setting.DiscoverConfig.SSDPAddress, nil)
	ssdpAddr.ConnectTextChanged(func(text string) {
		ui.setting.DiscoverConfig.SSDPAddress = text
		ui.setting.Save()
	})
	dConfLayout.AddRow3("SSDP Address", ssdpAddr)

	dTimeout := widgets.NewQSpinBox(nil)
	dTimeout.SetMinimum(1)
	dTimeout.SetValue(int(ui.setting.DiscoverConfig.Timeout / time.Second))
	dTimeout.ConnectValueChanged(func(value int) {
		ui.setting.DiscoverConfig.Timeout = time.Duration(value) * time.Second
		ui.setting.Save()
	})
	dConfLayout.AddRow3("Timeout (s)", dTimeout)

	listenPort := widgets.NewQSpinBox(nil)
	listenPort.SetRange(1, 65535)
	listenPort.SetValue(ui.setting.DiscoverConfig.ListenPort)
	listenPort.ConnectValueChanged(func(value int) {
		ui.setting.DiscoverConfig.ListenPort = value
		ui.setting.Save()
	})
	dConfLayout.AddRow3("Listen Port", listenPort)

	ui.QTabWidget.AddTab(dConfWidget, "Discover")
}

func (ui *SettingUI) initEffect() {
	dEffectWidget := widgets.NewQScrollArea(nil)
	dEffectWidget.SetWidgetResizable(true)
	dEffectLayout := widgets.NewQFormLayout(nil)
	dEffectWidget.SetLayout(dEffectLayout)

	effectCombo := widgets.NewQComboBox(nil)
	effectCombo.AddItems([]string{yeelight.EffectSmooth.String(), yeelight.EffectSudden.String()})
	effectCombo.SetCurrentText(string(ui.setting.Effect))
	effectCombo.ConnectCurrentTextChanged(func(text string) {
		ui.setting.Effect = yeelight.Effect(text)
		ui.setting.Save()
	})
	dEffectLayout.AddRow3("Effect", effectCombo)

	durationSpin := widgets.NewQSpinBox(nil)
	durationSpin.SetRange(0, 10000)
	durationSpin.SetValue(ui.setting.EffectDuration)
	durationSpin.ConnectValueChanged(func(value int) {
		ui.setting.EffectDuration = value
		ui.setting.Save()
	})
	dEffectLayout.AddRow3("Effect Duration (ms)", durationSpin)

	ui.QTabWidget.AddTab(dEffectWidget, "Effect")

}
