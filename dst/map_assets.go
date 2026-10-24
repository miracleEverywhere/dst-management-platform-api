package dst

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/tex"
	"dst-management-platform-api/utils"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"
)

const (
	maxMapAssetBytes     = 2 << 20
	maxMapAssetNameRunes = 128
	// MapTileTextureSide 地形噪声贴图的输出边长；96px + BestCompression 约 20KB/张。
	MapTileTextureSide = 96
)

type MapAssetData struct {
	Name        string
	Kind        string
	ImageBase64 string
	SHA256      string
	Width       int
	Height      int
}

// BuildMapIconAtlas 把 minimap 图标打包成一张 PNG 图集，rects 的 key 与 prefab.icon 一致（*.png）。
// 前端 WorldMapCanvas 直接按 rect 从 atlas 里裁图，不再逐个请求图片。
func BuildMapIconAtlas(images map[string]MapAssetData) MapIconAtlas {
	atlas := MapIconAtlas{Rects: map[string][]int{}}
	names := make([]string, 0, len(images))
	for name, asset := range images {
		if name != "" && asset.ImageBase64 != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return atlas
	}
	sort.Strings(names)
	if len(names) > maxMapAssetRefs {
		names = names[:maxMapAssetRefs]
	}

	const cell = 64
	const columns = 32
	rows := (len(names) + columns - 1) / columns
	canvas := image.NewNRGBA(image.Rect(0, 0, columns*cell, rows*cell))
	for index, name := range names {
		data, err := base64.StdEncoding.DecodeString(images[name].ImageBase64)
		if err != nil {
			continue
		}
		picture, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		x := (index % columns) * cell
		y := (index / columns) * cell
		if picture.Bounds().Dx() == cell && picture.Bounds().Dy() == cell {
			draw.Draw(canvas, image.Rect(x, y, x+cell, y+cell), picture, picture.Bounds().Min, draw.Over)
		} else {
			draw.Draw(canvas, image.Rect(x, y, x+cell, y+cell), resizeIcon(picture, cell), image.Point{}, draw.Over)
		}
		atlas.Rects[name] = []int{x, y, cell, cell}
	}

	var buffer bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buffer, canvas); err != nil {
		return MapIconAtlas{Rects: map[string][]int{}}
	}
	atlas.Width = canvas.Bounds().Dx()
	atlas.Height = canvas.Bounds().Dy()
	atlas.PNG = base64.StdEncoding.EncodeToString(buffer.Bytes())
	return atlas
}

func resizeIcon(source image.Image, size int) *image.NRGBA {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	target := image.NewNRGBA(image.Rect(0, 0, size, size))
	if width <= 0 || height <= 0 {
		return target
	}
	for y := 0; y < size; y++ {
		sourceY := bounds.Min.Y + y*height/size
		for x := 0; x < size; x++ {
			sourceX := bounds.Min.X + x*width/size
			target.Set(x, y, source.At(sourceX, sourceY))
		}
	}
	return target
}

type mapTileDef struct {
	Noise string
	Tint  string
}

var mapTileDefsCache struct {
	sync.Mutex
	loaded  bool
	modTime time.Time
	size    int64
	defs    map[string]mapTileDef
}

// loadMapTileDefs 从 scripts.zip 的 tiledefs.lua 解析 tile -> 噪声贴图/minimap 颜色。
// 解析失败只记日志并返回空表，地图仍可用纯色兜底；scripts.zip 变化时自动重新解析。
func loadMapTileDefs() map[string]mapTileDef {
	path := mapScriptsZipPath()
	if _, err := os.Stat(mapScriptsLuaPath()); err == nil {
		path = mapScriptsLuaPath()
	}
	info, statErr := os.Stat(path)

	mapTileDefsCache.Lock()
	defer mapTileDefsCache.Unlock()

	if statErr == nil && mapTileDefsCache.loaded &&
		mapTileDefsCache.modTime.Equal(info.ModTime()) && mapTileDefsCache.size == info.Size() {
		return mapTileDefsCache.defs
	}
	if statErr != nil && mapTileDefsCache.loaded {
		return mapTileDefsCache.defs
	}

	defs := readMapTileDefs()
	mapTileDefsCache.defs = defs
	mapTileDefsCache.loaded = true
	if statErr == nil {
		mapTileDefsCache.modTime = info.ModTime()
		mapTileDefsCache.size = info.Size()
	} else {
		mapTileDefsCache.modTime = time.Time{}
		mapTileDefsCache.size = 0
	}
	return defs
}

