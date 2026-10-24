package tex

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// atlasKey 是 Klei 图集的 key（xml）：引用的 TEX 文件名、是否直通 alpha、
// 以及各元素的相对 UV 范围。
type atlasKey struct {
	atlasFilename string
	straightAlpha bool
	elements      []atlasElement
}

type atlasElement struct {
	name           string
	u1, v1, u2, v2 float64
}

type xmlAtlas struct {
	XMLName xml.Name `xml:"Atlas"`
	Texture struct {
		Filename string `xml:"filename,attr"`
	} `xml:"Texture"`
	Elements struct {
		Elements []xmlElement `xml:"Element"`
	} `xml:"Elements"`
	// StraightAlpha 是 Stexatlaser 打包时写入的扩展字段，Klei 原生图集没有
	StraightAlpha string `xml:"StraightAlpha"`
}

type xmlElement struct {
	Name string `xml:"name,attr"`
	U1   string `xml:"u1,attr"`
	V1   string `xml:"v1,attr"`
	U2   string `xml:"u2,attr"`
	V2   string `xml:"v2,attr"`
}

// readAtlasKey 读取并校验 atlas key。
func readAtlasKey(path string) (*atlasKey, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 atlas key 失败: %w", err)
	}

	var doc xmlAtlas
	if err := xml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("解析 atlas key 失败: %w", err)
	}
	if doc.Texture.Filename == "" {
		return nil, fmt.Errorf("atlas key 缺少 Texture filename: %s", path)
	}
	if len(doc.Elements.Elements) == 0 {
		return nil, fmt.Errorf("atlas key 中没有 Element: %s", path)
	}

	key := &atlasKey{
		atlasFilename: doc.Texture.Filename,
		straightAlpha: strings.EqualFold(strings.TrimSpace(doc.StraightAlpha), "true"),
		elements:      make([]atlasElement, 0, len(doc.Elements.Elements)),
	}
	for _, item := range doc.Elements.Elements {
		if item.Name == "" {
			return nil, fmt.Errorf("atlas key 中存在缺少 name 的 Element: %s", path)
		}

		var coords [4]float64
		for i, raw := range []string{item.U1, item.V1, item.U2, item.V2} {
			value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
			if err != nil {
				return nil, fmt.Errorf("元素 %s 的 UV 坐标无效: %q", item.Name, raw)
			}
			coords[i] = value
		}

		key.elements = append(key.elements, atlasElement{
			name: item.Name,
			u1:   coords[0],
			v1:   coords[1],
			u2:   coords[2],
			v2:   coords[3],
		})
	}

	return key, nil
}
