#!/bin/bash

###########################################
# 用户自定义设置请修改下方变量，其他变量请不要修改 #
###########################################

# --------------- ↓可修改↓ --------------- #
# dmp-lite暴露端口，即网页打开时所用的端口
PORT=9999

# 虚拟内存大小，例如 1G 4G等
SWAPSIZE=2G

# 加速节点
ACCELERATION_SITE=(
	"gh-proxy.com"
	"github.dpik.top"
	"ghfast.top"
)
# --------------- ↑可修改↑ --------------- #

###########################################
#     下方变量请不要修改，否则可能会出现异常     #
###########################################

CURRENT_USER=$(whoami)
SCRIPT_FILE="$(cd "$(dirname "$0")" && pwd)/$(basename "$0")"
WORK_DIR=$(dirname "${SCRIPT_FILE}")
ExeFile="${WORK_DIR}/dmp-lite"
LOG_DIR="${WORK_DIR}/logs"
LOG_FILE="${LOG_DIR}/dmp-lite.log"

cd "${WORK_DIR}" || exit 1

DMP_GITHUB_HOME_URL="https://github.com/miracleEverywhere/dst-management-platform-api"
DMP_GITHUB_API_URL="https://api.github.com/repos/miracleEverywhere/dst-management-platform-api/releases/latest"
SCRIPT_GITHUB="https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/lite/run.sh"
DMP_HOME="https://miraclesses.top/"

# lite 发布流水线只产出 Linux 安装包
ARCHIVE_NAME="dmp-lite.tgz"

ACCELERATED_URL=""

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

if [[ "$(uname -s)" != "Linux" ]]; then
	echo_red "DMP Lite 目前仅支持 Linux，当前系统: $(uname -s)"
	exit 1
fi

if [[ ! "${PORT}" =~ ^[0-9]+$ ]] || ((PORT < 1 || PORT > 65535)); then
	echo_red "PORT 必须是 1-65535 之间的数字，当前端口: ${PORT}"
	exit 1
fi

# 保存原始参数，供 update_script 和 auto_start_dmp 使用
ORIGINAL_ARGS=("$@")
RUN_SH_CMD=$(printf '%q ' "$SCRIPT_FILE" "${ORIGINAL_ARGS[@]}")

SUDO=""
if [[ "$1" == "no-root" ]]; then
	SUDO="sudo"
	shift
	echo_yellow "以非root模式运行，需要root权限的操作将使用sudo"
elif [[ "${CURRENT_USER}" != "root" ]]; then
	echo_red "请使用root用户执行此脚本，或使用 ./run.sh no-root 以非root模式运行"
	exit 1
fi

# 设置全局stderr为红色并添加固定格式
function set_tty() {
	exec 3>&2
	exec 2> >(while read -r line; do echo_red "[$(date +'%F %T')] [ERROR] ${line}" >&2; done)
}

# 恢复stderr颜色
function unset_tty() {
	exec 2>&3
	exec 3>&-
}

# 定义一个函数来提示用户输入
function prompt_user() {
	clear
	echo_green "饥荒管理平台 Lite (DMP Lite)"
	echo_green "--- ${DMP_GITHUB_HOME_URL} ---"
	if [[ $(echo "${DMP_GITHUB_HOME_URL}" | tr '/' '\n' | grep -vc "^$") != "4" ]] ||
		[[ $(echo "${DMP_GITHUB_API_URL}" | tr '/' '\n' | grep -vc "^$") != "7" ]] ||
		[[ $(echo "${SCRIPT_GITHUB}" | tr '/' '\n' | grep -vc "^$") != "7" ]]; then
		echo_red_blink "饥荒管理平台 Lite run.sh 脚本可能被加速站点篡改，请切换加速站点重新下载"
		echo_cyan "${DMP_HOME}"
		echo_green "如有疑问，请查阅帮助文档"
	fi
	echo_yellow "————————————————————————————————————————————————————————————"
	echo_green "[0]: 下载并启动饥荒管理平台 Lite"
	echo_yellow "————————————————————————————————————————————————————————————"
	echo_green "[1]: 启动饥荒管理平台 Lite"
	echo_green "[2]: 关闭饥荒管理平台 Lite"
	echo_green "[3]: 重启饥荒管理平台 Lite"
	echo_yellow "————————————————————————————————————————————————————————————"
	echo_green "[4]: 更新饥荒管理平台 Lite"
	echo_green "[5]: 更新run.sh启动脚本"
	echo_yellow "————————————————————————————————————————————————————————————"
	echo_green "[6]: 设置虚拟内存"
	echo_green "[7]: 设置开机自启"
	echo_green "[8]: 退出脚本"
	echo_yellow "————————————————————————————————————————————————————————————"
	echo_yellow "请输入要执行的操作 [0-8]: "
}

