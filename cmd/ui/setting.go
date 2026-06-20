package main

import (
	"time"
	"yeelight/pkg/yeelight"

	"github.com/therecipe/qt/widgets"
)

const defaultSyncInterval = 1000

type Setting struct {
	DiscoverConfig *yeelight.DiscoverConfig
	Effect         yeelight.Effect
	EffectDuration int
	Sync           map[string]*DeviceSync // per-device screen-sync config, keyed by device ID
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
	return ui
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
