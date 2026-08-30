package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

func main() {
	raw, _ := os.ReadFile(os.Args[1])
	g, err := assets.ParsePGF(raw)
	if err != nil {
		panic(err)
	}
	b := &g.Banks[0]
	const cols = 30
	rows := (len(b.Images) + cols - 1) / cols
	im := image.NewRGBA(image.Rect(0, 0, cols*b.Width, rows*b.Height))
	for k, img := range b.Images {
		ox, oy := (k%cols)*b.Width, (k/cols)*b.Height
		for y := 0; y < b.Height; y++ {
			for x := 0; x < b.Width; x++ {
				c := g.Palette[img.Pixels[y*b.Width+x]]
				im.Set(ox+x, oy+y, color.RGBA{c.R, c.G, c.B, 255})
			}
		}
	}
	f, _ := os.Create(os.Args[2])
	png.Encode(f, im)
	f.Close()
	println("圖塊", len(b.Images), "尺寸", b.Width, b.Height, "每列", cols)
}
