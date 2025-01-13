#!/bin/bash

# 定义变量
DMP_HOME="/root"

# 由于install.go里没有安装wget，在启动容器的时候安装；或者在install.go进行安装
apt update
apt install -y wget unzip jq screen

cd $DMP_HOME || exit

if [ -e "DstMP.sdb" ]; then
	bit64=$(jq -r .bit64 DstMP.sdb)
else
	bit64="false"
fi

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

exec ./dmp -c -s ./config > $DMP_HOME/dmp.log 2>&1