func readMapTileDefs() map[string]mapTileDef {
	for _, path := range mapScriptsLuaPaths() {
		if content, err := os.ReadFile(path); err == nil {
			return parseMapTileDefs(string(content))
		}
	}

	zipPath := mapScriptsZipPath()
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		logger.Logger.Warnf("地图读取 scripts.zip / scripts 目录失败，跳过 tiledefs, err: %v", err)
		return map[string]mapTileDef{}
	}
	defer reader.Close()

	for _, file := range reader.File {
		name := strings.ToLower(filepath.ToSlash(file.Name))
		if name != "scripts/tiledefs.lua" {
			continue
		}
		content, err := readZipEntry(file, 8<<20)
		if err != nil {
			logger.Logger.Warnf("地图读取 tiledefs.lua 失败, err: %v", err)
			return map[string]mapTileDef{}
		}
		return parseMapTileDefs(string(content))
	}

	logger.Logger.Warn("地图 scripts.zip 中没有 scripts/tiledefs.lua")
	return map[string]mapTileDef{}
}

func parseMapTileDefs(script string) map[string]mapTileDef {
	defs := make(map[string]mapTileDef)

	L := utils.NewSafeLuaState()
	defer L.Close()

	tileManager := L.NewTable()
	L.SetField(tileManager, "RegisterTileRange", L.NewFunction(func(*lua.LState) int { return 0 }))
	L.SetField(tileManager, "AddTile", L.NewFunction(func(L *lua.LState) int {
		name := lua.LVAsString(L.Get(1))
		if name == "" {
			return 0
		}
		info := defs[name]
		top := L.GetTop()
		for i := 3; i <= top; i++ {
			style, ok := L.Get(i).(*lua.LTable)
			if !ok {
				continue
			}
			if info.Noise == "" {
				info.Noise = lua.LVAsString(style.RawGetString("noise_texture"))
			}
			if info.Tint == "" {
				if colors, ok := style.RawGetString("colors").(*lua.LTable); ok {
					if minimap, ok := colors.RawGetString("minimap_color").(*lua.LTable); ok {
						info.Tint = luaColorToHex(minimap)
					}
				}
			}
		}
		defs[name] = info
		return 0
	}))
	L.SetGlobal("TileManager", tileManager)
	L.SetGlobal("require", L.NewFunction(func(L *lua.LState) int {
		L.Push(tileManager)
		return 1
	}))
	L.SetGlobal("GROUND", L.NewTable())

	if err := L.DoString(script); err != nil {
		logger.Logger.Warnf("地图解析 tiledefs.lua 失败, err: %v", err)
	}
	return defs
}

func luaColorToHex(table *lua.LTable) string {
	channels := [4]int{}
	for i := 0; i < 4; i++ {
		value := int(lua.LVAsNumber(table.RawGetInt(i + 1)))
		if value < 0 {
			value = 0
		}
		if value > 255 {
			value = 255
		}
		channels[i] = value
	}
	return fmt.Sprintf("%02x%02x%02x%02x", channels[0], channels[1], channels[2], channels[3])
}

type mapAssetEntry struct {
	xmlName  string
	texName  string
	element  string
	priority int
}

var mapAssetIndexCache struct {
	sync.Mutex
	loaded    bool
	fromDir   bool
	signature string
	entries   map[string]mapAssetEntry
}

