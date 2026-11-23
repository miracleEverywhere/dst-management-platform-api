package models

// BannedMod 全局模组屏蔽名单条目。
// 对应存档根目录（与集群目录同级）下的 banned_mods.txt，对该机器上的所有集群生效。
// 数据库是本名单的唯一真相源，banned_mods.txt 只是投影给游戏引擎读取的产物。
type BannedMod struct {
	ID uint `gorm:"column:id;primaryKey" json:"id"`
	// ModID 创意工坊模组 ID，对应 banned_mods.txt 里的一行纯数字
	ModID int `gorm:"column:mod_id;not null;uniqueIndex:idx_banned_mods_mod_id" json:"modID"`
}

func (BannedMod) TableName() string {
	return "banned_mods"
}
