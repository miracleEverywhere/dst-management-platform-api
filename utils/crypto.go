package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// CompareFileSHA256 比较两个文件的SHA256哈希值
func CompareFileSHA256(file1, file2 string) bool {
	// 比较文件大小
	info1, err := os.Stat(file1)
	if err != nil {
		return false
	}

	info2, err := os.Stat(file2)
	if err != nil {
		return false
	}

	if info1.Size() != info2.Size() {
		return false
	}

	// 计算文件的哈希值
	hash1, err := calculateSHA256(file1)
	if err != nil {
		return false
	}

	hash2, err := calculateSHA256(file2)
	if err != nil {
		return false
	}

	return hash1 == hash2
}

// 计算文件的SHA256哈希值
func calculateSHA256(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
