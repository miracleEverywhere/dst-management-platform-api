#!/bin/bash

# 定义变量
DMP_HOME="/root"

cd $DMP_HOME || exit

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

# 构建启动命令
DMP_CMD="./dmp -bind $DMP_PORT -dbpath ./data -level ${LEVEL:-info}"

# 如果启用 TLS，追加证书和私钥参数
if [ "$TLS" = "true" ]; then
    TLS_CERT="${TLS_CERT:-/etc/ssl/dmp/fullchain.pem}"
    TLS_KEY="${TLS_KEY:-/etc/ssl/dmp/privkey.pem}"
    DMP_CMD="$DMP_CMD -cert $TLS_CERT -key $TLS_KEY"
    echo "TLS enabled, cert: $TLS_CERT, key: $TLS_KEY"
fi

# 启动 dmp 并获取其 PID
$DMP_CMD 2>&1 &
DMP_PID=$!  # 获取 dmp 进程的 PID

# 让脚本保持运行状态，直到收到信号
while true; do
    sleep 1
done
