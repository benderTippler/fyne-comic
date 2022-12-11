package config

type Config struct {
	Usable   bool               `json:"usable"`   //应用是否可用，当前应用是否开启功能，是否下架这个应用程序
	Notice   string             `json:"notice"`   //公告
	Donate   string             `json:"donate"`   //打赏图片链接
	IUpdate  bool               `json:"isUpdate"` //是否强制更新
	IsDonate bool               `json:"isDonate"` //是否开启打赏
	DownLoad string             `json:"downLoad"` //最新下载链接
	Modules  map[string]*Module `json:"modules"`
	Email    string             `json:"email"`
	Version  string             `json:"version"`
}

type Module struct {
	SiteName   string   `json:"siteName"`   //网站名称
	Target     string   `json:"target"`     // "http://mangastream.mobi"
	ModuleName string   `json:"moduleName"` //模块名称
	Enable     int      `json:"enable"`     //1.html、启用模块 (初始化和启动检测是否有新增)  2、关闭模块（删除模块数据，不再提供数据）
	Source     string   `json:"source"`     //资源来源和目标地址一一对应
	Collect    *Collect `json:"collect"`    //采集器配置
	Language   string   `json:"language"`   //语言 en 英文漫画， ch 中文漫画
}

type Collect struct {
	Page     *Selector `json:"page"`     //页码选择器
	List     *Selector `json:"list"`     //列表选择器
	Chapter  *Selector `json:"chapter"`  //章节选择器
	Resource *Selector `json:"resource"` //资源选择器
}

// 自定义选择器
type Selector struct {
	DataType  string            `json:"dataType"` //html类型 json数据类型 script
	Query     map[string]string // css选择器
	ScriptStr string            // 脚本拼接字符串  专门用于 script
	Api       *Api              //用于json数据
	Prefix    string            //链接前缀
	Url       string            //目标链接地址
	QueryType string            //选择器列类型
	Limiter   string            //分隔符
	Referer   string            //防盗链接
}

// 用于json格式类型
type Api struct {
	Url          string
	Method       string            //请求方式 GET 或者 POST
	FilesMap     map[string]string //字段提取
	Params       string            //请求参数
	Header       map[string]string //请求头部
	ParamsSource string
	ScriptStr    string
	Query        map[string]string // css选择器
}

type Result struct {
	Code int
	Data string
}
