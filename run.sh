#!/bin/bash

###########################################
# 用户自定义设置请修改下方变量，其他变量请不要修改 #
###########################################

# --------------- ↓可修改↓ --------------- #
# dmp暴露端口，即网页打开时所用的端口
PORT=80

# 数据库文件所在目录，例如：./config
CONFIG_DIR="./"

# 虚拟内存大小，例如 1G 4G等
SWAPSIZE=2G
# --------------- ↑可修改↑ --------------- #

###########################################
#     下方变量请不要修改，否则可能会出现异常     #
###########################################

USER=$(whoami)
ExeFile="$HOME/dmp"

DMP_GITHUB_HOME_URL="https://github.com/miracleEverywhere/dst-management-platform-api"
DMP_GITHUB_API_URL="https://api.github.com/repos/miracleEverywhere/dst-management-platform-api/releases/latest"
SCRIPT_GITHUB="https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/run.sh"

cd "$HOME" || exit

function echo_red() {
    echo -e "\033[0;31m$*\033[0m"
}

function echo_green() {
    echo -e "\033[0;32m$*\033[0m"
}

function echo_yellow() {
    echo -e "\033[0;33m$*\033[0m"
}

function echo_cyan() {
    echo -e "\033[0;36m$*\033[0m"
}

function echo_red_blink() {
    echo -e "\033[5;31m$*\033[0m"
}

# 检查用户，只能使用root执行
if [[ "${USER}" != "root" ]]; then
    echo_red "请使用root用户执行此脚本"
    exit 1
fi

if [ -z "$1" ]; then
    acceleration_index=0

else
    acceleration_index=$1
fi

# 设置全局stderr为红色并添加固定格式
function set_tty() {
    exec 2> >(while read -r line; do echo_red "[$(date +'%F %T')] [ERROR] ${line}" >&2; done)
}

# 恢复stderr颜色
function unset_tty() {
    exec 2> /dev/tty
}

# 定义一个函数来提示用户输入
function prompt_user() {
    clear
    echo_green "饥荒管理平台(DMP)"
    echo_green "--- ${DMP_GITHUB_HOME_URL} ---"
    if [[ $(echo "${DMP_GITHUB_HOME_URL}" | tr '/' '\n' | grep -vc "^$") != "4" ]] ||
       [[ $(echo "${DMP_GITHUB_API_URL}" | tr '/' '\n' | grep -vc "^$") != "7" ]] ||
       [[ $(echo "${SCRIPT_GITHUB}" | tr '/' '\n' | grep -vc "^$") != "6" ]]; then
        echo_red_blink "饥荒管理平台 run.sh 脚本可能被加速站点篡改，请切换加速站点重新下载"
    fi
    echo_yellow "————————————————————————————————————————————————————————————"
    echo_green "[0]: 下载并启动饥荒管理平台"
    echo_yellow "————————————————————————————————————————————————————————————"
    echo_green "[1]: 启动饥荒管理平台"
    echo_green "[2]: 关闭饥荒管理平台"
    echo_green "[3]: 重启饥荒管理平台"
    echo_yellow "————————————————————————————————————————————————————————————"
    echo_green "[4]: 更新饥荒管理平台"
    echo_green "[5]: 强制更新饥荒管理平台"
    echo_green "[6]: 更新run.sh启动脚本"
    echo_yellow "————————————————————————————————————————————————————————————"
    echo_green "[7]: 设置虚拟内存"
    echo_green "[8]: 更改端口"
    echo_green "[9]: 安装饥荒服务器"
    echo_green "Ctrl+C: 退出脚本"
    echo_yellow "————————————————————————————————————————————————————————————"
    echo_yellow "请输入要执行的操作 [0-9]: "
}

# 检查jq
function check_jq() {
    echo_cyan "正在检查jq命令"
    if ! jq --version >/dev/null 2>&1; then
        OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
        if [[ ${OS} == "ubuntu" ]]; then
            apt install -y jq
        else
            if grep -P "^ID_LIKE=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g" | grep rhel; then
                yum install -y jq
            fi
        fi
    fi
}

function check_curl() {
    echo_cyan "正在检查curl命令"
    if ! curl --version >/dev/null 2>&1; then
        OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
        if [[ ${OS} == "ubuntu" ]]; then
            apt install -y curl
        else
            if grep -P "^ID_LIKE=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g" | grep rhel; then
                yum install -y curl
            fi
        fi
    fi
}

