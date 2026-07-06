package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

var WorkDir string

func InitWorkDir(workDir string) error {
	var dir string
	if workDir != "" {
		abs, err := filepath.Abs(workDir)
		if err != nil {
			return err
		}
		dir = abs
	} else if runtime.GOOS == "windows" {
		exe, err := os.Executable()
		if err == nil {
			dir = filepath.Dir(exe)
		}
	}

	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir = cwd
	}

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}

	WorkDir = dir
	configureRuntimePaths()
	return nil
}

func Path(elem ...string) string {
	parts := append([]string{WorkDir}, elem...)
	return filepath.Join(parts...)
}

func configureRuntimePaths() {
	GameModSettingPath = filepath.Join("dst", "mods", "dedicated_server_mods_setup.lua")
	DSTLocalVersionPath = filepath.Join("dst", "version.txt")

	if runtime.GOOS == "windows" {
		ClusterPath = filepath.Join("klei", "DoNotStarveTogether")
		return
	}

	ClusterPath = filepath.Join(".klei", "DoNotStarveTogether")
}

func DSTInstallDir() string {
	if runtime.GOOS == "windows" {
		return Path("dst")
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(WorkDir, "dst")
	}
	return filepath.Join(home, "dst")
}

func SteamCMDExecutable() string {
	if runtime.GOOS == "windows" {
		return Path("steamcmd", "steamcmd.exe")
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(WorkDir, "steamcmd", "steamcmd.sh")
	}
	return filepath.Join(home, "steamcmd", "steamcmd.sh")
}
