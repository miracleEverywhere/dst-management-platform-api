package dst

import (
	"bytes"
	"compress/zlib"
	"context"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	lua "github.com/yuin/gopher-lua"
)

// 地图相关限制。
const (
	// maxMapSaveBytes 单个存档文件读取上限。
	maxMapSaveBytes = 64 << 20
	// maxMapDimension 地图单边格数上限。
	maxMapDimension = 2048
	// maxMapTiles 地图总格数上限。
	maxMapTiles = 4_194_304
	// maxMapObjects 实体数量上限。
	maxMapObjects = 200_000
	// maxMapPrefabs prefab 字典上限。
	maxMapPrefabs = 4096
	// maxMapTileSpecs 地形字典上限。
	maxMapTileSpecs = 4096
	// maxMapLabels 路牌数量上限。
	maxMapLabels = 512
	// maxMapLabelRunes 单条路牌文本 rune 上限。
	maxMapLabelRunes = 64
	// maxMapSkeletons 玩家遗骸数量上限。
	maxMapSkeletons = 512
	// maxMapSkeletonNameRunes 遗骸玩家名 rune 上限。
	maxMapSkeletonNameRunes = 48
	// maxMapSkeletonEquipSlots 单具遗骸装备槽上限。
	maxMapSkeletonEquipSlots = 5
	// maxMapWormholes 虫洞对数量上限。
	maxMapWormholes = 1024
	// maxMapPayloadBytes 单次响应的 JSON 上限。
	maxMapPayloadBytes = 32 << 20
	// maxMapPlayers 单世界返回的在线玩家上限。
	maxMapPlayers = 32
	// maxMapPlayerNameRunes 玩家名 rune 上限。
	maxMapPlayerNameRunes = 48

	// MapRequestTimeoutSeconds 单次请求的总超时（秒）。
	MapRequestTimeoutSeconds = 20
	// MapPlayersRequestTimeoutSeconds 仅刷新玩家位置的请求超时（秒）。
	MapPlayersRequestTimeoutSeconds = 10
	// MapLockWaitSeconds 同一世界等待生成锁的上限（秒）。
	MapLockWaitSeconds = 10
	// mapPlayersWaitMillis 等待玩家命令输出 marker 的上限（毫秒）。
	mapPlayersWaitMillis = 2000
	// mapPlayersPollMillis 轮询玩家日志的间隔（毫秒）。
	mapPlayersPollMillis = 50
	// mapLogTailLines 玩家探测回读的日志行数。
	mapLogTailLines = 400
)

// MapFull 是地图的返回结构：完整地形、全部实体和世界状态。
// 不在服务端生成 PNG，也不落库；每次请求实时从存档投影。
type MapFull struct {
	SchemaVersion int       `json:"schemaVersion"`
	WorldID       int       `json:"worldId"`
	WorldName     string    `json:"worldName"`
	GeneratedAt   time.Time `json:"generatedAt"`
	SaveName      string    `json:"saveName"`
	SaveModTime   time.Time `json:"saveModTime"`
	GameBuild     string    `json:"gameBuild,omitempty"`

	Width    int  `json:"width"`
	Height   int  `json:"height"`
	HasOcean bool `json:"hasOcean"`

	Clock MapClock `json:"clock"`
	World MapWorld `json:"world"`

	TileSpecs []MapTileSpec `json:"tileSpecs"`
	Tiles     string        `json:"tiles"`
	TileBytes int           `json:"tileBytes"`

	Prefabs     []MapPrefab    `json:"prefabs"`
	Objects     string         `json:"objects"`
	ObjectCount int            `json:"objectCount"`
	LayerCounts map[string]int `json:"layerCounts"`
	Wormholes   []MapWormhole  `json:"wormholes"`
	Labels      []MapLabel     `json:"labels"`
	Skeletons   []MapSkeleton  `json:"skeletons"`
	Assets      []MapAssetRef  `json:"assets"`
	IconAtlas   MapIconAtlas   `json:"iconAtlas"`
	Players     MapPlayers     `json:"players"`
}

type MapClock struct {
	Cycles         int     `json:"cycles"`
	Phase          string  `json:"phase,omitempty"`
	Season         string  `json:"season,omitempty"`
	DayInSeason    int     `json:"dayInSeason"`
	DaysInSeason   int     `json:"daysInSeason"`
	PhaseSeconds   float64 `json:"phaseSeconds,omitempty"`
	PhaseRemaining float64 `json:"phaseRemaining,omitempty"`
}

type MapWorld struct {
	Temperature    float64 `json:"temperature"`
	IsDay          bool    `json:"isDay"`
	Season         string  `json:"season,omitempty"`
	SeasonDaysGone int     `json:"seasonDaysGone"`
	SeasonDaysLeft int     `json:"seasonDaysLeft"`
	SeasonProgress float64 `json:"seasonProgress"`
}

type MapTileSpec struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Noise string `json:"noise,omitempty"`
	Tint  string `json:"tint,omitempty"`
}

type MapPrefab struct {
	Name  string `json:"name"`
	Layer string `json:"layer"`
	Icon  string `json:"icon,omitempty"`
	EN    string `json:"en,omitempty"`
	ZH    string `json:"zh,omitempty"`
}

type MapWormhole struct {
	X1     float64 `json:"x1"`
	Z1     float64 `json:"z1"`
	X2     float64 `json:"x2"`
	Z2     float64 `json:"z2"`
	Number int     `json:"number,omitempty"`
}

type MapLabel struct {
	Prefab string  `json:"prefab"`
	Text   string  `json:"text"`
	X      float64 `json:"x"`
	Z      float64 `json:"z"`
}

type MapSkeleton struct {
	X          float64           `json:"x"`
	Z          float64           `json:"z"`
	Name       string            `json:"name,omitempty"`
	Char       string            `json:"char,omitempty"`
	Cause      string            `json:"cause,omitempty"`
	UID        string            `json:"uid,omitempty"`
	AgeSeconds float64           `json:"ageSeconds,omitempty"`
	Equip      map[string]string `json:"equip,omitempty"`
}

type MapAssetRef struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256,omitempty"`
}

type MapIconAtlas struct {
	Name   string           `json:"name,omitempty"`
	Width  int              `json:"width,omitempty"`
	Height int              `json:"height,omitempty"`
	Rects  map[string][]int `json:"rects,omitempty"`
}

type MapPlayers struct {
	ProbedAt     time.Time   `json:"probedAt"`
	WorldRunning bool        `json:"worldRunning"`
	Error        string      `json:"error,omitempty"`
	List         []MapPlayer `json:"list"`
}

type MapPlayer struct {
	UID    string  `json:"uid"`
	Name   string  `json:"name,omitempty"`
	Prefab string  `json:"prefab,omitempty"`
	X      float64 `json:"x"`
	Z      float64 `json:"z"`
}

