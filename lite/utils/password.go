package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt 生成密码哈希
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

// ValidatePassword 校验明文密码是否与存储的 bcrypt 哈希匹配
func ValidatePassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
