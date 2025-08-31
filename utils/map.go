package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"regexp"
	"strconv"
)

type Data struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	Image  string `json:"image"`
}

func tileID2Color(tileID int) string {
	MAP := map[int]string{
		0:   "#000000", // 默认异常
		1:   "#F44336", // 边缘等
		2:   "#A1887F", // 卵石路
		3:   "#FFEFD5", // 矿区
		4:   "#F5DEB3", // 空
		5:   "#FFFACD", // 热带草原
		6:   "#66CDAA", // 长草
		7:   "#2E8B57", // 森林
		8:   "#4A148C", // 沼泽
		30:  "#FFA07A", // 落叶林
		31:  "#FFF9C4", // 沙漠
		42:  "#96CDCD", // 月岛1
		43:  "#96CDCD", // 月岛2
		44:  "#FFB6C1", // 奶奶岛
		201: "#1E88E5", // 浅海1
		202: "#1976D2", // 浅海2
		203: "#1565C0", // 中海
		204: "#0D47A1", // 深海
		205: "#F5FFFA", // 盐
		208: "#4DB6AC", // 水中木
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

// GenerateBackgroundMap filepath: 最新的存档文件 返回背景地图base64
func GenerateBackgroundMap(filepath string) Data {
	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		Logger.Error("打开存档文件失败", "err", err)
		return Data{}
	}

	var height, width int

	reHeight := regexp.MustCompile(`height=(\d+)`)
	reWidth := regexp.MustCompile(`width=(\d+)`)

	matchHeight := reHeight.FindSubmatch(fileContent)
	if len(matchHeight) >= 2 {
		height, err = strconv.Atoi(string(matchHeight[1]))
		if err != nil {
			Logger.Error("获取存档文件中height失败")
			return Data{}
		}
	} else {
		Logger.Error("获取存档文件中height失败")
		return Data{}
	}

	matchWidth := reWidth.FindSubmatch(fileContent)
	if len(matchWidth) >= 2 {
		width, err = strconv.Atoi(string(matchWidth[1]))
		if err != nil {
			Logger.Error("获取存档文件中width失败")
			return Data{}
		}
	} else {
		Logger.Error("获取存档文件中width失败")
		return Data{}
	}

	var tiles []byte

	// 匹配base64内容
	reTiles := regexp.MustCompile(`tiles="([A-Za-z0-9+/=]+)"`)
	matchTiles := reTiles.FindSubmatch(fileContent)
	if len(matchTiles) >= 2 {
		tiles = matchTiles[1]
	} else {
		Logger.Error("存档文件中没有找到tiles字段")
		return Data{}
	}

	tilesDecoded, err := base64.StdEncoding.DecodeString(string(tiles))
	if err != nil {
		Logger.Error("tiles字段解码失败", "err", err)
		return Data{}
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
			if X*y == 49*152 {
				Logger.Error(strconv.Itoa(tileIDs[index]))
			}
			img.Set(X, y, c)
		}
	}

	// 将图像编码为PNG格式的字节
	var buf bytes.Buffer
	if err = png.Encode(&buf, img); err != nil {
		Logger.Error("图片编码失败", "err", err)
		return Data{}
	}

	// 将PNG字节转换为Base64字符串
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	return Data{
		Height: height,
		Width:  width,
		Image:  base64Str,
	}
}
