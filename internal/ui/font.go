// Package ui 是呈現層。它認識 Ebiten 與圖形檔，不認識模擬規則——
// 規則全部在 internal/sim，這裡只讀 sim.World 的狀態並把玩家的操作
// 轉成 sim 的工具呼叫。
package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/chengshi_cht/internal/textfont"
)

// Font 是 24×24 的中文點陣字型。
//
// 為什麼是點陣不是 TTF：老遊戲的底圖要用最近鄰整數倍放大才銳利，
// 字如果走向量渲染，同一個畫面會出現兩種質感。而且發行包不必帶字型檔。
// 產生工具是 tools/build_font.py，字集從 translations/ 掃出來。
//
// ⚠ **不要為了塞進原版的小字位而縮小中文。** 中文筆畫多，縮到八像素
// 會糊成一團。正解是把畫布拉高、底圖放大，讓 24×24 有地方站。
type Font struct {
	img  *ebiten.Image
	size int
	cols int
	// glyph 記每個字在圖集裡的序號與**顯示寬度**（全形 24、半形 12）。
	glyph map[rune]glyphInfo
}

type glyphInfo struct {
	index int
	width int
}

// LoadFont 讀進內嵌的字型圖集並轉成 Ebiten 影像。
func LoadFont() (*Font, error) {
	a, err := textfont.Load()
	if err != nil {
		return nil, err
	}
	// 圖集是灰階：亮度就是筆畫的覆蓋率。轉成「白色 ＋ 以亮度當 alpha」，
	// 這樣 ColorScale 染色時背景才是透明的。
	//
	// ⚠ 直接把灰階圖丟給 Ebiten 的話 alpha 全是 255，每個字都會拖著一塊
	// 黑色方塊——而且因為字本身還是看得懂的，很容易被當成「設計就是這樣」。
	b := a.Image.Bounds()
	rgba := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g, _, _, _ := a.Image.At(x, y).RGBA()
			v := uint8(g >> 8)
			rgba.SetRGBA(x, y, color.RGBA{v, v, v, v})
		}
	}
	f := &Font{
		img:   ebiten.NewImageFromImage(rgba),
		size:  a.Size,
		cols:  a.Cols,
		glyph: make(map[rune]glyphInfo, len(a.Glyphs)),
	}
	for r, g := range a.Glyphs {
		f.glyph[r] = glyphInfo{index: g.Index, width: g.Width}
	}
	return f, nil
}

// Size 回傳一個全形字的邊長。
func (f *Font) Size() int { return f.size }

// Measure 算一段文字的像素寬度。半形佔一半。
func (f *Font) Measure(s string) int {
	w := 0
	for _, r := range s {
		if g, ok := f.glyph[r]; ok {
			w += g.width
		} else {
			w += f.size
		}
	}
	return w
}

// Draw 在 (x, y) 畫一段文字，y 是**上緣**不是基線。
//
// 沒烘進圖集的字會被跳過並留一個空位——寧可缺字也不要畫成豆腐塊，
// 因為缺字要靠肉眼發現，而豆腐塊會讓人以為是字型壞了。
// 重烘字型（tools/font.sh）就會補上。
func (f *Font) Draw(dst *ebiten.Image, s string, x, y int, c color.Color) {
	for _, r := range s {
		g, ok := f.glyph[r]
		if !ok {
			x += f.size
			continue
		}
		sx := (g.index % f.cols) * f.size
		sy := (g.index / f.cols) * f.size
		sub := f.img.SubImage(image.Rect(sx, sy, sx+g.width, sy+f.size)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		// 圖集是灰階遮罩（白＝有筆畫），用 ColorScale 染成想要的顏色。
		cr, cg, cb, ca := c.RGBA()
		op.ColorScale.Scale(
			float32(cr)/0xffff, float32(cg)/0xffff,
			float32(cb)/0xffff, float32(ca)/0xffff)
		dst.DrawImage(sub, op)
		x += g.width
	}
}

// DrawCentered 把文字置中畫在寬度 w 的區域裡。
func (f *Font) DrawCentered(dst *ebiten.Image, s string, x, y, w int, c color.Color) {
	f.Draw(dst, s, x+(w-f.Measure(s))/2, y, c)
}
