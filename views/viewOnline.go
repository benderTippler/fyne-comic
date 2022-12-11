package views

import (
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"hei-windows/consts"
	"hei-windows/modules/comic/types"
)

type ImageOline struct {
	Index        int
	CanvasObject fyne.CanvasObject
}

type viewOnline struct {
	ctx            context.Context
	Title          string
	left           []*types.Chapter //左边数据
	CurrentChapter int              //当前查看的漫画章节
	container      *fyne.Container
	topWt          *fyne.Container
	LeftWt         fyne.CanvasObject
	MainCtr        *fyne.Container
	Scroll         *container.Scroll
}

func NewViewOnline(ctx context.Context, title string) *viewOnline {
	return &viewOnline{
		ctx:            ctx,
		Title:          title,
		CurrentChapter: 0, //默认第一话
	}
}

// 设置左侧章节
func (v *viewOnline) SetLeft(left []*types.Chapter) {
	v.left = left
	v.container.Refresh()
}

func (v *viewOnline) SetTop(top fyne.CanvasObject) {
	v.topWt.RemoveAll()
	v.topWt.Add(top)
	v.container.Refresh()
}

// 渲染对应在线浏览界面
func (v *viewOnline) IntWindow() fyne.Window {
	wdd := fyne.CurrentApp().NewWindow("在线阅读-" + v.Title)
	wdd.Resize(fyne.NewSize(consts.ImageViewWidth, 700))
	v.MainCtr = v.mainLayout()
	v.Scroll = container.NewScroll(v.MainCtr)

	content := container.NewBorder(v.topLayout(), nil, v.leftLayout(), nil, v.Scroll)

	v.container = content
	wdd.SetContent(content)
	wdd.CenterOnScreen()
	wdd.SetFixedSize(true)
	wdd.Show()
	return wdd
}

// 主题漫画显示
func (v *viewOnline) mainLayout() *fyne.Container {
	return container.NewMax()
}

func (v *viewOnline) topLayout() fyne.CanvasObject {
	v.topWt = container.NewMax()
	return v.topWt
}

// 左边列表显示
func (v *viewOnline) leftLayout() fyne.CanvasObject {
	list := widget.NewList(
		func() int {
			return len(v.left)
		},
		func() fyne.CanvasObject {
			box := container.NewVBox(widget.NewLabel(""))
			box.Resize(fyne.NewSize(200, 100))
			return box
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*fyne.Container).Objects[0].(*widget.Label).SetText(v.left[id].Title)
		},
	)
	v.LeftWt = list
	box := container.NewScroll(container.NewPadded(v.LeftWt))
	box.SetMinSize(fyne.NewSize(100, consts.Height))
	return container.NewPadded(box)
}

func (v *viewOnline) Show(canvasObj fyne.CanvasObject) {
	v.MainCtr.RemoveAll()
	v.MainCtr.Add(canvasObj)
	v.Scroll.ScrollToTop()
	v.Scroll.Refresh()
	v.MainCtr.Refresh()
}
