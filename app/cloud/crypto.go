package cloud

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"dst-management-platform-api/constant"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	cloudKeyOnce sync.Once
	cloudKey     []byte
	cloudKeyErr  error
)

// loadOrCreateKey 读取 dmp_files/cloud.key，不存在时生成 32 字节随机密钥并写入 0600 文件。
func loadOrCreateKey() ([]byte, error) {
	cloudKeyOnce.Do(func() {
		cloudKey, cloudKeyErr = doLoadOrCreateKey(constant.CloudKeyPath)
	})
	if cloudKeyErr != nil {
		return nil, cloudKeyErr
	}

	return cloudKey, nil
}

func doLoadOrCreateKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("云凭据密钥长度不正确: %s", path)
		}
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		// 并发创建时另一个请求可能已经写入，重新读取即可。
		if errors.Is(err, os.ErrExist) {
			return doLoadOrCreateKey(path)
		}
		return nil, err
	}
	defer file.Close()

	if _, err = file.Write(key); err != nil {
		_ = os.Remove(path)
		return nil, err
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
