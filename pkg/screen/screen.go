// Package screen captures a region of the desktop and reduces it to an average
// color. Capture is platform-specific (see screen_linux.go / screen_windows.go
// / screen_darwin.go) because a Qt/X11 root-window grab returns black under
// XWayland; each backend shells out to a native tool (or GDI on Windows). This
// file holds the platform-independent averaging.
package screen

import (
	"image"
	"image/color"
	"math"
)

// maxSamples caps how many pixels Average reads per frame. A mood color does
// not need every pixel; sampling on a stride keeps a full-res capture (e.g.
// macOS, whose screencapture can't downscale at grab time) cheap.
const maxSamples = 4096

// Average captures the global pixel rect (x,y w×h) via the platform backend
// (capture, defined in screen_<goos>.go) and returns the mean color packed as
// (r<<16)|(g<<8)|b.
func Average(x, y, w, h int) (int, error) {
	img, err := capture(x, y, w, h)
	if err != nil {
		return 0, err
	}
	return averageImage(img), nil
}

// averageImage reduces an image to its mean color (packed RGB), sampling on a
// stride so a full-res frame stays cheap (see maxSamples).
func averageImage(img image.Image) int {
	b := img.Bounds()
	stride := 1
	if total := b.Dx() * b.Dy(); total > maxSamples {
		stride = int(math.Sqrt(float64(total) / maxSamples))
	}
	pixels := make([]int, 0, maxSamples)
	for y := b.Min.Y; y < b.Max.Y; y += stride {
		for x := b.Min.X; x < b.Max.X; x += stride {
			// NRGBA = un-premultiplied, so a non-opaque framebuffer alpha does
			// not darken the colors we report.
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			pixels = append(pixels, int(c.R)<<16|int(c.G)<<8|int(c.B))
		}
	}
	return mean(pixels)
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
