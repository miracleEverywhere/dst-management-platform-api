#!/bin/bash

# 定义变量
DMP_HOME="/root"
DMP_DB="./config/DstMP.sdb"

# 安装必要的依赖
apt update
apt install -y wget unzip jq screen

cd $DMP_HOME || exit

# 检查是否为64位启动
if [ -e "$DMP_DB" ]; then
	bit64=$(jq -r .bit64 "$DMP_DB")
else
	bit64="false"
fi

#安装对应的DST依赖
if [[ "$bit64" == "true" ]]; then
    apt install -y lib32gcc1
    apt install -y lib32gcc-s1
    apt install -y libcurl4-gnutls-dev
else
    dpkg --add-architecture i386
    apt update
    apt install -y lib32gcc1
    apt install -y lib32gcc-s1
    apt install -y libcurl4-gnutls-dev:i386
fi

#启动dmp
exec ./dmp -l "$DMP_PORT" -c -s ./config > $DMP_HOME/dmp.log 2>&1
