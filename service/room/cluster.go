package room

import (
	"bufio"
	"dst-management-platform-api/logger"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 查找包含 cluster.ini 文件的目录
func findClusterDir(path string) (string, error) {
	// 检查当前目录是否包含 cluster.ini
	clusterFile := filepath.Join(path, "cluster.ini")
	if _, err := os.Stat(clusterFile); err == nil {
		return path, nil
	}

	// 读取当前目录
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %v", err)
	}

	// 遍历子目录
	for _, entry := range entries {
		logger.Logger.Debug(entry.Name())
		if entry.IsDir() {
			subPath := filepath.Join(path, entry.Name())
			// 递归查找
			if result, err := findClusterDir(subPath); err == nil {
				return result, nil
			}
		}
	}

	return "", fmt.Errorf("未找到包含 cluster.ini 的目录")
}

func isSafeWorldDir(s string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9/]+$`, s)

	return matched
}

// 将ini文件读取为map
func parseIniToMap(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	configMap := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// 检查是否是节标题
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}

		// 解析键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			configMap[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return configMap, nil
}
