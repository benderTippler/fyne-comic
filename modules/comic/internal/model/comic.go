package model

import (
	"hei-windows/consts"
	"hei-windows/modules/comic/types"
)

type Comic struct {
	Id        int64  `json:"id"`
	Title     string `json:"title" xorm:"varchar(100) default('') notnull"`     // 标题
	DetailUrl string `json:"detailUrl" xorm:"varchar(250) default('') notnull"` //漫画详情d
	Md5Url    string `json:"md5Url" xorm:"varchar(50) default('') notnull"`     //漫画详情地址md5，确定唯一性
	Source    string `json:"source"`                                            //漫画采集来源 1.html、代表 md5(http://mangastream.mobi)
	Cover     string `json:"cover" xorm:"varchar(250) default('') notnull"`     //封面
	Language  string `json:"language" xorm:"varchar(250) default('') notnull"`  //漫画语言
}

func (c *Comic) Unmarshal(comics []*Comic) []*types.Comic {
	ctnComic := make([]*types.Comic, 0, len(comics))
	for _, v := range comics {
		comic := &types.Comic{
			Id:        v.Id,
			Title:     v.Title,
			DetailUrl: v.DetailUrl,
			Md5Url:    v.Md5Url,
			Source:    v.Source,
			Cover:     v.Cover,
			Language:  v.Language,
		}
		if v.Cover == "" {
			comic.Cover = consts.DefaultCover
		}
		ctnComic = append(ctnComic, comic)
	}
	return ctnComic
}

type Moudule struct {
	Id      int64  `json:"id"`
	Site    string `json:"Site" xorm:"varchar(100) default('') notnull unique"` //模块网站名称，md5后
	IsInit  int    `json:"IsInit" xorm:"default(0) notnull"`                    //0 未初始化，1.html、初始化已完毕 2、部分数据更新
	Moudule string `json:"moudule" xorm:"varchar(50) default('') notnull"`      //模块名称 comic代表漫画模块
	MaxPage int    `json:"maxPage" xorm:"default(0) notnull"`                   //当前网站的初始化后的最大页面，用于以后检查是否有新数据更新
}
