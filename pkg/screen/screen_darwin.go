//go:build darwin

package screen

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
)

// capture grabs the rect with the built-in `screencapture` tool: -x silences
// the shutter, -t png, -R sets the region. screencapture has no stdout mode, so
// it writes a temp file we read and remove. Average's stride sampling keeps the
// full-res frame cheap (screencapture can't downscale at grab time).
//
// First use raises the macOS Screen Recording permission prompt; sync stays
// black until the user grants it.
func capture(x, y, w, h int) (image.Image, error) {
	f, err := os.CreateTemp("", "yeelight-*.png")
	if err != nil {
		return nil, err
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	region := fmt.Sprintf("%d,%d,%d,%d", x, y, w, h)
	if err := exec.Command("screencapture", "-x", "-t", "png", "-R", region, name).Run(); err != nil {
		return nil, fmt.Errorf("screencapture %q: %w", region, err)
	}

	r, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return png.Decode(r)
}
