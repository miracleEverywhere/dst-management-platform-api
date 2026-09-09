#!/bin/bash

# 设置错误处理
set -e

# 定义变量
WORK_DIR=$(pwd)
STEAM_DIR="$WORK_DIR/steamcmd"
DST_DIR="$WORK_DIR/dst"
OS_TYPE=$(uname -s)

# 错误处理函数
function error_exit() {
    echo -e "==>dmp@@ 更新失败 @@dmp<=="
    exit 1
}

# 设置trap捕获所有错误
trap error_exit ERR

cd "${STEAM_DIR}" || error_exit
./steamcmd.sh +login anonymous +force_install_dir "${DST_DIR}" +app_update 343050 validate +quit || error_exit

if [[ "${OS_TYPE}" == "Darwin" ]]; then
	DST_BIN_DIR="$DST_DIR/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS"
    DST_BIN="dontstarve_dedicated_server_nullrenderer"
    cd "$DST_BIN_DIR" || error_exit
    timeout 1m ./$DST_BIN
fi

cd || true

# 安装完成
echo -e "==>dmp@@ 更新完成 @@dmp<=="