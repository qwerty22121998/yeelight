package main

import (
	"fmt"
	"math"
	"reflect"
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

// hsvToColorInt converts HSV (h in degrees 0..360, s and v in 0..1) to a packed
// (r<<16)|(g<<8)|b int. Used by music sync to turn a tone value into a color.
func hsvToColorInt(h, s, v float64) int {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return rgbToColorInt(int((r+m)*255), int((g+m)*255), int((b+m)*255))
}

func allNotNil(arr ...any) bool {
	for _, value := range arr {
		v := reflect.ValueOf(value)
		k := v.Kind()
		switch k {
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer,
			reflect.UnsafePointer, reflect.Interface, reflect.Slice:
			if v.IsNil() {
				return false
			}
		}
	}
	return true
}
