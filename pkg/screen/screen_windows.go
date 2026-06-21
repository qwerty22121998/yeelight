//go:build windows

package screen

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

// Windows has no clean screenshot CLI, so grab via GDI directly. syscall's
// LazyDLL (Windows-only) keeps this pure Go — no CGO, no extra dependency.
var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	getDC                  = user32.NewProc("GetDC")
	releaseDC              = user32.NewProc("ReleaseDC")
	createCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	deleteDC               = gdi32.NewProc("DeleteDC")
	deleteObject           = gdi32.NewProc("DeleteObject")
	selectObject           = gdi32.NewProc("SelectObject")
	setStretchBltMode      = gdi32.NewProc("SetStretchBltMode")
	stretchBlt             = gdi32.NewProc("StretchBlt")
	getDIBits              = gdi32.NewProc("GetDIBits")
)

const (
	srcCopy      = 0x00CC0020
	halftone     = 4 // best-quality StretchBlt shrink
	biRGB        = 0
	dibRGBColors = 0
)

// dst is the downscaled capture size: StretchBlt shrinks the screen rect into
// this small bitmap so we copy a few KB, not the full framebuffer.
const dstW, dstH = 128, 72

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

func capture(x, y, w, h int) (image.Image, error) {
	screenDC, _, _ := getDC.Call(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer releaseDC.Call(0, screenDC)

	memDC, _, _ := createCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer deleteDC.Call(memDC)

	bmp, _, _ := createCompatibleBitmap.Call(screenDC, dstW, dstH)
	if bmp == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap failed")
	}
	defer deleteObject.Call(bmp)

	old, _, _ := selectObject.Call(memDC, bmp)
	defer selectObject.Call(memDC, old)

	setStretchBltMode.Call(memDC, halftone)
	ret, _, _ := stretchBlt.Call(memDC, 0, 0, dstW, dstH,
		screenDC, uintptr(x), uintptr(y), uintptr(w), uintptr(h), srcCopy)
	if ret == 0 {
		return nil, fmt.Errorf("StretchBlt failed")
	}

	bi := bitmapInfo{Header: bitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:       dstW,
		Height:      -dstH, // negative => top-down rows
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
	}}
	buf := make([]byte, dstW*dstH*4)
	ret, _, _ = getDIBits.Call(memDC, bmp, 0, dstH,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), dibRGBColors)
	if ret == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}

	// 32-bit DIB is BGRA with undefined alpha; repack as opaque RGBA.
	img := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for i := 0; i < len(buf); i += 4 {
		img.Pix[i] = buf[i+2]   // R
		img.Pix[i+1] = buf[i+1] // G
		img.Pix[i+2] = buf[i]   // B
		img.Pix[i+3] = 255      // A
	}
	return img, nil
}
