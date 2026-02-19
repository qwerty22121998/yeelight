package main

import (
	"context"
	"fmt"
	"time"
	"yeelight/pkg/yeelight"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

type UI struct {
	devices        []*yeelight.Yeelight
	discoverConfig *yeelight.DiscoverConfig
	devicesTab     *widgets.QTabWidget
	root           *widgets.QMainWindow
	loadingDialog  *widgets.QDialog
}

func NewYeelightUI(root *widgets.QMainWindow) *UI {
	ui := &UI{
		discoverConfig: &yeelight.DiscoverConfig{},
		root:           root,
	}
	ui.discoverConfig.Sanitize()

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

func (ui *UI) deviceConfigUI(ctx context.Context, device *yeelight.Yeelight) widgets.QWidget_ITF {
	widget := widgets.NewQWidget(nil, 0)
	layout := widgets.NewQHBoxLayout()
	widget.SetLayout(layout)
	layout.AddWidget(ui.mainLightUI(ctx, device), 1, 0)
	layout.AddWidget(ui.ambientLightUI(device), 1, 0)
	return widget
}
func (ui *UI) mainLightUI(ctx context.Context, device *yeelight.Yeelight) widgets.QWidget_ITF {
	group := widgets.NewQGroupBox2("Main Light", nil)
	groupLayout := widgets.NewQVBoxLayout()
	groupLayout.SetContentsMargins(0, 0, 0, 0)
	group.SetLayout(groupLayout)
	sa := widgets.NewQScrollArea(nil)
	layout := widgets.NewQFormLayout(nil)
	sa.SetLayout(layout)
	groupLayout.AddWidget(sa, 1, 0)
	// power
	if device.Methods[yeelight.SetPower] {
		powerBtn := widgets.NewQPushButton(nil)
		powerBtn.SetCheckable(true)
		powerBtn.SetChecked(device.Props.Power == "on")
		powerBtn.SetText(device.Props.Power)
		powerBtn.ConnectClicked(func(checked bool) {
			power := "off"
			if checked {
				power = "on"
			}
			_, err := device.SendCommand(ctx, yeelight.C(yeelight.SetPower, power))
			if err != nil {
				powerBtn.SetCheckable(!checked)
				return
			}
			powerBtn.SetText(power)

		})
		layout.AddRow3("Power", powerBtn)
	}
	// brightness
	if device.Methods[yeelight.SetBright] {
		brightnessSlider := widgets.NewQSlider2(core.Qt__Horizontal, nil)
		brightnessSlider.SetRange(1, 100)
		brightnessSlider.SetValue(device.Props.Bright)
		brightnessSlider.ConnectSliderReleased(func() {
			value := brightnessSlider.Value()
			_, err := device.SendCommand(ctx, yeelight.C(yeelight.SetBright, value))
			if err != nil {
				brightnessSlider.SetValue(device.Props.Bright)
				return
			}
		})
		layout.AddRow3("Brightness", brightnessSlider)
	}
	// temperature
	if device.Methods[yeelight.SetCtAbx] {
		ctSlider := widgets.NewQSlider2(core.Qt__Horizontal, nil)
		ctSlider.SetRange(1700, 6500)
		ctSlider.SetValue(device.Props.Ct)
		ctSlider.ConnectSliderReleased(func() {
			value := ctSlider.Value()
			_, err := device.SendCommand(ctx, yeelight.C(yeelight.SetCtAbx, value))
			if err != nil {
				ctSlider.SetValue(device.Props.Ct)
				return
			}
		})
		layout.AddRow3("Color Temperature", ctSlider)
	}

	// color
	return group
}

func (ui *UI) ambientLightUI(device *yeelight.Yeelight) widgets.QWidget_ITF {
	group := widgets.NewQGroupBox2("Ambient Light", nil)
	groupLayout := widgets.NewQVBoxLayout()
	groupLayout.SetContentsMargins(0, 0, 0, 0)
	group.SetLayout(groupLayout)
	sa := widgets.NewQScrollArea(nil)
	layout := widgets.NewQFormLayout(nil)
	sa.SetLayout(layout)
	groupLayout.AddWidget(sa, 1, 0)
	// power
	if device.Methods[yeelight.BgSetPower] {
		powerBtn := widgets.NewQPushButton(nil)
		powerBtn.SetCheckable(true)
		powerBtn.SetChecked(device.Props.BgPower == "on")
		powerBtn.SetText(device.Props.BgPower)
		powerBtn.ConnectClicked(func(checked bool) {
			power := "off"
			if checked {
				power = "on"
			}
			_, err := device.SendCommand(context.Background(), yeelight.C(yeelight.BgSetPower, power))
			if err != nil {
				powerBtn.SetCheckable(!checked)
				return
			}
			powerBtn.SetText(power)

		})
		layout.AddRow3("Power", powerBtn)
	}
	// brightness
	if device.Methods[yeelight.BgSetBright] {
		brightnessSlider := widgets.NewQSlider2(core.Qt__Horizontal, nil)
		brightnessSlider.SetRange(1, 100)
		brightnessSlider.SetValue(device.Props.BgBright)
		brightnessSlider.ConnectSliderReleased(func() {
			value := brightnessSlider.Value()
			_, err := device.SendCommand(context.Background(), yeelight.C(yeelight.BgSetBright, value))
			if err != nil {
				brightnessSlider.SetValue(device.Props.BgBright)
				return
			}
		})
		layout.AddRow3("Brightness", brightnessSlider)
	}
	// temperature
	if device.Methods[yeelight.BgSetCtAbx] {
		ctSlider := widgets.NewQSlider2(core.Qt__Horizontal, nil)
		ctSlider.SetRange(1700, 6500)
		ctSlider.SetValue(device.Props.BgCt)
		ctSlider.ConnectSliderReleased(func() {
			value := ctSlider.Value()
			_, err := device.SendCommand(context.Background(), yeelight.C(yeelight.BgSetCtAbx, value))
			if err != nil {
				ctSlider.SetValue(device.Props.BgCt)
				return
			}
		})
		layout.AddRow3("Color Temperature", ctSlider)
	}
	// color
	if device.Methods[yeelight.BgSetRGB] {
		colorDialog := widgets.NewQColorDialog(nil)
		colorBtn := widgets.NewQPushButton2("Set Color", nil)

		updateColorBtnBackground := func() {
			colorBtn.SetStyleSheet(fmt.Sprintf("background-color: rgb(%d, %d, %d)", device.Props.BgRGB>>16&0xFF, device.Props.BgRGB>>8&0xFF, device.Props.BgRGB&0xFF))
		}

		updateColorBtnBackground()
		colorBtn.ConnectClicked(func(checked bool) {
			if colorDialog.Exec() == int(widgets.QDialog__Accepted) {
				color := colorDialog.CurrentColor()
				rgb := (color.Red() << 16) | (color.Green() << 8) | color.Blue()
				_, err := device.SendCommand(context.Background(), yeelight.C(yeelight.BgSetRGB, rgb))
				if err != nil {
					return
				}
				device.Props.BgRGB = rgb
				updateColorBtnBackground()
			}
		})
		layout.AddRow3("Color", colorBtn)
	}

	return group
}

func (ui *UI) devicesUI() widgets.QWidget_ITF {
	devicesTab := widgets.NewQTabWidget(nil)
	devicesTab.SetTabPosition(widgets.QTabWidget__West)
	ui.devicesTab = devicesTab
	return devicesTab
}

func (ui *UI) reRenderDevices(ctx context.Context) {
	ui.devicesTab.Clear()
	for _, device := range ui.devices {
		if err := device.FetchProps(ctx); err != nil {

			continue
		}
		ui.devicesTab.AddTab(ui.deviceConfigUI(ctx, device), device.IP)
	}
}

func (ui *UI) settingsUI() widgets.QWidget_ITF {
	settingsTab := widgets.NewQTabWidget(nil)
	settingsTab.SetTabPosition(widgets.QTabWidget__West)
	//discover
	discoverConfWidget := widgets.NewQWidget(nil, core.Qt__Widget)
	discoverConfLayout := widgets.NewQFormLayout(nil)
	discoverConfWidget.SetLayout(discoverConfLayout)
	discoverConfScroll := widgets.NewQScrollArea(nil)
	discoverConfScroll.SetWidgetResizable(true)
	discoverConfScroll.SetWidget(discoverConfWidget)

	discoverSSDPAddress := widgets.NewQLineEdit2(ui.discoverConfig.SSDPAddress, nil)
	discoverSSDPAddress.ConnectTextChanged(func(text string) {
		ui.discoverConfig.SSDPAddress = text
	})
	discoverTimeout := widgets.NewQSpinBox(nil)
	discoverTimeout.SetMinimum(1)
	discoverTimeout.ConnectValueChanged(func(value int) {
		ui.discoverConfig.Timeout = time.Duration(value) * time.Second
	})
	discoverTimeout.SetValue(int(ui.discoverConfig.Timeout / time.Second))
	discoverConfLayout.AddRow3("SSDP Address", discoverSSDPAddress)
	discoverConfLayout.AddRow3("Timeout (s)", discoverTimeout)

	settingsTab.AddTab(discoverConfScroll, "Discover")

	return settingsTab
}

func (ui *UI) scan(ctx context.Context) {
	for _, device := range ui.devices {
		device.Close()
	}
	ui.setLoading(true)
	defer ui.setLoading(false)
	devices, err := yeelight.Discover(ctx, ui.discoverConfig)
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
	functionTab.AddTab(ui.devicesUI(), "Devices")
	functionTab.AddTab(ui.settingsUI(), "Settings")

	layout.AddWidget(functionTab, 1, 0)
	layout.AddWidget(ui.functionBtnUI(ctx), 0, core.Qt__AlignBottom)

	go ui.scan(ctx)

	root.SetCentralWidget(widget)

}
