package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

type User struct {
	Username                  string   `json:"username"`
	Nickname                  string   `json:"nickname"`
	Role                      string   `json:"role"`
	Password                  string   `json:"password"`
	Disabled                  bool     `json:"disabled"`
	ClusterPermission         []string `json:"clusterPermission"`
	AnnounceID                int      `json:"announceID"`
	ClusterCreationProhibited bool     `json:"clusterCreationProhibited"`
}

type ClusterSetting struct {
	ClusterName        string `json:"clusterName"` // MyDediServer
	ClusterDisplayName string `json:"clusterDisplayName"`
	Name               string `json:"name"` // xxx长期档
	Description        string `json:"description"`
	GameMode           string `json:"gameMode"`
	PVP                bool   `json:"pvp"`
	PlayerNum          int    `json:"playerNum"`
	BackDays           int    `json:"backDays"`
	Vote               bool   `json:"vote"`
	Password           string `json:"password"`
	Token              string `json:"token"`
}

type World struct {
	ID                      int    `json:"id"`
	Name                    string `json:"name"`       // Master
	ScreenName              string `json:"screenName"` // DST_Master
	LevelData               string `json:"levelData"`
	IsMaster                bool   `json:"isMaster"`
	ServerPort              int    `json:"serverPort"`
	ClusterKey              string `json:"clusterKey"`
	ShardMasterIp           string `json:"shardMasterIp"`
	ShardMasterPort         int    `json:"shardMasterPort"`
	SteamMasterPort         int    `json:"steamMasterPort"`
	SteamAuthenticationPort int    `json:"steamAuthenticationPort"`
	EncodeUserPath          bool   `json:"encodeUserPath"`
	LastAliveTime           string `json:"lastAliveTime"`
}

type Cluster struct {
	ClusterSetting ClusterSetting `json:"clusterSetting"`
	Worlds         []World        `json:"worlds"`
	Mod            string         `json:"mod"`
	SysSetting     SysSetting     `json:"sysSetting"`
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

type AutoRestart struct {
	Enable bool   `json:"enable"`
	Time   string `json:"time"`
}

type ScheduledStartStop struct {
	Enable    bool   `json:"enable"`
	StartTime string `json:"startTime"`
	StopTime  string `json:"stopTime"`
}

type Players struct {
	UID      string `json:"uid"`
	NickName string `json:"nickName"`
	Prefab   string `json:"prefab"`
}

type Statistics struct {
	Timestamp int64     `json:"timestamp"`
	Num       int       `json:"num"`
	Players   []Players `json:"players"`
}

type SysMetrics struct {
	Timestamp   int64   `json:"timestamp"`
	Cpu         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	NetUplink   float64 `json:"netUplink"`
	NetDownlink float64 `json:"netDownlink"`
}

type Keepalive struct {
	Enable    bool `json:"enable"`
	Frequency int  `json:"frequency"`
}

type SchedulerSettingItem struct {
	// disable的原因是1.1.3版本之前都是默认打开的，新增配置后应该也是默认打开
	// 所以 disable=false
	Disable   bool `json:"disable"`
	Frequency int  `json:"frequency"`
}

type SchedulerSetting struct {
	PlayerGetFrequency int                  `json:"playerGetFrequency"`
	UIDMaintain        SchedulerSettingItem `json:"UIDMaintain"`
	SysMetricsGet      SchedulerSettingItem `json:"sysMetricsGet"`
	AutoUpdate         AutoUpdate           `json:"autoUpdate"`
	PlayerUpdateMod    SchedulerSettingItem `json:"playerUpdateMod"`
}

type SysSetting struct {
	AutoRestart        AutoRestart        `json:"autoRestart"`
	AutoAnnounce       []AutoAnnounce     `json:"autoAnnounce"`
	AutoBackup         AutoBackup         `json:"autoBackup"`
	Keepalive          Keepalive          `json:"keepalive"`
	ScheduledStartStop ScheduledStartStop `json:"scheduledStartStop"`
	Bit64              bool               `json:"bit64"`
	TickRate           int                `json:"tickRate"`
}

type Config struct {
	Users            []User           `json:"users"`
	JwtSecret        string           `json:"jwtSecret"`
	Clusters         []Cluster        `json:"clusters"`
	SchedulerSetting SchedulerSetting `json:"schedulerSetting"`
	Registered       bool             `json:"registered"`
}

func (config Config) Init() {
	config.JwtSecret = GenerateJWTSecret()
	config.SchedulerSetting = SchedulerSetting{
		PlayerGetFrequency: 30,
		UIDMaintain: SchedulerSettingItem{
			Disable:   false,
			Frequency: 5,
		},
		SysMetricsGet: SchedulerSettingItem{
			Disable:   false,
			Frequency: 0,
		},
		AutoUpdate: AutoUpdate{
			Time:   "06:19:23",
			Enable: true,
		},
		PlayerUpdateMod: SchedulerSettingItem{
			Disable:   false,
			Frequency: 10,
		},
	}
	config.Registered = false
	err := WriteConfig(config)
	if err != nil {
		Logger.Error("写入数据库失败", "err", err)
		panic("数据库初始化失败")
	}
}

func ReadConfig() (Config, error) {
	content, err := os.ReadFile(ConfDir + "/DstMP.sdb")
	if err != nil {
		return Config{}, err
	}

	jsonData := string(content)
	var config Config
	err = json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		return Config{}, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	return config, nil
}

func ReadBackupConfig(configPath string) (Config, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	jsonData := string(content)
	var config Config
	err = json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		return Config{}, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	return config, nil
}

func WriteConfig(config Config) error {
	data, err := json.MarshalIndent(config, "", "    ") // 格式化输出
	if err != nil {
		return fmt.Errorf("Error marshalling JSON:" + err.Error())
	}
	file, err := os.OpenFile(ConfDir+"/DstMP.sdb", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("Error opening file:" + err.Error())
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭文件失败", "err", err)
		}
	}(file) // 在函数结束时关闭文件
	// 写入 JSON 数据到文件
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("Error writing to file:" + err.Error())
	}
	return nil
}

func CheckConfig() {
	_ = EnsureDirExists(ConfDir)
	_, err := os.Stat(ConfDir + "/DstMP.sdb")
	if !os.IsNotExist(err) {
		Logger.Info("执行数据库检查中，发现数据库文件")
		_, err := ReadConfig()
		if err != nil {
			Logger.Error("执行数据库检查中，打开数据库文件失败", "err", err)
			panic("数据库检查未通过")
			return
		}
		Logger.Info("数据库检查完成")
		return
	}

	Logger.Info("执行数据库检查中，初始化数据库")
	var config Config
	config.Init()

	Logger.Info("数据库初始化完成")
}
