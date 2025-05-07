package utils

const ShutdownScreenCMD = "c_shutdown()"

/* ---- 重构 ---- */

const UpdateGameCMD = "cd ~/steamcmd ; ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit"

/* 以下是MacOS常量 Linux交叉编译：CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /root/dmp_darwin */

const MacDSTVersionCMD = "cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && strings dontstarve_dedicated_server_nullrenderer | grep -A 1 PRODUCTION | grep -E '\\d+'"
