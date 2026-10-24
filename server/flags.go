package server

import (
	"flag"
	"strings"
)

var (
	bindPort          int
	dbPath            string
	logLevel          string
	versionShow       bool
	consoleCmd        string
	cert              string
	key               string
	noHttpCompression bool
	pprof             bool
	stexInput         stringList
	stexOutput        string
)

// stringList 让同一个 flag 可以重复指定，用于 -console stex 的多份图集。
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func bindFlags() {
	flag.IntVar(&bindPort, "bind", 80, "DMP端口, 如: -bind 8080")
	flag.BoolVar(&noHttpCompression, "no-http-compress", false, "http响应gzip压缩文本数据，默认启动，关闭使用：-no-http-compress")
	flag.StringVar(&dbPath, "dbpath", "data", "数据库文件目录, 如: -dbpath ./data")
	flag.StringVar(&logLevel, "level", "info", "日志等级, 如: -level debug")
	flag.StringVar(&cert, "cert", "", "证书文件路径, 不填则启动http, 例如: /path/to/fullchain.pem")
	flag.StringVar(&key, "key", "", "私钥文件路径, 不填则启动http, 例如: /path/to/privkey.pem")
	flag.BoolVar(&versionShow, "v", false, "查看版本, 如: -v")
	flag.BoolVar(&pprof, "pprof", false, "是否开启pprof")
	flag.StringVar(&consoleCmd, "console", "", "控制台命令, 如: -console help")
	flag.Var(&stexInput, "i", "stex输入，可重复指定多个xml, 如: -i a.xml -i b.xml")
	flag.StringVar(&stexOutput, "o", "", "stex输出目录, 如: -o dmp_files/dst_images")
	flag.Parse()
}
