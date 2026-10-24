package utils

// Klei TEX（KTEX）图集解包，功能等价于 Stexatlaser 的 stex unpack 命令。
//
// 这里只包含解包所需的部分：读取 KTEX 主图、解码 DXT1/DXT3/DXT5/RGB/RGBA
// 像素、按 atlas key（xml）裁剪出各元素并输出 PNG。打包、压缩、mipmap
// 生成等其它能力不在此实现。

import (
	"bufio"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PixelFormat 是 KTEX 头中记录的像素格式。
type PixelFormat uint8

const (
	PixelFormatDXT1    PixelFormat = 0x00
	PixelFormatDXT3    PixelFormat = 0x01
	PixelFormatDXT5    PixelFormat = 0x02
	PixelFormatRGBA    PixelFormat = 0x04
	PixelFormatRGB     PixelFormat = 0x05
	PixelFormatETC2EAC PixelFormat = 0x11
)

const (
	ktexMagic      = "KTEX"
	ktexHeaderSize = 8
	mipMapMetaSize = 10
	// maxMipMapCount 只用于拦截明显异常的文件，正常图集不会超过 16 级
	maxMipMapCount = 32
)

// AtlasImage 是 KTEX 主图（第 0 级 mipmap）的解码结果。
// Pix 按 RGBA 顺序存储，alpha 保持 TEX 中的预乘值，由调用方决定是否反预乘。
type AtlasImage struct {
	Width       int
	Height      int
	Pix         []byte
	PixelFormat PixelFormat
}

type mipMapMeta struct {
	width  uint16
	height uint16
	pitch  uint16
	size   uint32
}

// ReadKTex 读取 path 指向的 KTEX 文件并解码其中的主图。
func ReadKTex(path string) (*AtlasImage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开 TEX 文件失败: %w", err)
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("读取 TEX 文件信息失败: %w", err)
	}

	head := make([]byte, ktexHeaderSize)
	if _, err := io.ReadFull(f, head); err != nil {
		return nil, fmt.Errorf("读取 TEX 头失败: %w", err)
	}
	if string(head[:4]) != ktexMagic {
		return nil, fmt.Errorf("不是有效的 TEX 文件: %s", path)
	}

	pixelFormat, mipMapCount := parseSpec(binary.LittleEndian.Uint32(head[4:8]))
	if !supportedPixelFormat(pixelFormat) {
		return nil, fmt.Errorf("不支持的像素格式: 0x%02X", pixelFormat)
	}
	if mipMapCount == 0 {
		return nil, fmt.Errorf("TEX 中没有图像数据: %s", path)
	}
	if mipMapCount > maxMipMapCount {
		return nil, fmt.Errorf("TEX 的 mipmap 数量异常: %d", mipMapCount)
	}

	metas := make([]mipMapMeta, mipMapCount)
	metaBuf := make([]byte, mipMapMetaSize)
	for i := range metas {
		if _, err := io.ReadFull(f, metaBuf); err != nil {
			return nil, fmt.Errorf("读取第 %d 级 mipmap 元数据失败: %w", i, err)
		}
		metas[i] = mipMapMeta{
			width:  binary.LittleEndian.Uint16(metaBuf[0:2]),
			height: binary.LittleEndian.Uint16(metaBuf[2:4]),
			pitch:  binary.LittleEndian.Uint16(metaBuf[4:6]),
			size:   binary.LittleEndian.Uint32(metaBuf[6:10]),
		}
	}

	main := metas[0]
	if main.width == 0 || main.height == 0 || main.size == 0 {
		return nil, fmt.Errorf("TEX 主图数据为空: %s", path)
	}
	// 声明的数据长度不能超过文件剩余长度，避免被损坏的头引向超大内存分配
	remaining := fileInfo.Size() - ktexHeaderSize - int64(len(metas))*mipMapMetaSize
	if int64(main.size) > remaining {
		return nil, fmt.Errorf("TEX 数据长度异常: 声明 %d 字节，剩余 %d 字节", main.size, remaining)
	}

	data := make([]byte, main.size)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, fmt.Errorf("读取 TEX 主图数据失败: %w", err)
	}

	width := int(main.width)
	height := int(main.height)
	pitch := int(main.pitch)
	if pitch == 0 {
		pitch = defaultPitch(width, pixelFormat)
	}

	pix, err := decodePixels(data, width, height, pitch, pixelFormat)
	if err != nil {
		return nil, err
	}

	return &AtlasImage{
		Width:       width,
		Height:      height,
		Pix:         pix,
		PixelFormat: pixelFormat,
	}, nil
}