var (
	// ErrMapSaveNotFound 表示世界还没有任何存档（从未启动过）。
	ErrMapSaveNotFound = errors.New("map save not found")
	// ErrMapParseFailed 表示存档存在但解析失败。
	ErrMapParseFailed = errors.New("map parse failed")
)

// GenerateFullMap 实时读取指定世界的最新存档并构建载荷。
// 不写数据库、不写文件；调用方负责并发保护与超时。
func (g *Game) GenerateFullMap(ctx context.Context, worldID int) (MapFull, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	world, err := g.getWorldByID(worldID)
	if err != nil {
		return MapFull{}, fmt.Errorf("%w: %v", ErrMapSaveNotFound, err)
	}

	metaPath, err := findLatestMetaFile(world.sessionPath)
	if err != nil {
		return MapFull{}, fmt.Errorf("%w: %v", ErrMapSaveNotFound, err)
	}
	savePath := strings.TrimSuffix(metaPath, ".meta")
	if savePath == metaPath {
		return MapFull{}, fmt.Errorf("%w: invalid meta path", ErrMapParseFailed)
	}

	save, err := loadWorldSave(ctx, savePath, metaPath)
	if err != nil && !errors.Is(err, ErrMapSaveNotFound) && ctx.Err() == nil {
		// 游戏可能正在写存档，稍等后重试一次；仍失败则按解析失败返回。
		time.Sleep(150 * time.Millisecond)
		save, err = loadWorldSave(ctx, savePath, metaPath)
	}
	if err != nil {
		if errors.Is(err, ErrMapSaveNotFound) {
			return MapFull{}, err
		}
		return MapFull{}, fmt.Errorf("%w: %v", ErrMapParseFailed, err)
	}
	if err := ctx.Err(); err != nil {
		return MapFull{}, fmt.Errorf("%w: %v", ErrMapParseFailed, err)
	}

	fullMap, err := buildMapFull(save, world)
	if err != nil {
		return MapFull{}, fmt.Errorf("%w: %v", ErrMapParseFailed, err)
	}
	if len(fullMap.Tiles) == 0 || len(fullMap.Objects) == 0 {
		return MapFull{}, fmt.Errorf("%w: empty map payload", ErrMapParseFailed)
	}

	encoded, err := json.Marshal(fullMap)
	if err != nil {
		return MapFull{}, fmt.Errorf("%w: %v", ErrMapParseFailed, err)
	}
	if len(encoded) > maxMapPayloadBytes {
		return MapFull{}, fmt.Errorf("%w: payload exceeds limit", ErrMapParseFailed)
	}

	logger.Logger.Infof(
		"地图生成成功, roomID: %d, worldID: %d, worldName: %s, width: %d, height: %d, objects: %d, prefabs: %d, payload: %d bytes",
		g.room.ID, world.ID, world.WorldName, fullMap.Width, fullMap.Height,
		fullMap.ObjectCount, len(fullMap.Prefabs), len(encoded),
	)

	return fullMap, nil
}

func buildMapFull(save *worldSave, world *worldSaveData) (MapFull, error) {
	if save == nil || world == nil {
		return MapFull{}, errors.New("nil save or world")
	}
	if save.Width < 1 || save.Height < 1 ||
		save.Width > maxMapDimension || save.Height > maxMapDimension {
		return MapFull{}, fmt.Errorf("invalid map size %dx%d", save.Width, save.Height)
	}
	tileCount := save.Width * save.Height
	if tileCount > maxMapTiles || tileCount != len(save.TileIDs) {
		return MapFull{}, fmt.Errorf("invalid tile count %d", len(save.TileIDs))
	}
	if len(save.Entities) > maxMapObjects {
		save.Entities = save.Entities[:maxMapObjects]
	}

	tiles, err := encodeMapTiles(save)
	if err != nil {
		return MapFull{}, err
	}
	prefabs, objects, objectCount, layerCounts, err := encodeMapObjects(save.Entities)
	if err != nil {
		return MapFull{}, err
	}

	fullMap := MapFull{
		SchemaVersion: 1,
		WorldID:       world.ID,
		WorldName:     world.WorldName,
		GeneratedAt:   time.Now().UTC(),
		SaveName:      save.SaveName,
		SaveModTime:   save.SaveModTime.UTC(),
		GameBuild:     save.GameBuild,
		Width:         save.Width,
		Height:        save.Height,
		HasOcean:      save.HasOcean,
		Clock:         save.Clock,
		World:         save.World,
		TileSpecs:     buildMapTileSpecs(save.TileNames, save.TileIDs),
		Tiles:         tiles,
		TileBytes:     2,
		Prefabs:       prefabs,
		Objects:       objects,
		ObjectCount:   objectCount,
		LayerCounts:   layerCounts,
		Wormholes:     save.Wormholes,
		Labels:        save.Labels,
		Skeletons:     save.Skeletons,
		Assets:        []MapAssetRef{},
		IconAtlas:     MapIconAtlas{},
		Players: MapPlayers{
			List: []MapPlayer{},
		},
	}
	if fullMap.Wormholes == nil {
		fullMap.Wormholes = []MapWormhole{}
	}
	if fullMap.Labels == nil {
		fullMap.Labels = []MapLabel{}
	}
	if fullMap.Skeletons == nil {
		fullMap.Skeletons = []MapSkeleton{}
	}
	return fullMap, nil
}

func encodeMapTiles(save *worldSave) (string, error) {
	raw := make([]byte, save.Width*save.Height*2)
	for row := 0; row < save.Height; row++ {
		dstRow := save.Height - 1 - row
		for col := 0; col < save.Width; col++ {
			id := save.TileIDs[row*save.Width+col]
			offset := (dstRow*save.Width + col) * 2
			binary.BigEndian.PutUint16(raw[offset:offset+2], id)
		}
	}
	return encodeMapBase64(raw)
}

