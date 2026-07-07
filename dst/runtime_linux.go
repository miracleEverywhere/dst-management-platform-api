//go:build !windows

package dst

import (
	"dst-management-platform-api/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

func KillRuntimeName(name string) error {
	return utils.BashCMD(fmt.Sprintf("screen -X -S %s quit", name))
}

func (g *Game) cleanupRuntime() {
	_ = utils.BashCMD("screen -wipe")
}

func (g *Game) cleanupRuntimeName(name string) error {
	return KillRuntimeName(name)
}

func (g *Game) prepareRuntimeFiles() {
	if !utils.CompareFileSHA256("dst/bin/lib32/steamclient.so", "steamcmd/linux32/steamclient.so") {
		replaceDSTSOFile()
	}
}

func (g *Game) startWorldProcess(world *worldSaveData) error {
	return utils.BashCMD(world.startCmd)
}

func (g *Game) stopWorldProcess(world *worldSaveData) error {
	err := g.sendConsoleCommand(world, "c_shutdown()")
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	return KillRuntimeName(world.screenName)
}

func (g *Game) isWorldRunning(world *worldSaveData) bool {
	cmd := fmt.Sprintf("ps -ef | grep %s | grep -v grep", world.screenName)
	return utils.BashCMD(cmd) == nil
}

func (g *Game) sendConsoleCommand(world *worldSaveData, cmd string) error {
	return utils.ScreenCMD(cmd, world.screenName)
}

func (g *Game) runningWorldNames() ([]string, error) {
	cmd := fmt.Sprintf("ps -ef | grep 'DMP_Cluster_%d_' | grep dontstarve_dedicated_server_nullrenderer | grep -v grep | awk '{print $14}'", g.room.ID)
	out, _, _ := utils.BashCMDOutput(cmd)
	screenNamesStr := strings.TrimSpace(out)
	if screenNamesStr == "" {
		return []string{}, nil
	}
	return strings.Split(screenNamesStr, "\n"), nil
}

func (g *Game) worldProcess(world *worldSaveData) (*process.Process, error) {
	cmd := fmt.Sprintf("ps -ef | grep dontstarve_dedicated_server_nullrenderer | grep Cluster_%d | grep %s | grep -v luajit | grep -vi screen | awk '{print $2}'", g.room.ID, world.WorldName)
	out, _, _ := utils.BashCMDOutput(cmd)
	if len(out) < 2 {
		return nil, fmt.Errorf("获取世界PID失败, 世界id: %d", world.ID)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return nil, err
	}
	return process.NewProcess(int32(pid))
}
