package views

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gogf/gf/v2/os/gtime"
	"hei-windows/icon"
	"hei-windows/modules"
)

var VLayoutUi = new(layoutUI)

type layoutUI struct {
	ctx      context.Context
	Wm       fyne.Window
	TopC     *fyne.Container
	MainC    *fyne.Container
	Page     *fyne.Container
	InputBt  *widget.Entry
	SearchBt *widget.Button
	Scroll   *container.Scroll
}

// 顶部布局 做个搜索框
func (v *layoutUI) topLayout() *fyne.Container {
	inputBt := widget.NewEntry()
	searchBt := widget.NewButtonWithIcon("搜索", theme.SearchIcon(), func() {})
	//设置图片
	img := canvas.NewImageFromResource(icon.ResourceIcon)
	img.Resize(fyne.NewSize(70, 79))
	img.SetMinSize(fyne.NewSize(70, 79))
	inputBt.SetPlaceHolder("请输出您要搜索的动漫名称")
	inputBt.Resize(fyne.NewSize(200, searchBt.MinSize().Height))
	ctr := container.NewGridWithColumns(3, widget.NewLabel(""), inputBt, container.NewHBox(searchBt))

	v.InputBt = inputBt
	v.SearchBt = searchBt

	return container.NewVBox(container.NewCenter(container.NewGridWithRows(1, img)), ctr)
}

// 底部布局
func (v *layoutUI) bottomLayout() fyne.CanvasObject {
	year := gtime.Now().Year()
	tips := fmt.Sprintf("版权归著作者所有，如有侵权请联系邮箱:%v @Copy%v年", modules.GlobalConfig.Email, year)
	label := widget.NewLabelWithStyle(tips, fyne.TextAlignCenter, fyne.TextStyle{})
	layout := container.NewCenter(label)
	return layout
}

// 渲染
func (v *layoutUI) InitLayoutUI(ctx context.Context) {
	v.ctx = ctx
	v.TopC = v.topLayout()

	v.MainC = container.NewMax()
	v.Page = container.NewMax()

	v.Scroll = container.NewScroll(container.NewBorder(nil, v.Page, nil, nil, v.MainC))
	content := container.NewBorder(v.TopC, v.bottomLayout(), nil, nil, v.Scroll)
	v.Wm.SetContent(content)
}

func (v *layoutUI) Show(object ...fyne.CanvasObject) {
	v.MainC.RemoveAll()
	for _, o := range object {
		v.MainC.Add(o)
	}
	v.MainC.Refresh()
}
