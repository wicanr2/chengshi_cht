package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// 原版介面上的兩個小記號。它們不是中文字，也不在中文字型的圖集裡，
// 所以直接用**量自原版畫面的位元組**畫。
//
// 量法：把 DOS 原版的截圖換算回 640×350 的內容座標，逐像素讀出來
// （workplace/dosbox/t2-00-default.png）。兩個都對得上 CP437：
//
//	glyphSun  ＝ 0x0F（太陽記號），編輯視窗標題列左端的關閉記號（8×14 字型）
//	glyphPlus ＝ 0x2B `+`，右下角的改變大小把手（8×8 字型）
//
// ⚠ 不要改成從中文字型畫那兩個記號：那是另一套字面，形狀與大小都不一樣，
// 而且畫出來「看起來也像個記號」——這種錯沒有症狀。
var (
	glyphSun  = []byte{0x18, 0x18, 0xdb, 0x3c, 0xe7, 0x3c, 0xdb, 0x18, 0x18}
	glyphPlus = []byte{0x18, 0x18, 0x7e, 0x18, 0x18}
)

// drawGlyph8 在原版座標 (x, y) 畫一個八像素寬的點陣記號，放大 UIScale 倍。
func drawGlyph8(dst *ebiten.Image, bits []byte, x, y int, c color.RGBA) {
	for row, b := range bits {
		for col := 0; col < 8; col++ {
			if b&(0x80>>uint(col)) == 0 {
				continue
			}
			fill(dst, x+col, y+row, 1, 1, c)
		}
	}
}
