//go:build linux

package screen

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
)

// grabScale is the grim downscale factor applied at capture time. A 2560-wide
// monitor becomes ~128px — plenty to average a mood color, and fast to decode.
const grabScale = "0.05"

// capture grabs the rect under Wayland (grim) or X11 (ImageMagick import).
// Detection is by $WAYLAND_DISPLAY, which the compositor sets under Wayland and
// is absent on a plain X11 session.
func capture(x, y, w, h int) (image.Image, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return captureGrim(x, y, w, h)
	}
	return captureImport(x, y, w, h)
}

func captureGrim(x, y, w, h int) (image.Image, error) {
	geom := fmt.Sprintf("%d,%d %dx%d", x, y, w, h)
	out, err := exec.Command("grim", "-g", geom, "-s", grabScale, "-").Output()
	if err != nil {
		return nil, fmt.Errorf("grim capture %q: %w", geom, err)
	}
	return png.Decode(bytes.NewReader(out))
}

// captureImport uses ImageMagick `import` against the X11 root window: grab
// root, crop to the rect, +repage to drop the crop offset, -resize 5% to
// downscale at capture (mirrors grim's -s), PNG to stdout.
func captureImport(x, y, w, h int) (image.Image, error) {
	crop := fmt.Sprintf("%dx%d+%d+%d", w, h, x, y)
	out, err := exec.Command("import", "-silent", "-window", "root",
		"-crop", crop, "+repage", "-resize", "5%", "png:-").Output()
	if err != nil {
		return nil, fmt.Errorf("import capture %q: %w (X11 screen sync needs ImageMagick)", crop, err)
	}
	return png.Decode(bytes.NewReader(out))
}
