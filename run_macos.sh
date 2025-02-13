#!/bin/zsh

########################################################
# 用户自定义设置请修改下方变量，其他变量请不要修改

# dmp暴露端口，即网页打开时所用的端口
PORT=80

# 数据库文件所在目录，例如：./config
CONFIG_DIR="./"

########################################################

# 下方变量请不要修改，否则可能会出现异常
ExeFile="$HOME/dmp"

if ! brew --version >/dev/null 2>&1; then
    echo -e "\e[31mbrew未安装 (brew NOT installed) \e[0m"
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

# 检查jq
function check_jq() {
    echo -e "\e[36m正在检查jq命令(Checking jq command) \e[0m"
    if ! jq --version >/dev/null 2>&1; then
        brew install jq
    fi
}

function check_curl() {
    echo -e "\e[36m正在检查curl命令(Checking curl command) \e[0m"
    if ! curl --version >/dev/null 2>&1; then
        brew install curl
    fi
}

function check_wget() {
    echo -e "\e[36m正在检查curl命令(Checking wget command) \e[0m"
    if ! wget --version >/dev/null 2>&1; then
        brew install wget
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
    check_wget
    # 原GitHub下载链接
    GITHUB_URL=$(curl -s https://api.github.com/repos/miracleEverywhere/dst-management-platform-api/releases/latest | jq -r ".assets[1].browser_download_url")
    # 加速站点，失效从 https://github.akams.cn/ 重新搜索。
    PRIMARY_PROXY="https://ghproxy.cc/"   # 主加速站点
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
            echo -e "\e[31m备用加速站点下载失败: wget 返回码为 $?, 尝试从 Gitee 下载\e[0m"
            # Gitee下载链接
            GITEE_URL=$(curl -s https://gitee.com/api/v5/repos/s763483966/dst-management-platform-api/releases/latest | jq -r ".assets[1].browser_download_url")
            # 尝试从 Gitee 下载
            echo -e "\e[36m尝试通过国内站点下载 Gitee\e[0m"
            if download "$GITEE_URL" 5 10; then
                echo -e "\e[32m从 Gitee 下载成功！\e[0m"
            else
                echo -e "\e[31m从 Gitee 下载失败: wget 返回码为 $?, 尝试从原 GitHub 链接下载\e[0m"

                # 尝试从原 GitHub 链接下载
                echo -e "\e[36m尝试通过原站点下载 GitHub\e[0m"
                if download "$GITHUB_URL" 5 10; then
                    echo -e "\e[32m从原 GitHub 链接下载成功！\e[0m"
                else
                    echo -e "\e[31m从原 GitHub 链接下载失败: wget 返回码为 $?, 下载失败！\e[0m"
                    exit 1
                fi
            fi
        fi
    fi

    tar zxvf dmp_darwin.tgz
    rm -f dmp_darwin.tgz
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
    rm -f dmp*
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
        stop_dmp
        clear_dmp
        install_dmp
        start_dmp
        check_dmp
        echo -e "\e[32m更新完成 (Update completed) \e[0m"
        break
        ;;
    *)
        echo -e "\e[31m无效输入，请输入 0, 1, 2, 3, 4 (Invalid input, please enter 0, 1, 2, 3, 4) \e[0m"
        continue
        ;;
    esac
done
