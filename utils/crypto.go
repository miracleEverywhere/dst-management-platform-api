package utils

import (
	"crypto/hmac"
	cRand "crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
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

func GenerateBcryptPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return string(hashed), err
}

func ValidatePassword(formPassword, dbPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(formPassword))
	if err != nil {
		return false
	}

	return true
}

const (
	nonceSize   = 1 // 1字节Nonce（8位）
	sigSize     = 4 // 4字节签名（32位）
	maxSigChars = 7 // Base32编码后最多7字符
)

func GenerateUpdateModID() string {
	key := []byte("x")
	data := []byte("y")

	// 1. 生成随机Nonce（使用crypto/rand）
	nonce := make([]byte, nonceSize)
	if _, err := cRand.Read(nonce); err != nil {
		return ""
	}

	// 2. 计算 HMAC-SHA256(Nonce || data)
	h := hmac.New(sha256.New, key)
	h.Write(nonce)
	h.Write(data)
	sig := h.Sum(nil)[:sigSize] // 取前4字节

	// 3. Base32编码并截断
	combined := append(nonce, sig...)
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(combined)
	if len(encoded) > maxSigChars {
		encoded = encoded[:maxSigChars]
	}
	return encoded
}

func VerifyUpdateModID(signature string) bool {
	key := []byte("x")
	data := []byte("y")
	// 1. 长度检查
	if len(signature) != maxSigChars {
		return false
	}

	// 2. Base32解码
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(signature)
	if err != nil {
		return false
	}

	// 3. 数据完整性检查
	if len(decoded) < nonceSize+sigSize/2 { // 至少需要Nonce+部分签名
		return false
	}

	// 4. 重新计算HMAC
	h := hmac.New(sha256.New, key)
	h.Write(decoded[:nonceSize])
	h.Write(data)
	expectedSig := h.Sum(nil)[:min(sigSize, len(decoded)-nonceSize)]

	// 5. 安全比对
	return hmac.Equal(expectedSig, decoded[nonceSize:])
}
