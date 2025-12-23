package dst

import (
	"bufio"
	"bytes"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MapData struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	Image  string `json:"image"`
}

func tileID2Color(tileID int) string {
	MAP := map[int]string{
		0:   "#000000", // 默认异常
		1:   "#546E7A", // 边缘等
		2:   "#A1887F", // 卵石路
		3:   "#FFEFD5", // 矿区
		4:   "#F5DEB3", // 没有地皮
		5:   "#FFFACD", // 热带草原
		6:   "#66CDAA", // 长草
		7:   "#2E8B57", // 森林
		8:   "#4A148C", // 沼泽
		13:  "#B2EBF2", // 蝙蝠
		14:  "#0091EA", // 蓝蘑菇
		15:  "#66BB6A", // 楼梯普通
		16:  "#8D6E63", // 圆石笋
		17:  "#9E9D24", // 荧光果普通
		18:  "#BA68C8", // 迷宫
		19:  "#E040FB", // 远古1
		20:  "#E040FB", // 远古2
		21:  "#E040FB", // 远古3
		22:  "#E040FB", // 远古4
		23:  "#E040FB", // 远古5
		24:  "#E57373", // 红蘑菇
		25:  "#C8E6C9", // 绿蘑菇
		30:  "#FFA07A", // 落叶林
		31:  "#FFF9C4", // 沙漠
		42:  "#96CDCD", // 月岛1
		43:  "#96CDCD", // 月岛2
		44:  "#FFB6C1", // 奶奶岛
		45:  "#FFB300", // 档案馆
		46:  "#4DB6AC", // 月亮蘑菇林
		201: "#1E88E5", // 浅海1
		202: "#1976D2", // 浅海2
		203: "#1565C0", // 中海
		204: "#0D47A1", // 深海
		205: "#F5FFFA", // 海盐
		208: "#00897B", // 水中木
	}

	if MAP[tileID] == "" {
		return "#000000"
	}

	return MAP[tileID]
}

func parseHexColor(s string) color.RGBA {
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{}
	}

	r, err := strconv.ParseUint(s[1:3], 16, 8)
	if err != nil {
		return color.RGBA{}
	}

	g, err := strconv.ParseUint(s[3:5], 16, 8)
	if err != nil {
		return color.RGBA{}
	}

	b, err := strconv.ParseUint(s[5:7], 16, 8)
	if err != nil {
		return color.RGBA{}
	}

	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func (g *Game) generateBackgroundMap(worldID int) (MapData, error) {
	world, err := g.getWorldByID(worldID)
	if err != nil {
		return MapData{}, err
	}
	sessionPath, err := findLatestMetaFile(world.sessionPath)
	if err != nil {
		return MapData{}, err
	}

	filepath := strings.Split(sessionPath, ".meta")[0]

	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		logger.Logger.Error("打开存档文件失败", "err", err)
		return MapData{}, err
	}

	var height, width int

	reHeight := regexp.MustCompile(`,height=(\d+),`)
	reWidth := regexp.MustCompile(`,width=(\d+),`)

	matchHeight := reHeight.FindSubmatch(fileContent)
	if len(matchHeight) >= 2 {
		height, err = strconv.Atoi(string(matchHeight[1]))
		if err != nil {
			logger.Logger.Error("获取存档文件中height失败")
			return MapData{}, err
		}
	} else {
		logger.Logger.Error("获取存档文件中height失败")
		return MapData{}, errors.New("获取存档文件中height失败")
	}

	matchWidth := reWidth.FindSubmatch(fileContent)
	if len(matchWidth) >= 2 {
		width, err = strconv.Atoi(string(matchWidth[1]))
		if err != nil {
			logger.Logger.Error("获取存档文件中width失败")
			return MapData{}, errors.New("获取存档文件中width失败")
		}
	} else {
		logger.Logger.Error("获取存档文件中width失败")
		return MapData{}, errors.New("获取存档文件中width失败")
	}

	var tiles []byte

	// 匹配base64内容
	reTiles := regexp.MustCompile(`tiles="([A-Za-z0-9+/=]+)"`)
	matchTiles := reTiles.FindSubmatch(fileContent)
	if len(matchTiles) >= 2 {
		tiles = matchTiles[1]
	} else {
		logger.Logger.Error("存档文件中没有找到tiles字段")
		return MapData{}, errors.New("存档文件中没有找到tiles字段")
	}

	tilesDecoded, err := base64.StdEncoding.DecodeString(string(tiles))
	if err != nil {
		logger.Logger.Error("tiles字段解码失败", "err", err)
		return MapData{}, errors.New("tiles字段解码失败")
	}

	if len(tilesDecoded)%2 != 0 {
		tilesDecoded = tilesDecoded[:len(tilesDecoded)-1]
	}

	var tileIDs []int

	for i := 0; i < len(tilesDecoded); i += 2 {
		if i+1 >= len(tilesDecoded) {
			break
		}
		tileId := int(tilesDecoded[i+1])
		tileIDs = append(tileIDs, tileId)
	}
	// 创建新图像
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充像素
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 计算当前像素index
			index := y*width + x
			// 解析16进制颜色
			c := parseHexColor(tileID2Color(tileIDs[index]))

			X := width - x - 1

			img.Set(X, y, c)
		}
	}

	// 将图像编码为PNG格式的字节
	var buf bytes.Buffer
	if err = png.Encode(&buf, img); err != nil {
		logger.Logger.Error("图片编码失败", "err", err)
		return MapData{}, err
	}

	// 将PNG字节转换为Base64字符串
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	return MapData{
		Height: height,
		Width:  width,
		Image:  base64Str,
	}, nil
}

