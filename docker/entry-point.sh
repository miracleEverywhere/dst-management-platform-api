#!/bin/bash

# 定义变量
DMP_HOME="/root"

# 由于install.go里没有安装wget，在启动容器的时候安装；或者在install.go进行安装
apt update
apt install -y wget unzip

cd $DMP_HOME || exit

exec ./dmp -c -s -d ./config > $DMP_HOME/dmp.log 2>&1