function check_strings() {
    echo_cyan "正在检查strings命令"
    if ! strings --version >/dev/null 2>&1; then
        OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
        if [[ ${OS} == "ubuntu" ]]; then
            apt install -y binutils
        else
            if grep -P "^ID_LIKE=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g" | grep rhel; then
                yum install -y binutils
            fi
        fi
    fi

}

# Ubuntu检查GLIBC, rhel需要下载文件手动安装
function check_glibc() {
    check_strings
    echo_cyan "正在检查GLIBC版本"
    OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
    if [[ ${OS} == "ubuntu" ]]; then
        if ! strings /lib/x86_64-linux-gnu/libc.so.6 | grep GLIBC_2.34 >/dev/null 2>&1; then
            apt update
            apt install -y libc6
        fi
    else
        echo_red "非Ubuntu系统，如GLIBC小于2.34，请手动升级"
    fi
}

# 下载函数
function download() {
    local url="$1"
    local output="$2"
    local retries="${3:-3}"
    local timeout="${4:-30}"
    
    echo_cyan "下载文件: $url"
    echo_cyan "保存到: $output"
    echo_cyan "超时时间: ${timeout}秒，重试次数: ${retries}"
    
    for i in $(seq 1 $retries); do
        echo_yellow "尝试第 $i 次下载..."
        unset_tty
        # 使用 wget 替换 curl
        if wget --timeout=$timeout --tries=1 --progress=bar -O "$output" "$url"; then
            set_tty
            echo_green "下载成功"
            return 0
        fi
        set_tty
        echo_red "第 $i 次下载失败"
        sleep 2
    done
    
    echo_red "所有下载尝试都失败"
    return 1
}

function install_dmp() {
    # 检查 jq
    if ! check_jq; then
        echo_red "jq安装失败，请检查网络连接"
        exit 1
    fi
    
    echo_cyan "开始安装 DMP..."
    
    # 获取最新版本信息
    echo_cyan "正在获取最新版本信息..."
    local latest_info
    if ! latest_info=$(wget -qO- "https://api.github.com/repos/miracleEverywhere/dst-management-platform-api/releases/latest"); then
        echo_red "获取最新版本信息失败，请检查网络连接"
        exit 1
    fi
    
    # 解析版本号和下载URL
    local latest_version
    local download_url
    if ! latest_version=$(echo "$latest_info" | jq -r '.tag_name' 2>/dev/null); then
        echo_red "解析版本号失败"
        exit 1
    fi
    
    if ! download_url=$(echo "$latest_info" | jq -r '.assets[] | select(.name=="dmp.tgz") | .browser_download_url' 2>/dev/null); then
        echo_red "解析下载链接失败"
        exit 1
    fi
    
    if [ -z "$latest_version" ] || [ -z "$download_url" ] || [ "$latest_version" = "null" ] || [ "$download_url" = "null" ]; then
        echo_red "未找到可用的版本信息"
        exit 1
    fi
    
    echo_green "最新版本: $latest_version"
    echo_cyan "下载地址: $download_url"
    
    # 构造镜像源URL
    local base_download_url="https://github.com/miracleEverywhere/dst-management-platform-api/releases/download/${latest_version}/dmp.tgz"
    local download_urls=(
        "https://gh.llkk.cc/$base_download_url"
        "https://github.dpik.top/$base_download_url"
        "https://ghfast.top/$base_download_url"
    )
    
    local mirror_names=(
        "镜像源1 (gh.llkk.cc)"
        "镜像源2 (github.dpik.top)" 
        "镜像源3 (ghfast.top)"
    )
    
    # 检查当前目录下是否已存在dmp文件
    if [ -e "dmp" ]; then
        echo_yellow "检测到当前目录下已存在dmp文件，正在删除..."
        rm -f "dmp"
        echo_green "已删除现有dmp文件"
    fi

    # 检查当前目录下是否已存在dmp.tgz文件
    if [ -e "dmp.tgz" ]; then
        echo_yellow "检测到当前目录下已存在dmp.tgz文件，正在删除..."
        rm -f "dmp.tgz"
        echo_green "已删除现有dmp.tgz文件"
    fi
    
    # 显示镜像源选择菜单
    echo_cyan "请选择下载镜像源："
    for i in "${!mirror_names[@]}"; do
        echo_green "$((i+1)). ${mirror_names[i]}"
    done
    
    local selected_mirror
    while true; do
        read -p "请输入选择 [1-3]: " selected_mirror
        
        case $selected_mirror in
            1|2|3)
                break
                ;;
            *)
                echo_red "无效选择，请输入 1-3 之间的数字"
                ;;
        esac
    done
    
    local download_success=false
    local output_file="dmp.tgz"
    
    # 使用选择的镜像源
    local mirror_index=$((selected_mirror-1))
    echo_cyan "使用镜像源：${mirror_names[mirror_index]}"
    echo_cyan "下载链接: ${download_urls[mirror_index]}"
    
    if download "${download_urls[mirror_index]}" "$output_file" 3 15; then
        echo_green "镜像源 $selected_mirror 下载成功"
        download_success=true
    else
        echo_red "镜像源 $selected_mirror 下载失败"
    fi

    # 处理下载的文件
    if [ "$download_success" = true ] && [ -f "$output_file" ]; then
        echo_cyan "正在解压文件..."
        if tar zxvf "$output_file"; then
            echo_green "文件解压成功"
            chmod 755 dmp
            echo_green "DMP 安装完成"
            echo_green "版本: $latest_version"
        else
            echo_red "文件解压失败"
            download_success=false
        fi
    else
        echo_red "镜像源下载失败"
        
        # 提供手动安装指南
        echo_cyan "请尝试以下手动安装方法:"
        echo_cyan "1. 手动下载最新版本: $download_url"
        echo_cyan "2. 将文件保存到当前目录并运行: tar zxvf dmp.tgz && chmod +x dmp"
        exit 1
    fi
}