// loadMapAssetIndex 缓存图集元素名称索引，避免每次按名字扫全部 XML。
// 优先扫描已解压的 data/images 和 data/minimap，其次回退到 databundles/images.zip。
func loadMapAssetIndex() (map[string]mapAssetEntry, bool) {
	dirs := mapAssetDirs()
	if len(dirs) > 0 {
		signature := mapAssetDirsSignature(dirs)
		mapAssetIndexCache.Lock()
		defer mapAssetIndexCache.Unlock()
		if mapAssetIndexCache.loaded && mapAssetIndexCache.fromDir &&
			mapAssetIndexCache.signature == signature {
			return mapAssetIndexCache.entries, true
		}
		entries := make(map[string]mapAssetEntry)
		for _, dir := range dirs {
			readAtlasIndexFromDir(dir, entries)
		}
		mapAssetIndexCache.loaded = true
		mapAssetIndexCache.fromDir = true
		mapAssetIndexCache.signature = signature
		mapAssetIndexCache.entries = entries
		return entries, true
	}

	zipPath := mapImagesZipPath()
	info, statErr := os.Stat(zipPath)

	mapAssetIndexCache.Lock()
	defer mapAssetIndexCache.Unlock()

	if statErr == nil && mapAssetIndexCache.loaded && !mapAssetIndexCache.fromDir &&
		mapAssetIndexCache.signature == zipSignature(info) {
		return mapAssetIndexCache.entries, false
	}
	if statErr != nil && mapAssetIndexCache.loaded && !mapAssetIndexCache.fromDir {
		return mapAssetIndexCache.entries, false
	}

	entries := make(map[string]mapAssetEntry)
	if statErr == nil {
		reader, err := zip.OpenReader(zipPath)
		if err != nil {
			logger.Logger.Warnf("地图读取 images.zip 失败, err: %v", err)
		} else {
			for _, file := range reader.File {
				name := strings.ToLower(filepath.ToSlash(file.Name))
				if !strings.HasPrefix(name, "images/") || !strings.HasSuffix(name, ".xml") {
					continue
				}
				content, err := readZipEntry(file, 8<<20)
				if err != nil {
					continue
				}
				var doc mapAtlasXML
				if xml.Unmarshal(content, &doc) != nil {
					continue
				}
				texName := strings.TrimSpace(doc.Texture.Filename)
				if texName == "" {
					continue
				}
				indexAtlasElements(entries, filepath.ToSlash(file.Name), texName, &doc)
			}
			_ = reader.Close()
		}
	}

	mapAssetIndexCache.loaded = true
	mapAssetIndexCache.fromDir = false
	mapAssetIndexCache.entries = entries
	if statErr == nil {
		mapAssetIndexCache.signature = zipSignature(info)
	} else {
		mapAssetIndexCache.signature = ""
	}
	return entries, false
}

func zipSignature(info os.FileInfo) string {
	return fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
}

func mapAssetDirsSignature(dirs []string) string {
	var builder strings.Builder
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			continue
		}
		fmt.Fprintf(&builder, "%s:%d:%d;", dir, info.ModTime().UnixNano(), info.Size())
	}
	return builder.String()
}

func readAtlasIndexFromDir(dir string, entries map[string]mapAssetEntry) {
	files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return
	}
	sort.Strings(files)
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc mapAtlasXML
		if xml.Unmarshal(content, &doc) != nil {
			continue
		}
		texName := strings.TrimSpace(doc.Texture.Filename)
		if texName == "" {
			continue
		}
		indexAtlasElements(entries, path, texName, &doc)
	}
}

func indexAtlasElements(entries map[string]mapAssetEntry, xmlName, texName string, doc *mapAtlasXML) {
	priority := atlasPriority(xmlName)
	for _, element := range doc.Elements.Elements {
		base := strings.TrimSuffix(element.Name, ".tex")
		key := strings.ToLower(base)
		if key == "" {
			continue
		}
		if existing, exists := entries[key]; exists && existing.priority <= priority {
			continue
		}
		entries[key] = mapAssetEntry{
			xmlName:  xmlName,
			texName:  texName,
			element:  element.Name,
			priority: priority,
		}
	}
}

// atlasPriority 让 minimap / 地形噪声图集优先于 inventoryimages 等大图集，
// 避免 prefab 同名图标误取物品栏图片。
func atlasPriority(xmlName string) int {
	name := strings.ToLower(filepath.Base(xmlName))
	switch {
	case strings.Contains(name, "minimap"):
		return 0
	case strings.Contains(name, "noise"):
		return 0
	case strings.Contains(name, "inventoryimages"):
		return 5
	case strings.Contains(name, "playerportrait"):
		return 4
	default:
		return 2
	}
}

