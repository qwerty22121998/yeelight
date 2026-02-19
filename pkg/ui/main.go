package main

import (
	"context"
	"os"
	"time"
	"yeelight/pkg/yeelight"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func init() {
}

func main() {
	ctx := context.Background()
	devices, err := yeelight.Discover(ctx, &yeelight.DiscoverConfig{
		Timeout: time.Second,
	})
	if err != nil {
		panic(err)
	}
	if len(devices) == 0 {
		panic("no devices")
	}
	if err := devices[0].FetchProps(ctx); err != nil {
		panic(err)
	}

	app := gtk.NewApplication("com.github.qwerty22121998.yeelight", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() {
		activate(ctx, app, devices[0])
	})
	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(ctx context.Context, app *gtk.Application, device *yeelight.Yeelight) {
	window := gtk.NewApplicationWindow(app)
	window.SetTitle("Yeelight Controller")
	dm := NewDeviceGUI(ctx, device)
	window.SetChild(dm)
	window.SetDefaultSize(400, 300)
	window.SetVisible(true)
}
