package api

import (
	"bufio"
	"dst-management-platform-api/lite/utils"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

func sysInfoGet(w http.ResponseWriter, r *http.Request) {
	var data struct {
		DmpMem uint64  `json:"dmpMem"`
		Cpu    float64 `json:"cpu"`
		Memory float64 `json:"memory"`
	}

	data.DmpMem = getRES()
	data.Cpu = cpuUsage()
	data.Memory = memoryUsage()

	writeJSON(w, 200, msgSuccess, data)
}

func gameVersionGet(w http.ResponseWriter, r *http.Request) {
	type DSTVersion struct {
		Local  int `json:"local"`
		Server int `json:"server"`
	}
	var dstVersion DSTVersion
	dstVersion.Server = int(utils.GameVersionServer.Load())
	dstVersion.Local = gameVersionLocal()

	writeJSON(w, 200, msgSuccess, dstVersion)
}

func connectCodeGet(w http.ResponseWriter, r *http.Request) {
	var cc string
	if utils.DB.Room.Password != "" {
		cc = fmt.Sprintf("c_connect('%s', %d, '%s')", utils.GetInternetIP(), utils.DB.Ground.ServerPort, utils.DB.Room.Password)
	} else {
		cc = fmt.Sprintf("c_connect('%s', %d)", utils.GetInternetIP(), utils.DB.Ground.ServerPort)
	}

	writeJSON(w, 200, msgSuccess, cc)
}

func execPost(w http.ResponseWriter, r *http.Request) {
	var form struct {
		CmdType   string `json:"cmdType"`
		WorldName string `json:"worldName"`
	}
	if err := bindJSON(r, &form); err != nil {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	if err := utils.GameExec(form.CmdType, form.WorldName, utils.DB.Room.StartType); err != nil {
		writeJSON(w, 201, "执行失败", nil)
		return
	}

	writeJSON(w, 200, "执行成功", nil)
}

func getRES() uint64 {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0
	}

	memoryInfo, err := p.MemoryInfo()
	if err != nil {
		return 0
	}

	return memoryInfo.RSS
}

func cpuUsage() float64 {
	percent, err := cpu.Percent(0, false)
	if err != nil || len(percent) == 0 {
		return 0
	}
	return percent[0]
}

func memoryUsage() float64 {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}
	return vmStat.UsedPercent
}

func gameVersionLocal() int {
	const versionPath = "dst/version.txt"
	version := 0
	file, err := os.Open(versionPath)
	if err != nil {
		utils.Logger.Error("获取游戏版本失败, err: " + err.Error())
		return version
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)

	if scanner.Scan() {
		// 读取第一行的文本
		line := scanner.Text()
		number, err := strconv.Atoi(line)
		if err != nil {
			utils.Logger.Error("获取游戏版本失败, err: " + err.Error())

			return version
		}
		version = number

		return version
	}

	// 如果扫描器遇到错误，返回错误
	if err = scanner.Err(); err != nil {
		utils.Logger.Error("获取游戏版本失败, err: " + err.Error())

		return version
	}

	return version
}
