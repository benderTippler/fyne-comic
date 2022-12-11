package types

// 漫画章节
type Chapter struct {
	Title      string
	ChapterUrl string
}

type Comic struct {
	Id        int64  `json:"id"`
	Title     string `json:"title"`     // 标题
	DetailUrl string `json:"detailUrl"` //漫画详情
	Md5Url    string `json:"md5Url"`    //漫画详情地址md5，确定唯一性
	Created   int64  `json:"updated"`
	Updated   int64  `json:"updated"`  // 更新时间
	Source    string `json:"source"`   //漫画采集来源 1.html、代表 http://mangastream.mobi
	IsStatus  int8   `json:"IsStatus"` //0 进行中 1.html 完结
	Cover     string `json:"cover"`    //封面
	Language  string `json:"language"`
}
