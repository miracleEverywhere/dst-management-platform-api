// Package tex 实现 Klei TEX（KTEX）图集的解包，功能等价于 Stexatlaser 的
// stex unpack 命令。
//
// 这里只包含解包所需的部分：读取 KTEX 主图、解码 DXT1/DXT3/DXT5/RGB/RGBA
// 像素、按 atlas key（xml）裁剪出各元素并输出 PNG。打包、压缩、mipmap
// 生成等其它能力不在此实现。
package tex

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
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
