package main

import (
	"context"
	"log/slog"
	"yeelight/pkg/yeelight"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
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
			Theme:          darkTheme(),
			MusicScheme:    defaultMusicScheme,
			MusicFloor:     defaultMusicFloor,
		},
		root: root,
	}
	ui.setting.DiscoverConfig.Sanitize()
	ui.setting.Load()

	return ui
}

// showStatus shows a transient message in the window status bar. Safe from any
// goroutine — it marshals to the GUI thread.
func (ui *UI) showStatus(msg string) {
	runOnUI(func() { ui.root.StatusBar().ShowMessage(msg, 5000) })
}

func (ui *UI) devicesUI() widgets.QWidget_ITF {
	devicesTab := widgets.NewQTabWidget(nil)
	ui.devicesTab = devicesTab
	devicesTab.AddTab(emptyState(), "No devices") // hint until the first scan returns
	return devicesTab
}

// tabLabel prefers the device's friendly name, falling back to its IP.
func tabLabel(d *yeelight.Device) string {
	if d.Name != "" {
		return d.Name
	}
	return d.IP
}

func emptyState() widgets.QWidget_ITF {
	w := widgets.NewQWidget(nil, core.Qt__Widget)
	l := widgets.NewQVBoxLayout()
	w.SetLayout(l)
	l.AddWidget(widgets.NewQLabel2(`No devices found. Click "Scan Device".`, nil, 0), 0, core.Qt__AlignCenter)
	return w
}

// reRenderDevices fetches each device's props (network, slow — runs on the
// caller's goroutine, never the GUI thread) then rebuilds the device tabs on
// the GUI thread.
func (ui *UI) reRenderDevices(ctx context.Context) {
	ready := make([]*yeelight.Device, 0, len(ui.devices))
	for _, device := range ui.devices {
		if err := device.FetchProps(ctx); err != nil {
			slog.Error("Failed to fetch device properties", "ip", device.IP, "error", err)
			continue
		}
		slog.Info("Device found", "ip", device.IP, "model", device.Model)
		ready = append(ready, device)
	}

	runOnUI(func() {
		ui.devicesTab.Clear()
		for _, device := range ready {
			ui.devicesTab.AddTab(NewDeviceUI(ctx, device, ui.setting, ui.showStatus), tabLabel(device))
		}
		if len(ready) == 0 {
			ui.devicesTab.AddTab(emptyState(), "No devices")
		}
	})
}

func (ui *UI) scan(ctx context.Context) {
	runOnUI(func() {
		ui.root.SetDisabled(true)
		ui.loadingProgress.SetRange(0, 0) // 0..0 = indeterminate "busy" animation
	})
	defer runOnUI(func() {
		ui.loadingProgress.SetRange(0, 1) // stop the busy animation
		ui.loadingProgress.SetValue(0)
		ui.root.SetDisabled(false)
	})

	for _, device := range ui.devices {
		device.Close()
	}

	devices, err := yeelight.Discover(ctx, ui.setting.DiscoverConfig)
	if err != nil {
		slog.Error("discovery failed", "error", err)
		ui.showStatus("Scan failed: " + err.Error())
		return
	}

	// Found nothing and haven't touched the firewall yet: the inbound reply
	// port may be blocked. Open it (this is what triggers the auth prompt) and
	// rescan once. When devices are already reachable we never prompt.
	if len(devices) == 0 && !ui.firewallTried {
		ui.firewallTried = true
		yeelight.EnsureFirewallPort(ctx, ui.setting.DiscoverConfig.ListenPort)
		devices, _ = yeelight.Discover(ctx, ui.setting.DiscoverConfig)
	}

	if len(devices) == 0 {
		ui.showStatus("No devices found")
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

	hideBtn := widgets.NewQPushButton2("Hide to Tray", nil)
	hideBtn.ConnectClicked(func(checked bool) {
		ui.root.Hide()
	})

	quitBtn := widgets.NewQPushButton2("Quit", nil)
	quitBtn.ConnectClicked(func(checked bool) {
		core.QCoreApplication_Exit(0)
	})

	loading := widgets.NewQProgressBar(nil)
	loading.SetRange(0, 1)
	ui.loadingProgress = loading

	layout.AddWidget(scanDeviceBtn, 0, 0)
	layout.AddWidget(hideBtn, 0, 0)
	layout.AddWidget(loading, 1, 0) // expands, pushing Quit to the far right
	layout.AddWidget(quitBtn, 0, 0)
	return widget
}

// appIcon paints a generic warm-glow lightbulb for the window and tray. It's a
// drawn placeholder, not the Yeelight brand mark — no asset to ship, no
// trademark to embed. ponytail: swap in gui.NewQIcon5("logo.png") if you want
// real branding.
func appIcon() *gui.QIcon {
	const sz = 64
	pix := gui.NewQPixmap2(core.NewQSize2(sz, sz))
	pix.Fill(gui.NewQColor3(0, 0, 0, 0)) // transparent

	p := gui.NewQPainter2(pix)
	p.SetRenderHint(gui.QPainter__Antialiasing, true)
	p.SetPen3(core.Qt__NoPen)

	// soft glow halo
	p.SetBrush(gui.NewQBrush3(gui.NewQColor3(255, 200, 70, 60), core.Qt__SolidPattern))
	p.DrawEllipse3(6, 4, 52, 52)
	// bulb glass
	p.SetBrush(gui.NewQBrush3(gui.NewQColor3(255, 196, 0, 255), core.Qt__SolidPattern))
	p.DrawEllipse3(18, 8, 28, 28)
	// screw base
	p.SetBrush(gui.NewQBrush3(gui.NewQColor3(120, 120, 130, 255), core.Qt__SolidPattern))
	p.DrawRoundedRect2(25, 36, 14, 18, 3, 3, core.Qt__AbsoluteSize)

	p.End()
	return gui.NewQIcon2(pix)
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
	tray := widgets.NewQSystemTrayIcon2(appIcon(), root)
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
	root.SetWindowIcon(appIcon())
	widget := widgets.NewQWidget(root, core.Qt__Widget)
	layout := widgets.NewQVBoxLayout()
	widget.SetLayout(layout)
	functionTab := widgets.NewQTabWidget(widget)
	functionTab.SetTabPosition(widgets.QTabWidget__West)
	functionTab.AddTab(ui.devicesUI(), "Devices")
	functionTab.AddTab(NewSettingUI(ui.setting), "Settings")
	functionTab.AddTab(logUI(), "Log")

	layout.AddWidget(functionTab, 1, 0)
	layout.AddWidget(ui.functionBtnUI(ctx), 0, core.Qt__AlignBottom)

	root.SetCentralWidget(widget)

	ui.initTray(root)

	// Closing the window hides to the tray (app keeps running) when a tray is
	// available; otherwise let the close quit normally.
	root.ConnectCloseEvent(func(e *gui.QCloseEvent) {
		if widgets.QSystemTrayIcon_IsSystemTrayAvailable() {
			root.Hide()
			e.Ignore()
			return
		}
		e.Accept()
	})

	go ui.scan(ctx) // auto-scan once on launch
}
