package main

import (
	"context"
	"os"

	"github.com/therecipe/qt/widgets"
)

const defaultStyle = `
QWidget {
}
`

func main() {
	ctx := context.Background()
	app := widgets.NewQApplication(len(os.Args), os.Args)
	app.SetStyleSheet(defaultStyle)
	mainWindow := widgets.NewQMainWindow(nil, 0)
	mainWindow.Show()

	ui := NewYeelightUI(mainWindow)
	ui.RenderMain(ctx, mainWindow)

	app.Exec()
}
