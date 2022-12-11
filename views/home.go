package views

import (
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"hei-windows/consts"
)

type appBox struct {
	ctx     context.Context
	app     fyne.App
	version string
	mainWm  fyne.Window
}

func NewAppBox() *appBox {
	return &appBox{
		version: "1.html.0",
	}
}

// 初始化主窗体
func (v *appBox) InitMainWindow(ctx context.Context, app fyne.App, icon *fyne.StaticResource) fyne.Window {
	v.ctx = ctx
	mainWm := app.NewWindow("漫画大师 " + consts.Version)
	//设置系统托盘
	if desk, ok := app.(desktop.App); ok {
		desk.SetSystemTrayIcon(icon)
		m := fyne.NewMenu("漫画大师",
			fyne.NewMenuItem("主界面", func() {
				mainWm.Show()
			}),
		)
		desk.SetSystemTrayMenu(m)
	}

	mainWm.SetIcon(icon)
	mainWm.SetCloseIntercept(func() {
		mainWm.Hide()
	})
	mainWm.Resize(fyne.NewSize(consts.WindowWidth, consts.WindowHeight)) //设置初始化窗体大小
	mainWm.SetFixedSize(true)
	return mainWm
}