// parseSpec 解析规格整数中的像素格式与 mipmap 数量。
// 两种头部版本都以「padding 位全高」作为判别依据，位序均为低位在前。
func parseSpec(spec uint32) (PixelFormat, int) {
	if spec>>14 == 0x3FFFF {
		// pre-caves: platform:3 pixelFormat:3 textureType:3 mipmapCount:4 flag:1 padding:18
		return PixelFormat((spec >> 3) & 0x7), int((spec >> 9) & 0xF)
	}
	// post-caves: platform:4 pixelFormat:5 textureType:4 mipMapCount:5 flags:2 padding:12
	return PixelFormat((spec >> 4) & 0x1F), int((spec >> 13) & 0x1F)
}

func supportedPixelFormat(format PixelFormat) bool {
	switch format {
	case PixelFormatDXT1, PixelFormatDXT3, PixelFormatDXT5,
		PixelFormatRGBA, PixelFormatRGB, PixelFormatETC2EAC:
		return true
	default:
		return false
	}
}

// defaultPitch 在 TEX 未记录行字节数（pitch 为 0）时按像素格式推算。
func defaultPitch(width int, format PixelFormat) int {
	switch format {
	case PixelFormatDXT1:
		return (width + 3) / 4 * 8
	case PixelFormatDXT3, PixelFormatDXT5:
		return (width + 3) / 4 * 16
	case PixelFormatRGB:
		return width * 3
	default:
		return width * 4
	}
}

// decodePixels 把 TEX 原始像素数据解码为 RGBA8888。
func decodePixels(data []byte, width, height, pitch int, format PixelFormat) ([]byte, error) {
	switch format {
	case PixelFormatDXT1, PixelFormatDXT3, PixelFormatDXT5:
		return decodeS3TC(data, width, height, format)
	case PixelFormatRGB:
		return decodeRaw(data, width, height, pitch, 3)
	case PixelFormatRGBA:
		return decodeRaw(data, width, height, pitch, 4)
	case PixelFormatETC2EAC:
		return nil, fmt.Errorf("暂不支持 ETC2EAC 像素格式")
	default:
		return nil, fmt.Errorf("不支持的像素格式: 0x%02X", format)
	}
}

// decodeRaw 解码未压缩的 RGB/RGBA 数据，pitch 为每行字节数。
func decodeRaw(data []byte, width, height, pitch, bytesPerPixel int) ([]byte, error) {
	if pitch < width*bytesPerPixel {
		pitch = width * bytesPerPixel
	}
	if len(data) < pitch*(height-1)+width*bytesPerPixel {
		return nil, fmt.Errorf("TEX 数据长度不足: 实际 %d 字节", len(data))
	}

	pix := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		row := data[y*pitch:]
		for x := 0; x < width; x++ {
			dst := (y*width + x) * 4
			src := x * bytesPerPixel
			pix[dst+0] = row[src+0]
			pix[dst+1] = row[src+1]
			pix[dst+2] = row[src+2]
			if bytesPerPixel == 4 {
				pix[dst+3] = row[src+3]
			} else {
				pix[dst+3] = 0xFF
			}
		}
	}
	return pix, nil
}

