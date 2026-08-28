// pgfdump 是 .PGF 圖形檔的探查工具（開發用，不是遊戲的一部分）。
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

func main() {
	bank := flag.Int("bank", 0, "要畫哪一個圖形庫")
	cols := flag.Int("cols", 32, "每列幾張")
	scale := flag.Int("scale", 4, "放大倍率")
	limit := flag.Int("n", 0, "只畫前 n 張（0 = 全部）")
	out := flag.String("o", "workplace/pgf/bank.png", "輸出")
	flag.Parse()

	raw, err := os.ReadFile(flag.Arg(0))
	if err != nil { fmt.Println(err); os.Exit(1) }
	g, err := assets.ParsePGF(raw)
	if err != nil { fmt.Println(err); os.Exit(1) }
	fmt.Printf("%s：%d 個圖形庫，%d bpp，風格編號 %d\n", g.Name, len(g.Banks), g.BitsPerPixel, g.StyleID)
	for i, b := range g.Banks {
		fmt.Printf("  #%2d %3d×%-3d ×%4d  旗標 0x%04x\n", i, b.Width, b.Height, len(b.Images), b.Flags)
	}
	if *bank >= len(g.Banks) { return }
	b := g.Banks[*bank]
	n := len(b.Images)
	if *limit > 0 && *limit < n { n = *limit }
	rows := (n + *cols - 1) / *cols
	img := image.NewRGBA(image.Rect(0, 0, *cols*b.Width**scale, rows*b.Height**scale))
	for i := 0; i < n; i++ {
		ox, oy := (i%*cols)*b.Width**scale, (i / *cols)*b.Height**scale
		for y := 0; y < b.Height; y++ {
			for x := 0; x < b.Width; x++ {
				c := g.Palette[b.Images[i].Pixels[y*b.Width+x]]
				for sy := 0; sy < *scale; sy++ {
					for sx := 0; sx < *scale; sx++ {
						img.Set(ox+x**scale+sx, oy+y**scale+sy, color.RGBA{c.R, c.G, c.B, 255})
					}
				}
			}
		}
	}
	f, _ := os.Create(*out)
	png.Encode(f, img)
	f.Close()
	fmt.Println("寫入", *out)
	_ = binary.LittleEndian
}
