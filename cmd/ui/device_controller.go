package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"yeelight/pkg/audio"
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
	mainLight    *lightUI
	ambientLight *lightUI
	effects      *widgets.QGroupBox
	info         *infoUI
}

// send runs cmd off the GUI thread (SendCommand blocks up to 5s — must never
// run on the GUI thread). On failure it reports to the status bar and, if
// given, runs onErr back on the GUI thread (e.g. to revert a slider).
func (d *DeviceUI) send(ctx context.Context, cmd yeelight.Command, onErr func()) {
	go func() {
		if _, err := d.device.SendCommand(ctx, cmd); err != nil {
			slog.Error("command failed", "ip", d.device.IP, "method", cmd.Method, "error", err)
			if yeelight.IsUnsupported(err) {
				d.status("Device does not support " + string(cmd.Method))
			} else {
				d.status("Command failed: " + err.Error())
			}
			if onErr != nil {
				runOnUI(onErr)
			}
		}
	}()
}

type lightUI struct {
	Widget           *widgets.QGroupBox
	Layout           *widgets.QFormLayout
	PowerBtn         *widgets.QCheckBox
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
	root := widgets.NewQHBoxLayout()
	ui.QWidget.SetLayout(root)

	ui.renderMainLight(ctx)
	ui.renderAmbientLight(ctx)
	ui.renderEffects(ctx)
	ui.renderInfo(ctx)

	// Left column: light groups stacked top-down, trailing stretch keeps each
	// box at its content height instead of inflating.
	left := widgets.NewQVBoxLayout()
	left.AddWidget(ui.mainLight.Widget, 0, 0)
	left.AddWidget(ui.ambientLight.Widget, 0, 0)
	left.AddWidget(ui.effects, 0, 0)
	left.AddStretch(1)
	leftW := widgets.NewQWidget(nil, 0)
	leftW.SetLayout(left)

	// Splitter so the user can drag the Info panel narrower/wider. Info content
	// scrolls (renderInfo), so it starts small instead of forcing its width.
	split := widgets.NewQSplitter2(core.Qt__Horizontal, nil)
	split.AddWidget(leftW)
	split.AddWidget(ui.info.Widget)
	split.SetStretchFactor(0, 1) // controls take new space
	split.SetStretchFactor(1, 0) // info keeps its width
	split.SetSizes([]int{560, 280})
	root.AddWidget(split, 1, 0)

	ui.update()
	updated := device.Updated()
	go func() {
		// Coalesce props pulses: apply the first immediately, then at most one
		// more per updateInterval. Without this, a fast prop stream (e.g. the
		// bulb echoing every music/screen-sync color change) floods the GUI
		// thread with ui.update — each rewrites stylesheets, which Qt reparses —
		// and the UI goes laggy. Leading edge + trailing so a burst still ends
		// on the latest state.
		var timerC <-chan time.Time
		var timer *time.Timer
		pending := false
		for {
			select {
			case <-ctx.Done():
				return
			case <-device.Done():
				return
			case <-updated:
				if timerC == nil {
					runOnUI(ui.update)
					timer = time.NewTimer(updateInterval)
					timerC = timer.C
				} else {
					pending = true
				}
			case <-timerC:
				if pending {
					runOnUI(ui.update)
					pending = false
					timer.Reset(updateInterval)
				} else {
					timerC = nil
				}
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

func powerLabel(on bool) string {
	if on {
		return "On"
	}
	return "Off"
}

// setPower reflects the device's on/off state on a power checkbox.
func setPower(btn *widgets.QCheckBox, p *string) {
	if btn == nil || p == nil {
		return
	}
	on := *p == "on"
	btn.SetChecked(on)
	btn.SetText(powerLabel(on))
}

func pct(v int) string    { return fmt.Sprintf("%d%%", v) }
func kelvin(v int) string { return fmt.Sprintf("%dK", v) }

// sliderRow builds a [slider | live value] composite. The value label tracks
// valueChanged, which fires for both user drags and programmatic SetValue, so
// it stays correct when the device pushes new state.
func sliderRow(s *widgets.QSlider, unit func(int) string) *widgets.QWidget {
	val := widgets.NewQLabel2(unit(s.Value()), nil, 0)
	s.ConnectValueChanged(func(v int) { val.SetText(unit(v)) })
	row := widgets.NewQWidget(nil, 0)
	h := widgets.NewQHBoxLayout()
	h.SetContentsMargins(0, 0, 0, 0)
	row.SetLayout(h)
	h.AddWidget(s, 1, 0)
	h.AddWidget(val, 0, 0)
	return row
}

// addSlider adds a sliderRow as a labelled form row.
func addSlider(layout *widgets.QFormLayout, label string, s *widgets.QSlider, unit func(int) string) {
	layout.AddRow3(label, sliderRow(s, unit))
}

// addColorControls lays out the color controls. The bulb is in EITHER
// color-temperature (white) OR RGB mode at any one time, so when both are
// supported they're presented as a White/Color toggle over a stacked widget —
// only the active mode's control shows. The toggle just swaps which control is
// visible; dragging the CT slider (set_ct_abx) or picking a color (set_rgb) is
// what actually switches the bulb's mode. mode is the device's current
// color_mode (2 == CT) used to pick the initial page.
// ponytail: view-only toggle, not synced from later props — an external app
// changing the mode won't move it. Add an update() sync if that ever matters.
func addColorControls(layout *widgets.QFormLayout, l *lightUI, mode *int) {
	var ctRow *widgets.QWidget
	if l.CTSlider != nil {
		ctRow = sliderRow(l.CTSlider, kelvin)
	}

	switch {
	case ctRow != nil && l.ColorBtn != nil:
		stack := widgets.NewQStackedWidget(nil)
		stack.AddWidget(ctRow)      // index 0 = white
		stack.AddWidget(l.ColorBtn) // index 1 = color

		white := widgets.NewQRadioButton2("White", nil)
		color := widgets.NewQRadioButton2("Color", nil)
		modeRow := widgets.NewQWidget(nil, 0)
		h := widgets.NewQHBoxLayout()
		h.SetContentsMargins(0, 0, 0, 0)
		modeRow.SetLayout(h)
		h.AddWidget(white, 0, 0)
		h.AddWidget(color, 0, 0)
		h.AddStretch(1)

		white.ConnectClicked(func(bool) { stack.SetCurrentIndex(0) })
		color.ConnectClicked(func(bool) { stack.SetCurrentIndex(1) })

		if mode != nil && *mode == 2 { // 2 == color temperature
			white.SetChecked(true)
			stack.SetCurrentIndex(0)
		} else {
			color.SetChecked(true)
			stack.SetCurrentIndex(1)
		}

		layout.AddRow3("Mode", modeRow)
		layout.AddRow5(stack)
	case ctRow != nil:
		layout.AddRow3("Color Temperature", ctRow)
	case l.ColorBtn != nil:
		layout.AddRow3("Color", l.ColorBtn)
	}
}

func (d *DeviceUI) update() {
	// Power IS synced here: lamp15 (and likely others) push power/bg_power in
	// async props, so external on/off must reflect in the UI. setPower skips
	// when the value is nil. Our own toggles call device.ApplyLocal so the
	// snapshot matches the user's intent even on firmware that never echoes
	// set_power — that's what stops a later props from reverting the switch.
	data := d.device.Snapshot()
	setPower(d.mainLight.PowerBtn, data.Power)
	setPower(d.ambientLight.PowerBtn, data.BgPower)
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
	box := widgets.NewQVBoxLayout()
	info.Widget.SetLayout(box)

	info.Layout = widgets.NewQFormLayout(nil)
	box.AddLayout(info.Layout, 0)

	info.Layout.AddRow3("IP:", widgets.NewQLabel2(d.device.IP, nil, 0))
	info.Layout.AddRow3("Name:", widgets.NewQLabel2(d.device.Name, nil, 0))
	info.Layout.AddRow3("ID:", widgets.NewQLabel2(d.device.ID, nil, 0))
	info.Layout.AddRow3("Model:", widgets.NewQLabel2(d.device.Model, nil, 0))
	info.Layout.AddRow3("Firmware version:", widgets.NewQLabel2(d.device.FWVersion, nil, 0))

	methods := make([]string, 0, len(d.device.Methods))
	for m, ok := range d.device.Methods {
		if ok {
			methods = append(methods, string(m))
		}
	}
	sort.Strings(methods)

	box.AddWidget(widgets.NewQLabel2("Supported methods:", nil, 0), 0, 0)
	// Qt has no native chip/pill widget. Closest: a QListWidget in IconMode —
	// items reflow as the panel is dragged (pairs with the splitter), and a
	// stylesheet rounds them into pills. It scrolls internally, so it fills the
	// remaining height instead of forcing the panel large.
	pills := widgets.NewQListWidget(nil)
	pills.SetViewMode(widgets.QListView__IconMode)
	pills.SetFlow(widgets.QListView__LeftToRight)
	pills.SetWrapping(true)
	pills.SetResizeMode(widgets.QListView__Adjust)
	pills.SetMovement(widgets.QListView__Static)
	pills.SetSpacing(3)
	pills.SetFocusPolicy(core.Qt__NoFocus)
	pills.SetSelectionMode(widgets.QAbstractItemView__NoSelection)
	pills.SetStyleSheet(`
QListWidget { border: none; background: transparent; }
QListWidget::item { background: ` + d.setting.Theme.Accent + `; color: white; border-radius: 9px; padding: 3px 8px; }`)
	for _, m := range methods {
		pills.AddItem(m)
	}
	box.AddWidget(pills, 1, 0)

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
		mainLight.PowerBtn = widgets.NewQCheckBox2("Off", nil)
		mainLight.Layout.AddRow3("Power", mainLight.PowerBtn)
		setPower(mainLight.PowerBtn, d.device.Snapshot().Power) // initial state from FetchProps

		mainLight.PowerBtn.ConnectClicked(func(checked bool) {
			mainLight.PowerBtn.SetText(powerLabel(checked))
			state := "off"
			if checked {
				state = "on"
			}
			old := d.device.Snapshot().Power
			d.device.ApplyLocal(yeelight.Data{Power: yeelight.Ptr(state)}) // so update() won't revert it
			d.send(ctx, yeelight.C(yeelight.SetPower, state, d.setting.Effect, d.setting.EffectDuration), func() {
				d.device.ApplyLocal(yeelight.Data{Power: old}) // send failed — roll back
				setPower(mainLight.PowerBtn, d.device.Snapshot().Power)
			})
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

		addSlider(mainLight.Layout, "Brightness", mainLight.BrightnessSlider, pct)
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
	}

	addColorControls(mainLight.Layout, mainLight, d.device.Snapshot().ColorMode)

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
		ambientLight.PowerBtn = widgets.NewQCheckBox2("Off", nil)
		ambientLight.Layout.AddRow3("Power", ambientLight.PowerBtn)
		setPower(ambientLight.PowerBtn, d.device.Snapshot().BgPower) // initial state from FetchProps

		ambientLight.PowerBtn.ConnectClicked(func(checked bool) {
			ambientLight.PowerBtn.SetText(powerLabel(checked))
			state := "off"
			if checked {
				state = "on"
			}
			old := d.device.Snapshot().BgPower
			d.device.ApplyLocal(yeelight.Data{BgPower: yeelight.Ptr(state)}) // so update() won't revert it
			d.send(ctx, yeelight.C(yeelight.BgSetPower, state, d.setting.Effect, d.setting.EffectDuration), func() {
				d.device.ApplyLocal(yeelight.Data{BgPower: old}) // send failed — roll back
				setPower(ambientLight.PowerBtn, d.device.Snapshot().BgPower)
			})
		})
	}

	if d.device.Methods[yeelight.BgSetBright] {
		ambientLight.BrightnessSlider = widgets.NewQSlider2(core.Qt__Horizontal, nil)
		ambientLight.BrightnessSlider.SetRange(1, 100)
		addSlider(ambientLight.Layout, "Brightness", ambientLight.BrightnessSlider, pct)

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
	}

	addColorControls(ambientLight.Layout, ambientLight, d.device.Snapshot().BgLMode)

	d.ambientLight = ambientLight
}

// renderEffects builds the "Effects" group: per-device automations that drive
// the ambient light from an external source (the screen, system audio). Runs
// after renderAmbientLight — it reuses d.ambientLight.ColorBtn as the live
// color readout.
func (d *DeviceUI) renderEffects(ctx context.Context) {
	effects := widgets.NewQGroupBox2("Effects", nil)
	layout := widgets.NewQFormLayout(nil)
	effects.SetLayout(layout)
	d.effects = effects

	// Screen and music sync both drive the one ambient light, so only one runs
	// at a time. Each block assigns its stopper; the other calls it (nil-safe)
	// before starting. Assigned below; closures capture the vars, not values.
	var stopScreenSync, stopMusicSync func()

	// Screen Sync (per-device): poll the chosen screen's average color into
	// the ambient light. Capture (pkg/screen, a subprocess) and the device
	// command run off the GUI thread so the UI never stalls; inFlight
	// serializes sends, since concurrent writes to the one TCP conn would
	// corrupt the stream. The QTimer and the screen list are set up in
	// ShowEvent: a QTimer only fires from the thread it is created in and
	// listScreens reads Qt globals — both need the GUI thread, but device
	// tabs are built on the scan goroutine.
	if d.device.Methods[yeelight.BgSetRGB] {
		cfg := d.setting.SyncFor(d.device.ID)

		screenCombo := widgets.NewQComboBox(nil)
		layout.AddRow3("Sync Screen", screenCombo)

		intervalSpin := widgets.NewQSpinBox(nil)
		intervalSpin.SetRange(100, 60000)
		intervalSpin.SetValue(cfg.Interval)
		layout.AddRow3("Sync Interval (ms)", intervalSpin)

		var inFlight atomic.Bool
		var timer *core.QTimer

		// Screen sync pushes one bg_set_rgb per tick — at the default 1s interval
		// that already sits at the bulb's ~60 cmd/min quota, and faster intervals
		// blow past it. Route updates through a music-mode channel (no quota). The
		// session is opened off the GUI thread (StartMusic blocks up to ~5s) when
		// sync turns on and closed when it turns off, so the bulb resumes
		// reporting props while idle.
		// ponytail: screen and music sync open independent sessions; enabling both
		// on one bulb makes their set_music calls fight. Rare (both drive the same
		// ambient light); add a shared ref-counted session if it ever matters.
		var (
			senderMu sync.Mutex
			sender   *rgbSender
			active   bool
		)
		startSync := func() {
			senderMu.Lock()
			active = true
			senderMu.Unlock()
			timer.Start(cfg.Interval)
			go func() {
				s, usingMusic := newRGBSender(ctx, d.device, yeelight.BgSetRGB)
				if !usingMusic {
					d.status("Bulb has no music mode — screen sync limited to ~1 update/s")
				}
				senderMu.Lock()
				if !active { // toggled off while we were connecting
					senderMu.Unlock()
					s.Close()
					return
				}
				sender = s
				senderMu.Unlock()
			}()
		}
		stopSync := func() {
			timer.Stop()
			senderMu.Lock()
			s := sender
			sender, active = nil, false
			senderMu.Unlock()
			if s != nil {
				go s.Close()
			}
		}

		syncBtn := widgets.NewQPushButton2("Sync", nil)
		syncBtn.SetCheckable(true)
		layout.AddRow3("Screen Sync", syncBtn)

		intervalSpin.ConnectValueChanged(func(value int) {
			cfg.Interval = value
			d.setting.Save()
			if timer != nil && timer.IsActive() {
				timer.SetInterval(value)
			}
		})

		// stopScreenSync turns screen sync off if it's on (used by music sync to
		// claim the ambient light). SetChecked won't fire ConnectClicked, so it
		// stops explicitly.
		stopScreenSync = func() {
			if !syncBtn.IsChecked() {
				return
			}
			syncBtn.SetChecked(false)
			if timer != nil {
				stopSync()
			}
			cfg.Enabled = false
			d.setting.Save()
		}

		syncBtn.ConnectClicked(func(checked bool) {
			if timer == nil {
				return
			}
			if checked {
				if stopMusicSync != nil {
					stopMusicSync() // mutually exclusive: only one sync at a time
				}
				startSync()
			} else {
				stopSync()
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
						senderMu.Lock()
						s := sender
						senderMu.Unlock()
						if s == nil {
							return // music session still connecting, or stopped
						}
						s.send(ctx, color)
						// Reflect the synced color on the ambient color button now —
						// the device doesn't reliably emit a props notification for
						// its own bg_set_rgb. Marshal to the GUI thread: this runs on
						// the capture goroutine.
						runOnUI(func() {
							d.ambientLight.ColorBtn.SetStyleSheet("background-color: " + colorIntToRGB(color))
						})
					}()
				})
				if cfg.Enabled {
					syncBtn.SetChecked(true)
					startSync()
				}
			})
		})
	}

	// Music Sync: drive the ambient light from system audio. Gated on bg color
	// only — set_music is tried at runtime, with a throttled fallback when the
	// bulb lacks music mode.
	if d.device.Methods[yeelight.BgSetRGB] {
		musicBtn := widgets.NewQPushButton2("Music Sync", nil)
		musicBtn.SetCheckable(true)
		layout.AddRow3("Music Sync", musicBtn)

		// Mode/scheme/sensitivity/saturation/floor live in one container so they
		// can hide as a unit — shown only while Music Sync is on. Shared across
		// devices (one value on Setting); runMusicSync reads them live per-tick.
		musicConfig := widgets.NewQWidget(nil, 0)
		cfgLayout := widgets.NewQFormLayout(nil)
		musicConfig.SetLayout(cfgLayout)

		modeCombo := widgets.NewQComboBox(nil)
		modeCombo.AddItems(musicModeNames)
		modeCombo.SetCurrentText(d.setting.MusicMode)
		modeCombo.ConnectCurrentTextChanged(func(text string) {
			d.setting.MusicMode = text
			d.setting.Save()
		})
		cfgLayout.AddRow3("Music Mode", modeCombo)

		schemeCombo := widgets.NewQComboBox(nil)
		schemeCombo.AddItems(musicSchemeNames)
		schemeCombo.SetCurrentText(d.setting.MusicScheme)
		schemeCombo.ConnectCurrentTextChanged(func(text string) {
			d.setting.MusicScheme = text
			d.setting.Save()
		})
		cfgLayout.AddRow3("Music Color", schemeCombo)

		sensSpin := widgets.NewQSpinBox(nil)
		sensSpin.SetRange(50, 300)
		sensSpin.SetValue(int(d.setting.MusicSensitivity * 100))
		sensSpin.ConnectValueChanged(func(value int) {
			d.setting.MusicSensitivity = float64(value) / 100
			d.setting.Save()
		})
		cfgLayout.AddRow3("Music Sensitivity (%)", sensSpin)

		satSpin := widgets.NewQSpinBox(nil)
		satSpin.SetRange(0, 100)
		satSpin.SetValue(int(d.setting.MusicSaturation * 100))
		satSpin.ConnectValueChanged(func(value int) {
			d.setting.MusicSaturation = float64(value) / 100
			d.setting.Save()
		})
		cfgLayout.AddRow3("Music Saturation (%)", satSpin)

		floorSpin := widgets.NewQSpinBox(nil)
		floorSpin.SetRange(0, 100)
		floorSpin.SetValue(int(d.setting.MusicFloor * 100))
		floorSpin.ConnectValueChanged(func(value int) {
			d.setting.MusicFloor = float64(value) / 100
			d.setting.Save()
		})
		cfgLayout.AddRow3("Music Brightness Floor (%)", floorSpin)

		// Visualizer: a scrolling waveform of recent levels, each bar tinted
		// with the color sent for that tick. Shown only while sync runs, and
		// updated at full audio rate even when the bulb itself updates slowly
		// (rate-limited fallback).
		viz := newWaveViz(128)
		viz.SetVisible(false)
		cfgLayout.AddRow3("", viz)

		musicConfig.SetVisible(false) // hidden until Music Sync is on
		layout.AddRow3("", musicConfig)

		// Config visibility mirrors the button's checked state. ConnectToggled
		// fires on programmatic SetChecked too (e.g. runMusicSync's stop on
		// stream end, or stopMusicSync), so the config hides whenever sync ends.
		musicBtn.ConnectToggled(func(checked bool) { musicConfig.SetVisible(checked) })

		var cancel context.CancelFunc
		// stopMusicSync turns music sync off if it's on (used by screen sync to
		// claim the ambient light). SetChecked drives ConnectToggled to hide the
		// config; the goroutine is cancelled explicitly since SetChecked does not
		// fire ConnectClicked.
		stopMusicSync = func() {
			if cancel != nil {
				cancel()
				cancel = nil
			}
			musicBtn.SetChecked(false)
		}

		musicBtn.ConnectClicked(func(checked bool) {
			if !checked {
				if cancel != nil {
					cancel()
					cancel = nil
				}
				return
			}
			if stopScreenSync != nil {
				stopScreenSync() // mutually exclusive: only one sync at a time
			}
			var mctx context.Context
			mctx, cancel = context.WithCancel(ctx)
			go d.runMusicSync(mctx, musicBtn, viz)
		})
	}
}

