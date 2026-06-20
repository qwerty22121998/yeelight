package main

import "github.com/therecipe/qt/core"

// Qt forbids touching widgets from any goroutine other than the GUI thread.
// Background work (discovery, device notifications, screen sync, async command
// sends) pushes closures here; a drain timer started on the GUI thread runs
// them in the right place.
//
// ponytail: 16ms poll-drain via QTimer — simplest correct option without moc
// codegen. If dispatch latency ever matters, replace with a moc signal.
var uiQueue = make(chan func(), 256)

// runOnUI schedules f to run on the Qt GUI thread. Safe from any goroutine.
func runOnUI(f func()) { uiQueue <- f }

// startUIDispatch must be called once, on the GUI thread (from main).
func startUIDispatch() {
	t := core.NewQTimer(nil)
	t.ConnectTimeout(func() {
		for {
			select {
			case f := <-uiQueue:
				f()
			default:
				return
			}
		}
	})
	t.Start(16)
}
