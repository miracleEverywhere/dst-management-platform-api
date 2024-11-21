
## :watermelon: 使用方法(Usage)
>**建议使用 Ubuntu 24系统，低版本系统可能会出现GLIBC版本报错**  
>**It is recommended to use the Ubuntu 24 system, as lower version systems may experience GLIBC version errors**
```shell
# 执行以下命令，根据系统提示输入并回车
# Please execute the following command according to the system prompts, enter the input and press Enter.
cd ~ && wget https://dmp-1257278878.cos.ap-chengdu.myqcloud.com/run.sh && chmod +x run.sh && ./run.sh
```
**更新方法(Update)**
```shell
cd ~ && ./run.sh
```
_根据提示输入4 (Input 4 according to the prompt)_
```shell
# root@VM-0-16-ubuntu:~# cd ~ && ./run.sh
# 请输入需要执行的操作(Please enter the operation to be performed): 
# [0]: 下载并启动服务(Download and start the service) 
# [1]: 启动服务(Start the service) 
# [2]: 关闭服务(Stop the service) 
# [3]: 重启服务(Restart the service) 
# [4]: 更新服务(Update the service)
```
如果下载了发行版，则执行以下命令：(If the release-version bin-file has been downloaded, execute the following command:)
```shell
# -c 为开启日志，建议开启
# The -c option is for enabling logging, it is recommended to enable it.
nohup ./dmp -c 2>&1 > dmp.log &
```
默认启动端口为80，如果您想修改，则修改启动命令：(The default port is 80. If you wish to modify it, please modify the startup command:)
```shell
# 修改端口为8888
# Change the port to 8888.
nohup ./dmp -c -l 8888 2>&1 > dmp.log &
```

---

## :grapes: 默认用户名密码(Default username and password)
>登录后请尽快修改密码(After logging in, please change your password as soon as possible)
>
>>admin/123456

---

## :cherries: 平台截图(DMP screenshot)
![home-en](docs/images/home-en.png)


![home-zh](docs/images/home-zh.png)

![home-en](docs/images/mobile-zh.png)


![home-zh](docs/images/mobile-en.png)


![room-en](docs/images/room-en.png)


![room-zh](docs/images/room-zh.png)


![player-en](docs/images/player-en.png)


![player-zh](docs/images/player-zh.png)


![statistics-en](docs/images/statistics-en.png)


![statistics-zh](docs/images/statistics-zh.png)


![menu-tools-en](docs/images/menu-tools-en.png)


![menu-tools-zh](docs/images/menu-tools-zh.png)

---

## :strawberry: 文件介绍(File Introduction)
```text
.
├── dmp             # 主程序
├── dmp.log         # 请求日志
├── dmpProcess.log  # 运行日志
├── DstMP.sdb       # 数据库
└── run.sh          # 运行脚本
```

---

## :peach: 项目介绍(Project Introduction)
```text
.
├── app                                 # 接口
│   ├── auth                            # 鉴权模块，包含登录、菜单等
│   │   ├── handlers.go
│   │   ├── i18n.go
│   │   └── routes.go
│   ├── externalApi                     # 外部接口
│   │   ├── externalApiUtils.go
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
├── README.md                           # 帮助文档
├── scheduler                           # 定时任务模块
│   ├── init.go
│   ├── player.go
│   ├── schedulerUtils.go
│   └── tools.go
└── utils                               # 工具集
    ├── constant.go                     # 一些路径和命令的常量
    ├── exceptions.go                   # 异常返回（今后可能会弃用）
    ├── install.go                      # 安装脚本
    ├── logger.go                       # 日志
    ├── scripts.go                      # 需要执行的shell脚本
    └── utils.go                        # 工具函数
```
##  :sparkling_heart: 致谢
本项目[前端页面](https://github.com/miracleEverywhere/dst-management-platform-api)基于**koi-ui**二次开发，感谢开源 [@yuxintao6](https://github.com/yuxintao6)  
The [front-end page](https://github.com/miracleEverywhere/dst-management-platform-api) of this project is based on the secondary development of **koi-ui**, thanks to open source  
[[koi-ui gitee]](https://gitee.com/BigCatHome/koi-ui)  
[[koi-ui github]](https://github.com/yuxintao6/koi-ui)  