func encodeMapObjects(entities []rawEntity) ([]MapPrefab, string, int, map[string]int, error) {
	prefabIndex := make(map[string]uint16)
	prefabs := make([]MapPrefab, 0, 256)
	layerCounts := make(map[string]int)
	objects := make([]byte, 0, len(entities)*10)
	objectCount := 0

	for _, entity := range entities {
		if math.IsNaN(entity.X) || math.IsNaN(entity.Z) ||
			math.IsInf(entity.X, 0) || math.IsInf(entity.Z, 0) ||
			math.Abs(entity.X) > 1e7 || math.Abs(entity.Z) > 1e7 {
			continue
		}

		meta := mapPrefabMeta(entity.Prefab)
		index, exists := prefabIndex[entity.Prefab]
		if !exists {
			if len(prefabs) >= maxMapPrefabs {
				continue
			}
			index = uint16(len(prefabs))
			prefabIndex[entity.Prefab] = index
			prefabs = append(prefabs, MapPrefab{
				Name:  entity.Prefab,
				Layer: meta.Layer,
				Icon:  meta.Icon,
				EN:    meta.EN,
				ZH:    meta.ZH,
			})
		}

		record := make([]byte, 10)
		binary.LittleEndian.PutUint16(record[0:2], index)
		binary.LittleEndian.PutUint32(record[2:6], uint32(int32(math.Round(entity.X*100))))
		binary.LittleEndian.PutUint32(record[6:10], uint32(int32(math.Round(entity.Z*100))))
		objects = append(objects, record...)
		objectCount++
		layerCounts[meta.Layer]++
	}

	if len(prefabs) == 0 || objectCount == 0 {
		return []MapPrefab{}, "", 0, layerCounts, errors.New("no map entities")
	}

	encoded, err := encodeMapBase64(objects)
	if err != nil {
		return nil, "", 0, nil, err
	}
	return prefabs, encoded, objectCount, layerCounts, nil
}

func buildMapTileSpecs(tileNames map[int]string, tileIDs []uint16) []MapTileSpec {
	idSet := make(map[int]struct{}, len(tileNames)+64)
	for id := range tileNames {
		idSet[id] = struct{}{}
	}
	for _, id := range tileIDs {
		idSet[int(id)] = struct{}{}
	}

	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	if len(ids) > maxMapTileSpecs {
		ids = ids[:maxMapTileSpecs]
	}

	specs := make([]MapTileSpec, 0, len(ids))
	for _, id := range ids {
		name := tileNames[id]
		if !isSafeMapTileName(name) {
			name = fmt.Sprintf("TILE_%d", id)
		}
		specs = append(specs, MapTileSpec{ID: id, Name: name})
	}
	return specs
}

func encodeMapBase64(raw []byte) (string, error) {
	var buffer bytes.Buffer
	writer := zlib.NewWriter(&buffer)
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

// rawEntity 是从存档实体段扫描出来、尚未进入 prefab 字典的实体。
type rawEntity struct {
	Prefab string
	X      float64
	Z      float64
}

type worldSave struct {
	Width     int
	Height    int
	HasOcean  bool
	TileIDs   []uint16
	TileNames map[int]string

	Entities []rawEntity

	Wormholes []MapWormhole
	Labels    []MapLabel
	Skeletons []MapSkeleton

	Clock MapClock
	World MapWorld

	GameBuild   string
	SaveName    string
	SaveModTime time.Time
}

type saveSection struct {
	Name string
	Body string
}

type metaInfo struct {
	Clock          MapClock
	Season         string
	SeasonDaysGone int
	SeasonDaysLeft int
	SeasonProgress float64
}

var (
	mapSectionRe = regexp.MustCompile(`tablefunctions\["([a-zA-Z0-9_]+)_fn"\] = function\(\)`)

	mapTilesRe   = regexp.MustCompile(`tiles="([A-Za-z0-9+/=]+)"`)
	mapWidthRe   = regexp.MustCompile(`[,{]width=(\d+)[,}]`)
	mapHeightRe  = regexp.MustCompile(`[,{]height=(\d+)[,}]`)
	mapOceanRe   = regexp.MustCompile(`[,{]has_ocean=(true|false)`)
	mapTileMapRe = regexp.MustCompile(`world_tile_map=\{([^}]*)\}`)

	mapBuildRe = regexp.MustCompile(`build_version="([^"]+)"`)
	mapTempRe  = regexp.MustCompile(`worldtemperature=\{([^}]*)\}`)
)

func loadWorldSave(ctx context.Context, savePath, metaPath string) (*worldSave, error) {
	info, err := os.Stat(savePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMapSaveNotFound
		}
		return nil, err
	}
	if info.Size() > int64(maxMapSaveBytes) {
		return nil, fmt.Errorf("save file exceeds limit: %d", info.Size())
	}

	raw, err := os.ReadFile(savePath)
	if err != nil {
		return nil, err
	}
	sections := splitSaveSections(string(raw))

	byName := make(map[string]string, len(sections))
	for _, section := range sections {
		byName[section.Name] = section.Body
	}

	mapBody, ok := byName["map"]
	if !ok || mapBody == "" {
		return nil, errors.New("save file has no map section")
	}

	save := &worldSave{
		SaveName:    filepath.Base(savePath),
		SaveModTime: info.ModTime(),
	}
	if err := parseMapSection(mapBody, save); err != nil {
		return nil, err
	}

	if metaBody := byName["meta"]; metaBody != "" {
		save.GameBuild = parseGameBuild(metaBody)
	}

	meta, metaErr := parseMetaFile(ctx, metaPath)
	if metaErr != nil {
		logger.Logger.Warnf("地图解析 .meta 失败，时钟/季节字段留空, err: %v", metaErr)
	}
	save.Clock = meta.Clock
	save.World = parseWorldState(byName["world_network"], meta)

	entities, wormholes, labels, skeletons, err := parseEntitySections(ctx, sections)
	if err != nil {
		return nil, err
	}
	save.Entities = entities
	save.Wormholes = wormholes
	save.Labels = labels
	save.Skeletons = skeletons

	return save, nil
}

// splitSaveSections 按 DST 存档的 tablefunctions 段切分；保留文件顺序。
func splitSaveSections(text string) []saveSection {
	matches := mapSectionRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}
	sections := make([]saveSection, 0, len(matches))
	for _, match := range matches {
		name := text[match[2]:match[3]]
		body, _ := extractLuaReturn(text, match[1])
		if body == "" {
			continue
		}
		sections = append(sections, saveSection{Name: name, Body: body})
	}
	return sections
}

// extractLuaReturn 从 from 位置开始找到 return {...}，并用花括号配对（跳过字符串）取完整表。
func extractLuaReturn(text string, from int) (string, int) {
	if from < 0 || from >= len(text) {
		return "", from
	}
	idx := strings.Index(text[from:], "\nreturn ")
	if idx < 0 {
		idx = strings.Index(text[from:], "return ")
	}
	if idx < 0 {
		return "", from
	}
	if text[from+idx] == '\n' {
		idx++
	}
	start := from + idx

	open := strings.Index(text[start:], "{")
	if open < 0 {
		return "", from
	}
	open += start

	depth := 0
	var quote byte
	escaped := false
	for i := open; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1], i + 1
			}
		}
	}
	return "", from
}

