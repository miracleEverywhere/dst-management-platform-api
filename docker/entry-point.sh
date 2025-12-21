#!/bin/bash

# 定义变量
DMP_HOME="/root"
DMP_DB="./config/DstMP.sdb"

# 安装必要的依赖
apt-get update
apt-get install -y wget unzip jq screen

cd $DMP_HOME || exit

# 检查是否为64位启动
if [ -e "$DMP_DB" ]; then
    bit64=$(jq -r .bit64 "$DMP_DB")
else
    bit64="false"
fi

# 安装对应的DST依赖
if [[ "$bit64" == "true" ]]; then
    apt-get install -y lib32gcc1
    apt-get install -y lib32gcc-s1
    apt-get install -y libcurl4-gnutls-dev
else
    dpkg --add-architecture i386
    apt-get update
    apt-get install -y lib32gcc1
    apt-get install -y lib32gcc-s1
    apt-get install -y libcurl4-gnutls-dev:i386
fi

# 定义 SIGTERM 信号处理函数
cleanup() {
    echo "Received SIGTERM, cleaning up..."
    # 发送停止信号给 dmp 进程
    if [[ -n "$DMP_PID" ]]; then
        kill "$DMP_PID"
        echo "Stopped dmp process with PID $DMP_PID"
    fi
    exit 0
}

# 捕获 SIGTERM 信号
trap cleanup SIGTERM

# 启动 dmp 并获取其 PID
./dmp -l "$DMP_PORT" -c -s ./config > "$DMP_HOME/dmp.log" 2>&1 &
DMP_PID=$!  # 获取 dmp 进程的 PID

# 让脚本保持运行状态，直到收到信号
while true; do
    sleep 1
done