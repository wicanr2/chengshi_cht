package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// cmdGfx 把一個 `.PGF` 的圖形庫倒成 PNG。
//
// 用途是**看**：圖形庫的用途（哪一庫是工具盤、哪一庫是圖層圖示）
// 只有畫出來才認得出來，長度與尺寸看不出來。
// docs/formats/03-pgf-graphics.md §5 那張表就是這樣認的。
func cmdGfx(args []string) {
	fs := flag.NewFlagSet("gfx", flag.ExitOnError)
	in := fs.String("file", "", "圖形檔（.PGF）")
	out := fs.String("out", "gfx", "輸出目錄")
	bank := fs.Int("bank", -1, "只倒這一庫（預設全部）")
	max := fs.Int("max", 4, "每一庫最多倒幾張（0 ＝ 全部）")
	fs.Parse(args)
	if *in == "" {
		fs.Usage()
		os.Exit(2)
	}
	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	g, err := loadAnyPGF(raw, *in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i, b := range g.Banks {
		if *bank >= 0 && i != *bank {
			continue
		}
		fmt.Printf("庫 %2d  %3d×%-3d ×%-4d 旗標 %#04x\n",
			i, b.Width, b.Height, len(b.Images), b.Flags)
		for j, im := range b.Images {
			if *max > 0 && j >= *max {
				break
			}
			p := filepath.Join(*out, fmt.Sprintf("bank%02d-%02d.png", i, j))
			if err := writePNG(p, g, b, im); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}

// loadAnyPGF 先當風格檔讀，不行再當基本檔。
//
// ⚠ 兩種版面差很多（風格檔有橫幅與集中的圖形庫表，基本檔的表是行內的），
// 而且**基本檔讀不出橫幅時的錯誤看起來像檔案壞掉**，不是像「版面不同」。
func loadAnyPGF(raw []byte, name string) (*assets.PGF, error) {
	if g, err := assets.ParsePGF(raw); err == nil {
		return g, nil
	}
	tile, bpp := 16, 4
	switch {
	case strings.Contains(strings.ToLower(name), "mcga"):
		tile, bpp = 8, 8
	case strings.Contains(strings.ToLower(name), "sega"):
		tile, bpp = 8, 4
	case strings.Contains(strings.ToLower(name), "mono"):
		tile, bpp = 16, 1
	}
	return assets.LoadPGFBase(raw, tile, bpp)
}

func writePNG(path string, g *assets.PGF, b assets.PGFBank, im assets.PGFImage) error {
	img := image.NewRGBA(image.Rect(0, 0, b.Width, b.Height))
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			v := int(im.Pixels[y*b.Width+x])
			if v >= len(g.Palette) {
				v %= len(g.Palette)
			}
			c := g.Palette[v]
			img.Set(x, y, color.RGBA{c.R, c.G, c.B, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
