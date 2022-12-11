package widget

import (
	"bufio"
	"context"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"golang.org/x/image/webp"
	_ "golang.org/x/image/webp"
	"hei-windows/consts"
	"image"
	"image/gif"
	"image/png"
	"time"
)

type imageView struct {
	ctx         context.Context
	title       string              //标题
	cover       string              //封面
	ButtonGroup []fyne.CanvasObject //按钮组，按照顺序加载
}

type ImageView struct {
	Index     int
	ImageView *imageView
	Container *fyne.Container
}

func NewImageView(ctx context.Context, title, cover string, btGroup []fyne.CanvasObject) *imageView {
	return &imageView{
		ctx:         ctx,
		title:       title,
		cover:       cover,
		ButtonGroup: btGroup,
	}
}

func (w *imageView) Renderer() (*fyne.Container, error) {
	//解决图片格式不兼容，改成文件流
	var cover string
	if !gstr.ContainsI(w.cover, "http") {
		cover = "http:" + w.cover
	} else {
		cover = w.cover
	}
	rsp, err := g.Client().Timeout(10*time.Second).Get(w.ctx, cover)
	defer rsp.Close()
	if err != nil {
		return nil, err
	}
	var tmp image.Image
	reader := bufio.NewReaderSize(rsp.Body, 32*1024)
	if gstr.ContainsI(cover, "webp") {
		tmp, err = webp.Decode(reader)
	} else if gstr.ContainsI(cover, "png") {
		tmp, err = png.Decode(reader)
	} else if gstr.ContainsI(cover, "gif") {
		tmp, err = gif.Decode(reader)
	} else {
		tmp, _, err = image.Decode(reader)
	}

	var card, btGroups *fyne.Container
	if err != nil {
		image := canvas.NewImageFromURI(storage.NewURI(cover))
		image.SetMinSize(fyne.NewSize(consts.Width, consts.Height))
		image.Resize(fyne.NewSize(consts.Width, consts.Height))
		title := canvas.NewText(gstr.StrLimitRune(w.title, 10, "..."), consts.RedColor)
		title.TextSize = 13
		card = container.NewCenter(image, title)
	} else {
		image := canvas.NewRasterFromImage(tmp)
		image.SetMinSize(fyne.NewSize(consts.Width, consts.Height))
		image.Resize(fyne.NewSize(consts.Width, consts.Height))
		title := canvas.NewText(gstr.StrLimitRune(w.title, 10, "..."), consts.RedColor)
		title.TextSize = 13
		card = container.NewCenter(image, title)
	}
	btGroups = container.NewGridWithColumns(len(w.ButtonGroup), w.ButtonGroup...)
	return container.NewVBox(card, btGroups), nil
}