// maxFlowSteps caps how many buffered colors go into one fallback color flow.
// Audio ticks land ~11/s, so a 1s window holds ~10; the bulb tweens through
// them autonomously and the flow stays roughly 1s of wall-clock either way.
const maxFlowSteps = 9

// rgbSender pushes a stream of RGB updates to a device's light. It prefers
// music mode (set_music opens a side channel with NO command-rate limit — the
// point of the whole exercise, since screen and music sync push far more than
// the bulb's ~60 cmd/min control-connection quota). If the bulb has no music
// mode it falls back to the control connection: it buffers the colors that
// arrive within minGap and flushes them as ONE color flow (bg_start_cf), so a
// single command/sec still animates the whole second instead of dropping all
// but one sample. Costs up to ~minGap of latency. Not safe for concurrent use.
type rgbSender struct {
	dev        *yeelight.Device
	method     yeelight.Method
	flowMethod yeelight.Method // start_cf variant matching method; fallback only
	music      *yeelight.Music
	minGap     time.Duration // >0 only in fallback
	last       time.Time
	buf        []int // colors awaiting the next fallback flush
}

// newRGBSender starts a sender for method (e.g. bg_set_rgb). usingMusic reports
// whether music mode was acquired; false means the batched fallback is in use.
// StartMusic blocks up to ~5s, so never call this on the GUI thread.
func newRGBSender(ctx context.Context, dev *yeelight.Device, method yeelight.Method) (s *rgbSender, usingMusic bool) {
	flow := yeelight.StartCf
	if method == yeelight.BgSetRGB {
		flow = yeelight.BgStartCf
	}
	s = &rgbSender{dev: dev, method: method, flowMethod: flow}
	if music, err := dev.StartMusic(ctx); err == nil {
		s.music = music
	} else {
		slog.Warn("music mode unavailable; batched-flow fallback", "ip", dev.IP, "error", err)
		s.minGap = time.Second
	}
	return s, s.music != nil
}

