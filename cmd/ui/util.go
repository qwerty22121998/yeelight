package main

import (
	"fmt"
)

func colorIntToRGB(c int) string {
	r := (c >> 16) & 0xFF
	g := (c >> 8) & 0xFF
	b := c & 0xFF
	return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
}

func rgbToColorInt(r, g, b int) int {
	return (r << 16) | (g << 8) | b
}
