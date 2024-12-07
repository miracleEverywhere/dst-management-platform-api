#!/bin/bash

USER=`whoami`
ExeFile="$HOME/dmp"

# 检查用户，只能使用root执行
if [[ "${USER}" != "root" ]];then
    echo  -e "\e[31m请使用root用户执行此脚本 (Please run this script as the root user) \e[0m"
    exit 1
fi

# 定义一个函数来提示用户输入
function prompt_user() {
    echo -e "\e[33m请输入需要执行的操作(Please enter the operation to be performed): \e[0m"
    echo -e "\e[32m[0]: 下载并启动服务(Download and start the service) \e[0m"
    echo -e "\e[32m[1]: 启动服务(Start the service) \e[0m"
    echo -e "\e[32m[2]: 关闭服务(Stop the service) \e[0m"
    echo -e "\e[32m[3]: 重启服务(Restart the service) \e[0m"
    echo -e "\e[32m[4]: 更新服务(Update the service) \e[0m"
}

# Ubuntu检查GLIBC, rhel需要下载文件手动安装
function check_glibc() {
    echo -e "\e[32m正在检查GLIBC版本(Checking GLIBC version) \e[0m"
    OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
    if [[ ${OS} == "ubuntu" ]]; then
        strings /lib/x86_64-linux-gnu/libc.so.6 | grep GLIBC_2.34
        if (($? != 0)); then
            apt install -y libc6
        fi
    else
        echo -e "\e[32m非Ubuntu系统，如GLIBC小于2.34，请手动升级(For systems other than Ubuntu, if the GLIBC version is less than 2.34, please upgrade manually) \e[0m"
    fi
}

# 安装主程序
function install_dmp() {
    wget https://dmp-1257278878.cos.ap-chengdu.myqcloud.com/dmp.tgz
    tar zxvf dmp.tgz
    rm -f dmp.tgz
    chmod +x $ExeFile
}

# 检查进程状态
function check_dmp() {
    ps -ef | grep dmp | grep -v grep > /dev/null
    if (($? == 0)); then
        echo -e "\e[32m启动成功 (Startup Success) \e[0m"
    else
        echo -e "\e[31m启动失败 (Startup Fail) \e[0m"
        exit 1
    fi
}

# 启动主程序
function start_dmp() {
    if [ -e $ExeFile ];then
        nohup $ExeFile -c 2>&1 > dmp.log &
    else
        install_dmp
        nohup $ExeFile -c 2>&1 > dmp.log &
    fi
}

# 关闭主程序
function stop_dmp() {
    pkill -9 dmp
    echo -e "\e[32m关闭成功 (Shutdown Success) \e[0m"
}

# 使用无限循环让用户输入命令
while true; do
    # 提示用户输入
    prompt_user
    # 读取用户输入
    read command
    # 使用 case 语句判断输入的命令
    case $command in
        0)
            check_glibc
            install_dmp
            start_dmp
            check_dmp
            break
            ;;
        1)
            check_glibc
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
            sleep 1
            check_glibc
            start_dmp
            check_dmp
            echo -e "\e[32m重启成功 (Restart Success) \e[0m"
            break
            ;;
        4)
            stop_dmp
            rm -f dmp*
            check_glibc
            install_dmp
            start_dmp
            check_dmp
            echo -e "\e[32m更新成功 (Restart Success) \e[0m"
            break
            ;;
        *)
            echo  -e "\e[31m无效输入，请输入 0, 1, 2, 3, 4 (Invalid input, please enter 0, 1, 2, 3, 4) \e[0m"
            continue
            ;;
    esac
done