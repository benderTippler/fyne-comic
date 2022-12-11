package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/crypto/gaes"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/encoding/gbase64"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcache"
	"github.com/xormplus/xorm"
	"hei-windows/config"
	"hei-windows/consts"
	"hei-windows/modules/comic"
	"hei-windows/util"
	"net"
	"time"
)

var GlobalConfig = new(config.Config)

var Boostrap = new(app)

type app struct {
	ctx   context.Context
	Db    *xorm.Engine
	Cache *gcache.Cache
}

func (app *app) Run(ctx context.Context) error {
	//初始数据库文件
	app.ctx = ctx
	dbFile := fmt.Sprintf("%v/hei-box.db", util.GetConfigDir())
	app.Db = NewXormEngine(dbFile)
	//初始化数据库表
	for _, v := range comic.ComicBean.GetBeans() {
		app.Db.Sync2(v)
	}
	//初始化内存缓存
	app.Cache = gcache.New()
	return nil
}

// 配置文件读取
func (app *app) LoadConfig(ctx context.Context) error {
	//初始化全局配置
	rep, err := g.Client().Get(ctx, consts.ServiceApi)
	if err != nil {
		return errors.New("应用验证服务器访问不通，请检查网络!")
	}

	var result = &config.Result{}

	err = json.Unmarshal(rep.ReadAll(), result)
	if err != nil {
		return errors.New("服务器数据格式错误")
	}

	if result.Code != 200 {
		return errors.New("服务器数据加密失败")
	}

	aesStr, err := gbase64.Decode([]byte(result.Data))
	if err != nil {
		return errors.New("服务器数据解密失败")
	}

	data, err := gaes.Decrypt(aesStr, []byte(gmd5.MustEncryptString(consts.Key)))
	if err != nil {
		return errors.New("服务器数据解密失败")
	}

	err = json.Unmarshal(data, GlobalConfig)
	if err != nil {
		return errors.New("服务器数据序列化失败")
	}
	return nil
}

// 网络是否连通
func (app *app) NetWorkStatus(site string) bool {
	_, err := net.DialTimeout("tcp", fmt.Sprintf("%v:443", site), time.Duration(1*time.Minute))
	if err != nil {
		return false
	}
	return true
}
