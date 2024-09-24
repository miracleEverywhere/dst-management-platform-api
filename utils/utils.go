package utils

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"os"
	"time"
)

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type AutoUpdate struct {
	Enable bool   `json:"enable"`
	Time   string `json:"time"`
}

type AutoAnnounce struct {
	Name      string `json:"name"`
	Enable    bool   `json:"enable"`
	Content   string `json:"content"`
	Frequency int    `json:"frequency"`
}

type AutoBackup struct {
	Enable bool   `json:"enable"`
	Time   string `json:"time"`
}

type Config struct {
	Username     string         `json:"username"`
	Password     string         `json:"password"`
	JwtSecret    string         `json:"jwtSecret"`
	AutoUpdate   AutoUpdate     `json:"autoUpdate"`
	AutoAnnounce []AutoAnnounce `json:"autoAnnounce"`
	AutoBackup   AutoBackup     `json:"autoBackup"`
}

func GenerateJWT(username string, jwtSecret []byte, expiration int) (string, error) {
	// 定义一个自定义的声明结构

	claims := Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expiration) * time.Hour).Unix(), // 过期时间
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateJWT(tokenString string, jwtSecret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func Sha512(input string) string {
	// 创建一个新的 SHA-512 哈希对象
	hasher := sha512.New()

	// 写入需要加密的数据
	hasher.Write([]byte(input))

	// 计算哈希值并返回十六进制字符串
	hashed := hasher.Sum(nil)
	return hex.EncodeToString(hashed)
}

func Base64Encode(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

func Base64Decode(input string) string {
	decodedData, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		fmt.Println("解码失败:", err)
		return ""
	}
	return string(decodedData)
}

func ReadConfig() (Config, error) {
	content, _ := os.ReadFile("DstMP.sdb")
	jsonData := Base64Decode(string(content))

	var config Config
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		return Config{}, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	return config, nil
}
