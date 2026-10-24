package tex

import (
	"encoding/binary"
	"fmt"
)

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
