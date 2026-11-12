package tools

import (
	"context"
	"crypto/sha256"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// mapContext 地图接口共用的房间上下文
type mapContext struct {
	room      *models.Room
	worlds    *[]models.World
	setting   *models.RoomSetting
	worldName string
}

// mapWorldContext 统一地图接口的 roomID/worldID 校验：参数、房间权限、世界归属。
// ok 为 false 时返回的 result 已经是可以直接写给前端的响应。
func (s *Service) mapWorldContext(actor service.Actor, roomID, worldID int) (mapContext, bool, service.Result) {
	if roomID <= 0 || worldID <= 0 {
		return mapContext{}, false, service.Fail(constant.CodeToolsBadRequest)
	}
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return mapContext{}, false, service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("地图获取房间信息失败, roomID: %d, err: %v", roomID, err)
		return mapContext{}, false, service.Fail(constant.CodeToolsDBError)
	}

	for _, world := range *worlds {
		if world.ID == worldID {
			return mapContext{
				room:      room,
				worlds:    worlds,
				setting:   roomSetting,
				worldName: world.WorldName,
			}, true, service.Result{}
		}
	}

	return mapContext{}, false, service.Fail(constant.CodeToolsBadRequest)
}

// Map 返回完整世界地图（实时从存档生成）以及可选的实时在线玩家。
func (s *Service) Map(actor service.Actor, ctx context.Context, roomID, worldID int, players string) service.Result {
	probePlayers := true
	if value := strings.TrimSpace(players); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || (parsed != 0 && parsed != 1) {
			return service.Fail(constant.CodeToolsBadRequest)
		}
		probePlayers = parsed == 1
	}

	mc, ok, fail := s.mapWorldContext(actor, roomID, worldID)
	if !ok {
		return fail
	}

	ctx, cancel := context.WithTimeout(ctx, dst.MapRequestTimeoutSeconds*time.Second)
	defer cancel()

	lockCtx, cancelLock := context.WithTimeout(ctx, dst.MapLockWaitSeconds*time.Second)
	release, err := cache.AcquireMap(lockCtx, roomID, worldID)
	cancelLock()
	if err != nil {
		logger.Logger.Warnf("地图等待生成锁超时, roomID: %d, worldID: %d, err: %v", roomID, worldID, err)
		return service.Fail(constant.CodeToolsMapBusy)
	}
	defer release()

	game := dst.NewGameController(mc.room, mc.worlds, mc.setting, actor.Lang)
	mapData, err := game.GenerateFullMap(ctx, worldID)
	if err != nil {
		switch {
		case errors.Is(err, dst.ErrMapSaveNotFound):
			logger.Logger.Warnf("地图存档不存在, roomID: %d, worldID: %d, worldName: %s, err: %v", roomID, worldID, mc.worldName, err)
			return service.Fail(constant.CodeToolsMapSaveNotFound)
		case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled):
			logger.Logger.Warnf("地图生成超时或被取消, roomID: %d, worldID: %d, worldName: %s", roomID, worldID, mc.worldName)
			return service.Fail(constant.CodeToolsMapBusy)
		default:
			logger.Logger.Errorf("地图生成失败, roomID: %d, worldID: %d, worldName: %s, err: %v", roomID, worldID, mc.worldName, err)
			return service.Fail(constant.CodeToolsMapParseFail)
		}
	}

	assetCtx, cancelAssets := context.WithTimeout(ctx, dst.MapAssetFillTimeoutSeconds*time.Second)
	s.fillMapAssets(assetCtx, &mapData)
	cancelAssets()

	if probePlayers {
		mapData.Players = game.PlayerPositions(ctx, worldID)
		if mapData.Players.Error != "" {
			logger.Logger.Warnf("地图在线玩家获取失败, roomID: %d, worldID: %d, err: %s", roomID, worldID, mapData.Players.Error)
		}
	} else {
		mapData.Players = dst.MapPlayers{
			ProbedAt:     time.Now().UTC(),
			WorldRunning: game.WorldUpStatus(worldID),
			List:         []dst.MapPlayer{},
		}
	}

	return service.Success(mapData)
}

// MapPlayers 只刷新在线玩家位置，不重新解析地图。
func (s *Service) MapPlayers(actor service.Actor, ctx context.Context, roomID, worldID int) service.Result {
	mc, ok, fail := s.mapWorldContext(actor, roomID, worldID)
	if !ok {
		return fail
	}

	ctx, cancel := context.WithTimeout(ctx, dst.MapPlayersRequestTimeoutSeconds*time.Second)
	defer cancel()

	lockCtx, cancelLock := context.WithTimeout(ctx, dst.MapLockWaitSeconds*time.Second)
	release, err := cache.AcquireMap(lockCtx, roomID, worldID)
	cancelLock()
	if err != nil {
		logger.Logger.Warnf("地图玩家刷新等待锁超时, roomID: %d, worldID: %d, err: %v", roomID, worldID, err)
		return service.Fail(constant.CodeToolsMapBusy)
	}
	defer release()

	game := dst.NewGameController(mc.room, mc.worlds, mc.setting, actor.Lang)
	players := game.PlayerPositions(ctx, worldID)
	if players.Error != "" {
		logger.Logger.Warnf("地图在线玩家刷新失败, roomID: %d, worldID: %d, worldName: %s, err: %s",
			roomID, worldID, mc.worldName, players.Error)
	}

	return service.Success(players)
}

