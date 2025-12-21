#!/bin/bash

# 定义变量
STEAM_DIR="$HOME/steamcmd"
DST_DIR="$HOME/dst"

# 工具函数
function install_ubuntu() {
    dpkg --add-architecture i386
    apt update -y
    apt install -y lib32gcc1
    apt install -y lib32gcc-s1
    apt install -y libcurl4-gnutls-dev:i386
    apt install -y screen
    apt install -y unzip
}

function install_rhel() {
    yum update -y
    yum -y install glibc.i686 libstdc++.i686 libcurl.i686
    yum -y install glibc libstdc++ libcurl
    yum -y install screen
    yum install -y unzip
}

function check_screen() {

    if ! which screen; then
        echo -e "screen命令安装失败"
        exit 1
    fi
}

# 安装依赖
OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
if [[ "${OS}" == "ubuntu" || "${OS}" == "debian" ]]; then
    install_ubuntu
else
    if grep -P "^ID_LIKE=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g" | grep rhel; then
        install_rhel
    else
        echo -e "系统不支持"
        exit 1
    fi
fi

# 检查screen命令
check_screen

# 下载安装包
cd "$HOME" || exit 1
rm -f steamcmd_linux.tar.gz
wget https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz

# 解压安装包
rm -rf "$STEAM_DIR"
mkdir -p "$STEAM_DIR"
tar -zxvf steamcmd_linux.tar.gz -C "$STEAM_DIR"

# PR77 清理可能损坏的acf文件
rm -rf "$DST_DIR/steamapps/appmanifest_343050.acf"

# 安装DST
cd "$STEAM_DIR" || exit 1
./steamcmd.sh +force_install_dir "$DST_DIR" +login anonymous +app_update 343050 validate +quit

cd "$HOME" || exit 1
cp steamcmd/linux32/libstdc++.so.6 dst/bin/lib32/
ln -s /usr/lib64/libcurl.so.4 dst/bin64/lib64/libcurl-gnutls.so.4
ln -s /usr/lib/libcurl.so.4 dst/bin/lib32/libcurl-gnutls.so.4

# luajit
cd "$HOME" || exit 1
cp dmp_files/luajit/* dst/bin64/
cat >dst/bin64/dontstarve_dedicated_server_nullrenderer_x64_luajit <<-"EOF"
            export LD_PRELOAD=./libpreload.so
            ./dontstarve_dedicated_server_nullrenderer_x64 "$@"
            unset LD_PRELOAD
EOF
chmod --reference=dst/bin64/dontstarve_dedicated_server_nullrenderer_x64 dst/bin64/dontstarve_dedicated_server_nullrenderer_x64_luajit

# 清理
cd "$HOME" || exit 1
rm -f steamcmd_linux.tar.gz
rm -f "$STEAM_DIR/install.log"

# 安装完成
echo -e "安装完成"
