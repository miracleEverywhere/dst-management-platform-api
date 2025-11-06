package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var StartTime = time.Now()

// EnsureDirExists 检查目录是否存在，如果不存在则创建
func EnsureDirExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("无法创建目录: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("检查目录时出错: %w", err)
	}

	return nil
}

// TruncAndWriteFile 将指定内容完整写入文件，如果文件已存在会清空原有内容，如果文件不存在会创建新文件
func TruncAndWriteFile(fileName string, fileContent string) error {
	fileContentByte := []byte(fileContent)
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("打开或创建文件时出错: %w", err)
	}
	defer file.Close()

	// 写入新数据
	_, err = file.Write(fileContentByte)
	if err != nil {
		return fmt.Errorf("写入数据时出错: %w", err)
	}

	return nil
}

// BashCMD 执行Linux Bash 命令
func BashCMD(cmd string) error {
	cmdExec := exec.Command("/bin/bash", "-c", cmd)
	err := cmdExec.Run()
	if err != nil {
		return err
	}
	return nil
}

// BashCMDOutput 执行Linux Bash 命令，并返回结果
func BashCMDOutput(cmd string) (string, string, error) {
	// 定义要执行的命令和参数
	cmdExec := exec.Command("/bin/bash", "-c", cmd)

	// 使用 bytes.Buffer 捕获命令的输出
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmdExec.Stdout = &stdout
	cmdExec.Stderr = &stderr

	// 执行命令
	err := cmdExec.Run()
	if err != nil {
		return "", stderr.String(), err
	}

	return stdout.String(), "", nil
}

// ScreenCMD 执行饥荒Console命令
func ScreenCMD(cmd string, screenName string) error {
	totalCMD := "screen -S \"" + screenName + "\" -p 0 -X stuff \"" + cmd + "\\n\""

	cmdExec := exec.Command("/bin/bash", "-c", totalCMD)
	err := cmdExec.Run()
	if err != nil {
		return err
	}
	return nil
}

// ScreenCMDOutput 执行饥荒Console命令，并从日志中获取输出
// 自动添加print命令，cmdIdentifier是该命令在日志中输出的唯一标识符
func ScreenCMDOutput(cmd string, cmdIdentifier string, screenName string, logPath string) (string, error) {
	totalCMD := "screen -S \"" + screenName + "\" -p 0 -X stuff \"print('" + cmdIdentifier + "' .. 'DMPSCREENCMD' .. tostring(" + cmd + "))\\n\""

	cmdExec := exec.Command("/bin/bash", "-c", totalCMD)
	err := cmdExec.Run()
	if err != nil {
		return "", err
	}

	// 等待日志打印
	time.Sleep(50 * time.Millisecond)

	logCmd := "tail -1000 " + logPath
	out, _, err := BashCMDOutput(logCmd)
	if err != nil {
		return "", err
	}

	for _, i := range strings.Split(out, "\n") {
		if strings.Contains(i, cmdIdentifier+"DMPSCREENCMD") {
			result := strings.Split(i, "DMPSCREENCMD")
			return strings.TrimSpace(result[1]), nil
		}
	}

	return "", fmt.Errorf("在日志中未找到对应输出")
}
