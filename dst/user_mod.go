package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 用户模组缓存的读写锁按用户隔离，避免一个用户复制大模组时阻塞其他用户。
var (
	userModCacheLockMap   = make(map[string]*sync.Mutex)
	userModCacheLockMapMu sync.Mutex
)

func lockUserModCache(username string) func() {
	key := utils.SafeUserDirName(username)

	userModCacheLockMapMu.Lock()
	lock, ok := userModCacheLockMap[key]
	if !ok {
		lock = &sync.Mutex{}
		userModCacheLockMap[key] = lock
	}
	userModCacheLockMapMu.Unlock()

	lock.Lock()

	return lock.Unlock
}

// userModMeta 用户模组缓存中 ugc 模组的元数据，来源为下载完成后的 acf 节点，
// 恢复时用于给房间的 appworkshop_322330.acf 补回同一个 WorkshopItemsInstalled 条目。
type userModMeta struct {
	ID      int               `json:"id"`
	SavedAt int64             `json:"savedAt"`
	ACF     map[string]string `json:"acf"`
	// ACFRoot 保存源 acf 的根节点字段（appid、SizeOnDisk 等），目标 acf 不存在时用于构造骨架
	ACFRoot map[string]string `json:"acfRoot,omitempty"`
}

func userModUgcDir(username string) string {
	return fmt.Sprintf("%s/ugc", utils.UserModsDir(username))
}

func userModUgcContentPath(username string, id int) string {
	return fmt.Sprintf("%s/%d", userModUgcDir(username), id)
}

func userModUgcMetaPath(username string, id int) string {
	return fmt.Sprintf("%s/%d.json", userModUgcDir(username), id)
}

func userModNotUgcDir(username string) string {
	return fmt.Sprintf("%s/not_ugc", utils.UserModsDir(username))
}

func userModNotUgcContentPath(username string, id int) string {
	return fmt.Sprintf("%s/workshop-%d", userModNotUgcDir(username), id)
}

// saveModToUserCache 把已下载成功的模组保存一份到用户名下，更新时覆盖旧副本。
func (g *Game) saveModToUserCache(username string, id int, ugc bool) error {
	if username == "" {
		return fmt.Errorf("用户名为空，无法保存模组缓存")
	}
	if id <= 0 {
		return fmt.Errorf("模组ID非法: %d", id)
	}

	unlock := lockUserModCache(username)
	defer unlock()

	var (
		src  string
		dst  string
		meta *userModMeta
	)
	if ugc {
		world := g.resolveUgcWorld(fmt.Sprintf("content/322330/%d", id))
		src = fmt.Sprintf("%s/%s/content/322330/%d", g.ugcPath, world, id)
		dst = userModUgcContentPath(username, id)

		// 与下载/恢复保持一致，读取 acf 时持有同一把锁，避免读到其他下载任务正在写入的文件
		modAcfMutex.Lock()
		meta = g.ugcModMeta(world, id)
		modAcfMutex.Unlock()
		if meta == nil || len(meta.ACF) == 0 {
			return fmt.Errorf("获取模组acf元数据失败, modID: %d", id)
		}
	} else {
		src = fmt.Sprintf("%s/workshop-%d", g.notUgcPath, id)
		dst = userModNotUgcContentPath(username, id)
	}

	if !utils.FileDirectoryExists(src) {
		return fmt.Errorf("模组源目录不存在: %s", src)
	}
	if err := utils.EnsureDirExists(filepath.Dir(dst)); err != nil {
		return err
	}
	if err := replaceDir(src, dst); err != nil {
		return err
	}

	if ugc {
		if err := writeUserModMeta(userModUgcMetaPath(username, id), meta); err != nil {
			// 保留已写入的内容，避免元数据写入失败时把旧缓存也一并清掉
			return err
		}
	}

	return nil
}

// getUserCachedMods 扫描用户缓存目录，返回缓存中的模组列表；同一 ID 同时存在时 ugc 优先。
func (g *Game) getUserCachedMods(username string) []DownloadedMod {
	mods := make([]DownloadedMod, 0)
	if username == "" {
		return mods
	}

	unlock := lockUserModCache(username)
	defer unlock()

	seen := make(map[int]struct{})

	ugcDirs, err := utils.GetDirs(userModUgcDir(username), false)
	if err != nil && !os.IsNotExist(err) {
		logger.Logger.Warnf("读取用户ugc模组缓存失败, username: %s, err: %v", username, err)
	}
	for _, dir := range ugcDirs {
		id, err := strconv.Atoi(dir)
		if err != nil || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		localSize := "0"
		meta, err := readUserModMeta(username, id)
		if err != nil {
			logger.Logger.Warnf("读取用户模组缓存元数据失败, username: %s, modID: %d, err: %v", username, id, err)
		} else if meta != nil {
			if size := meta.ACF["size"]; size != "" {
				localSize = size
			}
		}

		mods = append(mods, DownloadedMod{ID: id, LocalSize: localSize})
	}

	notUgcDirs, err := utils.GetDirs(userModNotUgcDir(username), false)
	if err != nil && !os.IsNotExist(err) {
		logger.Logger.Warnf("读取用户非ugc模组缓存失败, username: %s, err: %v", username, err)
	}
	for _, dir := range notUgcDirs {
		if !strings.HasPrefix(dir, "workshop-") {
			continue
		}
		id, err := strconv.Atoi(strings.TrimPrefix(dir, "workshop-"))
		if err != nil || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		mods = append(mods, DownloadedMod{ID: id, LocalSize: "0"})
	}

	return mods
}