# 检查进程状态
function check_dmp() {
    sleep 1
    if pgrep dmp >/dev/null; then
        echo_green "启动成功"
        echo_green "请在选项9安装饥荒服务器程序"
    else
        echo_red "启动失败"
        exit 1
    fi
}

# 启动主程序
function start_dmp() {
    # 检查端口是否被占用,如果被占用则退出
    port=$(ss -ltnp | awk -v port=${PORT} '$4 ~ ":"port"$" {print $4}')

    if [ -n "$port" ]; then
       echo_red "端口 $PORT 已被占用: $port", 修改 run.sh 中的 PORT 变量后重新运行
       exit 1
    fi

    check_glibc

    if [ -e "$ExeFile" ]; then
        nohup "$ExeFile" -c -l ${PORT} -s ${CONFIG_DIR} >dmp.log 2>&1 &
    else
        install_dmp
        nohup "$ExeFile" -c -l ${PORT} -s ${CONFIG_DIR} >dmp.log 2>&1 &
    fi
}

# 关闭主程序
function stop_dmp() {
    pkill -9 dmp
    echo_green "关闭成功"
    sleep 1
}

# 删除主程序、请求日志、运行日志、遗漏的压缩包
function clear_dmp() {
    echo_cyan "正在执行清理"
    rm -f dmp dmp.log dmpProcess.log
}

# 检查当前版本号
function get_current_version() {
    if [ -e "$ExeFile" ]; then
        CURRENT_VERSION=$("$ExeFile" -v | head -n1) # 获取输出的第一行作为版本号
    else
        CURRENT_VERSION="v0.0.0"
    fi
}

# 获取GitHub最新版本号
function get_latest_version() {
    check_jq
    check_curl
    LATEST_VERSION=$(curl -s -L ${DMP_GITHUB_API_URL} | jq -r .tag_name)
    if [[ -z "$LATEST_VERSION" ]]; then
        echo_red "无法获取最新版本号，请检查网络连接或GitHub API"
        exit 1
    fi
}

# 更新启动脚本
function update_script() {
    check_curl
    echo_cyan "正在更新脚本..."
    TEMP_FILE="/tmp/run.sh"
    # 使用固定 URL 替换动态生成逻辑
    url="https://ghfast.top/https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/run.sh"
    echo_yellow "正在从固定镜像下载脚本，如果下载失败请检查网络或URL有效性。"
    if download "${url}" "${TEMP_FILE}" 10; then
        if [ -e "${TEMP_FILE}" ]; then
            echo_green "run.sh下载成功"
        else
            echo_red "run.sh下载失败"
            exit 1
        fi
    else
        echo_red "run.sh下载失败"
        exit 1
    fi

    # 使用sed命令修改下载文件中的配置参数
    sed -i "s/^PORT=.*/PORT=${PORT}/" $TEMP_FILE
    sed -i "s/^SWAPSIZE=.*/SWAPSIZE=${SWAPSIZE}/" $TEMP_FILE
    sed -i "s#^CONFIG_DIR=.*#CONFIG_DIR=${CONFIG_DIR}#" $TEMP_FILE

    # 替换当前脚本
    mv -f "$TEMP_FILE" "$0" && chmod +x "$0"
    echo_green "脚本更新完成，3秒后重新启动..."
    sleep 3
    exec "$0"
}

