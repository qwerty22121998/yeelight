package main

import (
	"sync"

	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

// waveViz is a scrolling level-over-time audio visualizer. Each tick pushes a
// (level, color) sample into a fixed-size ring; paintEvent draws the recent
// history as vertical colored bars mirrored about the centre line, newest on
// the right — so the waveform scrolls left as the music plays. Replaces the
// single-value progress bar.
type waveViz struct {
	*widgets.QWidget

	mu    sync.Mutex
	level []float64 // 0..1, oldest first
	color []int     // packed rgb, parallel to level
	bars  int       // ring capacity (max bars shown)
}

func newWaveViz(bars int) *waveViz {
	v := &waveViz{
		QWidget: widgets.NewQWidget(nil, 0),
		bars:    bars,
	}
	v.SetMinimumHeight(56)
	v.ConnectPaintEvent(v.paint)
	return v
}

// push appends one sample and schedules a repaint. Call on the GUI thread.
func (v *waveViz) push(level float64, rgb int) {
	v.mu.Lock()
	if len(v.level) >= v.bars {
		v.level = v.level[1:]
		v.color = v.color[1:]
	}
	v.level = append(v.level, level)
	v.color = append(v.color, rgb)
	v.mu.Unlock()
	v.Update()
}

// reset empties the history. Call on the GUI thread.
func (v *waveViz) reset() {
	v.mu.Lock()
	v.level, v.color = nil, nil
	v.mu.Unlock()
	v.Update()
}

func (v *waveViz) paint(*gui.QPaintEvent) {
	v.mu.Lock()
	level := append([]float64(nil), v.level...)
	color := append([]int(nil), v.color...)
	v.mu.Unlock()

	p := gui.NewQPainter2(v)
	defer p.DestroyQPainter()

	w, h := v.Width(), v.Height()
	p.FillRect5(0, 0, w, h, gui.NewQColor3(18, 18, 18, 255))
	if v.bars == 0 || len(level) == 0 {
		return
	}

	barW := float64(w) / float64(v.bars) // fixed column width => fills then scrolls
	mid := float64(h) / 2
	for i, lv := range level {
		bh := lv * (float64(h) - 2)
		if bh < 1 {
			bh = 1
		}
		c := color[i]
		col := gui.NewQColor3((c>>16)&0xff, (c>>8)&0xff, c&0xff, 255)
		p.FillRect5(int(float64(i)*barW), int(mid-bh/2), int(barW)+1, int(bh), col)
	}
}