// ensureModAvailable 在启用模组前确保房间实际存在该模组，缺失时从用户缓存恢复。
func (g *Game) ensureModAvailable(username string, id int, ugc bool) error {
	if id <= 0 {
		return fmt.Errorf("模组ID非法: %d", id)
	}

	if !ugc {
		dstPath := fmt.Sprintf("%s/workshop-%d", g.notUgcPath, id)
		if utils.FileDirectoryExists(dstPath) {
			return nil
		}
		if username == "" {
			return fmt.Errorf("用户名为空，无法从缓存恢复模组")
		}

		unlock := lockUserModCache(username)
		defer unlock()

		if utils.FileDirectoryExists(dstPath) {
			return nil
		}
		src := userModNotUgcContentPath(username, id)
		if !utils.FileDirectoryExists(src) {
			return fmt.Errorf("用户缓存中不存在该模组: %d", id)
		}
		if err := utils.EnsureDirExists(filepath.Dir(dstPath)); err != nil {
			return err
		}
		return replaceDir(src, dstPath)
	}

	if g.ugcModAvailableInRoom(id) {
		return nil
	}
	if username == "" {
		return fmt.Errorf("用户名为空，无法从缓存恢复模组")
	}

	unlock := lockUserModCache(username)
	defer unlock()

	if g.ugcModAvailableInRoom(id) {
		return nil
	}

	meta, err := readUserModMeta(username, id)
	if err != nil {
		return err
	}
	if meta == nil || meta.ID != id || len(meta.ACF) == 0 {
		return fmt.Errorf("用户缓存模组元数据缺失或不完整, modID: %d", id)
	}
	src := userModUgcContentPath(username, id)
	if !utils.FileDirectoryExists(src) {
		return fmt.Errorf("用户缓存缺少模组内容, modID: %d", id)
	}

	modAcfMutex.Lock()
	defer modAcfMutex.Unlock()

	for _, world := range g.worldSaveData {
		contentPath := fmt.Sprintf("%s/%s/content/322330/%d", g.ugcPath, world.WorldName, id)
		if !utils.FileDirectoryExists(contentPath) {
			if err = utils.EnsureDirExists(filepath.Dir(contentPath)); err != nil {
				return err
			}
			if err = replaceDir(src, contentPath); err != nil {
				return err
			}
		}

		acfPath := fmt.Sprintf("%s/%s/appworkshop_322330.acf", g.ugcPath, world.WorldName)
		if err = ensureAcfEntry(acfPath, id, meta.ACF, meta.ACFRoot); err != nil {
			return err
		}
	}

	return nil
}

