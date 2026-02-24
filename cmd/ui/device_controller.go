package main

import (
	"context"
	"time"
	"yeelight/pkg/yeelight"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

const updateInterval = time.Second / 2

type DeviceUI struct {
	*widgets.QWidget
	device       *yeelight.Device
	setting      *Setting
	layout       *widgets.QGridLayout
	mainLight    *lightUI
	ambientLight *lightUI
	info         *infoUI
}

type lightUI struct {
	Widget           *widgets.QGroupBox
	Layout           *widgets.QFormLayout
	PowerBtn         *widgets.QPushButton
	BrightnessSlider *widgets.QSlider
	CTSlider         *widgets.QSlider
	ColorDialog      *widgets.QColorDialog
	ColorBtn         *widgets.QPushButton
}

type infoUI struct {
	Widget *widgets.QGroupBox
	Layout *widgets.QFormLayout
}

func NewDeviceUI(ctx context.Context, device *yeelight.Device, setting *Setting) *DeviceUI {
	ui := &DeviceUI{
		device:  device,
		setting: setting,
	}

	ui.QWidget = widgets.NewQWidget(nil, 0)
	ui.layout = widgets.NewQGridLayout2()
	ui.QWidget.SetLayout(ui.layout)

	ui.renderMainLight(ctx)
	ui.renderAmbientLight(ctx)
	ui.renderInfo(ctx)
	ui.layout.AddWidget2(ui.mainLight.Widget, 0, 0, 0)
	ui.layout.AddWidget2(ui.ambientLight.Widget, 1, 0, 0)
	ui.layout.AddWidget3(ui.info.Widget, 0, 1, 2, 1, 0)

	ui.update()
	updated := device.Updated()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-updated:
				ui.update()
			}
		}
	}()

	return ui
}

func (d *DeviceUI) update() {
	if allNotNil(d.mainLight.BrightnessSlider, d.device.Data.Bright) {
		d.mainLight.BrightnessSlider.SetValue(*d.device.Data.Bright)
	}
	if allNotNil(d.mainLight.CTSlider, d.device.Data.Ct) {
		d.mainLight.CTSlider.SetValue(*d.device.Data.Ct)
	}

	if allNotNil(d.mainLight.ColorBtn, d.device.Data.RGB) {
		d.mainLight.ColorBtn.SetStyleSheet("background-color: " + colorIntToRGB(*d.device.Data.RGB))
	}

	if allNotNil(d.ambientLight.BrightnessSlider, d.device.Data.BgBright) {
		d.ambientLight.BrightnessSlider.SetValue(*d.device.Data.BgBright)
	}

	if allNotNil(d.ambientLight.CTSlider, d.device.Data.BgCt) {
		d.ambientLight.CTSlider.SetValue(*d.device.Data.BgCt)
	}

	if allNotNil(d.ambientLight.ColorBtn, d.device.Data.BgRGB) {
		d.ambientLight.ColorBtn.SetStyleSheet("background-color: " + colorIntToRGB(*d.device.Data.BgRGB))
	}
}

func (d *DeviceUI) renderInfo(ctx context.Context) {
	info := &infoUI{}

	info.Widget = widgets.NewQGroupBox2("Info", nil)
	sa := widgets.NewQScrollArea(nil)
	sa.SetWidgetResizable(true)
	info.Widget.Layout().AddWidget(sa)
	info.Layout = widgets.NewQFormLayout(nil)
	info.Widget.SetLayout(info.Layout)

	info.Layout.AddRow3("IP:", widgets.NewQLabel2(d.device.IP, nil, 0))
	info.Layout.AddRow3("Name:", widgets.NewQLabel2(d.device.Name, nil, 0))
	info.Layout.AddRow3("ID:", widgets.NewQLabel2(d.device.ID, nil, 0))
	info.Layout.AddRow3("Model:", widgets.NewQLabel2(d.device.Model, nil, 0))
	info.Layout.AddRow3("Firmware version:", widgets.NewQLabel2(d.device.FWVersion, nil, 0))

	d.info = info
}