# 更改端口
function change_port() {
    echo_yellow "当前端口: ${PORT}"

    local new_port

    read -rp "请输入新端口号 (1024-65535): " new_port

    if [[ "$new_port" =~ ^[0-9]+$ && "$new_port" -ge 1024 && "$new_port" -le 65535 ]]; then

        if [[ "$new_port" -eq "$PORT" ]]; then
            echo_yellow "新端口与当前端口相同，无需修改"
            sleep 2
            return 0
        fi

        local port_in_use
        port_in_use=$(ss -ltnp 2>/dev/null | awk -v port="${new_port}" '$4 ~ ":"port"$" {print $4}')

        if [[ -n "$port_in_use" ]]; then
            echo_red "错误：端口 ${new_port} 已被占用！"
            echo_yellow "请选择其他端口，2秒后返回..."
            sleep 2
            return 1
        fi

        if ! sed -i "s|^PORT=.*|PORT=${new_port}|" "$0"; then
            echo_red "错误：更新脚本文件失败"
            sleep 2
            return 1
        fi

        PORT="${new_port}"

        echo_green "端口已成功更改为: ${new_port}"
        echo_yellow "提示: 需要重启 DMP 服务才能使新端口生效"
        echo_cyan "请返回主菜单选择 [1]启动 或 [3]重启 服务，3秒后自动返回主菜单..."
        sleep 3
        return 0
    else
        echo_red "无效端口号！请输入 1024-65535 范围内的数字。"
        sleep 2
        return 1
    fi
}

# 设置虚拟内存
function set_swap() {
    # 修改交换区命名
    SWAPFILE=/swapfile.img

    # 检查是否已经存在交换文件
    if [ -f $SWAPFILE ]; then
        echo_green "交换文件已存在，跳过创建步骤"
    else
        echo_cyan "创建交换文件..."
        sudo fallocate -l $SWAPSIZE $SWAPFILE
        sudo chmod 600 $SWAPFILE
        sudo mkswap $SWAPFILE
        sudo swapon $SWAPFILE
        echo_green "交换文件创建并启用成功"
    fi

    # 添加到 /etc/fstab 以便开机启动
    if ! grep -q "$SWAPFILE" /etc/fstab; then
        echo_cyan "将交换文件添加到 /etc/fstab "
        echo "$SWAPFILE none swap sw 0 0" | sudo tee -a /etc/fstab
        echo_green "交换文件已添加到开机启动"
    else
        echo_green "交换文件已在 /etc/fstab 中，跳过添加步骤"
    fi

    # 更改swap配置并持久化
    sysctl -w vm.swappiness=20
    sysctl -w vm.min_free_kbytes=100000
    echo -e 'vm.swappiness = 20\nvm.min_free_kbytes = 100000\n' > /etc/sysctl.d/dmp_swap.conf

    echo_green "系统swap设置成功"
}

