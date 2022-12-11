package modules

import (
	"github.com/xormplus/xorm/names"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xormplus/xorm"
)

func NewXormEngine(dbPath string) *xorm.Engine {
	var err error
	engine, err := xorm.NewSqlite3(dbPath)
	if err != nil {
		panic("数据库初始化失败: " + err.Error())
	}
	// 设置时区和数据库时区
	engine.TZLocation, _ = time.LoadLocation("Asia/Shanghai")
	engine.DatabaseTZ, _ = time.LoadLocation("Asia/Shanghai")

	// 控制台输出SQL语句
	engine.ShowSQL(false)

	// 设置连接池大小
	engine.SetMaxIdleConns(50)
	engine.SetMaxOpenConns(100)
	engine.SetMapper(names.GonicMapper{}) // 名称映射规则 驼峰式
	return engine
}