// ExtractMapAssets 用内置 TEX 解包从 images.zip 提取指定图标/地形贴图。
// 只返回成功找到的资源；找不到的 name 不出现在结果里，由调用方决定是否 404。
func ExtractMapAssets(ctx context.Context, gameBuild, kind string, names []string) map[string]MapAssetData {
	result := make(map[string]MapAssetData)
	if kind != "icon" && kind != "tile" {
		return result
	}

	needed := make(map[string]string, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || len([]rune(name)) > maxMapAssetNameRunes {
			continue
		}
		needed[strings.ToLower(strings.TrimSuffix(name, ".tex"))] = name
	}
	if len(needed) == 0 {
		return result
	}

	index, fromDir := loadMapAssetIndex()
	if len(index) == 0 && !fromDir {
		return result
	}

	type atlasTask struct {
		xmlName   string
		texName   string
		requested map[string]string // lower element -> requested original name
	}
	tasks := make(map[string]*atlasTask)
	for lower, original := range needed {
		candidates := []string{lower}
		if kind == "icon" {
			base := strings.TrimSuffix(lower, ".png")
			if base != lower {
				candidates = append(candidates, base)
			}
			// 少数图集把 minimap 图标命名为 minimap_icon_<prefab> 等形式。
			candidates = append(candidates, "minimap_icon_"+base, "minimap_"+base, base+"_icon")
		}

		var entry mapAssetEntry
		var elementKey string
		found := false
		for _, candidate := range candidates {
			if item, ok := index[candidate]; ok {
				entry = item
				elementKey = candidate
				found = true
				break
			}
		}
		if !found {
			continue
		}
		// 地图图标不使用 inventoryimages 的物品栏图片，避免拿到同名物品图标。
		if kind == "icon" && entry.priority >= 5 {
			continue
		}

		task := tasks[entry.xmlName]
		if task == nil {
			task = &atlasTask{
				xmlName:   entry.xmlName,
				texName:   entry.texName,
				requested: make(map[string]string),
			}
			tasks[entry.xmlName] = task
		}
		task.requested[elementKey] = original
	}
	if len(tasks) == 0 {
		if fromDir {
			appendStandaloneAssets(needed, kind, result)
		}
		return result
	}

	taskNames := make([]string, 0, len(tasks))
	for name := range tasks {
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)

	// 已解压的 dst/data/images：直接以游戏目录里的 XML/TEX 为输入，PNG 输出到临时目录。
	if fromDir {
		for _, xmlName := range taskNames {
			if err := ctx.Err(); err != nil {
				break
			}
			task := tasks[xmlName]
			xmlPath := task.xmlName
			if filepath.Base(task.texName) != task.texName {
				continue
			}
			content, err := os.ReadFile(xmlPath)
			if err != nil {
				continue
			}
			var doc mapAtlasXML
			if xml.Unmarshal(content, &doc) != nil {
				continue
			}
			texPath := filepath.Join(filepath.Dir(xmlPath), task.texName)
			if _, err := os.Stat(texPath); err != nil {
				continue
			}

			tempDir, err := os.MkdirTemp("", "dmp-map-asset-")
			if err != nil {
				continue
			}
			outputDir, _, unpackErr := tex.Unpack(xmlPath, tempDir)
			if unpackErr != nil {
				logger.Logger.Warnf("地图解包图集失败, atlas: %s, err: %v", xmlName, unpackErr)
				_ = os.RemoveAll(tempDir)
				continue
			}
			appendAtlasAssets(kind, outputDir, &doc, task.requested, result)
			_ = os.RemoveAll(tempDir)
		}

		// 部分地形噪声/图标是独立的 .tex（没有 atlas XML），直接按文件名转换。
		appendStandaloneAssets(needed, kind, result)
		return result
	}

	zipPath := mapImagesZipPath()
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		logger.Logger.Warnf("地图读取 images.zip 失败, err: %v", err)
		return result
	}
	defer reader.Close()

	entries := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		entries[strings.ToLower(filepath.ToSlash(file.Name))] = file
	}

	for _, xmlName := range taskNames {
		if err := ctx.Err(); err != nil {
			break
		}
		task := tasks[xmlName]
		xmlFile := entries[strings.ToLower(xmlName)]
		texEntry := entries[strings.ToLower(pathpkg.Join(pathpkg.Dir(xmlName), task.texName))]
		if xmlFile == nil || texEntry == nil {
			continue
		}

		content, err := readZipEntry(xmlFile, 8<<20)
		if err != nil {
			continue
		}
		var doc mapAtlasXML
		if xml.Unmarshal(content, &doc) != nil {
			continue
		}

		tempDir, err := os.MkdirTemp("", "dmp-map-asset-")
		if err != nil {
			continue
		}
		imageDir := filepath.Join(tempDir, "images")
		if err := os.MkdirAll(imageDir, 0o755); err != nil {
			_ = os.RemoveAll(tempDir)
			continue
		}
		xmlPath := filepath.Join(imageDir, filepath.Base(xmlName))
		texPath := filepath.Join(imageDir, filepath.Base(task.texName))
		if err := writeZipEntry(xmlFile, xmlPath); err != nil {
			_ = os.RemoveAll(tempDir)
			continue
		}
		if err := writeZipEntry(texEntry, texPath); err != nil {
			_ = os.RemoveAll(tempDir)
			continue
		}

		outputDir, _, unpackErr := tex.Unpack(xmlPath, tempDir)
		if unpackErr != nil {
			logger.Logger.Warnf("地图解包图集失败, atlas: %s, err: %v", xmlName, unpackErr)
			_ = os.RemoveAll(tempDir)
			continue
		}
		appendAtlasAssets(kind, outputDir, &doc, task.requested, result)
		_ = os.RemoveAll(tempDir)
	}

	return result
}

