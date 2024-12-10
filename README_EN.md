
## :watermelon: Usage
>**It is recommended to use the Ubuntu 24 system, as lower version systems may experience GLIBC version errors**
```shell
# Please execute the following command according to the system prompts, enter the input and press Enter.
cd ~ && wget https://dmp-1257278878.cos.ap-chengdu.myqcloud.com/run.sh && chmod +x run.sh && ./run.sh
```
**Update**
```shell
cd ~ && ./run.sh
```
_Input 4 according to the prompt_
```shell
# root@VM-0-16-ubuntu:~# cd ~ && ./run.sh
# 请输入需要执行的操作(Please enter the operation to be performed): 
# [0]: 下载并启动服务(Download and start the service) 
# [1]: 启动服务(Start the service) 
# [2]: 关闭服务(Stop the service) 
# [3]: 重启服务(Restart the service) 
# [4]: 更新服务(Update the service)
```
If the release-version bin-file has been downloaded, execute the following command:
```shell
# The -c option is for enabling logging, it is recommended to enable it.
nohup ./dmp -c > dmp.log 2>&1 &
```
The default port is 80. If you wish to modify it, please modify the startup command:
```shell
# Change the port to 8888.
nohup ./dmp -c -l 8888 > dmp.log 2>&1 &
```
You can also specify the storage directory for the database file  
```shell
# Enable console output, listen on port 8899, and set the storage location of DstMP.sdb to ./config/DstMP.sdb
nohup ./dmp -c -l 8899 -s ./config > dmp.log 2>&1 &
```
**Docker deployment**  
First, obtain the Docker image tag from the package page
```shell
# Bing port 80
docker run -p 80:80 -v /app/dmp/config:/root/config --name dmp -itd ghcr.io/miracleeverywhere/dst-management-platform-api:tag
```
```shell
# Bing port 8000
docker run -p 8000:80 -v /app/dmp/config:/root/config --name dmp -itd ghcr.io/miracleeverywhere/dst-management-platform-api:tag
```
---

## :grapes: Default username and password
>After logging in, please change your password as soon as possible
>
>>admin/123456

---

## :cherries: DMP screenshot
![home-en](docs/images/home-en.png)
  

![home-zh](docs/images/mobile-en.png)
  

![room-en](docs/images/room-en.png)
  

![player-en](docs/images/player-en.png)
  

![statistics-en](docs/images/statistics-en.png)
  

![menu-tools-en](docs/images/menu-tools-en.png)
  

---

## :strawberry: File Introduction
```text
.
├── dmp                 # Main
├── dmp.log             # Access Log
├── dmpProcess.log      # Runtime Log
├── DstMP.sdb           # Database
├── manual_install.sh   # DST manual install script
└── run.sh              # startup script
```

---

## :peach: Project Introduction
```text
.
├── app
│   ├── auth                    # Auth Module
│   ├── externalApi             # External Api
│   ├── home                    # Home Page
│   ├── logs                    # DST logs
│   ├── setting                 # Settings
│   └── tools                   # Tools
├── dist                        # Static Resources
│   ├── assets 
│   ├── index.html
│   ├── index.html.gz
│   └── vite.png
├── docker                      # Docker
│   ├── Dockerfile
│   └── entry-point.sh
├── docs                        # Docs
│   └── images
├── DstMP.sdb                   # Database
├── go.mod
├── go.sum
├── LICENSE
├── main.go
├── README.md
├── scheduler                   # Scheduler Tasks
│   ├── init.go
│   └── schedulerUtils.go
└── utils                       # Utils
    ├── constant.go
    ├── exceptions.go
    ├── install.go
    ├── logger.go
    ├── scripts.go
    └── utils.go
```
##  :sparkling_heart: Thanks
The [front-end page](https://github.com/miracleEverywhere/dst-management-platform-web) of this project is based on the secondary development of **koi-ui**, thanks to open source  
[[koi-ui gitee]](https://gitee.com/BigCatHome/koi-ui)  
[[koi-ui github]](https://github.com/yuxintao6/koi-ui)  
