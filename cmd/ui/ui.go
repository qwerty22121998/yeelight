package main

import (
	"context"
	"yeelight/pkg/yeelight"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

type UI struct {
	devices       []*yeelight.Device
	setting       *Setting
	devicesTab    *widgets.QTabWidget
	root          *widgets.QMainWindow
	loadingDialog *widgets.QDialog
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

	loadingDialog := widgets.NewQDialog(root, core.Qt__Dialog|core.Qt__WindowStaysOnTopHint)
	loadingDialogLayout := widgets.NewQVBoxLayout()
	loadingDialog.SetLayout(loadingDialogLayout)
	loadingDialog.SetModal(true)
	loadingLabel := widgets.NewQLabel2("Loading...", nil, 0)
	loadingDialogLayout.AddWidget(loadingLabel, 1, core.Qt__AlignCenter)
	ui.loadingDialog = loadingDialog

	return ui
}

func (ui *UI) setLoading(isLoading bool) {
	if isLoading {
		ui.loadingDialog.Show()
		ui.root.SetDisabled(isLoading)
		return
	}
	ui.loadingDialog.Hide()
	ui.root.SetDisabled(isLoading)
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
			continue
		}
		deviceUi := NewDeviceUI(ctx, device, ui.setting)
		ui.devicesTab.AddTab(deviceUi, device.IP)
	}
}

func (ui *UI) scan(ctx context.Context) {
	for _, device := range ui.devices {
		device.Close()
	}
	ui.setLoading(true)
	defer ui.setLoading(false)
	devices, err := yeelight.Discover(ctx, ui.setting.DiscoverConfig)
	if err != nil {
		return
	}
	ui.devices = devices
	ui.reRenderDevices(ctx)
}

func (ui *UI) functionBtnUI(ctx context.Context) widgets.QWidget_ITF {
	widget := widgets.NewQWidget(nil, core.Qt__Widget)
	layout := widgets.NewQHBoxLayout()
	widget.SetLayout(layout)
	scanDeviceBtn := widgets.NewQPushButton2("Scan Device", nil)
	scanDeviceBtn.ConnectClicked(func(checked bool) {
		go ui.scan(ctx)
	})
	layout.AddWidget(scanDeviceBtn, 1, core.Qt__AlignLeft)
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
