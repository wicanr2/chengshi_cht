// promocard 畫推廣影片的字卡，用的是**遊戲自己那套點陣字**
// （`internal/textfont` 的內嵌圖集），所以字卡與實機畫面的字長得一樣。
//
// 不用 ffmpeg 的 drawtext 是有原因的：主機上的 Noto CJK 是 `.ttc` 字型集，
// 繁體那一面是第 3 面，而 drawtext 只吃得到檔案吃不到面索引——拿預設面
// 畫出來的是日文字形，共用碼位的字（直、骨、今）會是錯的字形。
//
// 用法：
//
//	promocard -out card.png -big "城　市" -line "副標" -line "第二行"
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/textfont"
)

type lines []string

func (l *lines) String() string     { return strings.Join(*l, "／") }
func (l *lines) Set(s string) error { *l = append(*l, s); return nil }

func hex(s string, def color.RGBA) color.RGBA {
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	if err != nil {
		return def
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
}

func main() {
	out := flag.String("out", "", "輸出的 PNG")
	w := flag.Int("w", 960, "寬")
	h := flag.Int("h", 526, "高")
	bg := flag.String("bg", "000000", "底色")
	fgBig := flag.String("fg-big", "FFFF55", "主標顏色")
	fgLine := flag.String("fg", "FFFFFF", "內文顏色")
	rule := flag.String("rule", "55FFFF", "上下橫線顏色（空字串＝不畫）")
	big := flag.String("big", "", "主標（放大兩倍）")
	imgPath := flag.String("image", "", "嵌一張 PNG（置中，放在文字上方）")
	var ls lines
	flag.Var(&ls, "line", "內文，可重複")
	flag.Parse()
	if *out == "" || (*big == "" && len(ls) == 0) {
		fmt.Fprintln(os.Stderr, "用法：promocard -out card.png [-big 主標] [-line 內文…]")
		os.Exit(2)
	}

	a, err := textfont.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	src := a.Image

	canvas := image.NewRGBA(image.Rect(0, 0, *w, *h))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{hex(*bg, color.RGBA{0, 0, 0, 0xff})},
		image.Point{}, draw.Src)

	// 上下兩條橫線，呼應原版那條青色選單列。
	if *rule != "" {
		rc := hex(*rule, color.RGBA{0x55, 0xff, 0xff, 0xff})
		for _, y := range []int{34, *h - 36} {
			for x := 60; x < *w-60; x++ {
				for dy := 0; dy < 3; dy++ {
					canvas.SetRGBA(x, y+dy, rc)
				}
			}
		}
	}

	// drawText 把一段字畫在 (x, y)，scale 是整數倍放大（點陣字只能整數倍，
	// 非整數倍會把筆畫糊掉——這也是原版畫面本身的規矩）。
	drawText := func(s string, x, y, scale int, c color.RGBA) {
		for _, r := range s {
			g, ok := a.Glyphs[r]
			if !ok {
				x += a.Size * scale
				continue
			}
			sx := (g.Index % a.Cols) * a.Size
			sy := (g.Index / a.Cols) * a.Height
			for gy := 0; gy < a.Height; gy++ {
				for gx := 0; gx < g.Width; gx++ {
					pr, pg, pb, pa := src.At(sx+gx, sy+gy).RGBA()
					// 圖集是灰階遮罩：白＝有筆畫。底色透明與底色全黑兩種
					// 烘法都要吃得下，所以亮度與 alpha 相乘當覆蓋率。
					lum := (pr + pg + pb) / 3
					cov := int(lum>>8) * int(pa>>8) / 255
					if cov == 0 {
						continue
					}
					for by := 0; by < scale; by++ {
						for bx := 0; bx < scale; bx++ {
							px, py := x+gx*scale+bx, y+gy*scale+by
							if !(image.Point{px, py}).In(canvas.Bounds()) {
								continue
							}
							old := canvas.RGBAAt(px, py)
							blend := func(o, n uint8) uint8 {
								return uint8((int(o)*(255-cov) + int(n)*cov) / 255)
							}
							canvas.SetRGBA(px, py, color.RGBA{
								blend(old.R, c.R), blend(old.G, c.G), blend(old.B, c.B), 0xff})
						}
					}
				}
			}
			x += g.Width * scale
		}
	}

	// 嵌圖（六種顯示模式的對照格用這個）。
	var embedded image.Image
	if *imgPath != "" {
		ef, err := os.Open(*imgPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		embedded, err = png.Decode(ef)
		ef.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	// 版面：嵌圖、主標與內文整組垂直置中。
	lineH := a.Height + 16
	total := 0
	if embedded != nil {
		total += embedded.Bounds().Dy() + 18
	}
	if *big != "" {
		total += a.Height*2 + 28
	}
	total += len(ls) * lineH
	y := (*h - total) / 2

	if embedded != nil {
		b := embedded.Bounds()
		at := image.Pt((*w-b.Dx())/2, y)
		draw.Draw(canvas, image.Rectangle{at, at.Add(b.Size())}, embedded, b.Min, draw.Src)
		y += b.Dy() + 18
	}

	if *big != "" {
		drawText(*big, (*w-a.Measure(*big)*2)/2, y, 2, hex(*fgBig, color.RGBA{0xff, 0xff, 0x55, 0xff}))
		y += a.Height*2 + 28
	}
	for _, s := range ls {
		drawText(s, (*w-a.Measure(s))/2, y, 1, hex(*fgLine, color.RGBA{0xff, 0xff, 0xff, 0xff}))
		y += lineH
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, canvas); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s　%d×%d\n", *out, *w, *h)
}
