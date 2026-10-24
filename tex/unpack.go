package tex

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

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
