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

const (
	scrW      = 640
	scrH      = 350
	bytesPerRow = scrW / 8 // 80
	planeLen  = bytesPerRow * scrH // 28000
)

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

// at 回傳第 p 個平面、第 y 列、第 b 個位元組在解壓資料裡的位置。
func at(mode string, p, y, b int) int {
	if mode == "planar" {
		return p*planeLen + y*bytesPerRow + b
	}
	return (y*4+p)*bytesPerRow + b
}

func render(d []byte, mode string) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, scrW, scrH))
	for y := 0; y < scrH; y++ {
		for b := 0; b < bytesPerRow; b++ {
			var pl [4]byte
			for p := 0; p < 4; p++ {
				pl[p] = d[at(mode, p, y, b)]
			}
			for bit := 0; bit < 8; bit++ {
				sh := uint(7 - bit)
				idx := 0
				for p := 0; p < 4; p++ {
					idx |= int((pl[p]>>sh)&1) << uint(3-p)
				}
				im.Set(b*8+bit, y, ega[idx])
			}
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
	if len(d) != planeLen*4 {
		fmt.Fprintf(os.Stderr, "長度不是 %d，版面假設不成立\n", planeLen*4)
		os.Exit(1)
	}
	for _, m := range []string{"planar", "rowint"} {
		save(render(d, m), os.Args[2]+"-"+m+".png")
		fmt.Println("寫出", os.Args[2]+"-"+m+".png")
	}
}
