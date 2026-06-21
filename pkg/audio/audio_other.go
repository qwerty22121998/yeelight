//go:build !linux && !windows && !darwin

package audio

import (
	"context"
	"fmt"
	"runtime"
)

// Capture is unsupported here: Linux uses parec (audio_linux.go), Windows/macOS
// use miniaudio (audio_malgo.go). Other platforms have no backend.
func Capture(ctx context.Context) (<-chan Tick, error) {
	return nil, fmt.Errorf("audio capture unsupported on %s", runtime.GOOS)
}
