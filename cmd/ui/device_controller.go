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
	device  *yeelight.Device
	setting *Setting
	layout  *widgets.QHBoxLayout

	mlWidget *widgets.QScrollArea
	mlLayout *widgets.QFormLayout
	alWidget *widgets.QScrollArea
	alLayout *widgets.QFormLayout

	mlPower       *widgets.QPushButton
	mlBrightness  *widgets.QSlider
	mlCt          *widgets.QSlider
	mlColorDialog *widgets.QColorDialog
	mlColor       *widgets.QPushButton

	alPower       *widgets.QPushButton
	alBrightness  *widgets.QSlider
	alCt          *widgets.QSlider
	alColorDialog *widgets.QColorDialog
	alColor       *widgets.QPushButton
}

func NewDeviceUI(ctx context.Context, device *yeelight.Device, setting *Setting) *DeviceUI {
	ui := &DeviceUI{
		device:  device,
		setting: setting,
	}

	ui.QWidget = widgets.NewQWidget(nil, 0)
	ui.layout = widgets.NewQHBoxLayout()
	ui.QWidget.SetLayout(ui.layout)

	ui.initML(ctx)
	ui.initAL(ctx)
	ui.layout.AddWidget(ui.mlWidget, 1, 0)
	ui.layout.AddWidget(ui.alWidget, 1, 0)

	ui.update()
	updated := device.Updated()
	go func() {
		for {
			select {
			case <-updated:
				ui.update()
			}
		}
	}()

	return ui
}

func (d *DeviceUI) update() {
	if d.mlBrightness != nil {
		d.mlBrightness.SetValue(*d.device.Data.Bright)
	}
	if d.mlCt != nil {
		d.mlCt.SetValue(*d.device.Data.Ct)
	}
	if d.mlColor != nil {
		d.mlColor.SetStyleSheet("background-color: " + colorIntToRGB(*d.device.Data.RGB))
	}

	if d.alBrightness != nil {
		d.alBrightness.SetValue(*d.device.Data.BgBright)
	}

	if d.alCt != nil {
		d.alCt.SetValue(*d.device.Data.BgCt)
	}

	if d.alColor != nil {
		d.alColor.SetStyleSheet("background-color: " + colorIntToRGB(*d.device.Data.BgRGB))
	}
}

func (d *DeviceUI) initML(ctx context.Context) {
	d.mlWidget = widgets.NewQScrollArea(nil)
	d.mlLayout = widgets.NewQFormLayout(nil)
	d.mlWidget.SetLayout(d.mlLayout)

	if d.device.Methods[yeelight.SetPower] {
		d.mlPower = widgets.NewQPushButton2("Toggle", nil)
		d.mlLayout.AddRow3("Power", d.mlPower)

		d.mlPower.ConnectClicked(func(_ bool) {
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.Toggle))
			if err != nil {

			}
		})
	}

	if d.device.Methods[yeelight.SetBright] {
		d.mlBrightness = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		d.mlBrightness.SetRange(1, 100)
		d.mlBrightness.ConnectSliderReleased(func() {
			value := d.mlBrightness.Value()
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.SetBright, value, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				d.mlBrightness.SetValue(*d.device.Data.Bright)
				return
			}
		})

		d.mlLayout.AddRow3("Brightness", d.mlBrightness)
	}

	if d.device.Methods[yeelight.SetCtAbx] {
		d.mlCt = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		d.mlCt.SetRange(1700, 6500)
		d.mlCt.ConnectSliderReleased(func() {
			value := d.mlCt.Value()
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.SetCtAbx, value, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				d.mlCt.SetValue(*d.device.Data.Ct)
				return
			}
		})

		d.mlLayout.AddRow3("Color Temperature", d.mlCt)
	}

}

func (d *DeviceUI) initAL(ctx context.Context) {
	d.alWidget = widgets.NewQScrollArea(nil)
	d.alLayout = widgets.NewQFormLayout(nil)
	d.alWidget.SetLayout(d.alLayout)

	if d.device.Methods[yeelight.BgSetPower] {
		d.alPower = widgets.NewQPushButton2("Toggle", nil)
		d.alLayout.AddRow3("Power", d.alPower)

		d.alPower.ConnectClicked(func(_ bool) {
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgToggle))
			if err != nil {

			}
		})

	}

	if d.device.Methods[yeelight.BgSetBright] {
		d.alBrightness = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		d.alBrightness.SetRange(1, 100)
		d.alLayout.AddRow3("Brightness", d.alBrightness)

		d.alBrightness.ConnectSliderReleased(func() {
			value := d.alBrightness.Value()
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgSetBright, value, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				d.alBrightness.SetValue(*d.device.Data.BgBright)
				return
			}
		})
	}

	if d.device.Methods[yeelight.BgSetCtAbx] {
		d.alCt = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		d.alCt.SetRange(1700, 6500)
		d.alLayout.AddRow3("Color Temperature", d.alCt)

		d.alCt.ConnectSliderReleased(func() {
			value := d.alCt.Value()
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgSetCtAbx, value, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				d.alCt.SetValue(*d.device.Data.BgCt)
				return
			}
		})
	}

	if d.device.Methods[yeelight.BgSetRGB] {
		d.alColorDialog = widgets.NewQColorDialog(nil)
		d.alColorDialog.ConnectAccepted(func() {
			color := d.alColorDialog.CurrentColor()
			r, g, b := color.Red(), color.Green(), color.Blue()
			colorInt := rgbToColorInt(int(r), int(g), int(b))
			_, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgSetRGB, colorInt, d.setting.Effect, d.setting.EffectDuration))
			if err != nil {
				return
			}

		})

		d.alColor = widgets.NewQPushButton(nil)
		d.alColor.ConnectClicked(func(checked bool) {
			d.alColorDialog.Exec()
		})

		d.alLayout.AddRow3("Color", d.alColor)
	}

}
