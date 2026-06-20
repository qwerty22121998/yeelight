package main

import (
	"fmt"

	"github.com/therecipe/qt/widgets"
)

// Screen enumeration lives here (not in pkg/screen) because it reads display
// geometry from Qt, the UI toolkit. The actual capture/averaging is backend
// work and lives in pkg/screen.
//
// Uses QDesktopWidget rather than QGuiApplication.Screens() because this
// therecipe/qt build's Screens() binding panics in qtbox mode (it asserts
// []*QScreen on a value the backend hands back as []interface{}).

// listScreens returns one label per monitor for the selector, indexed by
// QDesktopWidget screen number. Call on the GUI thread.
func listScreens() []string {
	d := widgets.QApplication_Desktop()
	n := d.ScreenCount()
	out := make([]string, n)
	for i := 0; i < n; i++ {
		g := d.ScreenGeometry(i)
		out[i] = fmt.Sprintf("%d (%dx%d)", i, g.Width(), g.Height())
	}
	return out
}

// screenRect returns the global pixel rect of monitor idx. Reads Qt geometry,
// so call on the GUI thread. ok is false if idx is out of range.
func screenRect(idx int) (x, y, w, h int, ok bool) {
	d := widgets.QApplication_Desktop()
	if idx < 0 || idx >= d.ScreenCount() {
		return 0, 0, 0, 0, false
	}
	g := d.ScreenGeometry(idx)
	return g.X(), g.Y(), g.Width(), g.Height(), true
}
