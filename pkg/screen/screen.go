// Package screen captures a region of the desktop and reduces it to an average
// color. It is Wayland/wlroots only: it shells out to grim, because under
// XWayland a Qt/X11 root-window grab returns black. An X11 session would need a
// different capture backend.
package screen

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"os/exec"
)

// grabScale is the grim downscale factor applied at capture time. A 2560-wide
// monitor becomes ~128px — plenty to average a mood color, and fast to decode.
const grabScale = "0.05"

// Average captures the global pixel rect (x,y w×h) with grim and returns the
// mean color packed as (r<<16)|(g<<8)|b.
func Average(x, y, w, h int) (int, error) {
	geom := fmt.Sprintf("%d,%d %dx%d", x, y, w, h)
	out, err := exec.Command("grim", "-g", geom, "-s", grabScale, "-").Output()
	if err != nil {
		return 0, fmt.Errorf("grim capture %q: %w", geom, err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		return 0, fmt.Errorf("decode grim png: %w", err)
	}
	b := img.Bounds()
	pixels := make([]int, 0, b.Dx()*b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			// NRGBA = un-premultiplied, so a non-opaque framebuffer alpha does
			// not darken the colors we report.
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			pixels = append(pixels, int(c.R)<<16|int(c.G)<<8|int(c.B))
		}
	}
	return mean(pixels), nil
}

// mean averages packed RGB ints into one packed RGB int. Returns -1 for an
// empty slice.
func mean(pixels []int) int {
	if len(pixels) == 0 {
		return -1
	}
	var r, g, b int
	for _, c := range pixels {
		r += c >> 16 & 0xFF
		g += c >> 8 & 0xFF
		b += c & 0xFF
	}
	n := len(pixels)
	return (r/n)<<16 | (g/n)<<8 | (b / n)
}
