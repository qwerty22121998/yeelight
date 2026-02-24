package yeelight

func Ptr[T any](v T) *T {
	return &v
}

func RGBToInt(r int, g int, b int) int {
	return (r << 16) | (g << 8) | b
}
