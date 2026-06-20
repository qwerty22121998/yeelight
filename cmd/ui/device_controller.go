package main

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
	"yeelight/pkg/screen"
	"yeelight/pkg/yeelight"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

const updateInterval = time.Second / 2

type DeviceUI struct {
	*widgets.QWidget
	device       *yeelight.Device
	setting      *Setting
	status       func(string) // report errors to the status bar; marshals to the GUI thread
	layout       *widgets.QGridLayout
	mainLight    *lightUI
	ambientLight *lightUI
	info         *infoUI
}

// send runs cmd off the GUI thread (SendCommand blocks up to 5s — must never
// run on the GUI thread). On failure it reports to the status bar and, if
// given, runs onErr back on the GUI thread (e.g. to revert a slider).
func (d *DeviceUI) send(ctx context.Context, cmd yeelight.Command, onErr func()) {
	go func() {
		if _, err := d.device.SendCommand(ctx, cmd); err != nil {
			slog.Error("command failed", "ip", d.device.IP, "method", cmd.Method, "error", err)
			d.status("Command failed: " + err.Error())
			if onErr != nil {
				runOnUI(onErr)
			}
		}
	}()
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

func NewDeviceUI(ctx context.Context, device *yeelight.Device, setting *Setting, status func(string)) *DeviceUI {
	ui := &DeviceUI{
		device:  device,
		setting: setting,
		status:  status,
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
			case <-device.Done():
				return
			case <-updated:
				runOnUI(ui.update)
			}
		}
	}()

	return ui
}

// setSlider applies v to a slider unless the user is currently dragging it
// (a device notification mid-drag would fight the user's input).
func setSlider(s *widgets.QSlider, v *int) {
	if allNotNil(s, v) && !s.IsSliderDown() {
		s.SetValue(*v)
	}
}

