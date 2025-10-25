package utils

import (
	"dst-management-platform-api/models"
	"github.com/dgrijalva/jwt-go"
	"math/rand"
	"time"
)

func GenerateJWTSecret() string {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 26
	randomString := make([]byte, length)
	for i := range randomString {
		// 从字符集中随机选择一个字符
		randomString[i] = charset[r.Intn(len(charset))]
	}

	return string(randomString)
}

func GenerateJWT(user models.User, jwtSecret []byte, expiration int) (string, error) {
	type Claims struct {
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
		jwt.StandardClaims
	}

	claims := Claims{
		Username: user.Username,
		Nickname: user.Nickname,
		Role:     user.Role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expiration) * time.Hour).Unix(), // 过期时间
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
