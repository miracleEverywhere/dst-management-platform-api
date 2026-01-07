#!/bin/bash

# 定义变量
STEAM_DIR="$HOME/steamcmd"
DST_DIR="$HOME/dst"

# 改进的错误处理函数
function error_exit() {
    local message="${1:-安装失败}"
    echo -e "==>dmp@@ ${message} @@dmp<=="
    exit 1
}

# 工具函数
function install_ubuntu() {
    dpkg --add-architecture i386
    apt update -y
    apt install -y screen wget
    #apt install -y lib32gcc1 || true
    apt install -y lib32gcc-s1 || true
    apt install -y libcurl4-gnutls-dev:i386 || error_exit "安装libcurl失败"
    apt install -y libcurl4-gnutls-dev || true
}

function install_rhel() {
    yum update -y
    yum -y install glibc.i686 libstdc++.i686 libcurl.i686
    yum -y install glibc libstdc++ libcurl
    yum -y install screen wget
    ln -s /usr/lib/libcurl.so.4 /usr/lib/libcurl-gnutls.so.4
}

function check_screen() {
    if ! command -v screen > /dev/null 2>&1; then
        error_exit "screen命令未安装"
    fi
}

function check_wget() {
    if ! command -v wget > /dev/null 2>&1; then
        error_exit "wget命令未安装"
    fi
}

# 安装完成后的修复操作
function dst_fix() {
    echo "开始执行环境修复..."

    cd "$HOME" || error_exit "无法切换到HOME目录"

    # 一些必要的so文件
    cp steamcmd/linux32/libstdc++.so.6 dst/bin/lib32/
    if [[ "${OS}" == "ubuntu" || "${OS}" == "debian" ]]; then
        [ ! -L "dst/bin64/lib64/libcurl-gnutls.so.4" ] && ln -s /usr/lib/x86_64-linux-gnu/libcurl-gnutls.so.4 dst/bin64/lib64/libcurl-gnutls.so.4
        [ ! -L "dst/bin/lib32/libcurl-gnutls.so.4" ] && ln -s /usr/lib/i386-linux-gnu/libcurl-gnutls.so.4 dst/bin/lib32/libcurl-gnutls.so.4
    else
        [ ! -L "dst/bin64/lib64/libcurl-gnutls.so.4" ] && ln -s /usr/lib64/libcurl.so.4 dst/bin64/lib64/libcurl-gnutls.so.4
        [ ! -L "dst/bin/lib32/libcurl-gnutls.so.4" ] && ln -s /usr/lib/libcurl.so.4 dst/bin/lib32/libcurl-gnutls.so.4
    fi

    # luajit
    if [ ! -d "dmp_files/luajit" ]; then
    error_exit "缺少必要的luajit文件目录"
    fi
    cp dmp_files/luajit/* dst/bin64/
    cat >dst/bin64/dontstarve_dedicated_server_nullrenderer_x64_luajit <<-"EOF"
    export LD_PRELOAD=./libpreload.so
    ./dontstarve_dedicated_server_nullrenderer_x64 "$@"
    unset LD_PRELOAD
    EOF
    chmod --reference=dst/bin64/dontstarve_dedicated_server_nullrenderer_x64 dst/bin64/dontstarve_dedicated_server_nullrenderer_x64_luajit
    
    echo "环境修复完成"
}

# 安装DST
function install_dst_server() {
    local success=false
    local attempts=0
    local max_retries=3
    
    while [ $attempts -lt $max_retries ] && [ "$success" = false ]; do
        attempts=$((attempts + 1))
        echo "尝试安装DST服务器 (第${attempts}次)..."
        
        # 使用SteamCMD安装
        if ./steamcmd.sh +login anonymous +force_install_dir "$DST_DIR" +app_update 343050 validate +quit; then
            # 验证安装结果
            if [ -d "$DST_DIR/bin" ] && [ -f "$DST_DIR/bin/dontstarve_dedicated_server_nullrenderer" ]; then
                echo "✅ DST服务器安装成功"
                success=true
            else
                echo "⚠️ 安装完成但文件验证失败"
            fi
        else
            echo "❌ SteamCMD安装失败"
            if [ $attempts -lt $max_retries ]; then
                echo "等待6秒后重试..."
                sleep 6
            fi
        fi
    done
    
    if [ "$success" = true ]; then
        dst_fix
        echo -e "==>dmp@@ 安装完成 @@dmp<=="
        return 0
    else
        error_exit "已达到最大重试次数，安装失败"
        return 1
    fi
}

# 安装依赖
OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
if [[ "${OS}" == "ubuntu" || "${OS}" == "debian" ]]; then
    install_ubuntu
else
    if grep -P "^ID_LIKE=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g" | grep rhel > /dev/null 2>&1; then
        install_rhel
    else
        error_exit "不支持的Linux发行版"
    fi
fi

# 检查screen命令
check_screen

# 检查wget命令
check_wget

# 准备安装目录
cd "$HOME" || error_exit "无法切换到HOME目录"
if [[ "${DMP_IN_CONTAINER:-0}" != "1" ]]; then
    rm -rf "$STEAM_DIR"
fi
mkdir -p "$STEAM_DIR"

# 下载SteamCMD
if ! wget -q https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz; then
    error_exit "下载SteamCMD失败"
fi

# 解压并安装
tar -zxvf steamcmd_linux.tar.gz -C "$STEAM_DIR" || error_exit "解压SteamCMD失败"
cd "$STEAM_DIR" || error_exit "无法进入SteamCMD目录"

# 执行安装
install_dst_server

# 清理
rm -f "$HOME/steamcmd_linux.tar.gz"
echo "安装流程完成"