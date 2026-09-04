// promocard 畫推廣影片的字卡，用的是**遊戲自己那套點陣字**
// （`internal/textfont` 的內嵌圖集），所以字卡與實機畫面的字長得一樣。
//
// 不用 ffmpeg 的 drawtext 是有原因的：主機上的 Noto CJK 是 `.ttc` 字型集，
// 繁體那一面是第 3 面，而 drawtext 只吃得到檔案吃不到面索引——拿預設面
// 畫出來的是日文字形，共用碼位的字（直、骨、今）會是錯的字形。
//
// 用法：
//
//	promocard -out card.png -big "城　市" -line "副標" -line "#55FFFF|第二行"
//
// 每一行可以用 `#RRGGBB|` 開頭指定顏色；放不下畫布就直接失敗。
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

	// ⚠ **放不下就失敗**。字卡是一次性算出來的圖，畫超出畫布不會有任何
	// 錯誤——影片編完才會看到最後幾個字被切掉，而那時候要重跑整條管線。
	// 三種語言並排之後，英文那一行最容易超出去（半形一格 24 像素，
	// 960 寬只放得下 40 個）。
	const margin = 24
	for _, s := range ls {
		t := s
		if i := strings.IndexByte(t, '|'); i == 7 && strings.HasPrefix(t, "#") {
			t = t[i+1:]
		}
		if got := a.Measure(t); got > *w-2*margin {
			fmt.Fprintf(os.Stderr, "字卡放不下：%q 寬 %d，上限 %d（%s）\n",
				t, got, *w-2*margin, *out)
			os.Exit(1)
		}
	}
	if *big != "" && a.Measure(*big)*2 > *w-2*margin {
		fmt.Fprintf(os.Stderr, "主標放不下：%q（%s）\n", *big, *out)
		os.Exit(1)
	}

	// 版面：嵌圖、主標與內文整組垂直置中。
	//
	// ⚠ 行距是**算出來的不是寫死的**。三種語言並排之後一張卡最多七行，
	// 用固定的 16 會撐到上下兩條橫線外面去——而那不會報錯，要編完影片
	// 才看得到最後一行被線切過。所以由寬鬆往緊縮試，都放不下才失敗。
	fixed := 0
	if embedded != nil {
		fixed += embedded.Bounds().Dy() + 18
	}
	if *big != "" {
		fixed += a.Height*2 + 28
	}
	avail := *h - 2*(34+3) - 16 // 兩條橫線之間再留一點餘裕
	lineH, total := 0, 0
	for _, gap := range []int{16, 12, 10, 8, 6, 4} {
		lineH = a.Height + gap
		total = fixed + len(ls)*lineH
		if total <= avail {
			break
		}
	}
	if total > avail {
		fmt.Fprintf(os.Stderr, "字卡塞不下：%d 行加主標共 %d 像素，可用 %d（%s）\n",
			len(ls), total, avail, *out)
		os.Exit(1)
	}
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
		c := hex(*fgLine, color.RGBA{0xff, 0xff, 0xff, 0xff})
		// `#RRGGBB|文字` 讓同一張卡的三種語言各有自己的顏色。
		if i := strings.IndexByte(s, '|'); i == 7 && strings.HasPrefix(s, "#") {
			c, s = hex(s[:i], c), s[i+1:]
		}
		drawText(s, (*w-a.Measure(s))/2, y, 1, c)
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
