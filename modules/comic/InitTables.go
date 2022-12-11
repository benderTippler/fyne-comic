package comic

import (
	"hei-windows/modules/comic/internal/model"
)

var ComicBean = new(comic)

// 初始化对应漫画模块
type comic struct{}

func (c *comic) GetBeans() []interface{} {
	var beans = make([]interface{}, 0)
	beans = append(beans,
		model.Comic{},   //漫画主表
		model.Moudule{}, //模块化表
	)
	return beans
}
