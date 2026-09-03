#!/bin/bash

set -e

WORK_DIR=$(pwd)
STEAM_DIR="$WORK_DIR/steamcmd"
DST_DIR="$WORK_DIR/dst"
DST_SETTING_DIR="$HOME/.klei"

# 设置trap捕获所有错误
trap error_exit ERR

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
cd "$HOME"

# 将游戏存档目录统一到 Documents/Klei。空目录可以安全替换，非空目录则保留用户数据。
if [ -L "${DST_SETTING_DIR}" ]; then
    ln -sfn "$HOME/Documents/Klei" "${DST_SETTING_DIR}"
elif [ -d "${DST_SETTING_DIR}" ]; then
    if [ -z "$(find "${DST_SETTING_DIR}" -mindepth 1 -maxdepth 1 -print -quit)" ]; then
        rmdir "${DST_SETTING_DIR}"
        ln -s "$HOME/Documents/Klei" "${DST_SETTING_DIR}"
    else
        echo "${DST_SETTING_DIR} 已存在且包含文件，保留现有目录"
    fi
else
    ln -s "$HOME/Documents/Klei" "${DST_SETTING_DIR}"
fi

# 清理
cd "$WORK_DIR" || error_exit
rm -f steamcmd_osx.tar.gz

echo -e "==>dmp@@ 安装完成 @@dmp<=="
