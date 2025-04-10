package utils

type User struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
	Password string `json:"password"`
	Disabled bool   `json:"disabled"`
}

type RoomSettingCluster struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	GameMode    string `json:"gameMode"`
	PVP         bool   `json:"pvp"`
	PlayerNum   int    `json:"playerNum"`
	BackDays    int    `json:"backDays"`
	Vote        bool   `json:"vote"`
	Password    string `json:"password"`
	Token       string `json:"token"`
}

type RoomSettingWorld struct {
	ID                      int    `json:"id"`
	Name                    string `json:"name"`
	ServerPort              int    `json:"serverPort"`
	ClusterKey              string `json:"clusterKey"`
	ShardMasterIp           string `json:"shardMasterIp"`
	ShardMasterPort         int    `json:"shardMasterPort"`
	SteamMasterPort         int    `json:"steamMasterPort"`
	SteamAuthenticationPort int    `json:"steamAuthenticationPort"`
	EncodeUserPath          bool   `json:"encodeUserPath"`
}

type RoomSetting struct {
	Cluster RoomSettingCluster `json:"cluster"`
	Worlds  []RoomSettingWorld `json:"worlds"`
	Mod     string             `json:"mod"`
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
	Enable        bool   `json:"enable"`
	Frequency     int    `json:"frequency"`
	LastTime      string `json:"lastTime"`
	CavesLastTime string `json:"cavesLastTime"`
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
}

type SysSetting struct {
	SchedulerSetting SchedulerSetting `json:"schedulerSetting"`
	AutoUpdate       AutoUpdate       `json:"autoUpdate"`
	AutoAnnounce     []AutoAnnounce   `json:"autoAnnounce"`
	AutoBackup       AutoBackup       `json:"autoBackup"`
	Keepalive        Keepalive        `json:"keepalive"`
	Bit64            bool             `json:"bit64"`
	TickRate         int              `json:"tickRate"`
}

type Config struct {
	Users       []User      `json:"users"`
	JwtSecret   string      `json:"jwtSecret"`
	RoomSetting RoomSetting `json:"roomSetting"`
	SysSetting  SysSetting  `json:"sysSetting"`
	Platform    string      `json:"platform"`
	AnnouncedID int         `json:"announcedID"`
	Registered  bool        `json:"registered"`
}