func appendAtlasAssets(kind, outputDir string, doc *mapAtlasXML, requested map[string]string, result map[string]MapAssetData) {
	for _, element := range doc.Elements.Elements {
		base := strings.TrimSuffix(element.Name, ".tex")
		if base == "" {
			continue
		}
		outputName := base
		if name, ok := requested[strings.ToLower(base)]; ok {
			outputName = name
		}
		if _, exists := result[outputName]; exists {
			continue
		}
		pngPath := filepath.Join(outputDir, base+".png")
		data, err := os.ReadFile(pngPath)
		if err != nil || len(data) == 0 || len(data) > maxMapAssetBytes {
			continue
		}
		data, width, height, ok := normalizeMapAssetImage(kind, data)
		if !ok {
			continue
		}
		sum := sha256.Sum256(data)
		result[outputName] = MapAssetData{
			Name:        outputName,
			Kind:        kind,
			ImageBase64: base64.StdEncoding.EncodeToString(data),
			SHA256:      fmt.Sprintf("%x", sum),
			Width:       width,
			Height:      height,
		}
		if len(result) >= maxMapAssetRefs {
			return
		}
	}
}

func normalizeMapAssetImage(kind string, data []byte) ([]byte, int, int, bool) {
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, false
	}
	width := config.Width
	height := config.Height
	if kind != "tile" {
		return data, width, height, true
	}

	picture, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, false
	}
	if width > MapTileTextureSide || height > MapTileTextureSide {
		picture = resizeToFit(picture, MapTileTextureSide)
	}
	var buffer bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buffer, picture); err != nil {
		return nil, 0, 0, false
	}
	if buffer.Len() == 0 || buffer.Len() > maxMapAssetBytes {
		return nil, 0, 0, false
	}
	return buffer.Bytes(), picture.Bounds().Dx(), picture.Bounds().Dy(), true
}

// NormalizeMapAssetBase64 把缓存里的地形 PNG 再压到当前版本，兼容旧 256px 缓存。
func NormalizeMapAssetBase64(kind, image string) (string, int, int, bool) {
	data, err := base64.StdEncoding.DecodeString(image)
	if err != nil {
		return "", 0, 0, false
	}
	normalized, width, height, ok := normalizeMapAssetImage(kind, data)
	if !ok {
		return "", 0, 0, false
	}
	return base64.StdEncoding.EncodeToString(normalized), width, height, true
}

func resizeToFit(source image.Image, maxSide int) *image.NRGBA {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1))
	}
	targetWidth := width
	targetHeight := height
	if width > maxSide || height > maxSide {
		if width >= height {
			targetWidth = maxSide
			targetHeight = height * maxSide / width
		} else {
			targetHeight = maxSide
			targetWidth = width * maxSide / height
		}
	}
	if targetWidth < 1 {
		targetWidth = 1
	}
	if targetHeight < 1 {
		targetHeight = 1
	}
	target := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	for y := 0; y < targetHeight; y++ {
		sourceY := bounds.Min.Y + y*height/targetHeight
		for x := 0; x < targetWidth; x++ {
			sourceX := bounds.Min.X + x*width/targetWidth
			target.Set(x, y, source.At(sourceX, sourceY))
		}
	}
	return target
}

// appendStandaloneAssets 处理没有 atlas XML 的独立 .tex 文件。
func appendStandaloneAssets(needed map[string]string, kind string, result map[string]MapAssetData) {
	for _, original := range needed {
		if _, exists := result[original]; exists {
			continue
		}
		data, width, height, err := readStandaloneKTex(original)
		if err != nil {
			continue
		}
		data, width, height, ok := normalizeMapAssetImage(kind, data)
		if !ok {
			continue
		}
		sum := sha256.Sum256(data)
		result[original] = MapAssetData{
			Name:        original,
			Kind:        kind,
			ImageBase64: base64.StdEncoding.EncodeToString(data),
			SHA256:      fmt.Sprintf("%x", sum),
			Width:       width,
			Height:      height,
		}
		if len(result) >= maxMapAssetRefs {
			return
		}
	}
}

