// Package cloud 云账号、云收藏同步与云存档的业务实现，不依赖 gin。
package cloud

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"encoding/base64"
	"errors"
)

// ===== 凭据加密 =====

// credentialKey 返回启动时由 SystemDAO 从 system 表载入的云凭据主密钥。
// 密钥不再落盘为独立文件，跟随数据库一起备份与迁移。
func credentialKey() ([]byte, error) {
	key := cache.CloudKey
	if len(key) != constant.CloudKeyBytes {
		return nil, errors.New("云凭据密钥未初始化")
	}

	return key, nil
}

// encryptSecret 使用 AES-256-GCM 加密云凭据，AAD 绑定本地用户名，防止把 A 用户的 token 复制给 B。
func encryptSecret(key []byte, username, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), []byte(username))

	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func decryptSecret(key []byte, username, encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("云凭据密文长度不正确")
	}

	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], []byte(username))
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
