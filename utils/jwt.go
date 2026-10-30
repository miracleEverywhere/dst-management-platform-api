package utils

import (
	"crypto/rand"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	Role         string `json:"role"`
	TokenVersion int    `json:"tokenVersion"`
	jwt.RegisteredClaims
}

// GenerateJWTSecret 生成JWT密钥。
// 使用 crypto/rand 生成随机字节，避免 math/rand 的伪随机序列被预测。
func GenerateJWTSecret() (string, error) {
	buf := make([]byte, constant.JwtSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// GenerateJWT 生成 JWT Token
func GenerateJWT(user models.User, jwtSecret []byte, expirationHours int) (string, error) {
	claims := Claims{
		Username:     user.Username,
		Nickname:     user.Nickname,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "https://github.com/miracleEverywhere/dst-management-platform-api",
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT 验证 JWT Token
func ValidateJWT(tokenString string, jwtSecret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
