#!/bin/zsh

WORK_DIR=$(pwd)
ExeFile="${WORK_DIR}/dmp"

pkill -9 dmp

exec > logs/runtime.log

nohup "${ExeFile}" -bind 80 -dbpath data -level debug >/dev/null 2>&1 &