func (d *DeviceUI) renderMainLight(ctx context.Context) {

	mainLight := &lightUI{}

	mainLight.Widget = widgets.NewQGroupBox2("Main Light", nil)
	sa := widgets.NewQScrollArea(nil)
	sa.SetWidgetResizable(true)
	mainLight.Widget.Layout().AddWidget(sa)
	mainLight.Layout = widgets.NewQFormLayout(nil)
	mainLight.Widget.SetLayout(mainLight.Layout)

	if d.device.Methods[yeelight.SetPower] {
		mainLight.PowerBtn = widgets.NewQPushButton2("Toggle", nil)
		mainLight.Layout.AddRow3("Power", mainLight.PowerBtn)

		mainLight.PowerBtn.ConnectClicked(func(_ bool) {
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.Toggle))
			if err != nil {

			}
		})
	}

	if d.device.Methods[yeelight.SetBright] {
		mainLight.BrightnessSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		mainLight.BrightnessSlider.SetRange(1, 100)
		mainLight.BrightnessSlider.ConnectSliderReleased(func() {
			value := mainLight.BrightnessSlider.Value()
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.SetBright, value, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				mainLight.BrightnessSlider.SetValue(*d.device.Data.Bright)
				return
			}
		})

		mainLight.Layout.AddRow3("Brightness", mainLight.BrightnessSlider)
	}

	if d.device.Methods[yeelight.SetCtAbx] {
		mainLight.CTSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		mainLight.CTSlider.SetRange(1700, 6500)
		mainLight.CTSlider.ConnectSliderReleased(func() {
			value := mainLight.CTSlider.Value()
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.SetCtAbx, value, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				mainLight.CTSlider.SetValue(*d.device.Data.Ct)
				return
			}
		})

		mainLight.Layout.AddRow3("Color Temperature", mainLight.CTSlider)
	}

	d.mainLight = mainLight
}

func (d *DeviceUI) renderAmbientLight(ctx context.Context) {
	ambientLight := &lightUI{}
	ambientLight.Widget = widgets.NewQGroupBox2("Ambient Light", nil)
	sa := widgets.NewQScrollArea(nil)
	sa.SetWidgetResizable(true)
	ambientLight.Widget.Layout().AddWidget(sa)
	ambientLight.Layout = widgets.NewQFormLayout(nil)
	ambientLight.Widget.SetLayout(ambientLight.Layout)

	if d.device.Methods[yeelight.BgSetPower] {
		ambientLight.PowerBtn = widgets.NewQPushButton2("Toggle", nil)
		ambientLight.Layout.AddRow3("Power", ambientLight.PowerBtn)

		ambientLight.PowerBtn.ConnectClicked(func(_ bool) {
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgToggle))
			if err != nil {

			}
		})

	}

	if d.device.Methods[yeelight.BgSetBright] {
		ambientLight.BrightnessSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		ambientLight.BrightnessSlider.SetRange(1, 100)
		ambientLight.Layout.AddRow3("Brightness", ambientLight.BrightnessSlider)

		ambientLight.BrightnessSlider.ConnectSliderReleased(func() {
			value := ambientLight.BrightnessSlider.Value()
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgSetBright, value, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				ambientLight.BrightnessSlider.SetValue(*d.device.Data.BgBright)
				return
			}
		})
	}

	if d.device.Methods[yeelight.BgSetCtAbx] {
		ambientLight.CTSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		ambientLight.CTSlider.SetRange(1700, 6500)
		ambientLight.Layout.AddRow3("Color Temperature", ambientLight.CTSlider)

		ambientLight.CTSlider.ConnectSliderReleased(func() {
			value := ambientLight.CTSlider.Value()
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgSetCtAbx, value, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				ambientLight.CTSlider.SetValue(*d.device.Data.BgCt)
				return
			}
		})
	}

	if d.device.Methods[yeelight.BgSetRGB] {
		ambientLight.ColorDialog = widgets.NewQColorDialog(nil)
		ambientLight.ColorDialog.ConnectAccepted(func() {
			color := ambientLight.ColorDialog.CurrentColor()
			r, g, b := color.Red(), color.Green(), color.Blue()
			colorInt := rgbToColorInt(int(r), int(g), int(b))
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgSetRGB, colorInt, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				return
			}

		})

		ambientLight.ColorBtn = widgets.NewQPushButton(nil)
		ambientLight.ColorBtn.ConnectClicked(func(checked bool) {
			ambientLight.ColorDialog.Exec()
		})

		ambientLight.Layout.AddRow3("Color", ambientLight.ColorBtn)
	}

	d.ambientLight = ambientLight
}
