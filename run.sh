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

# 检查用户，只能使用root执行
if [[ "${USER}" != "root" ]]; then
    echo -e "\e[31m请使用root用户执行此脚本 (Please run this script as the root user) \e[0m"
    exit 1
fi

# 定义一个函数来提示用户输入
function prompt_user() {
    echo -e "\e[32m饥荒管理平台(DMP) \e[0m"
    echo -e "\e[32m--- https://github.com/miracleEverywhere/dst-management-platform-api --- \e[0m"
    echo -e "\e[33m———————————————————————————————————————————————————————————— \e[0m"
    echo -e "\e[32m[0]: 下载并启动服务(Download and start the service) \e[0m"
    echo -e "\e[33m———————————————————————————————————————————————————————————— \e[0m"
    echo -e "\e[32m[1]: 启动服务(Start the service) \e[0m"
    echo -e "\e[32m[2]: 关闭服务(Stop the service) \e[0m"
    echo -e "\e[32m[3]: 重启服务(Restart the service) \e[0m"
    echo -e "\e[33m———————————————————————————————————————————————————————————— \e[0m"
    echo -e "\e[32m[4]: 更新管理平台(Update management platform) \e[0m"
    echo -e "\e[32m[5]: 强制更新平台(Force update platform) \e[0m"
    echo -e "\e[32m[6]: 更新启动脚本(Update startup script) \e[0m"
    echo -e "\e[33m———————————————————————————————————————————————————————————— \e[0m"
    echo -e "\e[32m[7]: 设置虚拟内存(Setup swap) \e[0m"
    echo -e "\e[32m[8]: 退出脚本(Exit script) \e[0m"
    echo -e "\e[33m———————————————————————————————————————————————————————————— \e[0m"
    echo -e "\e[33m请输入选择(Please enter your selection) [0-8]:  \e[0m"
}

# 检查jq
function check_jq() {
    echo -e "\e[36m正在检查jq命令(Checking jq command) \e[0m"
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
    echo -e "\e[36m正在检查curl命令(Checking curl command) \e[0m"
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
    echo -e "\e[36m正在检查strings命令(Checking strings command) \e[0m"
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
    echo -e "\e[36m正在检查GLIBC版本(Checking GLIBC version) \e[0m"
    OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
    if [[ ${OS} == "ubuntu" ]]; then
        if ! strings /lib/x86_64-linux-gnu/libc.so.6 | grep GLIBC_2.34; then
            apt update
            apt install -y libc6
        fi
    else
        echo -e "\e[31m非Ubuntu系统，如GLIBC小于2.34，请手动升级(For systems other than Ubuntu, if the GLIBC version is less than 2.34, please upgrade manually) \e[0m"
    fi
}

# 下载函数:下载链接,尝试次数,超时时间(s)
function download() {
    local download_url="$1"
    local tries="$2"
    local timeout="$3"

    wget -q --show-progress --tries="$tries" --timeout="$timeout" "$download_url"

    return $? # 返回 wget 的退出状态
}