func coordinateToPx(size, a, b int) (int, int) {
	x := ((size*2 - a) * 323) / 1310
	y := ((size*2 + b) * 235) / 938

	return x, y
}

func (g *Game) getCoordinate(cmd string, worldID int) (int, int, error) {
	world, err := g.getWorldByID(worldID)
	if err != nil {
		return 0, 0, err
	}

	err = utils.ScreenCMD(cmd, world.screenName)
	if err != nil {
		return 0, 0, err
	}

	time.Sleep(100 * time.Millisecond)
	logPath := fmt.Sprintf("%s/server_log.txt", world.worldPath)
	// 打开文件
	file, err := os.Open(logPath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	// 使用缓冲读取器
	scanner := bufio.NewScanner(file)
	var lines []string
	var targetLineIndex int = -1

	// 先扫描文件并将所有行存入内存（适用于可以放入内存的文件）
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if strings.Contains(line, cmd) {
			// 记录最后一个匹配行的索引
			targetLineIndex = len(lines) - 1
		}
	}

	if targetLineIndex == -1 {
		return 0, 0, fmt.Errorf("未找到坐标信息")
	}

	// 检查是否有足够的后续行
	if targetLineIndex+3 >= len(lines) {
		return 0, 0, fmt.Errorf("找到目标行但没有足够的后续行")
	}

	// 提取坐标的三行
	coordLines := lines[targetLineIndex+1 : targetLineIndex+4]
	var x, y int
	var parseErr error

	// 解析第三行坐标
	nums := strings.Fields(coordLines[2])
	if len(nums) >= 4 {
		if strings.Contains(nums[1], ".") {
			a, err := strconv.ParseFloat(nums[1], 64)
			if err != nil {
				return 0, 0, fmt.Errorf("字符串转浮点数失败")
			}
			x = int(a)
		} else {
			x, parseErr = strconv.Atoi(nums[1])
			if parseErr != nil {
				return 0, 0, fmt.Errorf("解析x坐标失败")
			}
		}

		if strings.Contains(nums[3], ".") {
			a, err := strconv.ParseFloat(nums[3], 64)
			if err != nil {
				return 0, 0, fmt.Errorf("字符串转浮点数失败")
			}
			y = int(a)
		} else {
			y, parseErr = strconv.Atoi(nums[3])
			if parseErr != nil {
				return 0, 0, fmt.Errorf("解析y坐标失败")
			}
		}
	}

	return x, y, nil
}

type PrefabItem struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

