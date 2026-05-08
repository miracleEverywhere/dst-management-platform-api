package server

import (
	"flag"
)

var (
	bindPort    int
	dbPath      string
	logLevel    string
	versionShow bool
	consoleCmd  string
	cert        string
	key         string
)

func bindFlags() {
	flag.IntVar(&bindPort, "bind", 80, "DMP端口, 如: -bind 8080")
	flag.StringVar(&dbPath, "dbpath", "./data", "数据库文件目录, 如: -dbpath ./data")
	flag.StringVar(&logLevel, "level", "info", "日志等级, 如: -level debug")
	flag.StringVar(&cert, "cert", "", "证书文件路径, 不填则启动http, 例如: /path/to/fullchain.pem")
	flag.StringVar(&key, "key", "", "私钥文件路径, 不填则启动http, 例如: /path/to/privkey.pem")
	flag.BoolVar(&versionShow, "v", false, "查看版本, 如: -v")
	flag.StringVar(&consoleCmd, "console", "", "控制台命令, 如: -console reset_password, -console list_user")
	flag.Parse()
}