// decodeS3TC 解码 DXT1/DXT3/DXT5（S3TC/BC1-BC3）数据，块按从左到右、从上到下排列。
func decodeS3TC(data []byte, width, height int, format PixelFormat) ([]byte, error) {
	blockSize := 8
	if format != PixelFormatDXT1 {
		blockSize = 16
	}
	blocksX := (width + 3) / 4
	blocksY := (height + 3) / 4
	need := blocksX * blocksY * blockSize
	if len(data) < need {
		return nil, fmt.Errorf("TEX 数据长度不足: 需要 %d 字节，实际 %d 字节", need, len(data))
	}

	pix := make([]byte, width*height*4)
	// 一个 4x4 块共 16 个像素，按 RGBA 展开后是 64 字节
	var block [64]byte
	offset := 0
	for by := 0; by < blocksY; by++ {
		for bx := 0; bx < blocksX; bx++ {
			src := data[offset : offset+blockSize]
			switch format {
			case PixelFormatDXT1:
				decodeDXT1Block(src, block[:])
			case PixelFormatDXT3:
				decodeDXT3Block(src, block[:])
			default:
				decodeDXT5Block(src, block[:])
			}

			for py := 0; py < 4; py++ {
				y := by*4 + py
				if y >= height {
					break
				}
				for px := 0; px < 4; px++ {
					x := bx*4 + px
					if x >= width {
						break
					}
					dst := (y*width + x) * 4
					srcPixel := (py*4 + px) * 4
					copy(pix[dst:dst+4], block[srcPixel:srcPixel+4])
				}
			}
			offset += blockSize
		}
	}
	return pix, nil
}

// decodeDXT1Block 解码一个 DXT1 块（8 字节），结果写入 4x4 的 RGBA 缓冲。
func decodeDXT1Block(src, dst []byte) {
	decodeColorBlock(src, dst, true)
}

// decodeDXT3Block 解码一个 DXT3 块（16 字节）。
func decodeDXT3Block(src, dst []byte) {
	for i := 0; i < 16; i++ {
		nibble := src[i/2] & 0x0F
		if i%2 == 1 {
			nibble = src[i/2] >> 4
		}
		dst[i*4+3] = nibble * 0x11
	}
	decodeColorBlock(src[8:16], dst, false)
}

// decodeDXT5Block 解码一个 DXT5 块（16 字节）。
func decodeDXT5Block(src, dst []byte) {
	decodeAlphaBlock(src[:8], dst)
	decodeColorBlock(src[8:16], dst, false)
}

// decodeAlphaBlock 解码 DXT5 的 8 字节 alpha 块，每个像素 3 位索引。
func decodeAlphaBlock(src, dst []byte) {
	a0, a1 := src[0], src[1]
	var table [8]byte
	table[0], table[1] = a0, a1
	if a0 > a1 {
		for i := 2; i < 8; i++ {
			table[i] = byte(((8-i)*int(a0) + (i-1)*int(a1)) / 7)
		}
	} else {
		for i := 2; i < 6; i++ {
			table[i] = byte(((6-i)*int(a0) + (i-1)*int(a1)) / 5)
		}
		table[6], table[7] = 0, 0xFF
	}

	var bits uint64
	for i := 0; i < 6; i++ {
		bits |= uint64(src[2+i]) << (8 * i)
	}
	for i := 0; i < 16; i++ {
		dst[i*4+3] = table[(bits>>(3*i))&0x7]
	}
}

