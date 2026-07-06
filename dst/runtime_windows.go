//go:build windows

package dst

import (
	"dst-management-platform-api/utils"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type windowsWorldProcess struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

var windowsProcesses sync.Map

func KillRuntimeName(name string) error {
	if value, ok := windowsProcesses.Load(name); ok {
		proc := value.(*windowsWorldProcess)
		if proc.stdin != nil {
			_ = proc.stdin.Close()
		}
		if proc.cmd != nil && proc.cmd.Process != nil {
			_ = proc.cmd.Process.Kill()
		}
		windowsProcesses.Delete(name)
		return nil
	}

	clusterName, worldName, hasRuntimeParts := parseRuntimeName(name)
	processes, err := process.Processes()
	if err != nil {
		return err
	}
	for _, p := range processes {
		cmdline, err := p.Cmdline()
		if err != nil {
			continue
		}
		if strings.Contains(cmdline, name) || (hasRuntimeParts && strings.Contains(cmdline, clusterName) && strings.Contains(cmdline, worldName)) {
			_ = p.Kill()
		}
	}
	return nil
}

func parseRuntimeName(name string) (string, string, bool) {
	const prefix = "DMP_Cluster_"
	if !strings.HasPrefix(name, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(name, prefix)
	idx := strings.Index(rest, "_")
	if idx <= 0 || idx == len(rest)-1 {
		return "", "", false
	}
	return "Cluster_" + rest[:idx], rest[idx+1:], true
}

func (g *Game) cleanupRuntime() {}

func (g *Game) cleanupRuntimeName(name string) error {
	return KillRuntimeName(name)
}

func (g *Game) prepareRuntimeFiles() {}

func (g *Game) startWorldProcess(world *worldSaveData) error {
	exePath, workDir, err := windowsServerExecutable(g.setting.StartType)
	if err != nil {
		return err
	}

	args := []string{
		"-console",
		"-persistent_storage_root", utils.Path("klei"),
		"-conf_dir", "DoNotStarveTogether",
		"-cluster", g.clusterName,
		"-shard", world.WorldName,
	}

	cmd := exec.Command(exePath, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "DMP_HOME="+utils.WorkDir)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}

	windowsProcesses.Store(world.screenName, &windowsWorldProcess{
		cmd:   cmd,
		stdin: stdin,
	})

	go func() {
		_ = cmd.Wait()
		windowsProcesses.Delete(world.screenName)
	}()

	return nil
}

func windowsServerExecutable(startType string) (string, string, error) {
	if startType == "32-bit" {
		exePath := utils.Path("dst", "bin", "dontstarve_dedicated_server_nullrenderer.exe")
		if _, err := os.Stat(exePath); err != nil {
			return "", "", err
		}
		return exePath, filepath.Dir(exePath), nil
	}

	exePath := utils.Path("dst", "bin64", "dontstarve_dedicated_server_nullrenderer_x64.exe")
	if _, err := os.Stat(exePath); err != nil {
		return "", "", err
	}
	return exePath, filepath.Dir(exePath), nil
}

func (g *Game) stopWorldProcess(world *worldSaveData) error {
	if value, ok := windowsProcesses.Load(world.screenName); ok {
		proc := value.(*windowsWorldProcess)
		if proc.stdin != nil {
			_, _ = proc.stdin.Write([]byte("c_shutdown()\r\n"))
		}
	}

	time.Sleep(2 * time.Second)

	var retErr error
	if value, ok := windowsProcesses.Load(world.screenName); ok {
		proc := value.(*windowsWorldProcess)
		if proc.stdin != nil {
			_ = proc.stdin.Close()
		}
		if proc.cmd != nil && proc.cmd.Process != nil {
			retErr = proc.cmd.Process.Kill()
		}
		windowsProcesses.Delete(world.screenName)
	}

	processes, err := g.findWorldProcesses(world)
	if err != nil {
		return retErr
	}
	for _, p := range processes {
		if err := p.Kill(); err != nil && retErr == nil {
			retErr = err
		}
	}

	return retErr
}

func (g *Game) isWorldRunning(world *worldSaveData) bool {
	if value, ok := windowsProcesses.Load(world.screenName); ok {
		proc := value.(*windowsWorldProcess)
		if proc.cmd != nil && proc.cmd.Process != nil {
			return true
		}
	}

	processes, err := g.findWorldProcesses(world)
	return err == nil && len(processes) > 0
}

func (g *Game) sendConsoleCommand(world *worldSaveData, command string) error {
	value, ok := windowsProcesses.Load(world.screenName)
	if !ok {
		return fmt.Errorf("世界进程不是当前DMP实例启动，无法写入控制台: %s", world.WorldName)
	}

	proc := value.(*windowsWorldProcess)
	if proc.stdin == nil {
		return fmt.Errorf("世界控制台不可用: %s", world.WorldName)
	}

	_, err := proc.stdin.Write([]byte(command + "\r\n"))
	return err
}

func (g *Game) runningWorldNames() ([]string, error) {
	var names []string
	for _, world := range g.worldSaveData {
		if g.isWorldRunning(&world) {
			names = append(names, world.screenName)
		}
	}
	return names, nil
}

func (g *Game) worldProcess(world *worldSaveData) (*process.Process, error) {
	processes, err := g.findWorldProcesses(world)
	if err != nil {
		return nil, err
	}
	if len(processes) == 0 {
		return nil, fmt.Errorf("获取世界PID失败, 世界id: %d", world.ID)
	}
	return processes[0], nil
}

func (g *Game) findWorldProcesses(world *worldSaveData) ([]*process.Process, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var matched []*process.Process
	for _, p := range processes {
		name, _ := p.Name()
		if !strings.Contains(strings.ToLower(name), "dontstarve_dedicated_server") {
			continue
		}

		cmdline, err := p.Cmdline()
		if err != nil {
			continue
		}
		if strings.Contains(cmdline, g.clusterName) && strings.Contains(cmdline, world.WorldName) {
			matched = append(matched, p)
		}
	}
	return matched, nil
}
