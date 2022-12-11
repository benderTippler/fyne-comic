package controller

import (
	"bytes"
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/systray"
	"github.com/gogf/gf/v2/container/garray"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"hei-windows/consts"
	"hei-windows/modules"
	"hei-windows/modules/comic/collection"
	"hei-windows/modules/comic/internal/model"
	"hei-windows/modules/comic/types"
	"hei-windows/util"
	"hei-windows/views"
	widgetSeft "hei-windows/widget"
	"image"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type comic struct {
	ctx         context.Context
	mainWm      fyne.Window
	currentPage int
	searchStr   string
}

func NewComicC(window fyne.Window) *comic {
	return &comic{
		mainWm:      window,
		currentPage: 1,
	}
}

// 漫画模块所有组件注册
func (c *comic) Register(ctx context.Context) {
	c.ctx = ctx
	//初始化搜索组件
	c.search()
	//第一次初始化网站数据
	isInitFinish := c.initCollection()
	if isInitFinish {
		c.page()
	}
	//第二次 启动程序对比模块数据是否有更新，如果存在更新，后台采集数据，并通知用户
}

// 初始化网站列表数据到本地数据库
func (c *comic) initCollection() bool {
	//多么模块更新
	modulesList := modules.GlobalConfig.Modules
	for _, module := range modulesList {
		if module.Enable != 1 {
			continue
		}
		//目前一个网站，后续根据配置更新网站信息
		isHas, _ := modules.Boostrap.Db.Where("site = ? and max_page > ?", module.Source, 0).Exist(&model.Moudule{})
		if isHas {
			fmt.Println("模块信息已经初始化，不需要再次初始化了！")
			continue
		}
		//开启初始化动漫网站数据
		customPb := widgetSeft.NewCustomProgressBar(c.ctx, "数据初始化", "禁止操作", c.mainWm)
		//设置初始化进度条,并且监听状态变化
		customPb.Run().WatchProgressBarValue()
		comicCtn := collection.NewComicCtn(c.ctx, module, nil)

		if !comicCtn.StartUp(customPb) { //数据完成，写入标识符,标识漫画模块初始化完成
			//弹框提示，初始化数据失败
			dialog.ShowConfirm("温馨提示", "初始化数据出现错误", func(b bool) {
				if b == true { //重试
					comicCtn.StartUp(customPb)
				} else { //关闭应用
					systray.Quit()
				}
			}, c.mainWm)
		}
	}

	return true
}

// 查询渲染
func (c *comic) search() {
	//注册查询按钮回调时间
	layoutUi := views.VLayoutUi
	layoutUi.SearchBt.OnTapped = func() {
		c.searchStr = layoutUi.InputBt.Text
		c.page()
	}
}

// 添加按钮事件回调
func (c *comic) seeButtonCallBack(comic *types.Comic) func() {
	return func() {
		comicMo := modules.GlobalConfig.Modules[comic.Source]
		if comicMo.Enable == 2 {
			msg := fmt.Sprintf("漫画来源于 %v 网站,官方要求下架次资源，非常抱歉！", modules.GlobalConfig.Modules[comic.Source].SiteName)
			dialog.ShowConfirm("温馨提示", msg, func(b bool) {}, c.mainWm)
			return
		}

		//第一步、布局启动
		windowOnline := views.NewViewOnline(c.ctx, comic.Title)
		winWn := windowOnline.IntWindow()

		winWn.SetOnClosed(func() {
			modules.Boostrap.Cache.Clear(c.ctx)
		})

		progressBarChapter := widgetSeft.NewCustomProgressBar(c.ctx, fmt.Sprintf("%v 章节列表初始化", comic.Title), "禁止操作", winWn)
		progressBarChapter.Run().WatchProgressBarValue()
		//第二步、 左侧组件展示
		module := modules.GlobalConfig.Modules[comic.Source]
		comicCtn := collection.NewComicCtn(c.ctx, module, progressBarChapter)
		chapters, err := comicCtn.GetComicChapters(comic.DetailUrl)
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "错误提示",
				Content: "章节列表获取失败",
			})
			winWn.Close()
			return
		}
		if len(chapters) == 0 {
			progressBarChapter.SetProgressData(1, 1)
			dialog.ShowConfirm("温馨提示", fmt.Sprintf("%v 动漫未有章节更新", comic.Title), func(b bool) {
				winWn.Close()
			}, winWn)
			return
		}
		windowOnline.SetLeft(chapters)
		progressBarChapter.SetProgressData(1, 1)
		//第三步、章节数据对应图片初始化
		windowOnline.LeftWt.(*widget.List).OnSelected = func(id widget.ListItemID) {
			windowOnline.Scroll.Offset.Y = 0
			windowOnline.Scroll.Refresh()
			winWn.SetTitle("正在阅读：" + comic.Title + "-" + chapters[id].Title)
			if len(chapters) == 0 {
				return
			}
			progressBar := widgetSeft.NewCustomProgressBar(c.ctx, fmt.Sprintf("%v 数据初始化", chapters[id].Title), "禁止操作", winWn)
			progressBar.Run().WatchProgressBarValue()

			progressBar.SetDialogTitle(fmt.Sprintf("%v 开始图片采集", chapters[id].Title))
			progressBar.SetProgressData(0, 1)
			comicCtn2 := collection.NewComicCtn(c.ctx, module, progressBar)
			list, err := comicCtn2.GetResource(chapters[id].ChapterUrl)
			if err != nil { //提示手动重新加载

			}
			bjj := make([]*views.ImageOline, 0)
			wg2 := sync.WaitGroup{}
			taskViewChan := make(chan bool, 20)
			var intProgress int32 = 0
			for i, imageUrl := range list {
				wg2.Add(1)
				taskViewChan <- true
				go func(i int, imageUrl interface{}) {
					imgSingleUrl := imageUrl.(string)
				tryImage:
					var charpterImage *canvas.Image
					if comicMo.Collect.Resource.Referer != "" {
						gclient := g.Client()
						gclient.SetHeader("Referer", comicMo.Collect.Resource.Referer)
						rsp, err := gclient.Timeout(10*time.Minute).Get(c.ctx, imgSingleUrl)
						defer rsp.Close()
						if err != nil {
							fmt.Println("重试 err", i)
							goto tryImage
						}
						if rsp.StatusCode != http.StatusOK {
							fmt.Println("重试 失败", i)
							goto tryImage
						}
						charpterImage = canvas.NewImageFromReader(rsp.Body, gmd5.MustEncryptString(imgSingleUrl))
					} else {
						charpterImage = canvas.NewImageFromURI(storage.NewURI(imgSingleUrl))
					}
					if charpterImage == nil {
						fmt.Println("重试 charpterImage is nil", i)
						goto tryImage
					}

					r := bytes.NewReader(charpterImage.Resource.Content())
					img, _, err := image.Decode(r)
					if err != nil {
						fmt.Println("重试", i, err)
						goto tryImage
					}
					var imgVx, imgVy int
					//TODO::通知
					if img == nil {
						imgVx = 640
						imgVy = 500
					} else {
						imgVx = img.Bounds().Max.X
						imgVy = img.Bounds().Max.Y
					}

					imgSy := math.Ceil(gconv.Float64(consts.ImageViewHeight) * gconv.Float64(imgVy) / gconv.Float64(imgVx))
					charpterImage.SetMinSize(fyne.NewSize(consts.ImageViewHeight*0.9, gconv.Float32(imgSy)))
					bjj = append(bjj, &views.ImageOline{
						Index: i, CanvasObject: charpterImage,
					})
					fmt.Println(fmt.Sprintf("结束 %v 第 %v 张图片加载完成", chapters[id].Title, i+1))
					//进度条设置
					<-taskViewChan
					atomic.AddInt32(&intProgress, 1)
					progressBar.SetDialogTitle(fmt.Sprintf("%v 第 %v 张图片加载完成", chapters[id].Title, i+1))
					progressBar.SetProgressData((gconv.Float64(intProgress)/gconv.Float64(len(list)))*0.4+0.5, 1)
					defer wg2.Done()
				}(i, imageUrl)
			}
			wg2.Wait()
			//切片数组排序
			progressBar.SetDialogTitle(fmt.Sprintf("%v 开始渲染图片到应用载体上面", chapters[id].Title))
			sort.SliceStable(bjj, func(i, j int) bool {
				return bjj[i].Index < bjj[j].Index
			})
			box := container.NewGridWithColumns(1)
			for _, v := range bjj {
				box.Add(v.CanvasObject)
			}
			windowOnline.Show(box)
			progressBar.SetProgressData(1, 1)
		}
		windowOnline.LeftWt.(*widget.List).Select(windowOnline.CurrentChapter) //默认选中第一个
	}
}