# 加速节点选择
function generate_acceleration() {
	local origin_url="$1"
	# 输出加速节点让用户选择
	echo_cyan "饥荒管理平台部署在Github, 请选择一个加速节点进行加速:"
	for i in "${!ACCELERATION_SITE[@]}"; do
		echo_green "$((i + 1)). ${ACCELERATION_SITE[i]}"
	done
	acc_site_len=${#ACCELERATION_SITE[@]}
	echo_green "$((acc_site_len + 1)). 不使用加速节点直接下载"

	# 获取用户输入
	read -r -p "请输入选择 (1-$((acc_site_len + 1))): " selected_mirror

	# 输入验证和处理
	while true; do
		# 如果输入为空，使用默认值
		if [[ -z "$selected_mirror" ]]; then
			selected_mirror=1
			break
		fi
		# 验证是否为数字
		if [[ ! "$selected_mirror" =~ ^[0-9]+$ ]]; then
			echo_yellow "输入无效，请输入数字 (1-$((acc_site_len + 1)))"
			read -r -p "请重新输入: " selected_mirror
			continue
		fi
		# 验证范围
		if ((selected_mirror < 1 || selected_mirror > acc_site_len + 1)); then
			echo_yellow "选择超出范围，请输入 1 到 $((acc_site_len + 1)) 之间的数字"
			read -r -p "请重新输入: " selected_mirror
			continue
		fi
		break
	done

	# 根据选择设置URL
	if ((selected_mirror >= 1 && selected_mirror <= acc_site_len)); then
		index=$((selected_mirror - 1))
		ACCELERATED_URL="https://${ACCELERATION_SITE[index]}/${origin_url}"
		echo_green "已选择加速节点 ${selected_mirror}: ${ACCELERATION_SITE[index]}"
	elif ((selected_mirror == acc_site_len + 1)); then
		ACCELERATED_URL="${origin_url}"
		echo_green "已选择直接下载"
	fi
}

# 通用包安装函数
function install_pkg() {
	local pkg="$1"
	OS=$(grep -P "^ID=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g")
	if [[ ${OS} == "ubuntu" ]]; then
		${SUDO} apt install -y "$pkg"
	else
		if grep -P "^ID_LIKE=" /etc/os-release | awk -F'=' '{print($2)}' | sed "s/['\"]//g" | grep rhel; then
			${SUDO} yum install -y "$pkg"
		fi
	fi
}

# 检查jq
function check_jq() {
	echo_cyan "正在检查jq命令"
	if ! jq --version >/dev/null 2>&1; then
		install_pkg jq
		# 再次检查是否安装成功
		if ! jq --version >/dev/null 2>&1; then
			echo_red "jq 安装失败，请手动安装 jq 后重试"
			exit 1
		fi
	fi
}

function check_curl() {
	echo_cyan "正在检查curl命令"
	if ! curl --version >/dev/null 2>&1; then
		install_pkg curl
		# 再次检查是否安装成功
		if ! curl --version >/dev/null 2>&1; then
			echo_red "curl 安装失败，请手动安装 curl 后重试"
			exit 1
		fi
	fi
}

# 下载函数:下载链接,输出文件,超时时间(s)
function download() {
	# 显示详细进度
	local url="$1"
	local output="$2"
	local timeout="$3"
	curl -L --connect-timeout "${timeout}" --progress-bar -o "${output}" "${url}" 2>&1

	local curl_exit_code=$?

	if [ $curl_exit_code -eq 0 ]; then
		echo_green "下载完成: $output"
	else
		echo_red "下载失败 (退出码: $curl_exit_code)"
	fi

	return $curl_exit_code
}

# 安装 Lite 程序
function install_dmp() {
	check_jq
	check_curl

	# 获取GITHUB最新releases信息（一次请求获取所有数据）
	echo_cyan "正在获取下载信息"
	release_json=$(curl -s -L "${DMP_GITHUB_API_URL}")
	if [[ -z "$release_json" ]]; then
		echo_red "获取最新版本信息失败，请检查网络连接"
		exit 1
	fi
	github_url=$(echo "$release_json" | jq -r --arg name "$ARCHIVE_NAME" '.assets[] | select(.name == $name) | .browser_download_url')
	if [[ -z "$github_url" || "$github_url" == "null" ]]; then
		echo_red "未找到下载文件 ${ARCHIVE_NAME}，请检查网络连接"
		exit 1
	fi
	github_digest=$(echo "$release_json" | jq -r --arg name "$ARCHIVE_NAME" '.assets[] | select(.name == $name) | .digest' | awk -F':' '{print $2}')
	if [[ -z "$github_digest" || "$github_digest" == "null" ]]; then
		echo_red "获取安装包校验值失败，请检查网络连接"
		exit 1
	fi

	generate_acceleration "${github_url}"
	url=$ACCELERATED_URL

	# 开始下载
	echo_cyan "正在从${url}进行下载"
	if ! download "${url}" "${WORK_DIR}/${ARCHIVE_NAME}" 10; then
		echo_red "DMP Lite下载失败"
		exit 1
	fi

	# 验证 SHA-256
	if command -v sha256sum >/dev/null 2>&1; then
		file_digest=$(sha256sum "${WORK_DIR}/${ARCHIVE_NAME}" | awk '{print $1}')
	else
		file_digest=$(shasum -a 256 "${WORK_DIR}/${ARCHIVE_NAME}" | awk '{print $1}')
	fi
	if [[ "$github_digest" != "$file_digest" ]]; then
		rm -f "${WORK_DIR}/${ARCHIVE_NAME}"
		echo_red "DMP Lite下载失败，安装包校验值不匹配"
		exit 1
	fi

	# 先解压到临时目录，确认内容完整后再替换，避免更新中断留下半成品
	local tmp_dir
	tmp_dir=$(mktemp -d "${WORK_DIR}/.dmp-lite-install.XXXXXX")
	if [[ -z "$tmp_dir" ]]; then
		rm -f "${WORK_DIR}/${ARCHIVE_NAME}"
		echo_red "创建临时目录失败"
		exit 1
	fi
	if ! tar zxf "${WORK_DIR}/${ARCHIVE_NAME}" -C "${tmp_dir}"; then
		rm -rf "$tmp_dir"
		rm -f "${WORK_DIR}/${ARCHIVE_NAME}"
		echo_red "解压安装包失败"
		exit 1
	fi
	rm -f "${WORK_DIR}/${ARCHIVE_NAME}"

	if [[ ! -f "${tmp_dir}/dmp-lite" || ! -f "${tmp_dir}/dist-lite/index.html" ]]; then
		rm -rf "$tmp_dir"
		echo_red "安装包内容不完整"
		exit 1
	fi

	# 覆盖前先停止，避免二进制或前端目录被占用
	pkill -9 -x dmp-lite 2>/dev/null

	mv -f "${tmp_dir}/dmp-lite" "$ExeFile"
	rm -rf "${WORK_DIR}/dist-lite"
	mv -f "${tmp_dir}/dist-lite" "${WORK_DIR}/dist-lite"
	if [[ -f "${tmp_dir}/manual_install.sh" ]]; then
		mv -f "${tmp_dir}/manual_install.sh" "${WORK_DIR}/manual_install.sh"
	fi
	if [[ -f "${tmp_dir}/manual_update.sh" ]]; then
		mv -f "${tmp_dir}/manual_update.sh" "${WORK_DIR}/manual_update.sh"
	fi
	# 安装包内的 run.sh 不在这里覆盖，脚本更新走单独的菜单项
	chmod +x "$ExeFile" "${WORK_DIR}/manual_install.sh" "${WORK_DIR}/manual_update.sh" 2>/dev/null
	rm -rf "$tmp_dir"
	echo_green "DMP Lite安装完成"
}

# 检查进程状态
function check_dmp() {
	sleep 1
	if pgrep -x dmp-lite >/dev/null; then
		echo_green "启动成功"
	else
		get_panic_logs
		echo_red "启动失败"
		exit 1
	fi
}

# 启动 Lite 程序
function start_dmp() {
	stop_dmp

	# 检查端口是否被占用,如果被占用则退出
	if command -v ss >/dev/null 2>&1; then
		port=$(ss -ltn | awk -v port="${PORT}" '$4 ~ ":"port"$" {print $4}')
		if [ -n "$port" ]; then
			echo_red "端口 $PORT 已被占用: $port, 修改 run.sh 中的 PORT 变量后重新运行或检查饥荒管理平台是否正在运行"
			exit 1
		fi
	fi

	# Lite 程序不会自动安装游戏依赖，仅做启动前提示
	check_runtime_env

	if [[ ! -x "$ExeFile" || ! -f "${WORK_DIR}/dist-lite/index.html" ]]; then
		install_dmp
	fi

	mkdir -p "$LOG_DIR"
	nohup "$ExeFile" -bind "${PORT}" >"${LOG_FILE}" 2>&1 &
}

# 关闭 Lite 程序
function stop_dmp() {
	pkill -x dmp-lite 2>/dev/null
	sleep 1
	pkill -9 -x dmp-lite 2>/dev/null
	sleep 1
}

# 删除 Lite 程序、日志、遗漏的压缩包
function clear_dmp() {
	echo_cyan "正在执行清理"
	pkill -9 -x dmp-lite 2>/dev/null
	rm -f "$ExeFile" "${WORK_DIR}/${ARCHIVE_NAME}"
	rm -rf "$LOG_DIR" "${WORK_DIR}/dist-lite"
	rm -f "${WORK_DIR}/manual_install.sh" "${WORK_DIR}/manual_update.sh"
}

# 启动前检查 screen、SteamCMD 和 DST 文件，只提示不安装
function check_runtime_env() {
	if ! command -v screen >/dev/null 2>&1; then
		echo_yellow "未检测到 screen 命令，游戏进程管理将不可用，请先安装 screen"
	fi
	if [[ ! -d "${WORK_DIR}/dst" || ! -x "${WORK_DIR}/steamcmd/steamcmd.sh" ]]; then
		echo_yellow "未检测到 dst 或 steamcmd，请先安装游戏依赖（DMP Lite 不会自动安装）"
	fi
}

# 更新 Lite 程序
function update_dmp() {
	echo_yellow "DMP Lite 不支持版本比较，将直接下载并覆盖最新版本"
	stop_dmp
	install_dmp
	start_dmp
	check_dmp
	echo_green "更新完成"
}

# 更新启动脚本
function update_script() {
	check_curl

	# 生成加速链接
	generate_acceleration "${SCRIPT_GITHUB}"
	url=$ACCELERATED_URL

	TEMP_FILE="${WORK_DIR}/.run.sh.new"

	echo_cyan "正在从${url}进行下载"

	if ! download "${url}" "${TEMP_FILE}" 10; then
		rm -f "${TEMP_FILE}"
		echo_red "run.sh下载失败"
		exit 1
	fi
	if [[ ! -s "${TEMP_FILE}" ]] || ! grep -q "dmp-lite" "${TEMP_FILE}"; then
		rm -f "${TEMP_FILE}"
		echo_red "run.sh内容校验失败，请稍后重试"
		exit 1
	fi

	# 保留当前脚本中的自定义配置
	sed -i "s/^PORT=.*/PORT=${PORT}/" "$TEMP_FILE"
	sed -i "s/^SWAPSIZE=.*/SWAPSIZE=${SWAPSIZE}/" "$TEMP_FILE"
	chmod +x "$TEMP_FILE"

	# 替换当前脚本
	mv -f "$TEMP_FILE" "$SCRIPT_FILE" && chmod +x "$SCRIPT_FILE"
	echo_green "脚本更新完成，3 秒后重新启动..."
	sleep 3
	exec "$SCRIPT_FILE" "${ORIGINAL_ARGS[@]}"
}

# 设置虚拟内存
function set_swap() {
	SWAPFILE=/swapfile

	# 检查是否已经存在交换文件
	if [ -f $SWAPFILE ]; then
		echo_green "交换文件已存在，跳过创建步骤"
	else
		echo_cyan "创建交换文件..."
		${SUDO} fallocate -l $SWAPSIZE $SWAPFILE
		${SUDO} chmod 600 $SWAPFILE
		${SUDO} mkswap $SWAPFILE
		${SUDO} swapon $SWAPFILE
		echo_green "交换文件创建并启用成功"
	fi

	# 添加到 /etc/fstab 以便开机启动
	if ! grep -q "$SWAPFILE" /etc/fstab; then
		echo_cyan "将交换文件添加到 /etc/fstab "
		echo "$SWAPFILE none swap sw 0 0" | ${SUDO} tee -a /etc/fstab >/dev/null
		echo_green "交换文件已添加到开机启动"
	else
		echo_green "交换文件已在 /etc/fstab 中，跳过添加步骤"
	fi

	# 更改swap配置并持久化
	${SUDO} sysctl -w vm.swappiness=20
	${SUDO} sysctl -w vm.min_free_kbytes=100000
	echo -e 'vm.swappiness = 20\nvm.min_free_kbytes = 100000\n' | ${SUDO} tee /etc/sysctl.d/dmp_swap.conf >/dev/null

	echo_green "系统虚拟内存设置成功"
}

# 设置开机自启
function auto_start_dmp() {
	local work_dir_q
	work_dir_q=$(printf '%q' "$WORK_DIR")
	CRON_JOB="@reboot /bin/bash -c 'source /etc/profile && cd ${work_dir_q} && echo 1 | ${RUN_SH_CMD}'"

	# 检查 crontab 中是否已存在该命令
	if crontab -l 2>/dev/null | grep -Fq "$CRON_JOB"; then
		echo_yellow "已发现开机自启配置，请勿重复添加"
	else
		# 如果不存在，则添加到 crontab
		(
			crontab -l 2>/dev/null
			echo "$CRON_JOB"
		) | crontab -
		echo_green "已成功设置开机自启"
	fi
}

# 获取启动失败日志
function get_panic_logs() {
	if [ -f "$LOG_FILE" ]; then
		tail -n 50 "$LOG_FILE"
	fi
}

# 使用无限循环让用户输入命令
while true; do
	# 提示用户输入
	prompt_user
	# 读取用户输入，stdin 关闭时直接退出，避免非交互场景下死循环
	if ! read -r command; then
		exit 0
	fi
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
		start_dmp
		check_dmp
		echo_green "重启成功"
		unset_tty
		break
		;;
	4)
		set_tty
		update_dmp
		unset_tty
		break
		;;
	5)
		set_tty
		update_script
		unset_tty
		break
		;;
	6)
		set_tty
		set_swap
		unset_tty
		break
		;;
	7)
		set_tty
		auto_start_dmp
		unset_tty
		break
		;;
	8)
		exit 0
		;;
	*)
		echo_red "请输入正确的数字 [0-8]"
		continue
		;;
	esac
done
