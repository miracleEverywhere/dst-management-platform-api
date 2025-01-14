package utils

const ShInstall = `#!/bin/bash

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
    dpkg --add-architecture i386 2>&1 > /dev/null
	apt update 2>&1 > /dev/null
    apt install -y lib32gcc1 2>&1 > /dev/null
	apt install -y lib32gcc-s1 2>&1 > /dev/null
    apt install -y libcurl4-gnutls-dev:i386 2>&1 > /dev/null
    apt install -y screen 2>&1 > /dev/null
	apt install -y unzip 2>&1 > /dev/null
}

function install_rhel() {
    yum -y install glibc.i686 libstdc++.i686 libcurl.i686 2>&1 > /dev/null
    yum -y install screen 2>&1 > /dev/null
	yum install -y unzip 2>&1 > /dev/null
    ln -s /usr/lib/libcurl.so.4 /usr/lib/libcurl-gnutls.so.4 2>&1 > /dev/null
}

function check_screen() {
    which screen
    if (($? != 0)); then
        echo -e "0\tscreen命令安装失败\tScreen command installation failed" > /tmp/install_status
        exit 1
    fi
}


# 清空status文件
> /tmp/install_status

# 创建、备份目录
check_directory

# 安装依赖
echo -e "17\t正在安装依赖\tInstalling dependencies" > /tmp/install_status
OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
if [[ ${OS} == "ubuntu" ]]; then
    install_ubuntu
else
    OS_LIKE=$(grep -P "^ID_LIKE=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g" | grep rhel)
    if (($? == 0)); then
        install_rhel
    else
        echo -e "0\t系统不支持\tSystem not supported" > /tmp/install_status
        exit 1
    fi
fi

# 检查screen命令
check_screen

# 下载安装包
echo -e "29\t正在下载Steam安装包\tDownloading the Steam installation package" > /tmp/install_status
cd ~
wget https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz 2>&1 > /dev/null

# 解压安装包
echo -e "41\t正在解压Steam安装包\tExtracting the Steam installation package" > /tmp/install_status
mkdir -p $STEAM_DIR
tar -zxvf steamcmd_linux.tar.gz -C $STEAM_DIR 2>&1 > /dev/null

# 安装DST
echo -e "49\t正在下载Steam\tDownloading Steam" > /tmp/install_status
cd $STEAM_DIR
> install.log
./steamcmd.sh +force_install_dir "$DST_DIR" +login anonymous +app_update 343050 validate +quit | tee -a "install.log" 2>&1 > /dev/null &
PID=$!

while true
do
    ps -ef | grep $PID | grep -v grep
    if (($? == 0)); then
        tail -2 install.log | awk -F'progress: ' '{
          if ($1 ~ /0x61.*\)/) {
            percentage=$2; sub(/ .*/, "", percentage); printf "%.1f\t正在下载饥荒\tDownloading DST\n", 59 + percentage * 0.2 > "/tmp/install_status"
          } else if ($1 ~ /0x61.*\)/) {
            percentage=$2; sub(/ .*/, "", percentage); printf "%.1f\t正在安装饥荒\tInstalling DST\n", 79 + percentage * 0.2 > "/tmp/install_status"
          }
        }'
    else
        rm -f install.log
        break
    fi
    sleep 1
done

cp ~/steamcmd/linux32/libstdc++.so.6 ~/dst/bin/lib32/

# 解决无法自动下载MOD的问题
# 备份
mv ~/dst/bin/lib32/steamclient.so ~/dst/bin/lib32/steamclient.so.bak
mv ~/dst/steamclient.so ~/dst/steamclient.so.bak
# 替换
cp ~/steamcmd/linux32/steamclient.so ~/dst/bin/lib32/steamclient.so
cp ~/steamcmd/linux32/steamclient.so ~/dst/steamclient.so

# 初始化一些目录和文件
mkdir -p ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/Master
mkdir -p ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/Caves
mkdir -p ${DST_SETTING_DIR}/DMP_BACKUP
mkdir -p ${DST_SETTING_DIR}/DMP_MOD/not_ugc
mkdir -p ${DST_DIR}/ugc_mods/MyDediServer/Master/content/322330
mkdir -p ${DST_DIR}/ugc_mods/MyDediServer/Caves/content/322330
# UID MAP
> ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/uid_map.json
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
rm -f $0

# 安装完成
echo -e "100\t安装完成\tInstallation completed" > /tmp/install_status
`

const Install32Dependency = `
#!/bin/bash

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
	yum update
    yum -y install glibc.i686 libstdc++.i686 libcurl.i686
    yum -y install screen
	yum install -y unzip
    ln -s /usr/lib/libcurl.so.4 /usr/lib/libcurl-gnutls.so.4
}

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

rm -f $0
`

const Install64Dependency = `
#!/bin/bash

function install_ubuntu() {
	apt update
    apt install -y lib32gcc1     
	apt install -y lib32gcc-s1
    apt install -y libcurl4-gnutls-dev
}

function install_rhel() {
	yum update
    yum -y install glibc libstdc++ libcurl
    ln -s /usr/lib/libcurl.so.4 /usr/lib/libcurl-gnutls.so.4
}

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

rm -f $0
`
