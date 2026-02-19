package main

import (
	"context"
	"fmt"
	"log/slog"
	"yeelight/pkg/yeelight"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type DeviceGUI struct {
	gtk.Widgetter
	device     *yeelight.Yeelight
	errorLabel *gtk.Label
}

func NewDeviceGUI(ctx context.Context, device *yeelight.Yeelight) *DeviceGUI {
	dm := &DeviceGUI{
		device: device,
	}
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	dm.Widgetter = box

	box.Append(dm.label())
	controlBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	controlBox.Append(dm.mainLightControl(ctx))
	controlBox.Append(dm.ambientLightControl(ctx))
	box.Append(controlBox)
	return dm
}

func (d *DeviceGUI) label() gtk.Widgetter {
	b := gtk.NewBox(gtk.OrientationHorizontal, 5)

	label := gtk.NewLabel(d.device.IP)
	b.Append(label)
	errLabel := gtk.NewLabel("")
	d.errorLabel = errLabel
	b.Append(errLabel)
	return b
}

func (d *DeviceGUI) setError(err error) {
	if err == nil {
		d.errorLabel.SetLabel("success")
		return
	}
	d.errorLabel.SetLabel(err.Error())
}

func (d *DeviceGUI) command(ctx context.Context, method yeelight.Method, params ...any) bool {
	_, err := d.device.SendCommand(ctx, yeelight.Command{
		Method: method,
		Params: params,
	})
	d.setError(err)
	return err == nil
}

func (d *DeviceGUI) ambientLightControl(ctx context.Context) gtk.Widgetter {
	f := gtk.NewFrame("Ambient light")
	b := gtk.NewBox(gtk.OrientationVertical, 5)
	f.SetChild(b)

	if d.device.Methods[yeelight.BgSetPower] {
		powerLabel := gtk.NewLabel("Power")
		b.Append(powerLabel)
		powerSwitch := gtk.NewSwitch()
		powerSwitch.SetActive(d.device.Props.BgPower == "on")
		powerSwitch.SetTooltipText(d.device.Props.BgPower)
		powerSwitch.ConnectStateSet(func(state bool) (ok bool) {
			defer func() {
				powerSwitch.SetTooltipText(d.device.Props.BgPower)
			}()
			p := "off"
			if state {
				p = "on"
			}
			if !d.command(ctx, yeelight.BgSetPower, p) {
				slog.Info("error setting power, reverting switch state")
				powerSwitch.SetState(!state)
				return true
			}
			d.device.Props.BgPower = p
			return false
		})

		b.Append(powerSwitch)
	}

	if d.device.Methods[yeelight.BgSetBright] {
		brightLabel := gtk.NewLabel("Brightness")
		b.Append(brightLabel)
		brightBox := gtk.NewBox(gtk.OrientationVertical, 5)
		brightScale := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 1, 100, 1)
		brightScale.SetValue(float64(d.device.Props.BgBright))
		brightScale.SetTooltipText(fmt.Sprint(d.device.Props.BgBright))
		brightBox.Append(brightScale)
		brightController := gtk.NewGestureClick()
		brightController.SetPropagationPhase(gtk.PhaseCapture)
		brightController.ConnectReleased(func(nPress int, x, y float64) {
			defer func() {
				brightScale.SetTooltipText(fmt.Sprint(d.device.Props.BgBright))
			}()
			if d.command(ctx, yeelight.BgSetBright, int(brightScale.Value())) {
				d.device.Props.BgBright = int(brightScale.Value())
			}
		})
		brightBox.AddController(brightController)
		b.Append(brightBox)
	}

	if d.device.Methods[yeelight.BgSetCtAbx] {
		temparatureLabel := gtk.NewLabel("Temperature")
		b.Append(temparatureLabel)
		temperatureBox := gtk.NewBox(gtk.OrientationVertical, 5)
		temperatureScale := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 1700, 6500, 100)
		temperatureScale.SetValue(float64(d.device.Props.BgCt))
		temperatureScale.SetTooltipText(fmt.Sprint(d.device.Props.BgCt))
		temperatureBox.Append(temperatureScale)
		temperatureController := gtk.NewGestureClick()
		temperatureController.SetPropagationPhase(gtk.PhaseCapture)
		temperatureController.ConnectReleased(func(nPress int, x, y float64) {
			defer func() {
				temperatureScale.SetTooltipText(fmt.Sprint(d.device.Props.BgCt))
			}()
			if d.command(ctx, yeelight.BgSetCtAbx, int(temperatureScale.Value())) {
				d.device.Props.BgCt = int(temperatureScale.Value())
			}

		})
		temperatureBox.AddController(temperatureController)
		b.Append(temperatureBox)
	}

	if d.device.Methods[yeelight.BgSetRGB] {
		colorLabel := gtk.NewLabel("Color")
		b.Append(colorLabel)
		dialog := gtk.NewColorDialog()
		colorBtn := gtk.NewColorDialogButton(dialog)
		color := intToRGB(d.device.Props.BgRGB)
		colorBtn.SetRGBA(color)

		colorBtn.Connect("notify::rgba", func() {
			color := colorBtn.RGBA()
			if d.command(ctx, yeelight.BgSetRGB, rgbToInt(color)) {
				d.device.Props.BgRGB = rgbToInt(color)
			}
		})

		b.Append(colorBtn)

	}

	return f
}

