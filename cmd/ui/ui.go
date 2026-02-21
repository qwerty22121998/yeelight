package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"yeelight/pkg/yeelight"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

type UI struct {
	devices         []*yeelight.Device
	setting         *Setting
	devicesTab      *widgets.QTabWidget
	root            *widgets.QMainWindow
	loadingProgress *widgets.QProgressBar
}

func NewYeelightUI(root *widgets.QMainWindow) *UI {
	ui := &UI{
		setting: &Setting{
			DiscoverConfig: &yeelight.DiscoverConfig{},
			Effect:         yeelight.EffectSmooth,
			EffectDuration: 500,
		},
		root: root,
	}
	ui.setting.DiscoverConfig.Sanitize()

	return ui
}

func (ui *UI) loadingTimeout(ctx context.Context, dur time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, dur)
	defer cancel()
	ui.root.SetDisabled(true)
	defer ui.root.SetDisabled(false)

	progressStepDur := time.Millisecond * 500
	t := time.NewTicker(progressStepDur)
	fmt.Println("total steps:", int(dur/progressStepDur))
	ui.loadingProgress.SetMaximum(int(dur / progressStepDur))
	step := 1
	ui.loadingProgress.SetValue(step)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			step++
			fmt.Println("step:", step)
			ui.loadingProgress.SetValue(step)
		case <-ctx.Done():
			return
		}
	}
}

func (ui *UI) devicesUI() widgets.QWidget_ITF {
	devicesTab := widgets.NewQTabWidget(nil)
	ui.devicesTab = devicesTab
	return devicesTab
}

func (ui *UI) reRenderDevices(ctx context.Context) {
	ui.devicesTab.Clear()
	for _, device := range ui.devices {
		if err := device.FetchProps(ctx); err != nil {
			slog.Error("Failed to fetch device properties", "ip", device.IP, "error", err)
			continue
		}
		slog.Info("Device found", "ip", device.IP, "model", device.Model)
		deviceUi := NewDeviceUI(ctx, device, ui.setting)
		ui.devicesTab.AddTab(deviceUi, device.IP)
	}
}

func (ui *UI) scan(ctx context.Context) {
	for _, device := range ui.devices {
		device.Close()
	}

	go ui.loadingTimeout(ctx, ui.setting.DiscoverConfig.Timeout)
	devices, err := yeelight.Discover(ctx, ui.setting.DiscoverConfig)
	if err != nil {
		return
	}
	ui.devices = devices
	ui.reRenderDevices(ctx)
}

func (ui *UI) functionBtnUI(ctx context.Context) widgets.QWidget_ITF {
	widget := widgets.NewQWidget(nil, core.Qt__Widget)
	layout := widgets.NewQGridLayout2()
	widget.SetLayout(layout)
	scanDeviceBtn := widgets.NewQPushButton2("Scan Device", nil)
	scanDeviceBtn.ConnectClicked(func(checked bool) {
		go ui.scan(ctx)
	})

	loading := widgets.NewQProgressBar(nil)
	ui.loadingProgress = loading

	layout.AddWidget2(scanDeviceBtn, 0, 0, 0)
	layout.AddWidget2(loading, 0, 1, 0)
	return widget
}

func (ui *UI) RenderMain(ctx context.Context, root *widgets.QMainWindow) {
	ui.root = root
	widget := widgets.NewQWidget(root, core.Qt__Widget)
	layout := widgets.NewQVBoxLayout()
	widget.SetLayout(layout)
	functionTab := widgets.NewQTabWidget(widget)
	functionTab.SetTabPosition(widgets.QTabWidget__West)
	functionTab.AddTab(ui.devicesUI(), "Devices")
	functionTab.AddTab(NewSettingUI(ui.setting), "Settings")

	layout.AddWidget(functionTab, 1, 0)
	layout.AddWidget(ui.functionBtnUI(ctx), 0, core.Qt__AlignBottom)

	root.SetCentralWidget(widget)

}