// send pushes one update. In fallback it buffers the color and, once minGap has
// elapsed since the last flush, sends the whole buffer as one color flow.
func (s *rgbSender) send(ctx context.Context, rgb int) {
	if s.music != nil {
		s.music.Send(yeelight.C(s.method, rgb, "sudden", 0))
		return
	}
	s.buf = append(s.buf, rgb)
	if len(s.buf) > maxFlowSteps { // keep the most recent window
		s.buf = s.buf[len(s.buf)-maxFlowSteps:]
	}
	if time.Since(s.last) < s.minGap { // still filling this window
		return
	}
	s.flush(ctx)
}

// flush sends the buffered colors as one color flow spanning ~minGap and clears
// the buffer. The flow's last step stays lit (FlowStay) until the next flush.
// ponytail: brightness pinned to 100 — music/screen sync is a full-on visual
// mode and musicColor already encodes loudness into the RGB magnitude; this
// overrides any brightness the user set on the light while sync runs.
func (s *rgbSender) flush(ctx context.Context) {
	if len(s.buf) == 0 {
		return
	}
	step := max(int(s.minGap/time.Millisecond)/len(s.buf), 50) // 50ms = Yeelight's per-transition floor
	exprs := make([]yeelight.FlowExpression, len(s.buf))
	for i, rgb := range s.buf {
		exprs[i] = yeelight.FlowExpression{Duration: step, Mode: yeelight.FlowRGB, Value: rgb, Brightness: 100}
	}
	s.dev.SendCommand(ctx, yeelight.C(s.flowMethod, yeelight.ColorFlow{
		Count:      len(exprs),
		Action:     yeelight.FlowStay,
		Expression: exprs,
	}.Build()...))
	s.buf = s.buf[:0]
	s.last = time.Now()
}

