package main

// 顯示模式的解碼驗證工具。兩種比法：
//
//	go run ./tools/pgfmode shot <截圖> <.PGF> <圖塊編號...>
//	go run ./tools/pgfmode cmp  <A.PGF> <B.PGF>
//
// `shot` 把解出來的圖塊放大兩倍（DOSBox 把 320×200 拉成 640×400），
// 在原版截圖裡滑動搜尋逐像素完全相同的位置。
// `cmp` 拿兩個模式的第 0 庫逐格比色號陣列。
//
// `cmp` 是給「畫面上沒出現的圖塊」用的：同一份美術的兩種編碼，
// 色號陣列應該逐格相同，所以它涵蓋得到樣板比對涵蓋不到的圖塊。

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

func load(p string) *assets.PGF {
	raw, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}
	g, err := assets.ParsePGF(raw)
	if err != nil {
		panic(err)
	}
	return g
}

func main() {
	switch os.Args[1] {
	case "mini":
		// 兩個模式的 City Form 縮圖逐格比色號陣列，判準同 cmp。
		a, b := load(os.Args[2]), load(os.Args[3])
		if a.Mini == nil || b.Mini == nil {
			fmt.Printf("縮圖解不出來：A nil? %v／B nil? %v\n", a.Mini == nil, b.Mini == nil)
			return
		}
		fmt.Printf("A %s 縮圖 %dx%d／B %s 縮圖 %dx%d\n",
			os.Args[2], a.Mini.Width, a.Mini.Height,
			os.Args[3], b.Mini.Width, b.Mini.Height)
		if a.Mini.Width != b.Mini.Width || a.Mini.Height != b.Mini.Height {
			fmt.Println("尺寸不同，不能逐格比")
			return
		}
		same, diff := 0, 0
		for i := 0; i < 960; i++ {
			ta, tb := a.Mini.Tile(i), b.Mini.Tile(i)
			eq := len(ta) == len(tb)
			for j := range ta {
				if !eq || ta[j] != tb[j] {
					eq = false
					break
				}
			}
			if eq {
				same++
			} else {
				diff++
			}
		}
		fmt.Printf("縮圖逐格相同 %d 張、不同 %d 張\n", same, diff)
	case "minidump":
		g := load(os.Args[2])
		if g.Mini == nil {
			fmt.Println("沒有縮圖")
			return
		}
		fmt.Printf("%s 縮圖 %dx%d\n", os.Args[2], g.Mini.Width, g.Mini.Height)
		for _, n := range []int{0, 2, 37, 64} {
			t := g.Mini.Tile(n)
			fmt.Printf("  圖塊 %3d 色號 %v 顏色", n, t)
			for _, v := range t {
				c := g.Palette[v]
				fmt.Printf(" (%d,%d,%d)", c.R, c.G, c.B)
			}
			fmt.Println()
		}
	case "cmp":
		a, b := load(os.Args[2]), load(os.Args[3])
		ba, bb := a.Banks[0], b.Banks[0]
		fmt.Printf("A %s %dx%d %d 張／B %s %dx%d %d 張\n",
			os.Args[2], ba.Width, ba.Height, len(ba.Images),
			os.Args[3], bb.Width, bb.Height, len(bb.Images))
		if ba.Width != bb.Width || ba.Height != bb.Height {
			fmt.Println("尺寸不同，不能逐格比")
			return
		}
		same, diff := 0, 0
		var firstDiff = -1
		for i := 0; i < len(ba.Images) && i < len(bb.Images); i++ {
			eq := true
			for j := range ba.Images[i].Pixels {
				if ba.Images[i].Pixels[j] != bb.Images[i].Pixels[j] {
					eq = false
					break
				}
			}
			if eq {
				same++
			} else {
				diff++
				if firstDiff < 0 {
					firstDiff = i
				}
			}
		}
		fmt.Printf("色號陣列逐格相同 %d 張、不同 %d 張", same, diff)
		if firstDiff >= 0 {
			fmt.Printf("（第一張不同的是 %d）", firstDiff)
		}
		fmt.Println()
	case "shot":
		f, err := os.Open(os.Args[2])
		if err != nil {
			panic(err)
		}
		shot, err := png.Decode(f)
		f.Close()
		if err != nil {
			panic(err)
		}
		g := load(os.Args[3])
		b := g.Banks[0]
		sb := shot.Bounds()
		for _, a := range os.Args[4:] {
			n, _ := strconv.Atoi(a)
			px := b.Images[n].Pixels
			hits := 0
			var first image.Point
			for oy := sb.Min.Y; oy+b.Height*2 <= sb.Max.Y; oy++ {
				for ox := sb.Min.X; ox+b.Width*2 <= sb.Max.X; ox++ {
					ok := true
					for y := 0; y < b.Height*2 && ok; y++ {
						for x := 0; x < b.Width*2; x++ {
							c := g.Palette[px[(y/2)*b.Width+x/2]]
							r0, g0, b0, _ := shot.At(ox+x, oy+y).RGBA()
							if uint8(r0>>8) != c.R || uint8(g0>>8) != c.G || uint8(b0>>8) != c.B {
								ok = false
								break
							}
						}
					}
					if ok {
						if hits == 0 {
							first = image.Pt(ox, oy)
						}
						hits++
					}
				}
			}
			if hits > 0 {
				fmt.Printf("圖塊 %3d：逐像素完全相同的位置 %d 個，第一個在 %v\n", n, hits, first)
			} else {
				fmt.Printf("圖塊 %3d：**畫面上沒有出現**（不是解錯，是沒有涵蓋）\n", n)
			}
		}
	}
}
