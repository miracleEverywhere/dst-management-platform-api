#!/bin/bash

# 设置错误处理
set -e

# 错误处理函数
function error_exit() {
    echo -e "==>dmp@@ 更新失败 @@dmp<=="
    exit 1
}

# 设置trap捕获所有错误
trap error_exit ERR

cd steamcmd || error_exit
./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit || error_exit

cd || true

# 安装完成
echo -e "==>dmp@@ 更新完成 @@dmp<=="