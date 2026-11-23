package dst

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

// bannedModsMutex 保护全局屏蔽名单文件的写入。
// banned_mods.txt 位于存档根目录、被本机所有集群共享，不能用 Game 上的实例锁。
var bannedModsMutex sync.Mutex

// BannedModsPath 全局模组屏蔽名单文件路径，与集群目录同级
func BannedModsPath() string {
	return filepath.Join(utils.ClusterPath, constant.BannedModsFileName)
}

// SaveBannedMods 用给定 ID 覆盖写入 banned_mods.txt，每个 ID 一行。
// 名单的权威来源是数据库，本函数只负责把名单投影成游戏引擎读取的文件。
func SaveBannedMods(modIDs []int) error {
	bannedModsMutex.Lock()
	defer bannedModsMutex.Unlock()

	// WriteLinesFromSlice 内部是 os.Create，不会自动创建缺失的上级目录
	if err := os.MkdirAll(filepath.Dir(BannedModsPath()), os.ModePerm); err != nil {
		return fmt.Errorf("创建存档根目录失败: %w", err)
	}

	ids := NormalizeBannedModIDs(modIDs)
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		lines = append(lines, strconv.Itoa(id))
	}

	if err := utils.WriteLinesFromSlice(BannedModsPath(), lines); err != nil {
		return fmt.Errorf("写入屏蔽名单文件失败: %w", err)
	}

	return nil
}

// NormalizeBannedModIDs 过滤非法 ID，去重并按升序返回。
// 数据库写入与文件写入都复用它，保证两侧内容与顺序一致。
func NormalizeBannedModIDs(modIDs []int) []int {
	ids := make([]int, 0, len(modIDs))
	seen := make(map[int]struct{}, len(modIDs))
	for _, id := range modIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Ints(ids)

	return ids
}
