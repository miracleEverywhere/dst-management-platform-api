#!/bin/zsh

set -e

WORK_DIR=$(pwd)
STEAM_DIR="$WORK_DIR/steamcmd"
DST_DIR="$WORK_DIR/dst"
DST_SETTING_DIR="$HOME/.klei"

function error_exit() {
    echo -e "==>dmp@@ 安装失败 @@dmp<=="
    exit 1
}

if ! brew --version >/dev/null 2>&1; then
    echo -e "brew未安装\n"
    error_exit
fi


# 安装依赖
brew install screen

rm -f steamcmd_osx.tar.gz
rm -rf "$STEAM_DIR" "$DST_DIR"
mkdir "$STEAM_DIR" || error_exit
curl -O "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_osx.tar.gz"
tar zxvf steamcmd_osx.tar.gz -C "$STEAM_DIR"

# 安装DST
cd "$STEAM_DIR" || error_exit
./steamcmd.sh +force_install_dir "$DST_DIR" +login anonymous +app_update 343050 validate +quit || true
./steamcmd.sh +force_install_dir "$DST_DIR" +login anonymous +app_update 343050 validate +quit


# 初始化一些目录和文件
mkdir -p "$HOME/Documents/Klei/DoNotStarveTogether"

cd "$HOME/Documents/Klei/DoNotStarveTogether"
ln -s "${DST_SETTING_DIR}"/DoNotStarveTogether/MyDediServer .

# 清理
cd "$WORK_DIR" || error_exit
rm -f steamcmd_osx.tar.gz

echo -e "==>dmp@@ 安装完成 @@dmp<=="