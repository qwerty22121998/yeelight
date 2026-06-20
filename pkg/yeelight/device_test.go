package yeelight

import (
	"context"
	"sync"
	"testing"
)

// Snapshot must be race-free against concurrent prop updates from the listen()
// goroutine. Run with -race; fails if dataMu is removed.
func TestSnapshotRace(t *testing.T) {
	d := &Device{updatedChan: make(chan struct{})}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			d.updateData(context.Background(), Data{Bright: Ptr(i % 100)})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = d.Snapshot()
		}
	}()
	wg.Wait()
}
