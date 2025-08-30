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

func tileID2Color(tileID int) string {
	MAP := map[int]string{
		0: "#000000", // 异常值
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
func GenerateBackgroundMap(filepath string) string {
	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		Logger.Error("打开存档文件失败", "err", err)
		return ""
	}

	var height, width int

	reHeight := regexp.MustCompile(`height=(\d+)`)
	reWidth := regexp.MustCompile(`width=(\d+)`)

	matchHeight := reHeight.FindSubmatch(fileContent)
	if len(matchHeight) >= 2 {
		height, err = strconv.Atoi(string(matchHeight[1]))
		if err != nil {
			Logger.Error("获取存档文件中height失败")
			return ""
		}
	} else {
		Logger.Error("获取存档文件中height失败")
		return ""
	}

	matchWidth := reWidth.FindSubmatch(fileContent)
	if len(matchWidth) >= 2 {
		width, err = strconv.Atoi(string(matchWidth[1]))
		if err != nil {
			Logger.Error("获取存档文件中width失败")
			return ""
		}
	} else {
		Logger.Error("获取存档文件中width失败")
		return ""
	}

	var tiles []byte

	reTiles := regexp.MustCompile(`tiles="(.+)"`)
	matchTiles := reTiles.FindSubmatch(fileContent)
	if len(matchTiles) >= 2 {
		tiles = matchTiles[1]
	} else {
		Logger.Error("存档文件中没有找到tiles字段")
		return ""
	}

	tilesDecoded, err := base64.StdEncoding.DecodeString(string(tiles))
	if err != nil {
		Logger.Error("tiles字段解码失败", "err", err)
		return ""
	}

	if len(tilesDecoded)%2 != 0 {
		tilesDecoded = tilesDecoded[:len(tilesDecoded)-1]
	}

	var tileIDs []int

	for i := 0; i < len(tilesDecoded); i += 2 {
		if i+1 >= len(tilesDecoded) {
			break
		}
		tileId := (int(tilesDecoded[i+1]) << 8) | int(tilesDecoded[i])
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
			img.Set(x, y, c)
		}
	}

	// 将图像编码为PNG格式的字节
	var buf bytes.Buffer
	if err = png.Encode(&buf, img); err != nil {
		Logger.Error("图片编码失败", "err", err)
		return ""
	}

	// 将PNG字节转换为Base64字符串
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	return base64Str
}
