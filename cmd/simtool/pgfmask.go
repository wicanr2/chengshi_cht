package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// cmdPgfMask 驗「遮罩庫（旗標 0x0100）與前一庫配對」這件事。
//
// 版面上看得出來配對：庫 10–23 是一美術一遮罩交錯，尺寸與張數兩兩相同。
// 但**怎麼用**要有證據：遮罩的 1 是「透明」還是「不透明」？
// 這支工具算兩件事：
//
//   - 遮罩位元與「美術是色號 0」的一致率。相符率高就代表遮罩標的是背景。
//   - 把美術照遮罩去背合成到棋盤格上，倒成 PNG 用眼睛確認。
func cmdPgfMask(args []string) {
	fs := flag.NewFlagSet("pgfmask", flag.ExitOnError)
	in := fs.String("file", "", "圖形檔（.PGF）")
	out := fs.String("out", "gfx", "輸出目錄")
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
	os.MkdirAll(*out, 0o755)
	pal := make([]color.RGBA, 256)
	for i, c := range g.Palette {
		pal[i] = color.RGBA{c.R, c.G, c.B, 255}
	}
	// 先印每一庫的像素值分布：全 0 或全 1 的庫不會是遮罩。
	for i, b := range g.Banks {
		hist := map[uint8]int{}
		for _, im := range b.Images {
			for _, v := range im.Pixels {
				hist[v]++
			}
		}
		tot := 0
		for _, n := range hist {
			tot += n
		}
		fmt.Printf("庫 %2d %3d×%-3d ×%-3d 旗標 %#04x  相異色號 %2d",
			i, b.Width, b.Height, len(b.Images), b.Flags, len(hist))
		if len(hist) <= 3 {
			for v, n := range hist {
				fmt.Printf("  色%d %.0f%%", v, pct(n, tot))
			}
		}
		fmt.Println()
	}

	// 單平面而且沒有配對的庫，直接倒成黑白 PNG 看內容。
	for i, b := range g.Banks {
		if b.Flags&0x0100 == 0 {
			continue
		}
		if i > 0 {
			a := g.Banks[i-1]
			if a.Flags&0x0100 == 0 && a.Width == b.Width && a.Height == b.Height &&
				len(a.Images) == len(b.Images) {
				continue // 這一庫是前一庫的遮罩，另外處理
			}
		}
		img := image.NewRGBA(image.Rect(0, 0, b.Width*len(b.Images), b.Height))
		for k, im := range b.Images {
			for y := 0; y < b.Height; y++ {
				for x := 0; x < b.Width; x++ {
					c := color.RGBA{0, 0, 0, 255}
					if im.Pixels[y*b.Width+x] != 0 {
						c = color.RGBA{255, 255, 255, 255}
					}
					img.Set(k*b.Width+x, y, c)
				}
			}
		}
		f, err := os.Create(filepath.Join(*out, fmt.Sprintf("mono%02d.png", i)))
		if err == nil {
			png.Encode(f, img)
			f.Close()
			fmt.Printf("寫出 mono%02d.png（%d×%d ×%d）\n", i, b.Width, b.Height, len(b.Images))
		}
	}

	const maskFlag = 0x0100
	for i := 1; i < len(g.Banks); i++ {
		m := g.Banks[i]
		if m.Flags&maskFlag == 0 {
			continue
		}
		a := g.Banks[i-1]
		if a.Width != m.Width || a.Height != m.Height || len(a.Images) != len(m.Images) {
			fmt.Printf("庫 %d 是遮罩，但前一庫 %d×%d ×%d 對不上（%d×%d ×%d）\n",
				i, a.Width, a.Height, len(a.Images), m.Width, m.Height, len(m.Images))
			continue
		}
		// 一致率：遮罩位元 1 的地方，美術是不是色號 0。
		var one0, one1, zero0, zero1 int
		for k := range a.Images {
			ap, mp := a.Images[k].Pixels, m.Images[k].Pixels
			for j := range ap {
				switch {
				case mp[j] != 0 && ap[j] == 0:
					one0++
				case mp[j] != 0 && ap[j] != 0:
					one1++
				case mp[j] == 0 && ap[j] == 0:
					zero0++
				default:
					zero1++
				}
			}
		}
		tot := one0 + one1 + zero0 + zero1
		fmt.Printf("庫 %2d（遮罩）配 %2d：%3d×%-3d ×%-2d  "+
			"遮罩1∧美術0 %5.1f%%　遮罩1∧美術≠0 %5.1f%%　遮罩0∧美術0 %5.1f%%\n",
			i, i-1, a.Width, a.Height, len(a.Images),
			pct(one0, tot), pct(one1, tot), pct(zero0, tot))
		writeMasked(filepath.Join(*out,
			fmt.Sprintf("mask%02d-00.png", i)), &a, &m, 0, pal)
	}
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) * 100 / float64(d)
}

// writeMasked 把第 k 張美術照遮罩去背，合成到洋紅底上。
// 洋紅不是原版的顏色，是**故意**選的：透明處理錯了會整片洋紅，一眼看得出來。
func writeMasked(path string, a, m *assets.PGFBank, k int, pal []color.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, a.Width*2, a.Height))
	ap, mp := a.Images[k].Pixels, m.Images[k].Pixels
	bg := color.RGBA{0xff, 0x00, 0xff, 0xff}
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			j := y*a.Width + x
			// 左：原始美術。右：照遮罩去背之後。
			img.Set(x, y, pal[ap[j]])
			c := bg
			if mp[j] == 0 {
				c = pal[ap[j]]
			}
			img.Set(a.Width+x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	png.Encode(f, img)
}