// decodeColorBlock 解码 8 字节颜色块。dxt1 为 true 时按 DXT1 规则决定是否使用
// 四色模式以及透明色，同时写入 alpha；DXT3/DXT5 的 alpha 由各自的 alpha 块负责。
func decodeColorBlock(src, dst []byte, dxt1 bool) {
	c0 := binary.LittleEndian.Uint16(src[0:2])
	c1 := binary.LittleEndian.Uint16(src[2:4])

	var colors [4][3]byte
	rgb565(c0, &colors[0])
	rgb565(c1, &colors[1])

	// DXT3/DXT5 固定使用四色插值；DXT1 仅在 c0 > c1 时使用四色
	fourColor := !dxt1 || c0 > c1
	if fourColor {
		for i := 0; i < 3; i++ {
			colors[2][i] = byte((2*int(colors[0][i]) + int(colors[1][i])) / 3)
			colors[3][i] = byte((int(colors[0][i]) + 2*int(colors[1][i])) / 3)
		}
	} else {
		for i := 0; i < 3; i++ {
			colors[2][i] = byte((int(colors[0][i]) + int(colors[1][i])) / 2)
		}
		colors[3] = [3]byte{}
	}

	indices := binary.LittleEndian.Uint32(src[4:8])
	for i := 0; i < 16; i++ {
		color := colors[int(indices>>(2*i))&0x3]
		dst[i*4+0] = color[0]
		dst[i*4+1] = color[1]
		dst[i*4+2] = color[2]
		if dxt1 {
			if !fourColor && int(indices>>(2*i))&0x3 == 3 {
				dst[i*4+3] = 0
			} else {
				dst[i*4+3] = 0xFF
			}
		}
	}
}

// rgb565 把 16 位 RGB565 展开为 8 位 RGB，按位复制等价于四舍五入到 0-255。
func rgb565(value uint16, dst *[3]byte) {
	r := uint8(value>>11) & 0x1F
	g := uint8(value>>5) & 0x3F
	b := uint8(value) & 0x1F
	dst[0] = r<<3 | r>>2
	dst[1] = g<<2 | g>>4
	dst[2] = b<<3 | b>>2
}

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

// Unpack 解包 keyPath 指向的 TEX 图集，PNG 输出到 outputDir/<图集名>/ 下，
// 目录结构与 stex unpack 一致。返回值是实际输出目录和生成的图片数量。
func Unpack(keyPath, outputDir string) (string, int, error) {
	info, err := os.Stat(keyPath)
	if err != nil {
		return "", 0, fmt.Errorf("输入文件不存在: %s", keyPath)
	}
	if info.IsDir() || !strings.EqualFold(filepath.Ext(keyPath), ".xml") {
		return "", 0, fmt.Errorf("输入必须是一个 atlas key（.xml）文件: %s", keyPath)
	}

	key, err := readAtlasKey(keyPath)
	if err != nil {
		return "", 0, err
	}

	atlasPath, err := atlasFilePath(keyPath, key.atlasFilename)
	if err != nil {
		return "", 0, err
	}

	img, err := ReadKTex(atlasPath)
	if err != nil {
		return "", 0, err
	}

	atlasName := strings.TrimSuffix(filepath.Base(atlasPath), filepath.Ext(atlasPath))
	dir := filepath.Join(outputDir, atlasName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("创建输出目录失败: %w", err)
	}

	for _, element := range key.elements {
		name, err := elementFileName(element.name)
		if err != nil {
			return "", 0, err
		}

		rect, err := elementRect(element, img.Width, img.Height)
		if err != nil {
			return "", 0, err
		}

		picture := crop(img.Pix, img.Width, img.Height, rect)
		if !key.straightAlpha {
			demultiplyAlpha(picture.Pix)
		}

		if err := writePNG(filepath.Join(dir, name+".png"), picture); err != nil {
			return "", 0, err
		}
	}

	return dir, len(key.elements), nil
}

// atlasFilePath 解析 key 中引用的 TEX 路径。Klei 的 key 只写同目录下的文件名，
// 这里拒绝带目录的部分，避免 key 内容把读取引到其它路径。
func atlasFilePath(keyPath, filename string) (string, error) {
	if filename != filepath.Base(filename) {
		return "", fmt.Errorf("atlas key 中的纹理文件名不合法: %q", filename)
	}

	path := filepath.Join(filepath.Dir(keyPath), filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("找不到图集文件: %s", path)
	}

	return path, nil
}

