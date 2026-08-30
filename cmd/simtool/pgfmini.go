package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// cmdPgfMini 把地圖視窗的 960 張縮圖與自帶字型倒成 PNG。
//
// 這是 docs/formats/03-pgf-graphics.md §7 的證據產生器：縮圖的寬度沒有
// 寫在檔案裡，判準之一是「畫出來認得出是圖塊」，所以要看得到。
func cmdPgfMini(args []string) {
	fs := flag.NewFlagSet("pgfmini", flag.ExitOnError)
	in := fs.String("file", "", "圖形檔（.PGF）")
	out := fs.String("out", "gfx", "輸出目錄")
	zoom := fs.Int("zoom", 6, "放大倍率")
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
	if g.Mini == nil {
		fmt.Fprintln(os.Stderr, "這個檔沒有解出地圖縮圖")
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pal := make([]color.RGBA, 256)
	for i, c := range g.Palette {
		pal[i] = color.RGBA{c.R, c.G, c.B, 255}
	}
	m := g.Mini
	fmt.Printf("縮圖 %d×%d，%d 張；自帶字型 %d 份\n",
		m.Width, m.Height, len(m.Pixels)/(m.Width*m.Height), len(g.Fonts))

	// 縮圖排成 40 欄，格與格之間留一像素，不然 3×3 連在一起看不出分界。
	const cols = 40
	rows := (960 + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*(m.Width+1), rows*(m.Height+1)))
	for t := 0; t < 960; t++ {
		px := m.Tile(t)
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				sheet.Set((t%cols)*(m.Width+1)+x, (t/cols)*(m.Height+1)+y,
					pal[px[y*m.Width+x]])
			}
		}
	}
	writeZoom(filepath.Join(*out, "mini.png"), sheet, *zoom)

	for i, f := range g.Fonts {
		const fc = 32
		fr := (f.Count() + fc - 1) / fc
		img := image.NewRGBA(image.Rect(0, 0, fc*f.Width, fr*f.Height))
		for c := 0; c < f.Count(); c++ {
			for y := 0; y < f.Height; y++ {
				for x := 0; x < f.Width; x++ {
					v := f.Pixels[(c*f.Height+y)*f.Width+x]
					col := color.RGBA{0, 0, 0, 255}
					if v != 0 {
						col = color.RGBA{255, 255, 255, 255}
					}
					img.Set((c%fc)*f.Width+x, (c/fc)*f.Height+y, col)
				}
			}
		}
		writeZoom(filepath.Join(*out, fmt.Sprintf("font%dx%d.png", f.Width, f.Height)), img, *zoom)
		fmt.Printf("  字型 %d：%d×%d ×%d 字\n", i, f.Width, f.Height, f.Count())
	}
}

func writeZoom(path string, src *image.RGBA, zoom int) {
	if zoom < 1 {
		zoom = 1
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*zoom, b.Dy()*zoom))
	for y := 0; y < b.Dy()*zoom; y++ {
		for x := 0; x < b.Dx()*zoom; x++ {
			dst.Set(x, y, src.At(x/zoom, y/zoom))
		}
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, dst); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println("寫出", path)
}
