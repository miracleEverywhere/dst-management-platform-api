package utils

import (
	"flag"
)

var (
	BindPort    int
	DBPath      string
	LogLevel    string
	VersionShow bool
)

func BindFlags() {
	flag.IntVar(&BindPort, "bind", 80, "DMP端口, 如: -bind 8080")
	flag.StringVar(&DBPath, "dbpath", "./data", "数据库文件目录, 如: -dbpath ./data")
	flag.StringVar(&LogLevel, "level", "info", "日志等级, 如: -level debug")
	flag.BoolVar(&VersionShow, "v", false, "查看版本，如： -v")
	flag.Parse()
}
