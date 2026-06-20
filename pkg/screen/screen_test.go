package screen

import "testing"

func packRGB(r, g, b int) int { return r<<16 | g<<8 | b }

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