# 安装服务器
function install_dst() {
    read -p "您确定要安装 Don't Starve Together 服务器吗？(y/n): " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        echo_yellow "安装已取消."
        return
    fi

    echo_cyan "正在安装 Don't Starve Together 服务器..."
    sudo dpkg --add-architecture i386
    sudo apt-get update
    sudo apt-get install -y libcurl4-gnutls-dev:i386
    sudo apt-get install -y lib32gcc1
    sudo apt-get install -y lib32stdc++6
    sudo apt-get install -y libcurl4-gnutls-dev
    sudo apt-get install -y libgcc1
    sudo apt-get install -y libstdc++6
    sudo apt-get install -y screen
    sudo apt-get install -y unzip
    echo_green "环境依赖安装完毕"

    mkdir -p ~/.klei/DMP_BACKUP
    mkdir -p ~/.klei/DMP_MOD/not_ugc
    mkdir -p ~/.klei/DoNotStarveTogether/MyDediServer/Master
    mkdir -p ~/.klei/DoNotStarveTogether/MyDediServer/Caves
    touch ~/.klei/DoNotStarveTogether/MyDediServer/cluster_token.txt
    touch ~/.klei/DoNotStarveTogether/MyDediServer/adminlist.txt
    touch ~/.klei/DoNotStarveTogether/MyDediServer/blocklist.txt
    touch ~/.klei/DoNotStarveTogether/MyDediServer/whitelist.txt
    echo_green "饥荒初始文件夹创建完成"

    mkdir ~/steamcmd
    cd ~/steamcmd
    
    file_name="steamcmd_linux.tar.gz"
    check_for_file "$file_name"

    if [ $? -eq 0 ]; then
        echo_yellow "$file_name 存在，正在删除..."
        rm "$file_name"
    else
        echo_cyan "$file_name 不存在，继续下载steamcmd"
    fi

    wget https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz
    tar -xvzf steamcmd_linux.tar.gz
    ./steamcmd.sh +login anonymous +force_install_dir "$install_dir" +app_update 343050 validate +quit
    
    echo_cyan "正在验证服务器安装..."
    cd ~/dst/bin/ || {
        echo
        echo_red "======================================"
        echo_red "✘ 服务器安装验证失败！"
        echo_red "✘ 请重新安装！"
        echo_red "======================================"
        echo
        cd "$HOME" #返回root根目录
        exit 1
    }

    # 服务器安装验证通过后，执行MOD修复
    if [ -d ~/dst/bin/ ]; then
        echo_green "=================================================="
        echo_green "✅ 服务器安装验证通过！"
        echo_green "=================================================="
        
        echo_cyan "正在执行MOD修复..."
        cp ~/steamcmd/linux32/libstdc++.so.6 ~/dst/bin/lib32/
        cp ~/steamcmd/linux32/steamclient.so ~/dst/bin/lib32/
        echo_green "MOD更新bug已修复"
        
        echo_green "=================================================="
        echo_green "✅ Don't Starve Together 服务器安装完成！"
        echo_green "=================================================="
    else
        echo_red "=================================================="
        echo_red "✘ 服务器安装验证失败！"
        echo_red "✘ 请重新安装一次！"
        echo_red "=================================================="
        cd "$HOME" #返回root根目录
        exit 1
    fi

    # 无论成功还是失败，最后都返回root根目录
    cd "$HOME"
    echo
}

# 使用无限循环让用户输入命令
while true; do
    # 提示用户输入
    prompt_user
    # 读取用户输入
    read -r command
    # 使用 case 语句判断输入的命令
    case $command in
    0)
        set_tty
        clear_dmp
        install_dmp
        start_dmp
        check_dmp
        unset_tty
        break
        ;;
    1)
        set_tty
        start_dmp
        check_dmp
        unset_tty
        break
        ;;
    2)
        set_tty
        stop_dmp
        unset_tty
        break
        ;;
    3)
        set_tty
        stop_dmp
        start_dmp
        check_dmp
        echo_green "重启成功"
        unset_tty
        break
        ;;
    4)
        set_tty
        get_current_version
        get_latest_version
        if [[ "$(echo -e "$CURRENT_VERSION\n$LATEST_VERSION" | sort -V | head -n1)" == "$CURRENT_VERSION" && "$CURRENT_VERSION" != "$LATEST_VERSION" ]]; then
            echo_yellow "当前版本 ($CURRENT_VERSION) 小于最新版本 ($LATEST_VERSION)，即将更新"
            stop_dmp
            clear_dmp
            install_dmp
            start_dmp
            check_dmp
            echo_green "更新完成"
        else
            echo_green "当前版本 ($CURRENT_VERSION) 已是最新版本，无需更新"
        fi
        unset_tty
        break
        ;;
    5)
        set_tty
        stop_dmp
        clear_dmp
        install_dmp
        start_dmp
        check_dmp
        echo_green "强制更新完成"
        unset_tty
        break
        ;;
    6)
        set_tty
        update_script
        unset_tty
        break
        ;;
    7)
        set_tty
        set_swap
        unset_tty
        break
        ;;
    8)
        set_tty
        change_port
        unset_tty
        continue
        ;;
    9)
        install_dst
        unset_tty
        break
        ;;
    *)
        echo_red "请输入正确的数字 [0-9]"
        continue
        ;;
    esac
done
