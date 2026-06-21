package screen

import (
	"image"
	"testing"
)

func packRGB(r, g, b int) int { return r<<16 | g<<8 | b }

// A uniform image averages to its own color at any size — i.e. stride sampling
// (which kicks in past maxSamples) must not skew the result.
func TestAverageImageUniform(t *testing.T) {
	for _, size := range []int{4, 4096} { // below and well past maxSamples
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		want := packRGB(10, 200, 30)
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 10, 200, 30, 255
		}
		if got := averageImage(img); got != want {
			t.Fatalf("%dx%d: want %d, got %d", size, size, want, got)
		}
	}
}

func TestMean(t *testing.T) {
	if got := mean(nil); got != -1 {
		t.Fatalf("empty: want -1, got %d", got)
	}
	// avg of red(255,0,0) and blue(0,0,255) = (127,0,127)
	if got, want := mean([]int{packRGB(255, 0, 0), packRGB(0, 0, 255)}), packRGB(127, 0, 127); got != want {
		t.Fatalf("want %d, got %d", want, got)
	}
	// constant input averages to itself
	c := packRGB(10, 20, 30)
	if got := mean([]int{c, c, c}); got != c {
		t.Fatalf("constant: want %d, got %d", c, got)
	}
}
