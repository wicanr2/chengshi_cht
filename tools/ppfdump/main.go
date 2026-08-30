// ppfdump 把 `.PPF`（整幅畫面）解出來存成 PNG，兩種平面版面各存一份。
//
// `.PPF` 解壓後剛好 112000 位元組 ＝ 640 × 350 ÷ 8 × 4，也就是 EGA 高解析的
// 四個位元平面（docs/formats/02-dos-lzss.md）。位元組怎麼排還沒定案，所以
// 這支工具把兩種可能都畫出來，拿去跟 DOSBox 實跑的畫面比對再定。
//
//	planar  整個平面接整個平面（0..28000 全是位元 0）
//	rowint  逐列交錯（每列 80 位元組 × 四個平面）
//
// 用法：go run ./tools/ppfdump <檔.ppf> <輸出前綴>
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// 三種顯示模式的畫面尺寸。長度 ＝ 寬 × 高 ÷ 8 × 平面數（256 色是每像素
// 一個位元組），拿解壓後的長度就認得出來。
var modes = []struct {
	name   string
	w, h   int
	planes int // 0 ＝ 每像素一個位元組（256 色）
}{
	{"CEGA 640×350×4", 640, 350, 4},
	{"sega 320×200×4", 320, 200, 4},
	{"mcga 320×199×8bpp", 320, 199, 0},
}

// 標準 EGA 16 色。PPF 自己不帶調色盤，先用預設的，對拍之後再定。
var ega = [16]color.RGBA{
	{0x00, 0x00, 0x00, 0xff}, {0x00, 0x00, 0xaa, 0xff},
	{0x00, 0xaa, 0x00, 0xff}, {0x00, 0xaa, 0xaa, 0xff},
	{0xaa, 0x00, 0x00, 0xff}, {0xaa, 0x00, 0xaa, 0xff},
	{0xaa, 0x55, 0x00, 0xff}, {0xaa, 0xaa, 0xaa, 0xff},
	{0x55, 0x55, 0x55, 0xff}, {0x55, 0x55, 0xff, 0xff},
	{0x55, 0xff, 0x55, 0xff}, {0x55, 0xff, 0xff, 0xff},
	{0xff, 0x55, 0x55, 0xff}, {0xff, 0x55, 0xff, 0xff},
	{0xff, 0xff, 0x55, 0xff}, {0xff, 0xff, 0xff, 0xff},
}

// renderPlanar 畫逐列交錯、高位在前的位元平面。
func renderPlanar(d []byte, w, h, planes int) *image.RGBA {
	bpr := w / 8
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := y * planes * bpr
		for b := 0; b < bpr; b++ {
			for bit := 0; bit < 8; bit++ {
				sh := uint(7 - bit)
				idx := 0
				for p := 0; p < planes; p++ {
					idx |= int((d[row+p*bpr+b]>>sh)&1) << uint(planes-1-p)
				}
				im.Set(b*8+bit, y, ega[idx])
			}
		}
	}
	return im
}

// renderLinear 畫每像素一個位元組的 256 色畫面。調色盤還沒解，
// 先用灰階把版面畫出來——判斷版面對不對只需要看得出形狀。
func renderLinear(d []byte, w, h int) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := d[y*w+x]
			im.Set(x, y, color.RGBA{v, v, v, 0xff})
		}
	}
	return im
}

func save(im *image.RGBA, path string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, im); err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "用法：ppfdump <檔.ppf> <輸出前綴>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	d, err := assets.DecompressLZSS(raw)
	if err != nil {
		panic(err)
	}
	fmt.Printf("解出 %d 位元組\n", len(d))
	for _, m := range modes {
		want := m.w * m.h
		if m.planes > 0 {
			want = want / 8 * m.planes
		}
		if want != len(d) {
			continue
		}
		var im *image.RGBA
		if m.planes > 0 {
			im = renderPlanar(d, m.w, m.h, m.planes)
		} else {
			im = renderLinear(d, m.w, m.h)
		}
		save(im, os.Args[2]+".png")
		fmt.Printf("%s → %s\n", m.name, os.Args[2]+".png")
		return
	}
	fmt.Fprintf(os.Stderr, "長度 %d 對不上任何已知的版面\n", len(d))
	os.Exit(1)
}