func (c *comic) getImageViews(title string, page int) *fyne.Container {
	limit := consts.Columns * consts.Rows
	comics := make([]*model.Comic, 0)
	//TODO:: 数据抽离到业务层
	if title == "" {
		modules.Boostrap.Db.Limit(limit, (page-1)*limit).Find(&comics)
	} else {
		modules.Boostrap.Db.Limit(limit).Where("title like ?", title+"%").OrderBy("id desc").Find(&comics)
	}

	if len(comics) == 0 {
		fmt.Println("没有数据了")
		label := widget.NewLabelWithStyle("已经到底了，没有数据了", fyne.TextAlignCenter, fyne.TextStyle{})
		return container.NewCenter(label)
	}

	//第一步、 初始化列表图片组件，并且排序
	comicM := &model.Comic{}
	cltComics := comicM.Unmarshal(comics)
	fmt.Println("数据开始渲染：个数", len(cltComics))
	imageView := make([]widgetSeft.ImageView, 0, len(cltComics))
	var start int32 = 0
	wg := sync.WaitGroup{}
	for index, v := range cltComics {
		wg.Add(1)
		go func(index int, comic *types.Comic) {
			defer wg.Done()
			//自定义按钮组件,并且绑定特定事件
			btGroup := make([]fyne.CanvasObject, 0)
			seeBt := widget.NewButtonWithIcon("浏览", theme.VisibilityIcon(), c.seeButtonCallBack(comic))
			downloadBt := widget.NewButtonWithIcon("下载", theme.DownloadIcon(), c.downloadBt(comic))
			btGroup = append(btGroup, seeBt, downloadBt)
		tryCover:
			imageViewW := widgetSeft.NewImageView(c.ctx, comic.Title, comic.Cover, btGroup)
			contr, err := imageViewW.Renderer()
			if err != nil {
				goto tryCover
			}
			imageView = append(imageView, widgetSeft.ImageView{
				Index:     index,
				ImageView: imageViewW,
				Container: contr,
			})
			atomic.AddInt32(&start, 1)
			progressText := fmt.Sprintf("列表数据加载%.f%%", (gconv.Float64(start)/gconv.Float64(len(comics)))*100)
			label := widget.NewLabelWithStyle(progressText, fyne.TextAlignCenter, fyne.TextStyle{})
			views.VLayoutUi.Show(container.NewCenter(label))
		}(index, v)
	}
	wg.Wait()
	//切片数组排序 从大到小排序,并且处理模块，并且加载到容器上面
	sort.SliceStable(imageView, func(i, j int) bool {
		return imageView[i].Index > imageView[j].Index
	})
	canvasObj := make([]fyne.CanvasObject, 0, len(imageView))
	for _, v := range imageView {
		canvasObj = append(canvasObj, v.Container)
	}
	fmt.Println("数据结束渲染：个数", len(cltComics))
	return container.NewGridWithColumns(consts.Columns, canvasObj...)
}