// deleteUserCachedMod 删除用户缓存中的指定模组，ugc 与 not_ugc 两份副本都清理，
// 避免删除后另一份副本再次出现在已下载列表中；不存在时视为成功。
func (g *Game) deleteUserCachedMod(username string, id int) error {
	if username == "" {
		return fmt.Errorf("用户名为空，无法清理模组缓存")
	}
	if id <= 0 {
		return fmt.Errorf("模组ID非法: %d", id)
	}

	unlock := lockUserModCache(username)
	defer unlock()

	if err := utils.RemoveDir(userModUgcContentPath(username, id)); err != nil {
		return err
	}
	if err := os.Remove(userModUgcMetaPath(username, id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除模组缓存元数据失败: %w", err)
	}
	if err := utils.RemoveDir(userModNotUgcContentPath(username, id)); err != nil {
		return err
	}

	return nil
}

// ugcModAvailableInRoom 判断房间是否至少有一个分世界已经存在该 ugc 模组内容。
// 只要任一世界存在，ModEnable 就能通过 resolveUgcWorld 读到 modinfo，无需从缓存恢复。
func (g *Game) ugcModAvailableInRoom(id int) bool {
	for _, world := range g.worldSaveData {
		contentPath := fmt.Sprintf("%s/%s/content/322330/%d", g.ugcPath, world.WorldName, id)
		if utils.FileDirectoryExists(contentPath) {
			return true
		}
	}

	return false
}

// ensureAcfEntry 确保 acf 文件包含指定模组的 WorkshopItemsInstalled 条目，
// 文件不存在、为空或缺少节点时补齐；条目已存在则保持原样。
func ensureAcfEntry(acfPath string, id int, fields, rootFields map[string]string) error {
	if err := utils.EnsureDirExists(filepath.Dir(acfPath)); err != nil {
		return err
	}

	parser, err := utils.NewParser(acfPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("读取acf文件失败: %w", err)
		}
		parser = nil
	}
	if parser == nil || parser.Root == nil {
		parser = utils.NewEmptyParser()
		for k, v := range rootFields {
			parser.Root.List[k] = v
		}
	}

	if parser.Root.Name != "AppWorkshop" {
		return fmt.Errorf("未知的acf结构: %s", parser.Root.Name)
	}
	if !hasWorkshopItemsInstalledNode(parser.Root) {
		parser.Root.AddChild("WorkshopItemsInstalled")
	}

	acfID := strconv.Itoa(id)
	if _, err = parser.GetWorkshopItemsInstalled(acfID); err == nil {
		return nil
	}

	node := utils.NewNode(acfID)
	node.List = make(map[string]string, len(fields))
	for k, v := range fields {
		node.List[k] = v
	}
	if err = parser.AddWorkshopItemsInstalled(node); err != nil {
		return err
	}

	return utils.TruncAndWriteFile(acfPath, strings.Join(parser.Format(), "\n"))
}

func hasWorkshopItemsInstalledNode(root *utils.Node) bool {
	for _, child := range root.Children {
		if child.Name == "WorkshopItemsInstalled" {
			return true
		}
	}
	return false
}

// ugcModMeta 从下载完成后的 acf 中读取模组条目，作为用户缓存的恢复元数据。
func (g *Game) ugcModMeta(world string, id int) *userModMeta {
	if world == "" {
		return nil
	}

	acfPath := fmt.Sprintf("%s/%s/appworkshop_322330.acf", g.ugcPath, world)
	parser, err := utils.NewParser(acfPath)
	if err != nil {
		logger.Logger.Warnf("读取acf文件失败, path: %s, err: %v", acfPath, err)
		return nil
	}
	if parser.Root == nil {
		logger.Logger.Warnf("acf文件为空, path: %s", acfPath)
		return nil
	}

	node, err := parser.GetWorkshopItemsInstalled(strconv.Itoa(id))
	if err != nil {
		logger.Logger.Warnf("acf文件中未找到模组条目, modID: %d, err: %v", id, err)
		return nil
	}

	fields := make(map[string]string, len(node.List))
	for k, v := range node.List {
		fields[k] = v
	}

	rootFields := make(map[string]string, len(parser.Root.List))
	for k, v := range parser.Root.List {
		rootFields[k] = v
	}

	return &userModMeta{
		ID:      id,
		SavedAt: time.Now().UnixMilli(),
		ACF:     fields,
		ACFRoot: rootFields,
	}
}

// replaceDir 先复制到临时目录，再整体替换目标目录，避免列表读到半成品。
func replaceDir(src, dst string) error {
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	tmp := fmt.Sprintf("%s.tmp-%s", dst, suffix)
	old := fmt.Sprintf("%s.old-%s", dst, suffix)

	if err := utils.CopyDir(src, tmp); err != nil {
		_ = utils.RemoveDir(tmp)
		return err
	}

	hasOld := utils.FileDirectoryExists(dst)
	if hasOld {
		if err := os.Rename(dst, old); err != nil {
			_ = utils.RemoveDir(tmp)
			return fmt.Errorf("备份旧模组失败: %w", err)
		}
	}

	if err := os.Rename(tmp, dst); err != nil {
		if hasOld {
			_ = os.Rename(old, dst)
		}
		_ = utils.RemoveDir(tmp)
		return fmt.Errorf("替换模组目录失败: %w", err)
	}

	if hasOld {
		_ = utils.RemoveDir(old)
	}

	return nil
}

func writeUserModMeta(path string, meta *userModMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("序列化模组元数据失败: %w", err)
	}

	if err = utils.EnsureDirExists(filepath.Dir(path)); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, data, 0644); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("写入模组元数据失败: %w", err)
	}

	// Windows 下 os.Rename 不能覆盖已存在的文件，先删除旧文件
	_ = os.Remove(path)
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("保存模组元数据失败: %w", err)
	}

	return nil
}

func readUserModMeta(username string, id int) (*userModMeta, error) {
	data, err := os.ReadFile(userModUgcMetaPath(username, id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var meta userModMeta
	if err = json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