# 安装主程序
function install_dmp() {
    check_jq
    check_curl
    # 原GitHub下载链接
    GITHUB_URL=$(curl -s https://api.github.com/repos/miracleEverywhere/dst-management-platform-api/releases/latest | jq -r '.assets[] | select(.name == "dmp.tgz") | .browser_download_url')
    # 加速站点，失效从 https://github.akams.cn/ 重新搜索。
    PRIMARY_PROXY="https://github.acmsz.top/"   # 主加速站点
    SECONDARY_PROXY="https://ghproxy.cn/" # 备用加速站点
    # 尝试通过主加速站点下载 GitHub
    echo -e "\e[36m尝试通过主加速站点下载 GitHub\e[0m"
    if download "$PRIMARY_PROXY$GITHUB_URL" 5 10; then
        echo -e "\e[32m通过主加速站点下载成功！\e[0m"
    else
        echo -e "\e[31m主加速站点下载失败: wget 返回码为 $?, 尝试备用加速站点下载 GitHub\e[0m"

        # 尝试通过备用加速站点下载 GitHub
        echo -e "\e[36m尝试通过备用加速站点下载 GitHub\e[0m"
        if download "$SECONDARY_PROXY$GITHUB_URL" 5 10; then
            echo -e "\e[32m通过备用加速站点下载成功！\e[0m"
        else
            echo -e "\e[31m备用加速站点下载失败: wget 返回码为 $?\e[0m"
            exit 1
        fi
    fi

    tar zxvf dmp.tgz
    rm -f dmp.tgz
    chmod +x "$ExeFile"
}

# 检查进程状态
function check_dmp() {
    sleep 1
    if pgrep dmp >/dev/null; then
        echo -e "\e[32m启动成功 (Startup Success) \e[0m"
    else
        echo -e "\e[31m启动失败 (Startup Fail) \e[0m"
        exit 1
    fi
}

# 启动主程序
function start_dmp() {
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
    echo -e "\e[32m关闭成功 (Shutdown Success) \e[0m"
    sleep 1
}

# 删除主程序、请求日志、运行日志、遗漏的压缩包
function clear_dmp() {
    echo -e "\e[36m正在执行清理 (Cleaning Files) \e[0m"
    rm -f dmp dmp.log dmpProcess.log
}

# 检查当前版本号
function get_current_version() {
    if [ -e "$ExeFile" ]; then
        CURRENT_VERSION=$("$ExeFile" -v | head -n1) # 获取输出的第一行作为版本号
    else
        CURRENT_VERSION="0.0.0"
    fi
}

# 获取GitHub最新版本号
function get_latest_version() {
    check_jq
    check_curl
    LATEST_VERSION=$(curl -s https://api.github.com/repos/miracleEverywhere/dst-management-platform-api/releases/latest | jq -r .tag_name | grep -oP '(\d+\.)+\d+')
    if [[ -z "$LATEST_VERSION" ]]; then
        echo -e "\e[31m无法获取最新版本号，请检查网络连接或GitHub API (Failed to fetch the latest version, please check network or GitHub API) \e[0m"
        exit 1
    fi
}

# 更新启动脚本
update_script() {
    echo -e "\e[36m正在更新脚本... \e[0m"
    TEMP_FILE="/tmp/run.sh"
    SCRIPT_GITHUB="https://github.acmsz.top/https://github.com/miracleEverywhere/dst-management-platform-api/raw/refs/heads/master/run.sh"
    # 读取旧脚本中的 PORT 值
    OLD_PORT=$(grep "^PORT=" "$0" | cut -d'=' -f2)
    # 尝试从 GitHub 下载
    if curl --connect-timeout 10 -sL "$SCRIPT_GITHUB" -o "$TEMP_FILE"; then
        echo -e "\e[32m从 GitHub 下载成功！ \e[0m"
    else
        echo -e "\e[31m更新脚本失败：无法从GitHub下载脚本 \e[0m" >&2
        echo -e "\e[31m请前往github.akams.cn手动下载，原地址为：${SCRIPT_GITHUB} \e[0m"
        exit 1
    fi

    # 将旧 PORT 值写入新脚本
    sed -i "s/^PORT=.*/PORT=${OLD_PORT}/" "$TEMP_FILE"
    # 替换当前脚本
    mv -f "$TEMP_FILE" "$0" && chmod +x "$0"
    echo -e "\e[32m脚本更新完成，3 秒后重新启动... \e[0m"
    sleep 3
    exec "$0"
}

# 设置虚拟内存
function set_swap() {
    SWAPFILE=/swapfile

    # 检查是否已经存在交换文件
    if [ -f $SWAPFILE ]; then
        echo -e "\e[32m交换文件已存在，跳过创建步骤 \e[0m"
    else
        echo -e "\e[36m创建交换文件... \e[0m"
        sudo fallocate -l $SWAPSIZE $SWAPFILE
        sudo chmod 600 $SWAPFILE
        sudo mkswap $SWAPFILE
        sudo swapon $SWAPFILE
        echo -e "\e[32m交换文件创建并启用成功 \e[0m"
    fi

    # 添加到 /etc/fstab 以便开机启动
    if ! grep -q "$SWAPFILE" /etc/fstab; then
        echo -e "\e[36m将交换文件添加到 /etc/fstab  \e[0m"
        echo "$SWAPFILE none swap sw 0 0" | sudo tee -a /etc/fstab
        echo -e "\e[32m交换文件已添加到开机启动 \e[0m"
    else
        echo -e "\e[32m交换文件已在 /etc/fstab 中，跳过添加步骤 \e[0m"
    fi

    # 更改swap配置并持久化
    sysctl -w vm.swappiness=20
    sysctl -w vm.min_free_kbytes=100000
    echo -e 'vm.swappiness = 20\nvm.min_free_kbytes = 100000\n' > /etc/sysctl.d/dmp_swap.conf

    echo -e "\e[32m系统swap设置成功 (System swap setting completed) \e[0m"
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
        clear_dmp
        install_dmp
        start_dmp
        check_dmp
        break
        ;;
    1)
        start_dmp
        check_dmp
        break
        ;;
    2)
        stop_dmp
        break
        ;;
    3)
        stop_dmp
        start_dmp
        check_dmp
        echo -e "\e[32m重启成功 (Restart Success) \e[0m"
        break
        ;;
    4)
        get_current_version
        get_latest_version
        if [[ "$(echo -e "$CURRENT_VERSION\n$LATEST_VERSION" | sort -V | head -n1)" == "$CURRENT_VERSION" && "$CURRENT_VERSION" != "$LATEST_VERSION" ]]; then
            echo -e "\e[33m当前版本 ($CURRENT_VERSION) 小于最新版本 ($LATEST_VERSION)，即将更新 (Updating to the latest version) \e[0m"
            stop_dmp
            clear_dmp
            install_dmp
            start_dmp
            check_dmp
            echo -e "\e[32m更新完成 (Update completed) \e[0m"
        else
            echo -e "\e[32m当前版本 ($CURRENT_VERSION) 已是最新版本 ($LATEST_VERSION)，无需更新 (No update needed) \e[0m"
        fi
        break
        ;;
    5)
        stop_dmp
        clear_dmp
        install_dmp
        start_dmp
        check_dmp
        echo -e "\e[32m强制更新完成 (Force update completed) \e[0m"
        break
        ;;
    6)
        update_script
        break
        ;;
    7)
        set_swap # 调用设置虚拟内存的函数
        break
        ;;
    8)
        exit 0
        break
        ;;
    *)
        echo -e "\e[31m请输入正确的数字 [0-8](Please enter the correct number [0-8]) \e[0m"
        continue
        ;;
    esac
done
