//go:build !linux && !windows && !darwin

package screen

import (
	"fmt"
	"image"
	"runtime"
)

// capture has no backend here (only linux/windows/darwin are wired up).
func capture(x, y, w, h int) (image.Image, error) {
	return nil, fmt.Errorf("screen capture unsupported on %s", runtime.GOOS)
}