// fillMapAssets 把地形贴图直接填进 assets，并把 minimap 图标打包进 iconAtlas。
// 先查 map_images，缺失的再解包并回写；超时或资源缺失时保留空 PNG，由前端回退。
func (s *Service) fillMapAssets(ctx context.Context, mapData *dst.MapFull) {
	if mapData == nil || s.mapImageDao == nil {
		return
	}

	mapAssetExtractMutex.Lock()
	defer mapAssetExtractMutex.Unlock()

	gameBuild := strings.TrimSpace(mapData.GameBuild)
	if gameBuild == "" {
		gameBuild = "unknown"
	}

	icons := make([]string, 0, len(mapData.Prefabs))
	tiles := make([]string, 0, len(mapData.Assets))
	seen := make(map[string]struct{}, len(mapData.Prefabs)+len(mapData.Assets))
	for _, prefab := range mapData.Prefabs {
		name := prefab.Icon
		if name == "" {
			continue
		}
		if !strings.HasSuffix(name, ".png") {
			name += ".png"
		}
		key := mapImageKey("icon", name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		icons = append(icons, name)
	}
	for _, ref := range mapData.Assets {
		if ref.Name == "" {
			continue
		}
		key := mapImageKey("tile", ref.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tiles = append(tiles, ref.Name)
	}

	rows := make(map[string]models.MapImage, len(icons)+len(tiles))
	load := func(kind string, names []string) bool {
		if len(names) == 0 {
			return true
		}
		got, err := s.mapImageDao.GetMany(kind, gameBuild, names)
		if err != nil {
			logger.Logger.Warnf("地图图片查询失败, kind: %s, build: %s, err: %v", kind, gameBuild, err)
			return false
		}
		for _, row := range got {
			rows[mapImageKey(kind, row.Name)] = row
		}
		return true
	}
	if !load("icon", icons) || !load("tile", tiles) {
		return
	}

	missing := map[string][]string{"icon": {}, "tile": {}}
	for _, name := range icons {
		if row, ok := rows[mapImageKey("icon", name)]; !ok || row.Image == "" {
			missing["icon"] = append(missing["icon"], name)
		}
	}
	for _, name := range tiles {
		if row, ok := rows[mapImageKey("tile", name)]; !ok || row.Image == "" {
			missing["tile"] = append(missing["tile"], name)
		}
	}

	for _, kind := range []string{"icon", "tile"} {
		names := missing[kind]
		if len(names) == 0 {
			continue
		}
		assets := dst.ExtractMapAssets(ctx, kind, names)
		if len(assets) == 0 {
			continue
		}
		records := make([]models.MapImage, 0, len(assets))
		for _, asset := range assets {
			record := models.MapImage{
				Name:      asset.Name,
				Kind:      asset.Kind,
				GameBuild: gameBuild,
				Version:   models.MapImageVersion,
				Image:     asset.ImageBase64,
				Width:     asset.Width,
				Height:    asset.Height,
				SHA256:    asset.SHA256,
			}
			records = append(records, record)
			rows[mapImageKey(kind, asset.Name)] = record
		}
		if err := s.mapImageDao.UpsertMany(records); err != nil {
			logger.Logger.Warnf("地图图片缓存失败, kind: %s, build: %s, err: %v", kind, gameBuild, err)
		}
	}

	terrain := make([]dst.MapAssetRef, 0, len(tiles))
	normalizedRows := make([]models.MapImage, 0)
	for _, name := range tiles {
		key := mapImageKey("tile", name)
		row, ok := rows[key]
		if !ok || row.Image == "" {
			continue
		}
		// 兼容旧缓存：如果仍是 256px，按当前策略重压后再返回。
		if row.Width > dst.MapTileTextureSide || row.Height > dst.MapTileTextureSide || row.Width == 0 {
			if png, width, height, ok := dst.NormalizeMapAssetBase64("tile", row.Image); ok {
				row.Image = png
				row.Width = width
				row.Height = height
				row.Version = models.MapImageVersion
				if data, err := base64.StdEncoding.DecodeString(png); err == nil {
					row.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
				}
				rows[key] = row
				normalizedRows = append(normalizedRows, row)
			}
		}
		terrain = append(terrain, dst.MapAssetRef{
			Name: name,
			PNG:  row.Image,
		})
	}
	mapData.Assets = terrain
	if len(normalizedRows) > 0 {
		if err := s.mapImageDao.UpsertMany(normalizedRows); err != nil {
			logger.Logger.Warnf("地图旧图片重新压缩失败, build: %s, err: %v", gameBuild, err)
		}
	}

	iconImages := make(map[string]dst.MapAssetData, len(icons))
	for _, name := range icons {
		row, ok := rows[mapImageKey("icon", name)]
		if !ok || row.Image == "" {
			continue
		}
		iconImages[name] = dst.MapAssetData{
			Name:        name,
			Kind:        "icon",
			ImageBase64: row.Image,
			Width:       row.Width,
			Height:      row.Height,
			SHA256:      row.SHA256,
		}
	}
	mapData.IconAtlas = dst.BuildMapIconAtlas(iconImages)
}

func mapImageKey(kind, name string) string {
	return kind + "\x00" + strings.ToLower(name)
}

var mapAssetExtractMutex sync.Mutex
