package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/therecipe/qt/widgets"
)

// setupLogging tees slog into the in-app Log tab (logs) and, when it can open
// it, <UserConfigDir>/yeelight/yeelight.log (alongside config.toml, appending
// across runs). A file failure just drops that writer rather than crashing.
// ponytail: append-only, never rotated — add rotation only if it grows unbounded in practice.
func setupLogging() {
	writers := []io.Writer{logs}
	if dir, err := os.UserConfigDir(); err == nil {
		logDir := filepath.Join(dir, configDirName)
		if os.MkdirAll(logDir, 0o755) == nil {
			if f, err := os.OpenFile(filepath.Join(logDir, "yeelight.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
				writers = append(writers, f)
			}
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(writers...), nil)))
	slog.Info("logging started")
}

func main() {
	setupLogging()
	ctx := context.Background()
	app := widgets.NewQApplication(len(os.Args), os.Args)
	qApp = app
	platformDefaultStyle = currentStyleKey("", availableStyles()) // capture before any SetStyle, for the "Default" option
	mainWindow := widgets.NewQMainWindow(nil, 0)
	mainWindow.SetWindowTitle("Yeelight")
	startUIDispatch()
	mainWindow.Show()

	ui := NewYeelightUI(mainWindow)
	applyAppearance(ui.setting) // install the persisted (or default) style + palette
	ui.RenderMain(ctx, mainWindow)

	app.Exec()
}
