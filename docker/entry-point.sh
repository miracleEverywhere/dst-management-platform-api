#!/bin/bash

# 定义变量
DMP_HOME="/root"
STEAM_DIR="$DMP_HOME/steamcmd"
DST_DIR="$DMP_HOME/dst"
DST_SETTING_DIR="$DMP_HOME/.klei"

cd $DMP_HOME || exit

wget https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz

# 解压安装包
mkdir -p $STEAM_DIR
tar -zxvf steamcmd_linux.tar.gz -C $STEAM_DIR

# 安装DST
cd $STEAM_DIR || exit
./steamcmd.sh +force_install_dir "$DST_DIR" +login anonymous +app_update 343050 validate +quit

cp ${STEAM_DIR}/linux32/libstdc++.so.6 ~/dst/bin/lib32/
# 初始化一些目录和文件
mkdir -p ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/Master
mkdir -p ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/Caves
mkdir -p ${DST_SETTING_DIR}/DMP_BACKUP
# 管理员
# shellcheck disable=SC2188
> ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/adminlist.txt
# 黑名单
# shellcheck disable=SC2188
> ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/blocklist.txt
# 预留位
# shellcheck disable=SC2188
> ${DST_SETTING_DIR}/DoNotStarveTogether/MyDediServer/whitelist.txt

cd $DMP_HOME || exit
# shellcheck disable=SC2069
exec ./dmp -c 2>&1 > $DMP_HOME/dmp.log
