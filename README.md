
#### 使用方法
```shell
# 执行以下命令，根据系统提示输入并回车
wget https://dmp-1257278878.cos.ap-chengdu.myqcloud.com/run.sh && chmod +x run.sh && ./run.sh
```
如果下载了发行版，则执行以下命令：
```shell
# -c 为开启日志，建议开启
nohup ./dmp -c 2>&1 > dmp.log &
```
默认启动端口为80，如果您想修改，则修改启动命令：
```shell
# 修改端口为8888
nohup ./dmp -c -l 8888 2>&1 > dmp.log &
```

---

#### 默认用户名密码
>登录后请尽快修改密码  
>  
>>admin/123456

---

#### 平台截图
![](http://8.137.107.46/dmp/home-en.png)
![](http://8.137.107.46/dmp/home-zh.png)
![](http://8.137.107.46/dmp/room-en.png)
![](http://8.137.107.46/dmp/room-zh.png)
![](http://8.137.107.46/dmp/player-en.png)
![](http://8.137.107.46/dmp/player-zh.png)
![](http://8.137.107.46/dmp/statistics-en.png)
![](http://8.137.107.46/dmp/statistics-zh.png)
![](http://8.137.107.46/dmp/menu-tools-en.png)
![](http://8.137.107.46/dmp/menu-tools-zh.png)
---

#### 文件介绍
```text
.
├── dmp             # 主程序
├── dmp.log         # 日志
├── DstMP.sdb       # 数据库
└── run.sh          # 运行脚本
```

---

#### 项目介绍
```text
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