func readStandaloneKTex(name string) ([]byte, int, int, error) {
	base := strings.TrimSuffix(name, ".tex")
	base = strings.TrimSuffix(base, ".png")
	if base == "" || filepath.Base(base) != base {
		return nil, 0, 0, fmt.Errorf("invalid tex name")
	}

	dirs := []string{mapLevelTexturesDir(), mapImagesDir()}
	candidates := []string{base, strings.ToLower(base), strings.ToUpper(base)}
	for _, dir := range dirs {
		for _, candidate := range candidates {
			path := filepath.Join(dir, candidate+".tex")
			if _, err := os.Stat(path); err != nil {
				continue
			}

			atlas, err := tex.ReadKTex(path)
			if err != nil {
				return nil, 0, 0, err
			}
			if atlas.Width <= 0 || atlas.Height <= 0 || len(atlas.Pix) < atlas.Width*atlas.Height*4 {
				return nil, 0, 0, fmt.Errorf("invalid tex size")
			}

			picture := image.NewNRGBA(image.Rect(0, 0, atlas.Width, atlas.Height))
			for y := 0; y < atlas.Height; y++ {
				src := (atlas.Height - 1 - y) * atlas.Width * 4
				dst := y * picture.Stride
				copy(picture.Pix[dst:dst+atlas.Width*4], atlas.Pix[src:src+atlas.Width*4])
			}
			demultiplyNRGBA(picture.Pix)

			var buffer bytes.Buffer
			if err := png.Encode(&buffer, picture); err != nil {
				return nil, 0, 0, err
			}
			if buffer.Len() == 0 || buffer.Len() > maxMapAssetBytes {
				return nil, 0, 0, fmt.Errorf("tex png size invalid")
			}
			return buffer.Bytes(), atlas.Width, atlas.Height, nil
		}
	}

	return nil, 0, 0, fmt.Errorf("tex not found")
}

func demultiplyNRGBA(pix []byte) {
	for i := 0; i+3 < len(pix); i += 4 {
		alpha := pix[i+3]
		if alpha == 0 || alpha == 0xFF {
			continue
		}
		inverse := uint32(255*256) / uint32(alpha)
		for channel := 0; channel < 3; channel++ {
			value := uint32(pix[i+channel]) * inverse >> 8
			if value > 0xFF {
				value = 0xFF
			}
			pix[i+channel] = byte(value)
		}
	}
}

type mapAtlasXML struct {
	Texture struct {
		Filename string `xml:"filename,attr"`
	} `xml:"Texture"`
	Elements struct {
		Elements []struct {
			Name string `xml:"name,attr"`
		} `xml:"Element"`
	} `xml:"Elements"`
}

func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, limit))
}

func writeZipEntry(file *zip.File, target string) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	output, err := os.Create(target)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, reader)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func mapDataDir() string {
	if cache.OsType == constant.Darwin {
		return filepath.Join(constant.DarwinDstContentsPath, "data")
	}
	return filepath.Join(constant.GameMainPath, "data")
}

func mapImagesDir() string {
	return filepath.Join(mapDataDir(), "images")
}

func mapMinimapDir() string {
	return filepath.Join(mapDataDir(), "minimap")
}

func mapLevelTexturesDir() string {
	return filepath.Join(mapDataDir(), "levels", "textures")
}

func mapAssetDirs() []string {
	var dirs []string
	for _, dir := range []string{mapImagesDir(), mapMinimapDir(), mapLevelTexturesDir()} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func mapImagesZipPath() string {
	return filepath.Join(mapDataDir(), "databundles", "images.zip")
}

func mapScriptsLuaPaths() []string {
	return []string{
		filepath.Join(mapDataDir(), "scripts", "tiledefs.lua"),
		filepath.Join(mapDataDir(), "databundles", "scripts", "scripts", "tiledefs.lua"),
		filepath.Join(mapDataDir(), "scripts", "scripts", "tiledefs.lua"),
	}
}

func mapScriptsLuaPath() string {
	for _, path := range mapScriptsLuaPaths() {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return mapScriptsLuaPaths()[0]
}

func mapScriptsZipPath() string {
	return filepath.Join(mapDataDir(), "databundles", "scripts.zip")
}
