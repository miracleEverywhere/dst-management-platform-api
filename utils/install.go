package utils

const ManualInstall = `#!/bin/bash

# 定义变量
STEAM_DIR="$HOME/steamcmd"
DST_DIR="$HOME/dst"
DST_SETTING_DIR="$HOME/.klei"


# 工具函数
function check_directory() {
    if [ -d "$STEAM_DIR" ]; then
        mv $STEAM_DIR $STEAM_DIR.$(date +%s)
    fi

    if [ -d "$DST_DIR" ]; then
        mv $DST_DIR $DST_DIR.$(date +%s)
    fi

    if [ -d "$DST_SETTING_DIR" ]; then
        mv $DST_SETTING_DIR $DST_SETTING_DIR.$(date +%s)
    fi

}

function install_ubuntu() {
	dpkg --add-architecture i386
	apt update
    apt install -y lib32gcc1     
	apt install -y lib32gcc-s1
    apt install -y libcurl4-gnutls-dev:i386
    apt install -y screen
	apt install -y unzip
}

function install_rhel() {
    yum -y install glibc.i686 libstdc++.i686 libcurl.i686
    yum -y install screen
	yum install -y unzip 2>&1 > /dev/null
    ln -s /usr/lib/libcurl.so.4 /usr/lib/libcurl-gnutls.so.4
}

function check_screen() {
    which screen
    if (($? != 0)); then
        echo -e "screen命令安装失败\tScreen command installation failed"
        exit 1
    fi
}

# 创建、备份目录
check_directory

# 安装依赖
OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
if [[ ${OS} == "ubuntu" ]]; then
    install_ubuntu
else
    OS_LIKE=$(grep -P "^ID_LIKE=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g" | grep rhel)
    if (($? == 0)); then
        install_rhel
    else
        echo -e "系统不支持\tSystem not supported"
        exit 1
    fi
fi

# 检查screen命令
check_screen

# 下载安装包
cd ~
wget https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz

# 解压安装包
mkdir -p $STEAM_DIR
tar -zxvf steamcmd_linux.tar.gz -C $STEAM_DIR

# 安装DST
cd $STEAM_DIR
./steamcmd.sh +force_install_dir "$DST_DIR" +login anonymous +app_update 343050 validate +quit

cp ~/steamcmd/linux32/libstdc++.so.6 ~/dst/bin/lib32/
# 修复MOD更新bug
cp ~/steamcmd/linux32/steamclient.so ~/dst/bin/lib32/
# 初始化一些目录和文件
mkdir -p ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/Master
mkdir -p ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/Caves
mkdir -p ${DST_SETTING_DIR}/DMP_BACKUP
# 管理员
> ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/adminlist.txt
# 黑名单
> ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/blocklist.txt
# 预留位
> ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/whitelist.txt

# 清理
cd ~
rm -f steamcmd_linux.tar.gz
rm -f $STEAM_DIR/install.log

# 安装完成
echo -e "安装完成\tInstallation completed"
`
