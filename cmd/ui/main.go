package main

import (
	"context"
	"os"

	"github.com/therecipe/qt/widgets"
)

func main() {
	ctx := context.Background()
	app := widgets.NewQApplication(len(os.Args), os.Args)
	mainWindow := widgets.NewQMainWindow(nil, 0)
	mainWindow.SetWindowTitle("Yeelight")
	startUIDispatch()
	mainWindow.Show()

	ui := NewYeelightUI(mainWindow)
	ui.RenderMain(ctx, mainWindow)

	app.Exec()
}