func parseMapSection(body string, save *worldSave) error {
	widthMatch := mapWidthRe.FindStringSubmatch(body)
	heightMatch := mapHeightRe.FindStringSubmatch(body)
	if len(widthMatch) < 2 || len(heightMatch) < 2 {
		return errors.New("save map section has no width/height")
	}
	width, err := strconv.Atoi(widthMatch[1])
	if err != nil {
		return fmt.Errorf("invalid map width: %w", err)
	}
	height, err := strconv.Atoi(heightMatch[1])
	if err != nil {
		return fmt.Errorf("invalid map height: %w", err)
	}
	if width < 1 || height < 1 || width > maxMapDimension || height > maxMapDimension {
		return fmt.Errorf("invalid map size: %dx%d", width, height)
	}
	if width*height > maxMapTiles {
		return fmt.Errorf("map tiles exceed limit: %dx%d", width, height)
	}

	tilesMatch := mapTilesRe.FindStringSubmatch(body)
	if len(tilesMatch) < 2 {
		return errors.New("save map section has no tiles")
	}
	tilesDecoded, err := base64.StdEncoding.DecodeString(tilesMatch[1])
	if err != nil {
		return fmt.Errorf("decode tiles: %w", err)
	}
	if len(tilesDecoded) < 8 || string(tilesDecoded[:4]) != "VRSN" {
		return errors.New("invalid tiles header")
	}
	expected := width * height * 2
	if len(tilesDecoded)-8 < expected {
		return fmt.Errorf("tiles too short: %d < %d", len(tilesDecoded)-8, expected)
	}

	tileIDs := make([]uint16, width*height)
	for i := range tileIDs {
		offset := 8 + i*2
		tileIDs[i] = binary.BigEndian.Uint16(tilesDecoded[offset : offset+2])
	}

	save.Width = width
	save.Height = height
	save.TileIDs = tileIDs
	if match := mapOceanRe.FindStringSubmatch(body); len(match) >= 2 {
		save.HasOcean = match[1] == "true"
	}

	if tileMapMatch := mapTileMapRe.FindStringSubmatch(body); len(tileMapMatch) >= 2 {
		save.TileNames = parseTileNames(tileMapMatch[1])
	}
	if save.TileNames == nil {
		save.TileNames = map[int]string{}
	}

	return nil
}

func parseTileNames(raw string) map[int]string {
	names := make(map[int]string)
	for _, pair := range strings.Split(raw, ",") {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		id, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil || !isSafeMapTileName(name) {
			continue
		}
		names[id] = name
	}
	return names
}

func parseMetaFile(ctx context.Context, metaPath string) (metaInfo, error) {
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		return metaInfo{}, err
	}
	body, _ := extractLuaReturn(string(raw), 0)
	if body == "" {
		return metaInfo{}, errors.New("meta file has no return table")
	}

	L := utils.NewSafeLuaState()
	defer L.Close()
	L.SetContext(ctx)
	if err := L.DoString(body); err != nil {
		return metaInfo{}, err
	}
	value := L.Get(-1)
	L.Pop(1)
	root, ok := value.(*lua.LTable)
	if !ok {
		return metaInfo{}, errors.New("meta table is invalid")
	}

	info := metaInfo{}
	if clock := luaTable(root.RawGetString("clock")); clock != nil {
		info.Clock.Cycles = int(luaNumber(clock, "cycles"))
		info.Clock.Phase = luaString(clock, "phase")
		info.Clock.PhaseSeconds = luaNumber(clock, "totaltimeinphase")
		info.Clock.PhaseRemaining = luaNumber(clock, "remainingtimeinphase")
	}
	if seasons := luaTable(root.RawGetString("seasons")); seasons != nil {
		info.Season = luaString(seasons, "season")
		info.SeasonDaysGone = int(luaNumber(seasons, "elapseddaysinseason"))
		total := int(luaNumber(seasons, "totaldaysinseason"))
		info.Clock.DayInSeason = info.SeasonDaysGone
		info.Clock.DaysInSeason = total
		info.Clock.Season = info.Season
		if total > 0 {
			info.SeasonDaysLeft = total - info.SeasonDaysGone
			if info.SeasonDaysLeft < 0 {
				info.SeasonDaysLeft = 0
			}
			info.SeasonProgress = float64(info.SeasonDaysGone) / float64(total)
			if info.SeasonProgress > 1 {
				info.SeasonProgress = 1
			}
		}
	}
	return info, nil
}

func parseGameBuild(body string) string {
	if match := mapBuildRe.FindStringSubmatch(body); len(match) >= 2 {
		return match[1]
	}
	return ""
}

func parseWorldState(body string, meta metaInfo) MapWorld {
	world := MapWorld{
		Season:         meta.Season,
		SeasonDaysGone: meta.SeasonDaysGone,
		SeasonDaysLeft: meta.SeasonDaysLeft,
		SeasonProgress: meta.SeasonProgress,
	}
	if match := mapTempRe.FindStringSubmatch(body); len(match) >= 2 {
		fields := match[1]
		if value, ok := parseRegexFloat(fields, `seasontemperature=([-0-9.]+)`); ok {
			world.Temperature = value
		}
		world.IsDay = strings.Contains(fields, "daylight=true")
		if season := parseRegexString(fields, `season="([a-zA-Z_]+)"`); season != "" {
			world.Season = season
		}
	}
	return world
}

func parseRegexFloat(text, pattern string) (float64, bool) {
	match := regexp.MustCompile(pattern).FindStringSubmatch(text)
	if len(match) < 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	return value, err == nil
}