func (d *DeviceGUI) mainLightControl(ctx context.Context) gtk.Widgetter {
	f := gtk.NewFrame("Main light")
	b := gtk.NewBox(gtk.OrientationVertical, 5)
	f.SetChild(b)

	if d.device.Methods[yeelight.SetPower] {
		powerLabel := gtk.NewLabel("Power")
		b.Append(powerLabel)
		powerSwitch := gtk.NewSwitch()
		powerSwitch.SetActive(d.device.Props.Power == "on")
		powerSwitch.SetTooltipText(d.device.Props.Power)
		powerSwitch.ConnectStateSet(func(state bool) (ok bool) {
			defer func() {
				powerSwitch.SetTooltipText(d.device.Props.Power)
			}()
			p := "off"
			if state {
				p = "on"
			}
			if !d.command(ctx, yeelight.SetPower, p) {
				slog.Info("error setting power, reverting switch state")
				powerSwitch.SetState(!state)
				return true
			}
			d.device.Props.Power = p
			return false
		})

		b.Append(powerSwitch)
	}

	if d.device.Methods[yeelight.SetBright] {
		brightLabel := gtk.NewLabel("Brightness")
		b.Append(brightLabel)
		brightBox := gtk.NewBox(gtk.OrientationVertical, 5)
		brightScale := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 1, 100, 1)
		brightScale.SetValue(float64(d.device.Props.Bright))
		brightScale.SetTooltipText(fmt.Sprint(d.device.Props.Bright))
		brightBox.Append(brightScale)
		brightController := gtk.NewGestureClick()
		brightController.SetPropagationPhase(gtk.PhaseCapture)
		brightController.ConnectReleased(func(nPress int, x, y float64) {
			defer func() {
				brightScale.SetTooltipText(fmt.Sprint(d.device.Props.Bright))
			}()
			if d.command(ctx, yeelight.SetBright, int(brightScale.Value())) {
				d.device.Props.Bright = int(brightScale.Value())
			}
		})
		brightBox.AddController(brightController)
		b.Append(brightBox)
	}

	if d.device.Methods[yeelight.SetCtAbx] {
		temparatureLabel := gtk.NewLabel("Temperature")
		b.Append(temparatureLabel)
		temperatureBox := gtk.NewBox(gtk.OrientationVertical, 5)
		temperatureScale := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 1700, 6500, 100)
		temperatureScale.SetValue(float64(d.device.Props.Ct))
		temperatureScale.SetTooltipText(fmt.Sprint(d.device.Props.Ct))
		temperatureBox.Append(temperatureScale)
		temperatureController := gtk.NewGestureClick()
		temperatureController.SetPropagationPhase(gtk.PhaseCapture)
		temperatureController.ConnectReleased(func(nPress int, x, y float64) {
			defer func() {
				temperatureScale.SetTooltipText(fmt.Sprint(d.device.Props.Ct))
			}()
			if d.command(ctx, yeelight.SetCtAbx, int(temperatureScale.Value())) {
				d.device.Props.Ct = int(temperatureScale.Value())
			}

		})
		temperatureBox.AddController(temperatureController)
		b.Append(temperatureBox)
	}
	if d.device.Methods[yeelight.SetRGB] {
		colorLabel := gtk.NewLabel("Color")
		b.Append(colorLabel)
		dialog := gtk.NewColorDialog()
		colorBtn := gtk.NewColorDialogButton(dialog)
		color := intToRGB(d.device.Props.RGB)
		colorBtn.SetRGBA(color)
		colorBtn.Connect("notify::rgba", func() {
			color := colorBtn.RGBA()
			if d.command(ctx, yeelight.SetRGB, rgbToInt(color)) {
				d.device.Props.RGB = rgbToInt(color)
			}
		})
		b.Append(colorBtn)

	}

	return f
}
