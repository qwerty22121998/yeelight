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
	tray            *widgets.QSystemTrayIcon
	firewallTried   bool
}

func NewYeelightUI(root *widgets.QMainWindow) *UI {
	ui := &UI{
		setting: &Setting{
			DiscoverConfig: &yeelight.DiscoverConfig{},
			Effect:         yeelight.EffectSmooth,
			EffectDuration: 500,
			Sync:           map[string]*DeviceSync{},
		},
		root: root,
	}
	ui.setting.DiscoverConfig.Sanitize()
	ui.setting.Load()

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

	// Found nothing and haven't touched the firewall yet: the inbound reply
	// port may be blocked. Open it (this is what triggers the auth prompt) and
	// rescan once. When devices are already reachable we never prompt.
	if len(devices) == 0 && !ui.firewallTried {
		ui.firewallTried = true
		yeelight.EnsureFirewallPort(ctx, ui.setting.DiscoverConfig.ListenPort)
		go ui.loadingTimeout(ctx, ui.setting.DiscoverConfig.Timeout)
		devices, _ = yeelight.Discover(ctx, ui.setting.DiscoverConfig)
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

	hideBtn := widgets.NewQPushButton2("Hide to Tray", nil)
	hideBtn.ConnectClicked(func(checked bool) {
		ui.root.Hide()
	})

	loading := widgets.NewQProgressBar(nil)
	ui.loadingProgress = loading

	layout.AddWidget2(scanDeviceBtn, 0, 0, 0)
	layout.AddWidget2(hideBtn, 0, 1, 0)
	layout.AddWidget2(loading, 0, 2, 0)
	return widget
}

// initTray adds a system-tray icon so the window can be hidden and restored.
// QSystemTrayIcon works on X11 (XEmbed) and Wayland (StatusNotifierItem over
// DBus). ponytail: on Wayland it needs an SNI host running (e.g. waybar's tray
// module on Hyprland); without one the icon won't appear — restore via the
// taskbar or relaunch.
func (ui *UI) initTray(root *widgets.QMainWindow) {
	if !widgets.QSystemTrayIcon_IsSystemTrayAvailable() {
		slog.Warn("system tray unavailable; need an SNI host (e.g. waybar tray) on Wayland")
	}
	icon := widgets.QApplication_Style().StandardIcon(widgets.QStyle__SP_ComputerIcon, nil, nil)
	tray := widgets.NewQSystemTrayIcon2(icon, root)
	tray.SetToolTip("Yeelight")

	restore := func() {
		root.ShowNormal()
		root.Raise()
		root.ActivateWindow()
	}

	menu := widgets.NewQMenu(nil)
	menu.AddAction("Show").ConnectTriggered(func(bool) { restore() })
	menu.AddAction("Quit").ConnectTriggered(func(bool) { core.QCoreApplication_Exit(0) })
	tray.SetContextMenu(menu)

	tray.ConnectActivated(func(reason widgets.QSystemTrayIcon__ActivationReason) {
		if reason == widgets.QSystemTrayIcon__Trigger || reason == widgets.QSystemTrayIcon__DoubleClick {
			restore()
		}
	})
	tray.Show()
	ui.tray = tray
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

	ui.initTray(root)
}