func (g *Game) countPrefabs(worldID int) []PrefabItem {
	world, err := g.getWorldByID(worldID)
	if err != nil {
		return []PrefabItem{}
	}

	screenName := world.screenName
	logPath := fmt.Sprintf("%s/server_log.txt", world.worldPath)

	prefabs := []PrefabItem{
		{
			Code: "walrus_camp",
		},
		{
			Code: "wasphive",
		},
		{
			Code: "ruins_statue_mage",
		},
		{
			Code: "archive_moon_statue",
		},
	}

	cmd1 := "print('=== world prefabs counting start ===')"
	err = utils.ScreenCMD(cmd1, screenName)
	if err != nil {
		logger.Logger.Error("统计世界失败", "err", err)
		return prefabs
	}

	for _, prefab := range prefabs {
		cmd := fmt.Sprintf("c_countprefabs('%s')", prefab.Code)
		_ = utils.ScreenCMD(cmd, screenName)
		time.Sleep(50 * time.Millisecond)
	}

	cmd2 := "print('=== world prefabs counting finish ===')"
	err = utils.ScreenCMD(cmd2, screenName)
	if err != nil {
		logger.Logger.Error("统计世界失败", "err", err)
		return prefabs
	}
	time.Sleep(100 * time.Millisecond)

	file, err := os.Open(logPath)
	if err != nil {
		logger.Logger.Error("统计世界失败", "err", err)
		return prefabs
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Logger.Error("文件关闭失败", "err", err)
		}
	}(file)

	// 逐行读取文件
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var usefulLines []string

	var foundFinish bool
	var foundStart bool

	// 反向遍历行
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.Contains(line, cmd2) {
			foundFinish = true
			continue
		}

		if foundFinish {
			usefulLines = append(usefulLines, line)
		}

		// 检查是否包含关键字
		if strings.Contains(line, cmd1) {
			foundStart = true
			break
		}
	}

	if !foundStart {
		logger.Logger.Error("没有发现开始标记")
		return prefabs
	}

	// 正则表达式匹配模式
	pattern := `There are\s+(\d+)\s+(\w+)\s+in the world`
	re := regexp.MustCompile(pattern)

	// 查找匹配的行并提取所需字段
	for _, line := range usefulLines {
		if matches := re.FindStringSubmatch(line); matches != nil {
			for index, prefab := range prefabs {
				if prefab.Code+"s" == matches[2] {
					count, err := strconv.Atoi(matches[1])
					if err != nil {
						count = 0
					}
					prefabs[index].Count = count
				}
			}
		}
	}

	return prefabs
}

type PlayerPosition struct {
	UID        string `json:"uid"`
	Nickname   string `json:"nickname"`
	Prefab     string `json:"prefab"`
	Coordinate struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"coordinate"`
}

func (g *Game) playerPosition(worldID int) []PlayerPosition {
	world, err := g.getWorldByID(worldID)
	if err != nil {
		return []PlayerPosition{}
	}

	screenName := world.screenName
	logPath := fmt.Sprintf("%s/server_log.txt", world.worldPath)

	db.PlayersStatisticMutex.Lock()
	defer db.PlayersStatisticMutex.Unlock()

	var Players []PlayerPosition

	if len(db.PlayersStatistic[g.room.ID]) > 0 {
		players := db.PlayersStatistic[g.room.ID][len(db.PlayersStatistic[g.room.ID])-1].PlayerInfo
		for _, player := range players {
			Players = append(Players, PlayerPosition{
				UID:      player.UID,
				Nickname: player.Nickname,
				Prefab:   player.Prefab,
			})
		}
	} else {
		return []PlayerPosition{}
	}

	for index, player := range Players {
		ts := time.Now().UnixNano()

		cmd := fmt.Sprintf("print('==== DMP Start %s [%d] Start DMP ====')", player.UID, ts)
		err := utils.ScreenCMD(cmd, screenName)
		if err != nil {
			logger.Logger.Warn("执行获取玩家坐标失败，跳过", "err", err)
			continue
		}

		time.Sleep(50 * time.Millisecond)

		cmd = fmt.Sprintf("print(UserToPlayer('%s').Transform:GetWorldPosition())", player.UID)
		err = utils.ScreenCMD(cmd, screenName)
		if err != nil {
			logger.Logger.Warn("执行获取玩家坐标失败，跳过", "err", err)
			continue
		}

		time.Sleep(50 * time.Millisecond)

		cmd = fmt.Sprintf("print('==== DMP End %s [%d] End DMP ====')", player.UID, ts)
		err = utils.ScreenCMD(cmd, screenName)
		if err != nil {
			logger.Logger.Warn("执行获取玩家坐标失败，跳过", "err", err)
			continue
		}

		time.Sleep(50 * time.Millisecond)

		data := utils.GetFileLastNLines(logPath, 100)
		var lines []string
		for i := len(data) - 1; i >= 0; i-- {
			lines = append(lines, data[i])
		}

		pattern := `(-?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)\s+([-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)\s+(-?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)`
		re := regexp.MustCompile(pattern)

		var endFound bool

		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf("==== DMP End %s [%d] End DMP ====", player.UID, ts)) {
				endFound = true
				continue
			}
			if endFound {
				endFound = false
				if matches := re.FindStringSubmatch(line); matches != nil {
					x, err := strconv.ParseFloat(matches[1], 64)
					if err != nil {
						break
					}
					y, err := strconv.ParseFloat(matches[3], 64)
					if err != nil {
						break
					}
					Players[index].Coordinate.X = int(x)
					Players[index].Coordinate.Y = int(y)
				}
			}

		}

	}

	var returnData []PlayerPosition

	for _, player := range Players {
		if player.Coordinate.Y != 0 {
			returnData = append(returnData, player)
		}
	}

	return returnData
}