// Close leaves music mode if it was entered.
func (s *rgbSender) Close() {
	if s.music != nil {
		s.music.Stop(context.Background())
	}
}

// musicScheme maps a tick's tone (0..1) to a hue window: bass sits at start,
// treble at start+span (degrees). Picked in Settings → Effect.
type musicScheme struct{ start, span float64 }

const (
	defaultMusicScheme      = "Spectrum"
	defaultMusicFloor       = 0.2
	defaultMusicMode        = "Spectrum"
	defaultMusicSensitivity = 1.0
	defaultMusicSaturation  = 1.0

	// pulseCycleTicks is how many ticks (~11/s) one Beat Pulse hue sweep takes.
	pulseCycleTicks = 40 // ~3.6s per full cycle
	// strobeThreshold is the loudness above which Strobe flashes white.
	strobeThreshold = 0.6
)

// musicSchemeNames is the picker order; musicSchemes is the lookup. Spectrum is
// the default — the original hardcoded red→blue mapping.
var (
	musicSchemeNames = []string{"Spectrum", "Rainbow", "Warm", "Cool", "Fire", "Ocean", "Neon"}
	musicSchemes     = map[string]musicScheme{
		"Spectrum": {0, 240},   // red → blue
		"Rainbow":  {0, 360},   // full wheel
		"Warm":     {0, 60},    // red → yellow
		"Cool":     {180, 120}, // cyan → purple
		"Fire":     {0, 40},    // red → orange
		"Ocean":    {180, 60},  // cyan → blue
		"Neon":     {300, 180}, // magenta → cyan
	}

	// musicModeNames is the mode-picker order. Spectrum is the default.
	musicModeNames = []string{"Spectrum", "Beat Pulse", "Strobe", "Steady"}
)

