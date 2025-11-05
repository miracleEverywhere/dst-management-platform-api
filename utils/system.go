package utils

import (
	"fmt"
	"os"
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

// EnsureFileExists 检查文件是否存在，如果不存在则创建空文件
func EnsureFileExists(filePath string) error {
	// 检查文件是否存在
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// 文件不存在，创建一个空文件
		file, errCreate := os.Create(filePath)
		if errCreate != nil {
			return errCreate
		}
		errClose := file.Close()
		if errClose != nil {
			return errClose
		}
	} else if err != nil {
		// 其他错误
		return err
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
