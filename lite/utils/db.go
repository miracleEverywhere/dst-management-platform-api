package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const (
	dbPath string = "./dmp-lite.sdb"
)

var (
	DB   db
	DBMu sync.RWMutex
)

type db struct {
	User   User   `json:"user"`
	Room   Room   `json:"room"`
	Ground World  `json:"ground"`
	Cave   World  `json:"cave"`
	Mod    string `json:"mod"`
	System System `json:"system"`
}

type System struct {
	JwtSecret string `json:"jwtSecret"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Room struct {
	GameName    string `json:"gameName"`
	Description string `json:"description"`
	GameMode    string `json:"gameMode"`
	Pvp         bool   `json:"pvp"`
	MaxPlayer   int    `json:"maxPlayer"`
	MaxRollBack int    `json:"maxRollBack"`
	Vote        bool   `json:"vote"`
	PauseEmpty  bool   `json:"pauseEmpty"`
	Password    string `json:"password"`
	Token       string `json:"token"`
	MasterIP    string `json:"masterIP"`
	MasterPort  int    `json:"masterPort"`
	ClusterKey  string `json:"clusterKey"`
	Whitelist   string `json:"whitelist"`
	Adminlist   string `json:"adminlist"`
	Blocklist   string `json:"blocklist"`
	StartType   string `json:"startType"`
}

type World struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	ServerPort         int    `json:"serverPort"`
	MasterServerPort   int    `json:"masterServerPort"`
	AuthenticationPort int    `json:"authenticationPort"`
	IsMaster           bool   `json:"isMaster"`
	EncodeUserPath     bool   `json:"encodeUserPath"`
	LevelData          string `json:"levelData"`
	Status             bool   `json:"status"`
}

// SaveLocked 将 db 序列化为 JSON、Base64 编码后原子写入 dbPath。
// 调用方必须已持有 DBMu 写锁。
func (d *db) SaveLocked() error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return writeFileAtomic(dbPath, []byte(encoded))
}

// Read 加写锁后从 dbPath 还原 db，文件不存在时创建空库
func (d *db) Read() error {
	DBMu.Lock()
	defer DBMu.Unlock()

	return d.readLocked()
}

// readLocked 从 dbPath 读取 Base64 内容，解码后按 JSON 整体替换 db。
// 文件不存在视为首次启动，初始化为空字段配置并立即创建数据库文件。
// 调用方必须已持有 DBMu 写锁。
func (d *db) readLocked() error {
	encoded, err := os.ReadFile(dbPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// 首次启动：重置为空字段配置，并落盘创建数据库文件
			secret, genErr := GenerateJWTSecret()
			if genErr != nil {
				return genErr
			}
			*d = db{
				System: System{
					JwtSecret: secret,
				},
				Room: Room{
					GameName:    "",
					Description: "",
					GameMode:    "endless",
					Pvp:         false,
					MaxPlayer:   6,
					MaxRollBack: 10,
					Vote:        false,
					PauseEmpty:  true,
					Password:    "",
					Token:       "",
					MasterIP:    "127.0.0.1",
					MasterPort:  21001,
					ClusterKey:  "",
					StartType:   "32-bit",
				},
				Ground: World{
					ID:                 101,
					Name:               "Master",
					ServerPort:         11001,
					MasterServerPort:   31001,
					AuthenticationPort: 41001,
					IsMaster:           true,
					EncodeUserPath:     true,
					LevelData:          "",
				},
				Cave: World{
					ID:                 102,
					Name:               "Caves",
					ServerPort:         11002,
					MasterServerPort:   31002,
					AuthenticationPort: 41002,
					IsMaster:           false,
					EncodeUserPath:     true,
					LevelData:          "",
				},
			}
			return d.SaveLocked()
		}
		return err
	}

	data, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(encoded)))
	if err != nil {
		return err
	}

	// 先解析到临时值，避免文件内容不完整时把现有配置改坏
	var loaded db
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	*d = loaded
	return nil
}

// writeFileAtomic 先写同目录临时文件再改名，避免进程中断留下半截数据库文件
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".dmp-lite-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	// 任何一步失败都清理临时文件，避免残留
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	return nil
}