// elementFileName 返回元素对应的 PNG 文件名（不含扩展名）。
// Klei 的元素名通常带 .tex 后缀，输出时需要去掉；同时拒绝可能逃出输出目录的名字。
func elementFileName(elementName string) (string, error) {
	name := strings.TrimSuffix(elementName, ".tex")
	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\:`) || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("元素名不合法: %q", elementName)
	}

	return name, nil
}

// pixelRect 是主图上的像素区域，x/y 为左上角，w/h 为宽高。
type pixelRect struct {
	x, y, w, h int
}

// elementRect 把相对 UV 坐标换算成主图上的像素区域。
// 换算方式与 Stexatlaser 一致：按 round(v*size-0.5) 取整，再因为 TEX 数据
// 自下而上存储而做垂直翻转，最后像 QImage::copy 一样裁掉越界部分。
func elementRect(element atlasElement, width, height int) (pixelRect, error) {
	x0 := int(math.Round(element.u1*float64(width) - 0.5))
	x1 := int(math.Round(element.u2*float64(width) - 0.5))
	y0 := int(math.Round(element.v1*float64(height) - 0.5))
	y1 := int(math.Round(element.v2*float64(height) - 0.5))
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}

	rect := pixelRect{
		x: x0,
		y: (height - 1) - y1,
		w: x1 - x0 + 1,
		h: y1 - y0 + 1,
	}
	if rect.x < 0 {
		rect.w += rect.x
		rect.x = 0
	}
	if rect.y < 0 {
		rect.h += rect.y
		rect.y = 0
	}
	if rect.x+rect.w > width {
		rect.w = width - rect.x
	}
	if rect.y+rect.h > height {
		rect.h = height - rect.y
	}
	if rect.w <= 0 || rect.h <= 0 {
		return pixelRect{}, fmt.Errorf("元素 %s 超出图集范围", element.name)
	}

	return rect, nil
}

// crop 从主图中裁出指定区域。rect 使用自上而下的图像坐标，而主图数据是
// 自下而上存储的，所以取行时要按 height-1-y 做垂直翻转。
func crop(pix []byte, width, height int, rect pixelRect) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, rect.w, rect.h))
	for y := 0; y < rect.h; y++ {
		srcRow := height - 1 - (rect.y + y)
		src := (srcRow*width + rect.x) * 4
		dst := y * picture.Stride
		copy(picture.Pix[dst:dst+rect.w*4], pix[src:src+rect.w*4])
	}

	return picture
}

// demultiplyAlpha 把预乘 alpha 还原为直通 alpha，采用 Qt qUnpremultiply 的定点算法
// （255*256/alpha）。DXT5 压缩会让个别像素的颜色值略大于 alpha，使反预乘结果超过
// 255，这里按 255 截断，避免颜色溢出到相邻通道。
func demultiplyAlpha(pix []byte) {
	for i := 0; i < len(pix); i += 4 {
		alpha := pix[i+3]
		if alpha == 0 || alpha == 0xFF {
			continue
		}
		inverse := uint32(255*256) / uint32(alpha)
		pix[i+0] = unpremultiply(pix[i+0], inverse)
		pix[i+1] = unpremultiply(pix[i+1], inverse)
		pix[i+2] = unpremultiply(pix[i+2], inverse)
	}
}

func unpremultiply(value byte, inverse uint32) byte {
	result := uint32(value) * inverse >> 8
	if result > 0xFF {
		return 0xFF
	}
	return byte(result)
}

func writePNG(path string, picture *image.NRGBA) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建图片失败: %w", err)
	}

	writer := bufio.NewWriter(file)
	encodeErr := png.Encode(writer, picture)
	flushErr := writer.Flush()
	closeErr := file.Close()

	if encodeErr != nil {
		return fmt.Errorf("编码图片失败: %w", encodeErr)
	}
	if flushErr != nil {
		return fmt.Errorf("写入图片失败: %w", flushErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭图片失败: %w", closeErr)
	}

	return nil
}
