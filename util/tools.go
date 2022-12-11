package util

import (
	"os"
	"os/user"
)

func GetConfigDir() string {
	// 获取配置文件夹路径路径
	userInfo, err := user.Current()
	if err != nil {
		panic("百宝箱 配置文件夹路径获取失败" + err.Error())
	}
	var homeDir = userInfo.HomeDir
	// 判断 homeDir/GTools 文件夹是否存在
	var gtDir = homeDir + "/hei-box"
	_, err = os.Stat(gtDir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(gtDir, os.ModePerm)
			if err != nil {
				panic("百宝箱配置文件夹创建失败")
			}
		} else {
			panic("百宝箱配置文件夹不存在--" + err.Error())
		}
	}
	return gtDir
}