func (c *comic) page() {
	box := container.NewVBox()
	var total int64
	if c.searchStr != "" {
		total, _ = modules.Boostrap.Db.Where("title like ?", c.searchStr+"%").Count(&model.Comic{})
	} else {
		total, _ = modules.Boostrap.Db.Count(&model.Comic{})
	}

	//第一步、初始化页码
	limit := consts.Columns * consts.Rows
	pageData := util.Paginator(c.currentPage, limit, total)
	pagesHbox := container.NewHBox()
	for i := 0; i < 5; i++ {
		pagesHbox.Add(widget.NewButton("1.html", nil))
	}
	currPagebd := binding.BindInt(&c.currentPage)
	currPagebd.AddListener(binding.NewDataListener(func() {
		garray.NewIntArrayFrom(pageData.Pages).Iterator(func(k, v int) bool {
			btn := pagesHbox.Objects[k].(*widget.Button)
			btn.SetText(gconv.String(v))
			btn.OnTapped = func() {
				btn.Importance = widget.LowImportance
				c.currentPage = v
				c.page()
			}
			return true
		})
	}))
	btnFirsData := widget.NewButton("首页", func() {
		c.currentPage = 1
		c.page()
	})
	btnPrev := widget.NewButton("上一页", func() {
		if c.currentPage < 2 {
			return
		}
		c.currentPage = c.currentPage - 1
		c.page()
	})
	btnNext := widget.NewButton("下一页", func() {
		if c.currentPage > pageData.Totalpages-1 {
			return
		}
		c.currentPage = c.currentPage + 1
		c.page()
	})
	btnLast := widget.NewButton("尾页", func() {
		c.currentPage = pageData.Totalpages
		c.page()
	})
	showLabel := widget.NewLabel(fmt.Sprintf("第%v页/总共%v页", c.currentPage, pageData.Totalpages))
	bottom := container.NewHBox(btnFirsData, btnPrev, pagesHbox, btnNext, btnLast, showLabel)
	if pageData.Totalpages < limit {
		bottom.Hide()
	} else {
		bottom.Show()
	}
	views.VLayoutUi.Page.RemoveAll()
	views.VLayoutUi.Page.Add(container.NewCenter(bottom))

	//第二步、 初始化数据
	list := c.getImageViews(c.searchStr, c.currentPage)
	box.Add(list)
	views.VLayoutUi.Show(box)

}

func (c *comic) downloadBt(comic *types.Comic) func() {
	return func() {
		dialog.ShowConfirm("温馨提示", "暂时不提供下载功能", func(b bool) {

		}, c.mainWm)
	}
}
