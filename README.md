# [中文文档] | [[README]](docs/README_EN.md)

# DMP 饥荒管理平台
![GitHub Release](https://img.shields.io/github/v/release/miracleEverywhere/dst-management-platform-api)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/miracleEverywhere/dst-management-platform-api/total)
![GitHub commit activity](https://img.shields.io/github/commit-activity/t/miracleEverywhere/dst-management-platform-api)
![GitHub License](https://img.shields.io/github/license/miracleEverywhere/dst-management-platform-api)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/miracleEverywhere/dst-management-platform-api)
<div>
    <img src="docs/images/logo.svg" height="100" alt="DMP"/>
    <h3>:sparkling_heart: 支持多房间、多用户、多权限管理 :sparkling_heart:</h3>
    <h3>:star2: 一键开启你的饥荒之旅 :star2:</h3>
    <h3>:tada: 欢迎加群：<a href="https://qun.qq.com/universal-share/share?ac=1&authKey=ePe2g%2Bq16q8tSAdeJwOXC08NnAKn%2BfmwKeTdf8oS3pD5DzrPKQkoS6eAAD6UivHk&busi_data=eyJncm91cENvZGUiOiI3MzM5NDg2NDQiLCJ0b2tlbiI6Ii9CTmFVWTZOUTNvNUFuaG4rNTdaSnAvQ3U1aERkSUgxcFdCelB1OEhDNWtYNjlvRGhQZnU4allOcWcvcHM4b3IiLCJ1aW4iOiI3NjM0ODM5NjYifQ%3D%3D&data=qjh1K6Pelvxvj6Yl-qeFNEF3jJbc7EJMEC6Edt3ULjtM9WSkvbe0PKTd2q2Qp0v8wA6hXmL-sN-ziKjuf2zEXA&svctype=4&tempid=h5_group_info">733948644</a> :tada:</h3>
</div>

---

## :strawberry: 详细文档
本页面的帮助文档仅为简单介绍  
**详细教程请前往** [https://miraclesses.top](https://miraclesses.top)  
文档包含快速开始、详细文档等，萌新大佬均可无痛阅读  

## :warning: 2.×版本与1.×版本不兼容
**1.×版本已于2025年7月31日停止维护，模组搜索等功能无法使用，[点击查看迁移文档](docs/README_migration_zh.md)**

## :watermelon: 使用方法
>**建议使用 Ubuntu 24系统，低版本系统可能会出现GLIBC版本报错**  
```shell
# 执行以下命令，下载脚本（使用加速节点）需要使用jq命令
cd && rm -f run.sh && wget https://ghfast.top/https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/run.sh && chmod +x run.sh && ./run.sh  
#或者  
cd && rm -f run.sh && wget https://gh.llkk.cc/https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/run.sh && chmod +x run.sh && ./run.sh
```
```shell
# 执行以下命令，下载脚本（不使用加速节点，自带梯子）
cd ~ && wget https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/run.sh && chmod +x run.sh
```
```shell
# 自定义启动端口（8080改为你要用的端口），请手动修改run.sh文件或者
sed -i 's/^PORT=.*/PORT=8080/' run.sh
```
```shell
# 根据系统提示输入并回车
./run.sh
```
**更新方法**
```shell
cd ~ && ./run.sh
```
根据提示输入4
```shell
# root@VM-0-16-ubuntu:~# cd ~ && ./run.sh
饥荒管理平台(DMP)
--- https://github.com/miracleEverywhere/dst-management-platform-api ---
————————————————————————————————————————————————————————————
[0]: 下载并启动服务(Download and start the service)
————————————————————————————————————————————————————————————
[1]: 启动服务(Start the service)
[2]: 关闭服务(Stop the service)
[3]: 重启服务(Restart the service)
————————————————————————————————————————————————————————————
[4]: 更新管理平台(Update management platform)
[5]: 强制更新平台(Force update platform)
[6]: 更新启动脚本(Update startup script)
————————————————————————————————————————————————————————————
[7]: 设置虚拟内存(Setup swap)
[8]: 退出脚本(Exit script)
————————————————————————————————————————————————————————————
请输入选择(Please enter your selection) [0-8]:
```
如果下载了发行版(不建议，请使用run.sh脚本启动)，则执行以下命令：
```shell
# -c 为开启日志，建议开启
nohup ./dmp -c > dmp.log 2>&1 &
```
默认启动端口为80，如果您想修改，则修改启动命令：
```shell
# 修改端口为8888
nohup ./dmp -c -l 8888 > dmp.log 2>&1 &
```
也可以指定数据库文件的存储目录  
```shell
# 开启控制台输出，监听8899端口，DstMP.sdb的存储位置为 ./config/DstMP.sdb
nohup ./dmp -c -l 8899 -s ./config > dmp.log 2>&1 &
```
**docker部署方式**  
首先在package页面获取docker镜像tag  
建议映射config、dst和.klei目录  

```shell
# 绑定80端口 映射到/app目录下
docker run -itd --name dmp -p 80:80 \
-v /app/config:/root/config \
-v /app/dst:/root/dst \
-v /app/.klei:/root/.klei \
-v /app/dmp_files:/root/dmp_files \
-v /app/steamcmd:/root/steamcmd \
-v /etc/localtime:/etc/localtime:ro \
-v /etc/timezone:/etc/timezone:ro \
ghcr.io/miracleeverywhere/dst-management-platform-api:latest
```
```shell
# 绑定8000端口 映射到/app目录下
docker run -itd --name dmp -p 8000:80 \
-v /app/config:/root/config \
-v /app/dst:/root/dst \
-v /app/.klei:/root/.klei \
-v /app/dmp_files:/root/dmp_files \
-v /app/steamcmd:/root/steamcmd \
-v /etc/localtime:/etc/localtime:ro \
-v /etc/timezone:/etc/timezone:ro \
ghcr.io/miracleeverywhere/dst-management-platform-api:latest
```
```shell
# 使用host网络，并绑定8080端口
docker run -itd --name dmp --net=host \
-e DMP_PORT=8080 \
-v /app/config:/root/config \
-v /app/dst:/root/dst \
-v /app/.klei:/root/.klei \
-v /app/dmp_files:/root/dmp_files \
-v /app/steamcmd:/root/steamcmd \
-v /etc/localtime:/etc/localtime:ro \
-v /etc/timezone:/etc/timezone:ro \
ghcr.io/miracleeverywhere/dst-management-platform-api:latest
```
**docker更新**  
停止旧版本容器，拉取新版本镜像，使用上述启动命令启动即可  
如果有映射config、dst和.klei目录，则无需重复安装游戏等操作  

**MacOS安装(测试版，不提供支持)**  
> 只支持M系列CPU的Mac
```shell
cd ~ && wget https://github.com/miracleEverywhere/dst-management-platform-api/raw/refs/heads/master/run.sh && chmod +x run_macos.sh
# 请输入需要执行的操作(Please enter the operation to be performed): 
# [0]: 下载并启动服务(Download and start the service) 
# [1]: 启动服务(Start the service) 
# [2]: 关闭服务(Stop the service) 
# [3]: 重启服务(Restart the service) 
# [4]: 更新服务(Update the service) 
```
选择0下载并启动，启动完成后运行manual_install.sh脚本安装游戏，无法在页面进行安装
```shell
./manual_install.sh
```
>注意：MacOS由于系统原因，模组配置暂不可用，需要点击设置-模组-添加模组页面的导出按钮，点击后会在桌面生成名为dmp_exported_mod的目录，用户需使用 **访达** 将改目录中的模组复制到~/dst/dontstarve_dedicated_server_nullrenderer/Contents/mods目录下。更新模组需要在设置-模组-添加模组页面删除对应要更新的模组，然后重新下载该模组，执行导出和复制操作后，重启游戏服务器。

---

## :balloon: 服务器选择

1. 如果你想玩半纯净档，只有showme、5格装备栏等模组，其实2C2G5M(2个核心，2G内存，5Mbps带宽)的机器完全够用，当然要开启SWAP(即虚拟内存，后续会讲到如何开启)。

2. 如果你想多加几个大型模组，例如棱镜、勋章等，这些模组对服务器的性能就有所要求，一般推荐2C4G5M及以上的云服务器，并同时开启SWAP。

3. 作者推荐[汉堡云服务器](https://hbyidc.com/recommend/OKkxTzgMP6k7)，专为饥荒打造！官方合作商「汉堡云」带来高性能游戏服务器，首月低至6折！

    - 云产品：NAT云（游戏云电脑）/ 云服务器ECS等多款上云产品
    - 地域选择： 湖北襄阳 / 内蒙古呼和浩特 / 江苏宿迁 / 中国香港 / 美国洛杉矶
    - CPU选择：从E5入门CPU到英特尔铂金CPU再到i9-12900K消费级CPU，均有涉猎
    - 新用户尊享DMP渠道价，云服务器ECS/轻量云LS/游戏云GYC产品首月6折

**优惠码**：`DMPqd6666` [点我立即购买](https://hbyidc.com/recommend/OKkxTzgMP6k7)

[![作者推荐](https://hbyidc.com/upload/common/default/6d9086b769c66efba611d361259371501745528167%5E%E6%B1%89%E5%A0%A1%E4%BA%911020-02.png)](https://hbyidc.com/recommend/OKkxTzgMP6k7)

## :cherries: 平台截图
![zh-home](docs/images/zh-home.png)
  

![zh-room](docs/images/zh-room.png)
  

![zh-mod](docs/images/zh-mod.png)
  

![zh-backup](docs/images/zh-backup.png)


![zh-location](docs/images/zh-location.png)


![zh-logs](docs/images/zh-logs.png)


![zh-clusters](docs/images/zh-clusters.png) 

---

## :love_letter: 代码贡献
欢迎所有开发者贡献代码，以改善和扩展这个项目！

![https://github.com/miracleEverywhere/dst-management-platform-api/graphs/contributors](https://contrib.rocks/image?repo=miracleEverywhere/dst-management-platform-api)

---

##  :sparkling_heart: 致谢
本项目[前端页面](https://github.com/miracleEverywhere/dst-management-platform-web)基于[koi-ui](https://github.com/KoiKite/koi-ui)二次开发，感谢开源 [@KoiKite](https://github.com/KoiKite)  

感谢加速站点[github.akams.cn](https://github.akams.cn/) 