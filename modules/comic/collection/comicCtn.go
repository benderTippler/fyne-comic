package collection

import (
	"context"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/gocolly/colly"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gclient"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"hei-windows/config"
	"hei-windows/modules"
	"hei-windows/modules/comic/internal/model"
	"hei-windows/modules/comic/types"
	"hei-windows/widget"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

var (
	maxTaskNum             = 20
	taskChan               = make(chan bool, maxTaskNum)
	wg                     = sync.WaitGroup{}
	comicSet               = gset.New(true)
	shareNums              = 500
	taskInsertChan         = make(chan bool, 20)
	module                 = "comic"
	start          float64 = 0
)

type comicCtn struct {
	ctx         context.Context
	module      *config.Module
	progressBar *widget.CustomProgressBar
}

func NewComicCtn(ctx context.Context, module *config.Module, progressBar *widget.CustomProgressBar) *comicCtn {
	return &comicCtn{
		ctx:         ctx,
		module:      module,
		progressBar: progressBar,
	}
}

// 启动初始化采集器
func (c *comicCtn) StartUp(views *widget.CustomProgressBar) bool {
	//清除对应模块数据
	_, err := modules.Boostrap.Db.Where("source = ?", c.module.Source).Delete(model.Comic{})
	if err != nil {
		title := fmt.Sprintf("收集%v网站信息出现未知错误,请关闭重新启动！", c.module.SiteName)
		views.SetDialogTitle(title)
		return false
	}
	maxPage, err := c.GetMaxPage()
	if err != nil {
		title := fmt.Sprintf("收集%v网站信息出现未知错误,请关闭重新启动！", c.module.SiteName)
		views.SetDialogTitle(title)
		return false
	}
	views.SetProgressData(0.05, 1)
	var intProgress int32 = 0
	//第一步、循环采集数据 0.5
	title := fmt.Sprintf("( ^_^ ) 正在收集%v网站信息,请耐心等待！", c.module.SiteName)
	views.SetDialogTitle(title)
	for startPage := 1; startPage <= maxPage; startPage++ {
		wg.Add(1)
		taskChan <- true
		go func(startPage int) {
			defer wg.Done()
		tryOne:
			comic, err := c.GetComicList(startPage)
			if err != nil {
				fmt.Println("采集失败", startPage)
				goto tryOne
			}
			for _, v := range comic {
				comicSet.Add(v)
			}
			<-taskChan
			atomic.AddInt32(&intProgress, 1)
			views.SetProgressData((gconv.Float64(intProgress)/gconv.Float64(maxPage))*0.5+0.05, 1)
		}(startPage)
	}
	wg.Wait()
	//第二步、数据入库 0.2
	intProgress = 0
	title = fmt.Sprintf("( ^_^ ) 正在将%v网站数据同步到本地,请耐心等待！", c.module.SiteName)
	views.SetDialogTitle(title)
	wgInsert := sync.WaitGroup{}
	sliceS := c.cutStringSlice(comicSet.Slice(), shareNums)
	for i, v := range sliceS {
		wgInsert.Add(1)
		taskInsertChan <- true
		go func(index int, v []interface{}) {
			defer wgInsert.Done()
		tryInsert:
			_, err = modules.Boostrap.Db.Insert(v)
			if err != nil {
				goto tryInsert
			}
			<-taskInsertChan
			atomic.AddInt32(&intProgress, 1)
			views.SetProgressData((gconv.Float64(intProgress)/gconv.Float64(len(sliceS)))*0.2+0.6, 1)
		}(i, v)
	}
	wgInsert.Wait()
	comicSet.Clear()
	siteMd5 := c.module.Source
	moduleBean := &model.Moudule{
		Site:    siteMd5,
		IsInit:  1,
		Moudule: module,
		MaxPage: maxPage,
	}
	isExc := true
	isHas, err := modules.Boostrap.Db.Where("Site = ?", siteMd5).Exist(moduleBean)
	if !isHas {
		_, err = modules.Boostrap.Db.Insert(moduleBean)
		if err != nil {
			isExc = false
		}
	} else {
		_, err = modules.Boostrap.Db.Where("Site = ?", siteMd5).Update(moduleBean)
		if err != nil {
			isExc = false
		}
	}
	if isExc {
		title = fmt.Sprintf("( ^_^ ), %v网站数据同步成功，请搜索自己喜欢的漫画吧! ", c.module.SiteName)
		views.SetProgressData(1, 1)
	} else {
		title = fmt.Sprintf("( ^_^ ), %v网站数据同步失败,请重试！", c.module.SiteName)
		views.SetProgressData(0, 1)
		return isExc
	}
	views.SetDialogTitle(title)
	return true
}

// 获取最大页码
func (c *comicCtn) GetMaxPage() (int, error) {
	var maxPage int
	Page := c.module.Collect.Page
	//json 接口请求
	switch Page.DataType {
	case "jsonp": //解析接口jsonp数据结构
		api := Page.Api
		client := g.Client()
		method := gstr.ToLower(api.Method)
		var rep *gclient.Response
		var err error
		switch method {
		case "get":
			rep, err = client.Get(c.ctx, api.Url)
			defer rep.Close()
			break
		case "post":
			rep, err = client.Post(c.ctx, api.Url, api.Params)
			defer rep.Close()
			break
		default:
			return 0, errors.New("请求方式不允许")
			break
		}

		if err != nil {
			return maxPage, err
		}
		callback := rep.Request.URL.Query().Get("callback")
		bodyStr := rep.ReadAllString()
		bodyStr = gstr.TrimLeft(gstr.TrimLeft(gstr.TrimRight(bodyStr, ");"), callback), "(")
		jsonResult := gjson.New(bodyStr)
		for k, v := range api.FilesMap {
			if k == "PageCount" {
				items := gstr.Split(v, "|")
				if len(items) == 2 {
					maxPage = jsonResult.Get(items[0]).Int()
				} else {
					maxPage = jsonResult.Get(v).Int()
				}
			}
		}
		break
	case "json": //解析接口json数据结构
		api := Page.Api
		client := g.Client()
		method := gstr.ToLower(api.Method)
		var rep *gclient.Response
		var err error
		switch method {
		case "get":
			rep, err = client.Get(c.ctx, api.Url)
			defer rep.Close()
			break
		case "post":
			rep, err = client.Post(c.ctx, api.Url)
			defer rep.Close()
			break
		default:
			return 0, errors.New("请求方式不允许")
			break
		}

		if err != nil {
			return maxPage, err
		}
		bodyStr := rep.ReadAllString()
		jsonResult := gjson.New(bodyStr)
		for k, v := range api.FilesMap {
			if k == "PageCount" {
				items := gstr.Split(v, "|")
				if len(items) == 2 {
					maxPage = gconv.Int(math.Ceil(jsonResult.Get(items[0]).Float64() / gconv.Float64(items[1])))
				} else {
					maxPage = jsonResult.Get(v).Int()
				}
			}
		}
		break
	case "html":
		cy := colly.NewCollector()
		Node := Page.Query["Node"]
		cy.OnHTML(Node, func(e *colly.HTMLElement) {
			sampleRegexp := regexp.MustCompile(`(\+)?\d+(\.\d+)?`)
			chapter := sampleRegexp.FindString(e.Text)
			maxPage = gconv.Int(chapter)
		})
		err := cy.Visit(Page.Url)
		if err != nil {
			return maxPage, err
		}
		break
	case "check":
		maxPage = gconv.Int(Page.Query["StartEnd"])
		var isCheckEnd bool
		for {
			if isCheckEnd {
				break
			}
			maxPage++
			cy := colly.NewCollector()
			Node := Page.Query["List"]
			cy.OnHTML(Node, func(e *colly.HTMLElement) {
				if e.DOM.Children().Size() == 0 {
					isCheckEnd = true
				}
			})
			cy.Visit(fmt.Sprintf(Page.Url, maxPage))
		}
		break
	}
	return maxPage, nil
}

// 获取列表信息
func (c *comicCtn) GetComicList(page int) ([]*model.Comic, error) {
	comicList := make([]*model.Comic, 0)
	List := c.module.Collect.List

	//json 接口请求
	switch List.DataType {
	case "jsonp": //解析接口jsonp数据结构
		targetUrl := fmt.Sprintf(List.Api.Url, page)
		api := List.Api
		client := g.Client()
		method := gstr.ToLower(api.Method)
		var rep *gclient.Response
		var err error
		switch method {
		case "get":
			rep, err = client.Get(c.ctx, targetUrl)
			defer rep.Close()
			break
		case "post":
			rep, err = client.Post(c.ctx, targetUrl, api.Params)
			defer rep.Close()
			break
		default:
			return comicList, errors.New("请求方式不允许")
			break
		}

		if err != nil {
			return comicList, err
		}
		callback := rep.Request.URL.Query().Get("callback")
		bodyStr := rep.ReadAllString()
		bodyStr = gstr.TrimLeft(gstr.TrimLeft(gstr.TrimRight(bodyStr, ");"), callback), "(")
		jsonResult := gjson.New(bodyStr)

		var list *gvar.Var
		for k, v := range api.FilesMap {
			if k == "List" {
				list = jsonResult.Get(v)
			}
		}

		for _, vlist := range list.Maps() {
			comic := &model.Comic{}

			title := vlist[api.FilesMap["Title"]]
			comic.Title = title.(string)

			var cover interface{}
			itemsCover := gstr.Split(api.FilesMap["Cover"], "|")
			if len(itemsCover) == 2 {
				cover = itemsCover[1] + vlist[itemsCover[0]].(string)
			} else {
				cover = vlist[itemsCover[0]]
			}

			comic.Cover = cover.(string)

			var detailUrl interface{}
			items := gstr.Split(api.FilesMap["DetailUrl"], "|")
			if len(items) == 2 {
				if !gstr.ContainsI(vlist[items[0]].(string), "http") {
					detailUrl = items[1] + vlist[items[0]].(string)
				} else {
					detailUrl = vlist[items[0]].(string)
				}
			} else {
				detailUrl = vlist[items[0]]
			}
			fmt.Println(vlist["comic_url"].(string))

			comic.DetailUrl = detailUrl.(string)

			comic.Md5Url = gmd5.MustEncryptString(comic.DetailUrl)
			comic.Source = gmd5.MustEncryptString(c.module.Target)

			comic.Language = c.module.Language
			comicList = append(comicList, comic)
		}

		break
	case "html":
		url := fmt.Sprintf(List.Url, page)
		cy := colly.NewCollector()
		cy.SetRequestTimeout(10 * time.Minute)
		mhm := make([]*model.Comic, 0)
		ListS := List.Query["List"]
		Node := List.Query["Node"]
		Title := List.Query["Title"]
		Cover := List.Query["Cover"]
		DetailUrl := List.Query["DetailUrl"]
		cy.OnHTML(ListS, func(e *colly.HTMLElement) {
			e.ForEach(Node, func(i int, item *colly.HTMLElement) {
				var cover, detailUrl string
				title := item.ChildText(Title)
				itemsTitle := gstr.Split(Cover, "|")
				if len(itemsTitle) == 2 {
					cover = itemsTitle[1] + item.ChildAttr(itemsTitle[0], "src")
				} else {
					cover = item.ChildAttr(itemsTitle[0], "src")
				}

				itemsDetailUrl := gstr.Split(DetailUrl, "|")
				if len(itemsDetailUrl) == 2 {
					detailUrl = itemsDetailUrl[1] + item.ChildAttr(itemsDetailUrl[0], "href")
				} else {
					detailUrl = item.ChildAttr(itemsDetailUrl[0], "href")
				}

				mhm = append(mhm, &model.Comic{
					Title:     title,
					Cover:     cover,
					DetailUrl: detailUrl,
					Source:    c.module.Source,
					Language:  c.module.Language,
					Md5Url:    gmd5.MustEncryptString(detailUrl),
				})
			})
		})
		err := cy.Visit(url)
		return mhm, err
		break
	case "json": //解析接口jsonp数据结构
		targetUrl := fmt.Sprintf(List.Api.Url, page)
		api := List.Api
		client := g.Client()
		method := gstr.ToLower(api.Method)
		var rep *gclient.Response
		var err error
		switch method {
		case "get":
			rep, err = client.Get(c.ctx, targetUrl)
			defer rep.Close()
			break
		case "post":
			rep, err = client.Post(c.ctx, targetUrl)
			defer rep.Close()
			break
		default:
			return comicList, errors.New("请求方式不允许")
			break
		}

		if err != nil {
			return comicList, err
		}
		bodyStr := rep.ReadAllString()
		jsonResult := gjson.New(bodyStr)

		var list *gvar.Var
		for k, v := range api.FilesMap {
			if k == "List" {
				list = jsonResult.Get(v)
			}
		}

		for _, vlist := range list.Maps() {
			comic := &model.Comic{}

			title := vlist[api.FilesMap["Title"]]
			comic.Title = title.(string)

			var cover interface{}
			itemsCover := gstr.Split(api.FilesMap["Cover"], "|")
			if len(itemsCover) == 2 {
				cover = itemsCover[1] + vlist[itemsCover[0]].(string)
			} else {
				cover = vlist[itemsCover[0]]
			}

			comic.Cover = cover.(string)

			var detailUrl interface{}
			items := gstr.Split(api.FilesMap["DetailUrl"], "|")
			if len(items) == 2 {
				detailUrl = fmt.Sprintf("%v%v", items[1], vlist[items[0]])
			} else {
				detailUrl = vlist[items[0]]
			}

			comic.DetailUrl = detailUrl.(string)

			comic.Md5Url = gmd5.MustEncryptString(comic.DetailUrl)
			comic.Source = c.module.Source

			comic.Language = c.module.Language
			comicList = append(comicList, comic)
		}

		break
	}

	return comicList, nil
}

// 获取漫画的章节
func (c *comicCtn) GetComicChapters(url string) ([]*types.Chapter, error) {

	//一些网站很坑人，格式不统一化，这里做兼容性处理  开始

	if gstr.Contains(url, "www.dmzj.com") { //这里特殊处理漫画之家  新版网站
		chapters := make([]*types.Chapter, 0)
		cy := colly.NewCollector()
		cy.OnHTML("div.tab-content:nth-child(3) > ul", func(e *colly.HTMLElement) {
			e.ForEach("li", func(i int, item *colly.HTMLElement) {
				chapterUrl := item.ChildAttr("a", "href")
				title := item.ChildText(".list_con_zj")
				chapters = append(chapters, &types.Chapter{
					Title:      title,
					ChapterUrl: chapterUrl,
				})
			})
		})
		err := cy.Visit(url)
		return chapters, err
	}

	//一些网站很坑人，格式不统一化，这里做兼容性处理  结束

	chapters := make([]*types.Chapter, 0)
	Chapter := c.module.Collect.Chapter
	switch Chapter.DataType {
	case "script":
		cy := colly.NewCollector()
		cy.OnHTML(Chapter.Query["Script"], func(e *colly.HTMLElement) {
			all := e.DOM.Text()
			all += Chapter.ScriptStr
			vm := goja.New()
			all = gstr.ReplaceByMap(all, map[string]string{
				"window.__NUXT__": "window",
			})
			_, err := vm.RunString(all)
			if err != nil {
				return
			}
			List := vm.Get("List").Export().([]interface{})
			for _, mv := range List {
				chapters = append(chapters, &types.Chapter{
					Title:      mv.(map[string]interface{})["Title"].(string),
					ChapterUrl: mv.(map[string]interface{})["ChapterUrl"].(string),
				})
			}
		})
		err := cy.Visit(url)
		if err != nil {
			return chapters, err
		}
		break
	case "jsonp":
		break
	case "json":
		break
	case "html":
		cy := colly.NewCollector()
		List := Chapter.Query["List"]
		Node := Chapter.Query["Node"]
		cy.OnHTML(List, func(e *colly.HTMLElement) {
			e.ForEach(Node, func(i int, item *colly.HTMLElement) {
				var title, chapterUrl string
				if Node == Chapter.Query["ChapterUrl"] {
					chapterUrl = item.Attr("href")
				} else {
					chapterUrl = item.ChildAttr(Chapter.Query["ChapterUrl"], "href")
				}
				if Node == Chapter.Query["Title"] {
					title = gstr.ReplaceByMap(item.Text, map[string]string{
						"  ": "",
						"話":  "话",
					})
				} else {
					title = item.ChildText(Chapter.Query["Title"])
				}
				chapters = append(chapters, &types.Chapter{
					Title:      title,
					ChapterUrl: c.module.Collect.Chapter.Prefix + chapterUrl,
				})
			})
		})
		err := cy.Visit(url)
		if err != nil {
			return nil, err
		}
		break
	}
	return chapters, nil
}

// 获取 章节对应的漫画资源
func (c *comicCtn) GetResource(url string) ([]string, error) {
	resource := make([]string, 0)
	//一些网站很坑人，格式不统一化，这里做兼容性处理  开始
	if gstr.Contains(url, "www.dmzj.com") { //这里特殊处理漫画之家  新版网站
		var ImageList = make([]interface{}, 0)
		cy := colly.NewCollector()
		cy.OnHTML("head > script:nth-child(11)", func(e *colly.HTMLElement) {
			all := e.DOM.Text()
			all += `
    var img_prefix = 'https://images.dmzj.com/';
    pages = pages.replace(/\n/g,"");
    pages = pages.replace(/\r/g,"|");
    var info = eval("(" + pages + ")");
    var imageS = (info["page_url"].split('|'))
    var List = new Array()
    imageS.forEach(function(element){
        List.push(img_prefix+element)
    })
`
			vm := goja.New()
			vm.RunString(all)
			ImageList = vm.Get("List").Export().([]interface{})
			for _, value := range ImageList {
				resource = append(resource, value.(string))
			}

		})
		err := cy.Visit(url)
		return resource, err
	}

	//http://mangabz.com 特殊处理
	if gstr.ContainsI(url, "mangabz.com") {
		var js string
		cy := colly.NewCollector()
		cy.OnHTML(c.module.Collect.Resource.Query["Script"], func(e *colly.HTMLElement) {
			js = e.Text
		})
		cy.Visit(url)
		js = gstr.ReplaceByMap(js, map[string]string{
			"reseturl(window.location.href, MANGABZ_CURL.substring(0, MANGABZ_CURL.length - 1));": "",
		})

		vm := goja.New()
		_, err := vm.RunString(js)
		if err != nil {
			return resource, err
		}
		domain := vm.Get("MANGABZ_COOKIEDOMAIN").Export().(string)
		curl := vm.Get("MANGABZ_CURL").Export().(string)
		cid := vm.Get("MANGABZ_CID").Export().(int64)
		maxPage := vm.Get("MANGABZ_IMAGE_COUNT").Export().(int64)
		dt := vm.Get("MANGABZ_VIEWSIGN_DT").Export().(string)
		sign := vm.Get("MANGABZ_VIEWSIGN").Export().(string)
		mid := vm.Get("MANGABZ_MID").Export()
		var intProgress int32
		taskImgChan := make(chan bool, 20)
		wg2 := sync.WaitGroup{}
		for page := int64(1); page <= maxPage; page++ {
			wg2.Add(1)
			taskImgChan <- true
			go func(page int64) {
				defer wg2.Done()
			tryOne:
				dierc := fmt.Sprintf("http://%v%vchapterimage.ashx?cid=%v&page=%v&key=&_cid=%v&_mid=%v&_dt=%v&_sign=%v", domain, curl, cid, page, cid, mid, dt, sign)
				method := "GET"
				client := &http.Client{
					Timeout: 10 * time.Minute,
				}
				req, err := http.NewRequest(method, dierc, nil)
				if err != nil {
					fmt.Println(err)
					goto tryOne
				}
				req.Header.Add("Referer", "http://mangabz.com/")
				res, err := client.Do(req)
				if err != nil {
					fmt.Println(err)
					goto tryOne
				}
				defer res.Body.Close()
				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					fmt.Println(err)
					goto tryOne
				}
				vm2 := goja.New()
				_, err = vm2.RunString(fmt.Sprintf("var List = %v", gstr.Trim(string(body))))
				if err != nil {
					fmt.Println(err)
					goto tryOne
				}
				list := vm2.Get("List").Export()
				for _, v := range list.([]interface{}) {
					resource = append(resource, v.(string))
				}
				<-taskImgChan
				atomic.AddInt32(&intProgress, 1)
				c.progressBar.SetDialogTitle(fmt.Sprintf("正在采集%v张图片加载完成", page))
				c.progressBar.SetProgressData((gconv.Float64(intProgress)/gconv.Float64(maxPage))*0.5, 1)
				fmt.Println("结束解析第", page, "页", "总共", maxPage)
			}(page)
		}
		wg2.Wait()
		return resource, nil
	}
	//一些网站很坑人，格式不统一化，这里做兼容性处理  结束

	Resource := c.module.Collect.Resource
	switch Resource.DataType {
	case "script":
		cy := colly.NewCollector()
		cy.OnHTML(Resource.Query["Script"], func(e *colly.HTMLElement) {
			all := e.DOM.Text()
			all += Resource.ScriptStr
			vm := goja.New()
			all = gstr.ReplaceByMap(all, map[string]string{
				"window.__NUXT__": "window",
			})
			_, err := vm.RunString(all)
			if err != nil {
				return
			}
			List := vm.Get("List").Export().([]interface{})
			for _, v := range List {
				resource = append(resource, gconv.String(v))
			}
		})
		err := cy.Visit(url)
		if err != nil {
			return resource, err
		}
		break
	case "jsonp":
		break
	case "json":
		var params string = "" //请求参数
		api := Resource.Api
		client := g.Client()
		method := gstr.ToLower(api.Method)
		var rep *gclient.Response
		var err error
		if api.ParamsSource == "script" {
			cy := colly.NewCollector()
			cy.OnHTML(api.Query["Script"], func(e *colly.HTMLElement) {
				all := e.DOM.Text()
				all += api.ScriptStr
				vm := goja.New()
				all = gstr.ReplaceByMap(all, map[string]string{
					"document.getElementById": "",
				})
				_, err = vm.RunString(all)
				if err != nil {
					return
				}
				params = vm.Get("Params").Export().(string)
			})
			err = cy.Visit(url)
			if err != nil {
				return resource, err
			}
		}
		switch method {
		case "get":
			rep, err = client.Get(c.ctx, api.Url, params)
			defer rep.Close()
			break
		case "post":
			rep, err = client.Post(c.ctx, api.Url, params)
			defer rep.Close()
			break
		default:
			return resource, errors.New("请求方式不允许")
			break
		}
		jsonResult := gjson.New(rep.ReadAllString())
		var list *gvar.Var
		var listType string
		for k, v := range api.FilesMap {
			if k == "List" {
				item := gstr.Split(v, "|")
				list = jsonResult.Get(item[0])
				if len(item) == 2 {
					listType = item[1]
				}
			}
		}
		if listType == "slice" {
			resource = list.Strings()
		} else { //其他类型处理

		}
		return resource, nil
		break
	case "html":
		cy := colly.NewCollector()
		List := Resource.Query["List"]
		if Resource.QueryType == "standard" {
			Node := Resource.Query["Node"]
			ImgUrl := Resource.Query["ImgUrl"]
			cy.OnHTML(List, func(e *colly.HTMLElement) {
				e.ForEach(Node, func(i int, element *colly.HTMLElement) {
					var src string
					if ImgUrl != "" {
						src = element.ChildAttr(ImgUrl, "src")
						if src == "" {
							src = element.ChildAttr(ImgUrl, Resource.Query["Attr"])
						}
					} else {
						src = element.Attr("src")
						if src == "" {
							src = element.Attr(Resource.Query["Attr"])
						}
					}
					resource = append(resource, src)
				})
			})
		} else if Resource.QueryType == "text" {
			cy.OnHTML(List, func(e *colly.HTMLElement) {
				resource = gstr.Explode(Resource.Limiter, e.Text)
			})
		}
		err := cy.Visit(url)
		if err != nil {
			return resource, err
		}
		break
	}
	//速度很快，直接标记50%
	c.progressBar.SetProgressData(0.5, 1)
	return resource, nil
}

// 切片分割
func (c *comicCtn) cutStringSlice(slice []interface{}, shareNums int) [][]interface{} {
	sliceLen := len(slice)
	if sliceLen == 0 {
		panic("ccc")
		return nil
	}
	totalShareNums := math.Ceil(float64(sliceLen) / float64(shareNums))
	resSlice := make([][]interface{}, 0, int(totalShareNums))

	for i := 0; i < sliceLen; i += shareNums {
		endIndex := i + shareNums
		if endIndex > sliceLen {
			endIndex = sliceLen
		}
		resSlice = append(resSlice, slice[i:endIndex])
	}
	return resSlice
}
