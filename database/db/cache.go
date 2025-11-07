package db

import (
	"os"
)

var (
	JwtSecret  string
	CurrentDir string
)

func init() {
	setCurrentDir()
}

func setCurrentDir() {
	var err error
	CurrentDir, err = os.Getwd()
	if err != nil {
		panic("获取工作路径失败")
	}
}
