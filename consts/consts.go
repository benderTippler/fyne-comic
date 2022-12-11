package consts

import (
	"github.com/gogf/gf/v2/util/gconv"
	"image/color"
	"time"
)

// 全家组件类型
type WidgetType int

var (
	RedColor = color.NRGBA{R: 255, G: 0, B: 0, A: 205}
)

// 窗体配置文件
var (
	Version         = "v1.0.0"
	anteil  float32 = 0.4
	Width           = 420 * anteil
	Height          = 580 * anteil

	// 漫画列表配置
	Rows    = 2
	Columns = 6

	WindowWidth          = Width * (gconv.Float32(Columns) + 0.6)
	WindowHeight float32 = 760
	CacheTime            = 24 * time.Hour //这里是内存缓存时间，应用关闭会失效

	ImageViewWidth  float32 = 1200
	ImageViewHeight float32 = 700

	DefaultCover = "" //默认封面

	ServiceApi = "http://bender.tpddns.cn:5555/update"
	//ServiceApi = "http://127.0.0.1:8080/update"

	Key = "comic-hei-box" //加密key
)