// schemeHue resolves a scheme name (unknown falls back to the default) and maps
// tone to its hue. Read per-tick so a scheme change applies without restarting.
func schemeHue(name string, tone float64) float64 {
	s, ok := musicSchemes[name]
	if !ok {
		s = musicSchemes[defaultMusicScheme]
	}
	return s.start + tone*s.span
}

// musicValue maps loudness (0..1) to a brightness value, with floor as the
// quiet-passage minimum so silence dims rather than going black. floor is
// clamped to 0..1.
func musicValue(floor, level float64) float64 {
	if floor < 0 {
		floor = 0
	}
	if floor > 1 {
		floor = 1
	}
	return floor + (1-floor)*level
}

// musicColor maps one audio tick to a packed RGB color per the user's music
// settings. tickN counts ticks since sync started; Beat Pulse uses it to cycle
// hue over time. Pure (modulo the scheme table) so it's unit-testable. Read
// per-tick, so every setting applies live without restarting sync.
func musicColor(s *Setting, t audio.Tick, tickN int) int {
	sat := clamp01(s.MusicSaturation)
	// Sensitivity scales loudness before it drives brightness; <=0 means unity.
	sens := s.MusicSensitivity
	if sens <= 0 {
		sens = 1
	}
	level := math.Min(t.Level*sens, 1)

	switch s.MusicMode {
	case "Beat Pulse":
		// Hue sweeps the scheme on a timer; loudness pulses brightness.
		phase := math.Mod(float64(tickN)/pulseCycleTicks, 1)
		return hsvToColorInt(schemeHue(s.MusicScheme, phase), sat, musicValue(s.MusicFloor, level))
	case "Strobe":
		// White flash on loud beats, floor-dim white otherwise.
		if level >= strobeThreshold {
			return hsvToColorInt(0, 0, 1)
		}
		return hsvToColorInt(0, 0, clamp01(s.MusicFloor))
	case "Steady":
		// Fixed color (scheme start), loudness → brightness.
		return hsvToColorInt(schemeHue(s.MusicScheme, 0), sat, musicValue(s.MusicFloor, level))
	default: // "Spectrum": tone → hue, loudness → brightness.
		return hsvToColorInt(schemeHue(s.MusicScheme, t.Tone), sat, musicValue(s.MusicFloor, level))
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// runMusicSync captures system audio and maps each tick to an ambient color
// until ctx is cancelled or the stream ends. It prefers music mode (no command
// rate limit); if the bulb rejects set_music it falls back to throttled
// control-connection sends. The visualizer updates at full audio rate
// regardless. On setup failure it reports to the status bar and un-checks.
func (d *DeviceUI) runMusicSync(ctx context.Context, btn *widgets.QPushButton, viz *waveViz) {
	stop := func() {
		runOnUI(func() {
			btn.SetChecked(false)
			viz.SetVisible(false)
			viz.reset()
		})
	}

	ticks, err := audio.Capture(ctx)
	if err != nil {
		slog.Error("audio capture failed", "error", err)
		d.status("Audio capture failed: " + err.Error())
		stop()
		return
	}

	// Push updates through a music-mode channel (no command-rate limit); if the
	// bulb lacks music mode, newRGBSender falls back to a throttled control
	// connection.
	sender, usingMusic := newRGBSender(ctx, d.device, yeelight.BgSetRGB)
	defer sender.Close()
	if !usingMusic {
		d.status("Bulb has no music mode — batched fallback (~1 flow/s, slight lag)")
	}

	runOnUI(func() { viz.SetVisible(true) })

	tickN := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.device.Done():
			return
		case t, ok := <-ticks:
			if !ok {
				d.status("Music sync stopped: audio stream ended")
				stop()
				return
			}
			// Map the tick to a color per the live music settings (mode,
			// scheme, sensitivity, saturation, floor). tickN drives Beat Pulse.
			rgb := musicColor(d.setting, t, tickN)
			tickN++
			level := t.Level
			runOnUI(func() { viz.push(level, rgb) })
			sender.send(ctx, rgb)
		}
	}
}
