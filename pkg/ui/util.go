package main

import "github.com/diamondburned/gotk4/pkg/gdk/v4"

func intToRGB(rgb int) *gdk.RGBA {
	r := float32((rgb >> 16) & 0xFF)
	g := float32((rgb >> 8) & 0xFF)
	b := float32(rgb & 0xFF)
	color := gdk.NewRGBA(r/255, g/255, b/255, 1)
	return &color
}

func rgbToInt(color *gdk.RGBA) int {
	r := int(color.Red() * 255)
	g := int(color.Green() * 255)
	b := int(color.Blue() * 255)
	return (r << 16) | (g << 8) | b
}
