package server

import (
	"flag"
)

var (
	bindPort    int
	dbPath      string
	logLevel    string
	versionShow bool
)

func bindFlags() {
	flag.IntVar(&bindPort, "bind", 80, "DMP端口, 如: -bind 8080")
	flag.StringVar(&dbPath, "dbpath", "./data", "数据库文件目录, 如: -dbpath ./data")
	flag.StringVar(&logLevel, "level", "info", "日志等级, 如: -level debug")
	flag.BoolVar(&versionShow, "v", false, "查看版本，如： -v")
	flag.Parse()
}
