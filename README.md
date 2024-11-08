
#### 使用方法
```shell
wget http://8.137.107.46/dmp/run.sh && chmod +x run.sh && ./run.sh
```

#### 默认用户名密码
登录后请尽快修改密码

admin/123456


#### 文件介绍
```
.
├── dmp             # 主程序
├── dmp.log         # 日志
├── DstMP.sdb       # 数据库
└── run.sh          # 运行脚本
```

#### 项目介绍
```
.
├── app                                 # 接口
│   ├── auth                            # 鉴权模块，包含登录、菜单等
│   │   ├── handlers.go
│   │   ├── i18n.go
│   │   └── routes.go
│   ├── home                            # 主页模块
│   │   ├── handlers.go
│   │   ├── homeUtils.go
│   │   ├── i18n.go
│   │   └── routes.go
│   ├── logs                            # 日志模块
│   │   ├── handlers.go
│   │   ├── i18n.go
│   │   ├── logsUtils.go
│   │   └── routes.go
│   ├── setting                         # 设置模块
│   │   ├── handlers.go
│   │   ├── i18n.go
│   │   ├── routes.go
│   │   └── settingUtils.go
│   └── tools                           # 工具模块
│       ├── handlers.go
│       ├── i18n.go
│       ├── routes.go
│       └── toolsUtils.go
├── dist                                # 前端静态资源
├── DstMP.sdb                           # 数据库
├── go.mod
├── go.sum
├── LICENSE
├── main.go                             # 程序入口
├── README.md
├── scheduler                           # 定时任务模块
│   ├── init.go
│   ├── player.go
│   ├── schedulerUtils.go
│   └── tools.go
└── utils                               # 工具集
    ├── constant.go                     # 一些路径和命令的常量
    ├── exceptions.go                   # 异常返回（今后可能会弃用）
    ├── install.go                      # 预留
    ├── scripts.go                      # 需要执行的shell脚本
    └── utils.go                        # 工具函数
```