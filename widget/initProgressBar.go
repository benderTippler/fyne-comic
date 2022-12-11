package widget

import (
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"hei-windows/consts"
)

type CustomProgressBar struct {
	ctx          context.Context
	wd           fyne.Window
	title        string
	dismiss      string
	progress     *widget.ProgressBar
	dialog       dialog.Dialog
	labelCtr     *fyne.Container
	isClosed     bool
	progressData chan *ProgressData
}

type ProgressData struct {
	start float64
	total float64
}

func NewCustomProgressBar(ctx context.Context, title, dismiss string, wd fyne.Window) *CustomProgressBar {
	return &CustomProgressBar{
		ctx:          ctx,
		title:        title,
		dismiss:      dismiss,
		wd:           wd,
		progress:     widget.NewProgressBar(),
		progressData: make(chan *ProgressData, 1),
	}
}

// 运行
func (w *CustomProgressBar) Run() *CustomProgressBar {
	canvasOb := canvas.NewText("<-(^w^)-> 正在启动采集器", nil)
	w.labelCtr = container.New(layout.NewCenterLayout(), canvasOb)
	box := container.NewVBox(w.labelCtr, w.progress)
	custom := dialog.NewCustom(w.title, w.dismiss, box, w.wd)
	custom.Resize(fyne.NewSize(600, 80))
	custom.SetOnClosed(func() {
		if w.isClosed {
			custom.Resize(fyne.NewSize(0, 0))
		} else {
			custom.Show()
		}
	})
	custom.Show()
	w.dialog = custom
	return w
}

// 监听数据变化
func (w *CustomProgressBar) WatchProgressBarValue() {
	go func() {
		for {
			data := <-w.progressData
			if data.start > 1 {
				break
			}
			w.setProgressBarValue(data.start, data.total)
		}
	}()
}

func (w *CustomProgressBar) SetProgressData(start float64, total float64) {
	data := &ProgressData{
		start: start,
		total: total,
	}
	w.progressData <- data
}

// 动态计算进度条变化
func (w *CustomProgressBar) setProgressBarValue(start float64, total float64) bool {
	isFinished := false
	value := start / total
	w.progress.SetValue(value)
	if value >= 1 {
		w.dialog.SetDismissText("关闭")
		w.isClosed = true
		isFinished = true
		w.dialog.Hide()
	}
	return isFinished
}

func (w *CustomProgressBar) SetDialogTitle(title string) {
	w.labelCtr.RemoveAll()
	showTitle := canvas.NewText(title, consts.RedColor)
	w.labelCtr.Add(showTitle)
}
