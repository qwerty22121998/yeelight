package main

import (
	"fmt"
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