func parseRegexString(text, pattern string) string {
	match := regexp.MustCompile(pattern).FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func luaTable(value lua.LValue) *lua.LTable {
	if value == lua.LNil {
		return nil
	}
	table, ok := value.(*lua.LTable)
	if !ok {
		return nil
	}
	return table
}

func luaNumber(table *lua.LTable, key string) float64 {
	if table == nil {
		return 0
	}
	return float64(lua.LVAsNumber(table.RawGetString(key)))
}

func luaString(table *lua.LTable, key string) string {
	if table == nil {
		return ""
	}
	return lua.LVAsString(table.RawGetString(key))
}

func luaBool(table *lua.LTable, key string) bool {
	if table == nil {
		return false
	}
	return lua.LVAsBool(table.RawGetString(key))
}

// parseEntitySections 遍历 656 个 ents_*_fn 段：
//   - 普通实体：扫描顶层 x/z，跳过 data 嵌套；
//   - 虫洞/路牌/玩家遗骸：用沙箱 Lua 读取嵌套字段。
func parseEntitySections(ctx context.Context, sections []saveSection) ([]rawEntity, []MapWormhole, []MapLabel, []MapSkeleton, error) {
	entities := make([]rawEntity, 0, 16000)
	wormholes := make([]MapWormhole, 0)
	labels := make([]MapLabel, 0)
	skeletons := make([]MapSkeleton, 0)

	for _, section := range sections {
		if !strings.HasPrefix(section.Name, "ents_") {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, nil, err
		}

		prefab := strings.TrimPrefix(section.Name, "ents_")
		switch prefab {
		case "wormhole":
			parsed, err := parseWormholes(ctx, section.Body)
			if err != nil {
				logger.Logger.Warnf("地图解析虫洞失败，跳过, err: %v", err)
				continue
			}
			wormholes = append(wormholes, parsed...)
			continue
		case "arrowsign_post", "arrowsign":
			parsed, err := parseLabels(ctx, section.Body)
			if err != nil {
				logger.Logger.Warnf("地图解析路牌失败，跳过, err: %v", err)
				continue
			}
			labels = append(labels, parsed...)
			continue
		case "skeleton_player":
			parsed, err := parseSkeletons(ctx, section.Body)
			if err != nil {
				logger.Logger.Warnf("地图解析玩家遗骸失败，跳过, err: %v", err)
				continue
			}
			skeletons = append(skeletons, parsed...)
			continue
		default:
			remaining := maxMapObjects - len(entities)
			if remaining <= 0 {
				continue
			}
			parsed, dropped := scanEntityBody(section.Body, remaining, prefab)
			entities = append(entities, parsed...)
			if dropped > 0 {
				logger.Logger.Warnf("地图  实体扫描越界/非法，已跳过, prefab: %s, dropped: %d", prefab, dropped)
			}
		}
	}

	if len(wormholes) > maxMapWormholes {
		wormholes = wormholes[:maxMapWormholes]
	}
	if len(labels) > maxMapLabels {
		labels = labels[:maxMapLabels]
	}
	if len(skeletons) > maxMapSkeletons {
		skeletons = skeletons[:maxMapSkeletons]
	}

	return entities, wormholes, labels, skeletons, nil
}

// scanEntityBody 只读取条目层（数组元素）的 x/z，data 里的坐标全部跳过。
func scanEntityBody(body string, limit int, prefab string) ([]rawEntity, int) {
	if limit <= 0 || body == "" {
		return nil, 0
	}

	entities := make([]rawEntity, 0, 64)
	dropped := 0
	depth := 0
	var quote byte
	escaped := false
	var x, z float64
	hasX, hasZ := false, false

	flush := func() {
		if hasX && hasZ {
			if math.IsNaN(x) || math.IsNaN(z) || math.IsInf(x, 0) || math.IsInf(z, 0) ||
				math.Abs(x) > 1e7 || math.Abs(z) > 1e7 {
				dropped++
				return
			}
			if len(entities) < limit {
				entities = append(entities, rawEntity{Prefab: prefab, X: x, Z: z})
			} else {
				dropped++
			}
		}
	}

	for i := 0; i < len(body); i++ {
		c := body[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}

		switch c {
		case '{':
			depth++
			if depth == 2 {
				x, z = 0, 0
				hasX, hasZ = false, false
			}
		case '}':
			if depth == 2 {
				flush()
			}
			if depth > 0 {
				depth--
			}
		default:
			if depth != 2 {
				continue
			}
			if (c != 'x' && c != 'z') || i+1 >= len(body) || body[i+1] != '=' {
				continue
			}
			if !isLuaKeyBoundary(body, i) {
				continue
			}
			value, next, ok := scanLuaNumber(body, i+2)
			if !ok {
				continue
			}
			if c == 'x' {
				x, hasX = value, true
			} else {
				z, hasZ = value, true
			}
			i = next - 1
		}
	}

	return entities, dropped
}

func isLuaKeyBoundary(body string, index int) bool {
	if index <= 0 {
		return true
	}
	prev := body[index-1]
	return !((prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') ||
		(prev >= '0' && prev <= '9') || prev == '_')
}

func scanLuaNumber(body string, start int) (float64, int, bool) {
	i := start
	for i < len(body) && (body[i] == ' ' || body[i] == '\t') {
		i++
	}
	begin := i
	if i < len(body) && (body[i] == '-' || body[i] == '+') {
		i++
	}
	hasDigit := false
	for i < len(body) {
		c := body[i]
		if c >= '0' && c <= '9' {
			hasDigit = true
			i++
			continue
		}
		if c == '.' {
			i++
			continue
		}
		if c == 'e' || c == 'E' {
			i++
			if i < len(body) && (body[i] == '-' || body[i] == '+') {
				i++
			}
			continue
		}
		break
	}
	if !hasDigit {
		return 0, start, false
	}
	value, err := strconv.ParseFloat(body[begin:i], 64)
	if err != nil {
		return 0, start, false
	}
	return value, i, true
}

func withLuaTable(ctx context.Context, body string, fn func(*lua.LTable) error) error {
	L := utils.NewSafeLuaState()
	defer L.Close()
	L.SetContext(ctx)
	if err := L.DoString(body); err != nil {
		return err
	}
	value := L.Get(-1)
	L.Pop(1)
	table, ok := value.(*lua.LTable)
	if !ok {
		return errors.New("lua section is not a table")
	}
	return fn(table)
}

func parseWormholes(ctx context.Context, body string) ([]MapWormhole, error) {
	type wormholeNode struct {
		x, z   float64
		target int
	}
	nodes := make(map[int]wormholeNode)
	ids := make([]int, 0)

	err := withLuaTable(ctx, body, func(table *lua.LTable) error {
		length := table.Len()
		for i := 1; i <= length; i++ {
			entry := luaTable(table.RawGetInt(i))
			if entry == nil {
				continue
			}
			id := int(luaNumber(entry, "id"))
			if id == 0 {
				continue
			}
			node := wormholeNode{
				x: luaNumber(entry, "x"),
				z: luaNumber(entry, "z"),
			}
			if !validMapCoord(node.x) || !validMapCoord(node.z) {
				continue
			}
			data := luaTable(entry.RawGetString("data"))
			if data == nil {
				continue
			}
			teleporter := luaTable(data.RawGetString("teleporter"))
			node.target = int(luaNumber(teleporter, "target"))
			nodes[id] = node
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Ints(ids)
	seen := make(map[[2]int]bool)
	result := make([]MapWormhole, 0)
	for _, id := range ids {
		node := nodes[id]
		if node.target == 0 || node.target == id {
			continue
		}
		target, ok := nodes[node.target]
		if !ok {
			continue
		}
		if !validMapCoord(target.x) || !validMapCoord(target.z) {
			continue
		}
		left, right := id, node.target
		if left > right {
			left, right = right, left
		}
		key := [2]int{left, right}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, MapWormhole{
			X1: node.x, Z1: node.z,
			X2: target.x, Z2: target.z,
		})
	}
	return result, nil
}

func parseLabels(ctx context.Context, body string) ([]MapLabel, error) {
	result := make([]MapLabel, 0)
	err := withLuaTable(ctx, body, func(table *lua.LTable) error {
		length := table.Len()
		for i := 1; i <= length; i++ {
			entry := luaTable(table.RawGetInt(i))
			if entry == nil {
				continue
			}
			data := luaTable(entry.RawGetString("data"))
			if data == nil {
				continue
			}
			writeable := luaTable(data.RawGetString("writeable"))
			text := strings.TrimSpace(luaString(writeable, "text"))
			if text == "" || !utf8.ValidString(text) {
				continue
			}
			text = truncateRunes(text, maxMapLabelRunes)
			x := luaNumber(entry, "x")
			z := luaNumber(entry, "z")
			if !validMapCoord(x) || !validMapCoord(z) {
				continue
			}
			result = append(result, MapLabel{
				Prefab: "arrowsign_post",
				Text:   text,
				X:      x,
				Z:      z,
			})
			if len(result) >= maxMapLabels {
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func parseSkeletons(ctx context.Context, body string) ([]MapSkeleton, error) {
	result := make([]MapSkeleton, 0)
	err := withLuaTable(ctx, body, func(table *lua.LTable) error {
		length := table.Len()
		for i := 1; i <= length; i++ {
			entry := luaTable(table.RawGetInt(i))
			if entry == nil {
				continue
			}
			data := luaTable(entry.RawGetString("data"))
			if data == nil {
				continue
			}
			avatar := luaTable(data.RawGetString("playeravatardata"))

			name := luaString(avatar, "name")
			if name == "" {
				name = luaString(data, "playername")
			}
			if !utf8.ValidString(name) {
				name = ""
			}
			name = truncateRunes(name, maxMapSkeletonNameRunes)

			uid := luaString(data, "userid")
			if !validPlayerUID(uid) {
				uid = ""
			}

			equip := make(map[string]string)
			var equipTable *lua.LTable
			if avatar != nil {
				equipTable = luaTable(avatar.RawGetString("equip"))
			}
			if equipTable == nil {
				equipTable = luaTable(data.RawGetString("equip"))
			}
			if equipTable != nil {
				equipTable.ForEach(func(key, value lua.LValue) {
					if len(equip) >= maxMapSkeletonEquipSlots {
						return
					}
					slot := lua.LVAsString(key)
					item := lua.LVAsString(value)
					if item == "" || !isSafeMapToken(slot) || !isSafeMapToken(item) {
						return
					}
					equip[slot] = item
				})
			}

			x := luaNumber(entry, "x")
			z := luaNumber(entry, "z")
			if !validMapCoord(x) || !validMapCoord(z) {
				continue
			}
			age := luaNumber(data, "age")
			if math.IsNaN(age) || math.IsInf(age, 0) || age < 0 || age > 1e9 {
				age = 0
			}

			skeleton := MapSkeleton{
				X:          x,
				Z:          z,
				Name:       name,
				Char:       safeMapTokenOrEmpty(luaString(data, "char")),
				Cause:      safeMapTokenOrEmpty(luaString(data, "cause")),
				UID:        uid,
				AgeSeconds: age,
			}
			if len(equip) > 0 {
				skeleton.Equip = equip
			}
			result = append(result, skeleton)
			if len(result) >= maxMapSkeletons {
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type mapPrefabInfo struct {
	Layer string
	Icon  string
	EN    string
	ZH    string
}

// mapCuratedPrefabs 是人工维护的高价值 prefab：地标/Boss、常见资源与建筑。
// 未收录的 prefab 由下面的命名规则兜底，不会阻塞地图生成。
var mapCuratedPrefabs = map[string]mapPrefabInfo{
	"pigking":                     {Layer: "landmark", ZH: "猪王", EN: "Pig King"},
	"multiplayer_portal":          {Layer: "landmark", ZH: "出生门", EN: "Portal"},
	"multiplayer_portal_moonrock": {Layer: "landmark", ZH: "月石传送门", EN: "Moonrock Portal"},
	"moonbase":                    {Layer: "landmark", ZH: "月台", EN: "Moon Base"},
	"oasislake":                   {Layer: "landmark", ZH: "绿洲", EN: "Oasis"},
	"antlion":                     {Layer: "landmark", ZH: "蚁狮", EN: "Antlion"},
	"beequeenhivegrown":           {Layer: "landmark", ZH: "巨大蜂窝", EN: "Gigantic Beehive"},
	"dragonfly":                   {Layer: "landmark", ZH: "龙蝇", EN: "Dragonfly"},
	"beequeen":                    {Layer: "landmark", ZH: "蜂后", EN: "Bee Queen"},
	"spiderqueen":                 {Layer: "landmark", ZH: "蜘蛛女王", EN: "Spider Queen"},
	"hermithouse":                 {Layer: "landmark", ZH: "奶奶岛", EN: "Hermit House"},
	"crabking":                    {Layer: "landmark", ZH: "帝王蟹", EN: "Crab King"},
	"malbatross":                  {Layer: "landmark", ZH: "邪天翁", EN: "Malbatross"},
	"toadstool":                   {Layer: "landmark", ZH: "蟾蜍王", EN: "Toadstool"},
	"klaus":                       {Layer: "landmark", ZH: "克劳斯", EN: "Klaus"},
	"bearger":                     {Layer: "landmark", ZH: "熊獾", EN: "Bearger"},
	"deerclops":                   {Layer: "landmark", ZH: "巨鹿", EN: "Deerclops"},
	"moose":                       {Layer: "landmark", ZH: "麋鹿", EN: "Moose"},
	"guardian":                    {Layer: "landmark", ZH: "远古守护者", EN: "Ancient Guardian"},
	"fuelweaver":                  {Layer: "landmark", ZH: "远古织影者", EN: "Fuelweaver"},
	"celestialchampion":           {Layer: "landmark", ZH: "天体英雄", EN: "Celestial Champion"},
	"ruins_statue_mage":           {Layer: "landmark", ZH: "远古雕像", EN: "Ancient Statue"},
	"archive_moon_statue":         {Layer: "landmark", ZH: "远古月亮雕像", EN: "Archive Moon Statue"},
	"cave_entrance_open":          {Layer: "landmark", ZH: "洞穴入口", EN: "Cave Entrance"},
	"lunarrift_portal":            {Layer: "landmark", ZH: "月亮裂隙", EN: "Lunar Rift"},
	"shadowrift":                  {Layer: "landmark", ZH: "暗影裂隙", EN: "Shadow Rift"},
	"carnival_host":               {Layer: "landmark", ZH: "乌鸦嘉年华", EN: "Carnival Host"},
	"yot_catcoonshrine":           {Layer: "landmark", ZH: "浣猫神龛", EN: "Catcoon Shrine"},
	"monkeyqueen":                 {Layer: "landmark", ZH: "猴王", EN: "Monkey Queen"},
	"monkeyisland_portal":         {Layer: "landmark", ZH: "猴岛传送门", EN: "Monkey Island Portal"},
	"terrarium":                   {Layer: "landmark", ZH: "盒中泰拉", EN: "Terrarium"},
	"charlie_lecturn":             {Layer: "landmark", ZH: "查理讲台", EN: "Charlie Lectern"},

	"evergreen":        {Layer: "resource", ZH: "常青树", EN: "Evergreen"},
	"evergreen_sparse": {Layer: "resource", ZH: "稀疏常青树", EN: "Sparse Evergreen"},
	"deciduoustree":    {Layer: "resource", ZH: "落叶树", EN: "Deciduous Tree"},
	"twiggytree":       {Layer: "resource", ZH: "白桦树", EN: "Birchnut Tree"},
	"marsh_tree":       {Layer: "resource", ZH: "沼泽树", EN: "Marsh Tree"},
	"sapling":          {Layer: "resource", ZH: "树苗", EN: "Sapling"},
	"grass":            {Layer: "resource", ZH: "草", EN: "Grass"},
	"berrybush":        {Layer: "resource", ZH: "浆果丛", EN: "Berry Bush"},
	"reeds":            {Layer: "resource", ZH: "芦苇", EN: "Reeds"},
	"cactus":           {Layer: "resource", ZH: "仙人掌", EN: "Cactus"},
	"flower":           {Layer: "resource", ZH: "花", EN: "Flower"},
	"flower_evil":      {Layer: "resource", ZH: "恶魔花", EN: "Evil Flower"},
	"mushroom":         {Layer: "resource", ZH: "蘑菇", EN: "Mushroom"},
	"blue_mushroom":    {Layer: "resource", ZH: "蓝蘑菇", EN: "Blue Mushroom"},
	"green_mushroom":   {Layer: "resource", ZH: "绿蘑菇", EN: "Green Mushroom"},
	"red_mushroom":     {Layer: "resource", ZH: "红蘑菇", EN: "Red Mushroom"},
	"rock1":            {Layer: "resource", ZH: "石头", EN: "Rock"},
	"rock2":            {Layer: "resource", ZH: "石头", EN: "Rock"},
	"rock_flintless":   {Layer: "resource", ZH: "无燧石矿", EN: "Flintless Rock"},
	"flint":            {Layer: "resource", ZH: "燧石", EN: "Flint"},
	"goldnugget":       {Layer: "resource", ZH: "金块", EN: "Gold Nugget"},
	"nitre":            {Layer: "resource", ZH: "硝石", EN: "Nitre"},
	"marble":           {Layer: "resource", ZH: "大理石", EN: "Marble"},
	"moonrocknugget":   {Layer: "resource", ZH: "月石", EN: "Moon Rock"},
	"kelp":             {Layer: "resource", ZH: "海带", EN: "Kelp"},
	"bullkelp_plant":   {Layer: "resource", ZH: "公牛海带", EN: "Bull Kelp"},
	"coral":            {Layer: "resource", ZH: "珊瑚", EN: "Coral"},
	"pond":             {Layer: "resource", ZH: "池塘", EN: "Pond"},
	"pond_mos":         {Layer: "resource", ZH: "沼泽池塘", EN: "Marsh Pond"},

	"spiderden":    {Layer: "nest", ZH: "蜘蛛巢", EN: "Spider Den"},
	"spiderden_2":  {Layer: "nest", ZH: "蜘蛛巢", EN: "Spider Den"},
	"spiderden_3":  {Layer: "nest", ZH: "蜘蛛巢", EN: "Spider Den"},
	"wasphive":     {Layer: "nest", ZH: "杀人蜂巢", EN: "Wasp Hive"},
	"beehive":      {Layer: "nest", ZH: "蜂巢", EN: "Beehive"},
	"beebox":       {Layer: "nest", ZH: "蜂箱", EN: "Bee Box"},
	"houndmound":   {Layer: "nest", ZH: "猎犬巢", EN: "Hound Mound"},
	"mound":        {Layer: "nest", ZH: "土堆", EN: "Mound"},
	"rabbithole":   {Layer: "nest", ZH: "兔子洞", EN: "Rabbit Hole"},
	"catcoonden":   {Layer: "nest", ZH: "浣猫窝", EN: "Catcoon Den"},
	"tallbirdnest": {Layer: "nest", ZH: "高鸟巢", EN: "Tallbird Nest"},
	"walrus_camp":  {Layer: "nest", ZH: "海象巢穴", EN: "Walrus Camp"},
	"mermhouse":    {Layer: "nest", ZH: "鱼人屋", EN: "Merm House"},

	"treasurechest":  {Layer: "structure", ZH: "箱子", EN: "Chest"},
	"icebox":         {Layer: "structure", ZH: "冰箱", EN: "Ice Box"},
	"cookpot":        {Layer: "structure", ZH: "烹饪锅", EN: "Crock Pot"},
	"firepit":        {Layer: "structure", ZH: "石头火堆", EN: "Fire Pit"},
	"coldfirepit":    {Layer: "structure", ZH: "吸热火堆", EN: "Endothermic Fire Pit"},
	"tent":           {Layer: "structure", ZH: "帐篷", EN: "Tent"},
	"sciencemachine": {Layer: "structure", ZH: "科学机器", EN: "Science Machine"},
	"alchemymachine": {Layer: "structure", ZH: "炼金引擎", EN: "Alchemy Engine"},
	"birdcage":       {Layer: "structure", ZH: "鸟笼", EN: "Bird Cage"},
	"lightning_rod":  {Layer: "structure", ZH: "避雷针", EN: "Lightning Rod"},
	"wall_stone":     {Layer: "structure", ZH: "石墙", EN: "Stone Wall"},
	"wall_wood":      {Layer: "structure", ZH: "木墙", EN: "Wood Wall"},
	"fence":          {Layer: "structure", ZH: "栅栏", EN: "Fence"},
	"sign":           {Layer: "structure", ZH: "路牌", EN: "Sign"},
	"arrowsign_post": {Layer: "structure", ZH: "方向路牌", EN: "Directional Sign"},
	"boat":           {Layer: "structure", ZH: "船", EN: "Boat"},
	"dock_woodposts": {Layer: "structure", ZH: "码头", EN: "Dock"},
}

// mapItemPrefabs 是掉落物/物品类 prefab；数量多，不逐个写中文名。
var mapItemPrefabs = map[string]bool{
	"log": true, "twigs": true, "cutgrass": true, "rocks": true, "flint": true,
	"goldnugget": true, "nitre": true, "charcoal": true, "boards": true,
	"cutstone": true, "rope": true, "silk": true, "papyrus": true, "gears": true,
	"marble": true, "houndstooth": true, "stinger": true, "horn": true,
	"spoiled_food": true, "seeds": true, "carrot": true, "berries": true,
	"meat": true, "smallmeat": true, "fish": true, "feather_crow": true,
	"feather_robin": true, "feather_robin_winter": true, "tumbleweed": true,
	"axe": true, "pickaxe": true, "shovel": true, "hammer": true,
	"torch": true, "lantern": true, "umbrella": true, "backpack": true,
}

func mapPrefabMeta(name string) mapPrefabInfo {
	lower := strings.ToLower(name)
	if info, ok := mapCuratedPrefabs[lower]; ok {
		if info.Icon == "" && isSafeMapToken(lower) {
			info.Icon = lower
		}
		return info
	}

	info := mapPrefabInfo{Layer: "other"}
	if isSafeMapToken(lower) {
		info.Icon = lower
	}

	switch {
	case mapItemPrefabs[lower] || strings.HasSuffix(lower, "_item") ||
		strings.HasPrefix(lower, "winter_ornament") || strings.HasPrefix(lower, "trinket_") ||
		strings.HasPrefix(lower, "bluegem") || strings.HasPrefix(lower, "redgem") ||
		strings.HasPrefix(lower, "purplegem") || strings.HasPrefix(lower, "yellowgem") ||
		strings.HasPrefix(lower, "orangegem") || strings.HasPrefix(lower, "greengem"):
		info.Layer = "item"
	case strings.HasPrefix(lower, "turf_"):
		info.Layer = "structure"
	case containsAny(lower, "spiderden", "hive", "mound", "den", "nest", "beebox"):
		info.Layer = "nest"
	case containsAny(lower, "wall", "chest", "farm", "cookpot", "firepit", "tent",
		"machine", "cage", "rod", "icebox", "box", "sign", "door", "gate",
		"table", "station", "fence"):
		info.Layer = "structure"
	case containsAny(lower, "tree", "rock", "bush", "grass", "sapling", "mushroom",
		"reeds", "flower", "kelp", "cactus", "berry", "twig", "flint", "boulder",
		"salt", "coral", "shell", "vine", "fern", "lichen", "plant"):
		info.Layer = "resource"
	}

	if !validMapLayer(info.Layer) {
		info.Layer = "other"
	}
	return info
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func validMapLayer(layer string) bool {
	switch layer {
	case "landmark", "structure", "nest", "resource", "other", "item":
		return true
	default:
		return false
	}
}

func isSafeMapToken(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' {
			continue
		}
		return false
	}
	return true
}

func isSafeMapTileName(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '_' {
			continue
		}
		return false
	}
	return true
}

func safeMapTokenOrEmpty(value string) string {
	if isSafeMapToken(value) {
		return value
	}
	return ""
}

var mapPlayerUIDRe = regexp.MustCompile(`^KU_[A-Za-z0-9_-]{1,125}$`)

func validPlayerUID(uid string) bool {
	return mapPlayerUIDRe.MatchString(uid)
}

// validMapCoord 过滤 NaN/Inf 和明显越界的世界坐标。
func validMapCoord(value float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	return math.Abs(value) <= 1e7
}

func truncateRunes(value string, max int) string {
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}

// PlayerPositions 在本次请求内实时探测指定世界的在线玩家。
// 世界未运行时不发命令，返回 worldRunning=false；探测失败只写 Error，不影响地图。
func (g *Game) PlayerPositions(ctx context.Context, worldID int) MapPlayers {
	result := MapPlayers{
		ProbedAt: time.Now().UTC(),
		List:     []MapPlayer{},
	}

	world, err := g.getWorldByID(worldID)
	if err != nil {
		result.Error = "world not found"
		return result
	}
	if !g.WorldUpStatus(worldID) {
		return result
	}
	result.WorldRunning = true

	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	startMarker := "===DMPP-" + nonce + "==="
	endMarker := "===DMPP-END-" + nonce + "==="

	// DST 控制台逐行执行，整段 Lua 必须压成一行；玩家名只作为 format 参数，不参与源码拼接。
	command := "print('" + startMarker + "') local ps=AllPlayers for i=1,#ps do " +
		"local p=ps[i] local x,y,z=p.Transform:GetWorldPosition() " +
		"local name=tostring(p.name or ''):gsub('%c',' ') " +
		"print(string.format('DMPP|%.2f|%.2f|%s|%s|%s',x,z,p.prefab or '',p.userid or '',name)) " +
		"end print('" + endMarker + "')"

	if err := utils.ScreenCMD(command, world.screenName); err != nil {
		result.Error = "send command failed: " + err.Error()
		logger.Logger.Warnf("地图玩家探测命令发送失败, roomID: %d, worldID: %d, err: %v", g.room.ID, worldID, err)
		return result
	}

	logPath := fmt.Sprintf("%s/server_log.txt", world.worldPath)
	ticker := time.NewTicker(time.Duration(mapPlayersPollMillis) * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(time.Duration(mapPlayersWaitMillis) * time.Millisecond)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			result.Error = "context canceled"
			return result
		case <-timeout.C:
			result.Error = "player probe timeout"
			logger.Logger.Warnf("地图玩家探测超时, roomID: %d, worldID: %d", g.room.ID, worldID)
			return result
		case <-ticker.C:
			lines := utils.GetFileLastNLines(logPath, mapLogTailLines)
			if players, ok := parsePlayerLines(lines, startMarker, endMarker); ok {
				result.List = players
				result.ProbedAt = time.Now().UTC()
				return result
			}
		}
	}
}

func parsePlayerLines(lines []string, startMarker, endMarker string) ([]MapPlayer, bool) {
	players := make([]MapPlayer, 0)
	inBlock := false
	foundEnd := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasSuffix(trimmed, startMarker) {
				inBlock = true
			}
			continue
		}
		if strings.HasSuffix(trimmed, endMarker) {
			foundEnd = true
			break
		}
		player, ok := parsePlayerLine(line)
		if !ok {
			continue
		}
		players = append(players, player)
		if len(players) >= maxMapPlayers {
			break
		}
	}

	if !foundEnd {
		return nil, false
	}
	return players, true
}

func parsePlayerLine(line string) (MapPlayer, bool) {
	index := strings.Index(line, "DMPP|")
	if index < 0 {
		return MapPlayer{}, false
	}
	parts := strings.SplitN(line[index+len("DMPP|"):], "|", 5)
	if len(parts) != 5 {
		return MapPlayer{}, false
	}

	x, errX := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	z, errZ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if errX != nil || errZ != nil ||
		x < -1e5 || x > 1e5 || z < -1e5 || z > 1e5 {
		return MapPlayer{}, false
	}

	uid := strings.TrimSpace(parts[3])
	if !validPlayerUID(uid) {
		return MapPlayer{}, false
	}
	prefab := safeMapTokenOrEmpty(strings.TrimSpace(parts[2]))
	name := strings.TrimSpace(parts[4])
	if !utf8.ValidString(name) {
		name = ""
	}
	name = truncateRunes(name, maxMapPlayerNameRunes)

	return MapPlayer{
		UID:    uid,
		Name:   name,
		Prefab: prefab,
		X:      x,
		Z:      z,
	}, true
}
