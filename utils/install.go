package utils

const ManualInstall = `#!/bin/bash

# 定义变量
STEAM_DIR="$HOME/steamcmd"
DST_DIR="$HOME/dst"
DST_SETTING_DIR="$HOME/.klei"


# 工具函数
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

function check_screen() {
    which screen
    if (($? != 0)); then
        echo -e "screen命令安装失败\tScreen command installation failed"
        exit 1
    fi
}

# 安装依赖
OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
if [[ "${OS}" == "ubuntu" || "${OS}" == "debian" ]]; then
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
rm -f steamcmd_linux.tar.gz
wget https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz

# 解压安装包
rm -rf $STEAM_DIR
mkdir -p $STEAM_DIR
tar -zxvf steamcmd_linux.tar.gz -C $STEAM_DIR

#清理可能损坏的acf文件
rm -rf $DST_DIR/steamapps/appmanifest_343050.acf

# 安装DST
cd $STEAM_DIR
./steamcmd.sh +force_install_dir "$DST_DIR" +login anonymous +app_update 343050 validate +quit

cp ~/steamcmd/linux32/libstdc++.so.6 ~/dst/bin/lib32/

# 清理
cd ~
rm -f steamcmd_linux.tar.gz
rm -f $STEAM_DIR/install.log

# 安装完成
echo -e "安装完成\tInstallation completed"
`

const ManualInstallMac = `
#!/bin/zsh

if ! brew --version >/dev/null 2>&1; then
    echo -e "\e[31mbrew未安装 (brew NOT installed) \e[0m"
    exit 1
fi

# 定义变量
STEAM_DIR="$HOME/steamcmd"
DST_DIR="$HOME/dst"
DST_SETTING_DIR="$HOME/.klei"

# 安装依赖
brew install unzip wget screen curl grep

rm -f steamcmd_osx.tar.gz
rm -rf $STEAM_DIR
mkdir $STEAM_DIR
curl -O "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_osx.tar.gz"
tar zxvf steamcmd_osx.tar.gz -C steamcmd

# 安装DST
cd $STEAM_DIR
./steamcmd.sh +force_install_dir "$DST_DIR" +login anonymous +app_update 343050 validate +quit


# 初始化一些目录和文件
mkdir -p $HOME/Documents/Klei/DoNotStarveTogether

cd $HOME/Documents/Klei/DoNotStarveTogether
ln -s ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer .

# 清理
cd ~
rm -f steamcmd_osx.tar.gz
`