func (d *DeviceUI) update() {
	data := d.device.Snapshot()
	setSlider(d.mainLight.BrightnessSlider, data.Bright)
	setSlider(d.mainLight.CTSlider, data.Ct)
	if allNotNil(d.mainLight.ColorBtn, data.RGB) {
		d.mainLight.ColorBtn.SetStyleSheet("background-color: " + colorIntToRGB(*data.RGB))
	}

	setSlider(d.ambientLight.BrightnessSlider, data.BgBright)
	setSlider(d.ambientLight.CTSlider, data.BgCt)
	if allNotNil(d.ambientLight.ColorBtn, data.BgRGB) {
		d.ambientLight.ColorBtn.SetStyleSheet("background-color: " + colorIntToRGB(*data.BgRGB))
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
			d.send(ctx, yeelight.C(yeelight.Toggle), nil)
		})
	}

	if d.device.Methods[yeelight.SetBright] {
		mainLight.BrightnessSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		mainLight.BrightnessSlider.SetRange(1, 100)
		mainLight.BrightnessSlider.ConnectSliderReleased(func() {
			value := mainLight.BrightnessSlider.Value()
			d.send(ctx, yeelight.C(yeelight.SetBright, value, d.setting.Effect, d.setting.EffectDuration), func() {
				setSlider(mainLight.BrightnessSlider, d.device.Snapshot().Bright)
			})
		})

		mainLight.Layout.AddRow3("Brightness", mainLight.BrightnessSlider)
	}

	if d.device.Methods[yeelight.SetCtAbx] {
		mainLight.CTSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		mainLight.CTSlider.SetRange(1700, 6500)
		mainLight.CTSlider.ConnectSliderReleased(func() {
			value := mainLight.CTSlider.Value()
			d.send(ctx, yeelight.C(yeelight.SetCtAbx, value, d.setting.Effect, d.setting.EffectDuration), func() {
				setSlider(mainLight.CTSlider, d.device.Snapshot().Ct)
			})
		})

		mainLight.Layout.AddRow3("Color Temperature", mainLight.CTSlider)
	}

	if d.device.Methods[yeelight.SetRGB] {
		mainLight.ColorDialog = widgets.NewQColorDialog(nil)
		mainLight.ColorDialog.ConnectAccepted(func() {
			color := mainLight.ColorDialog.CurrentColor()
			colorInt := rgbToColorInt(color.Red(), color.Green(), color.Blue())
			d.send(ctx, yeelight.C(yeelight.SetRGB, colorInt, d.setting.Effect, d.setting.EffectDuration), nil)
		})

		mainLight.ColorBtn = widgets.NewQPushButton(nil)
		mainLight.ColorBtn.ConnectClicked(func(checked bool) {
			mainLight.ColorDialog.Exec()
		})
		mainLight.Layout.AddRow3("Color", mainLight.ColorBtn)
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
			d.send(ctx, yeelight.C(yeelight.BgToggle), nil)
		})
	}

	if d.device.Methods[yeelight.BgSetBright] {
		ambientLight.BrightnessSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		ambientLight.BrightnessSlider.SetRange(1, 100)
		ambientLight.Layout.AddRow3("Brightness", ambientLight.BrightnessSlider)

		ambientLight.BrightnessSlider.ConnectSliderReleased(func() {
			value := ambientLight.BrightnessSlider.Value()
			d.send(ctx, yeelight.C(yeelight.BgSetBright, value, d.setting.Effect, d.setting.EffectDuration), func() {
				setSlider(ambientLight.BrightnessSlider, d.device.Snapshot().BgBright)
			})
		})
	}

	if d.device.Methods[yeelight.BgSetCtAbx] {
		ambientLight.CTSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		ambientLight.CTSlider.SetRange(1700, 6500)
		ambientLight.Layout.AddRow3("Color Temperature", ambientLight.CTSlider)

		ambientLight.CTSlider.ConnectSliderReleased(func() {
			value := ambientLight.CTSlider.Value()
			d.send(ctx, yeelight.C(yeelight.BgSetCtAbx, value, d.setting.Effect, d.setting.EffectDuration), func() {
				setSlider(ambientLight.CTSlider, d.device.Snapshot().BgCt)
			})
		})
	}

	if d.device.Methods[yeelight.BgSetRGB] {
		ambientLight.ColorDialog = widgets.NewQColorDialog(nil)
		ambientLight.ColorDialog.ConnectAccepted(func() {
			color := ambientLight.ColorDialog.CurrentColor()
			colorInt := rgbToColorInt(color.Red(), color.Green(), color.Blue())
			d.send(ctx, yeelight.C(yeelight.BgSetRGB, colorInt, d.setting.Effect, d.setting.EffectDuration), nil)
		})

		ambientLight.ColorBtn = widgets.NewQPushButton(nil)
		ambientLight.ColorBtn.ConnectClicked(func(checked bool) {
			ambientLight.ColorDialog.Exec()
		})

		ambientLight.Layout.AddRow3("Color", ambientLight.ColorBtn)

		// Screen Sync (per-device): poll the chosen screen's average color into
		// the ambient light. Capture (pkg/screen, a subprocess) and the device
		// command run off the GUI thread so the UI never stalls; inFlight
		// serializes sends, since concurrent writes to the one TCP conn would
		// corrupt the stream. The QTimer and the screen list are set up in
		// ShowEvent: a QTimer only fires from the thread it is created in and
		// listScreens reads Qt globals — both need the GUI thread, but device
		// tabs are built on the scan goroutine.
		cfg := d.setting.SyncFor(d.device.ID)

		screenCombo := widgets.NewQComboBox(nil)
		ambientLight.Layout.AddRow3("Sync Screen", screenCombo)

		intervalSpin := widgets.NewQSpinBox(nil)
		intervalSpin.SetRange(100, 60000)
		intervalSpin.SetValue(cfg.Interval)
		ambientLight.Layout.AddRow3("Sync Interval (ms)", intervalSpin)

		var inFlight atomic.Bool
		var timer *core.QTimer
		syncBtn := widgets.NewQPushButton2("Sync", nil)
		syncBtn.SetCheckable(true)
		ambientLight.Layout.AddRow3("Screen Sync", syncBtn)

		intervalSpin.ConnectValueChanged(func(value int) {
			cfg.Interval = value
			d.setting.Save()
			if timer != nil && timer.IsActive() {
				timer.SetInterval(value)
			}
		})

		syncBtn.ConnectClicked(func(checked bool) {
			if timer == nil {
				return
			}
			if checked {
				timer.Start(cfg.Interval)
			} else {
				timer.Stop()
			}
			cfg.Enabled = checked
			d.setting.Save()
		})

		var bootOnce sync.Once
		syncBtn.ConnectShowEvent(func(e *gui.QShowEvent) {
			syncBtn.ShowEventDefault(e)
			bootOnce.Do(func() {
				screenCombo.AddItems(listScreens())
				screenCombo.SetCurrentIndex(cfg.ScreenIndex)
				screenCombo.ConnectCurrentIndexChanged(func(index int) {
					cfg.ScreenIndex = index
					d.setting.Save()
				})

				timer = core.NewQTimer(nil)
				timer.ConnectTimeout(func() {
					if !inFlight.CompareAndSwap(false, true) {
						return
					}
					// Read screen geometry on the GUI thread (Qt), then capture +
					// send off-thread.
					x, y, w, h, ok := screenRect(cfg.ScreenIndex)
					if !ok {
						inFlight.Store(false)
						return
					}
					go func() {
						defer inFlight.Store(false)
						color, err := screen.Average(x, y, w, h)
						if err != nil {
							slog.Warn("screen sync capture failed", "error", err)
							return
						}
						if _, err := d.device.SendCommand(ctx, yeelight.C(yeelight.BgSetRGB, color, d.setting.Effect, d.setting.EffectDuration)); err != nil {
							return
						}
						// Reflect the synced color on the ambient color button now —
						// the device doesn't reliably emit a props notification for
						// its own bg_set_rgb. Marshal to the GUI thread: this runs on
						// the capture goroutine.
						runOnUI(func() {
							ambientLight.ColorBtn.SetStyleSheet("background-color: " + colorIntToRGB(color))
						})
					}()
				})
				if cfg.Enabled {
					syncBtn.SetChecked(true)
					timer.Start(cfg.Interval)
				}
			})
		})
	}

	d.ambientLight = ambientLight
}
