package main

import (
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"fyne.io/systray"
	_ "fyne.io/x/fyne/layout"
	"github.com/gogf/gf/v2/util/gconv"
	"hei-windows/consts"
	"hei-windows/icon"
	"hei-windows/modules"
	"hei-windows/modules/comic/controller"
	"hei-windows/theme"
	"hei-windows/views"
	"net/url"
	"time"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
func main() {

	//初始化数据库和一些前置配置
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	app := app.NewWithID("百宝箱")

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "错误提示",
				Content: gconv.String(err),
			})
			time.Sleep(2 * time.Second)
		}
	}()

	//设置主题 前置设置主题，防止乱码
	mytheme := &theme.MyTheme{}
	app.Settings().SetTheme(mytheme)
	app.SetIcon(icon.ResourceIcon)

	boostrap := modules.Boostrap
	err := boostrap.Run(ctx)
	if err != nil {
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "错误提示",
			Content: err.Error(),
		})
		time.Sleep(2 * time.Second)
		return
	}

	//每隔30分钟请求一下配置
	go func() {
		for {
			if err = boostrap.LoadConfig(ctx); err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "错误提示",
					Content: err.Error(),
				})
				time.Sleep(2 * time.Second)
				return
			}
			time.Sleep(30 * time.Minute)
		}
	}()

	//初始化UI
	appBox := views.NewAppBox()
	mainWm := appBox.InitMainWindow(ctx, app, icon.ResourceIcon)
	mainWm.SetMaster()
	mainWm.CenterOnScreen()

	config := modules.GlobalConfig

	if !config.Usable {
		tips := dialog.NewInformation("系统提示", "因不可抗拒原因，应用停止对外提供服务", mainWm)
		tips.SetOnClosed(func() {
			systray.Quit()
		})
		tips.Show()
	}
	var isPassChan = make(chan bool, 1)
	//应用升级检查
	if config.IUpdate {
		if config.Version != consts.Version {
			updateTips := dialog.NewConfirm("系统升级", "当前应用不可用，需要强制升级", func(b bool) {
				if b {
					url, _ := url.Parse(config.DownLoad)
					fyne.CurrentApp().OpenURL(url)
					systray.Quit()
				} else {
					systray.Quit()
				}
			}, mainWm)
			updateTips.Show()
		}
	} else {
		isPassChan <- true
	}

	if config.Notice != "" && config.Usable && !config.IUpdate {
		title := widget.NewLabel(config.Notice)
		box := container.NewVBox(title)
		if config.IsDonate {
			image := canvas.NewImageFromURI(storage.NewURI(config.Donate))
			image.FillMode = canvas.ImageFillOriginal
			box.Add(widget.NewCard("打赏码", "", image))
		}
		updateTips := dialog.NewCustom("免责声明", "关闭", box, mainWm)
		updateTips.Show()
	}

	//网络检测
	go func() {
		for {
			time.Sleep(10 * time.Second)
			if !boostrap.NetWorkStatus("baidu.com") {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "错误提示",
					Content: "请检查是否链接网络",
				})
				time.Sleep(60 * time.Second)
				systray.Quit()
				break
			}
		}
	}()

	//设置布局
	layoutUi := views.VLayoutUi
	layoutUi.Wm = mainWm
	layoutUi.InitLayoutUI(ctx)

	//网络检测通过，采取执行数据渲染
	go func() {
		<-isPassChan
		controller.NewComicC(mainWm).Register(ctx)
	}()
	mainWm.ShowAndRun()
	boostrap.Db.Close()
}
