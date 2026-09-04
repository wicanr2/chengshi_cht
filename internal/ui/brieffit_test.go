package ui

import (
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
)

// 圖片訊息與劇本簡介要放得進那個對話框。
//
// 這一條沒有測試釘著的話會**靜靜地溢出**：太寬的那一行畫到框線外面去，
// 太多行的最後一句被「繼續」按鈕蓋掉。兩種都不會讓程式出錯，也不會讓
// 別的測試變紅，而且只有切到那個語言、玩到那個劇本才看得到。
//
// 判準用原版像素：框寬 briefW=304，半形一格 8、全形 16，所以一行最多 38 個
// 半形寬。行距 15，第一行的字底在 92，按鈕上緣在 briefButtonY=222，
// 所以最多 (222-92)/15 = 8 行。
func TestPictureMessagesFitTheBriefDialog(t *testing.T) {
	const (
		maxHalfWidths = briefW / 8
		maxLines      = (briefButtonY - 92) / 15
	)
	for _, style := range []string{"", "asia", "medi", "west", "fusa", "feur", "moon"} {
		for _, lang := range []i18n.Lang{i18n.ZhHant, i18n.ZhHans, i18n.Ja} {
			c, err := i18n.LoadLang(style, lang)
			if err != nil {
				t.Fatalf("%q/%s：%v", style, lang, err)
			}
			for idx := 0; c.Has(i18n.SecPicture, idx); idx++ {
				lines := splitLines(c.S(i18n.SecPicture, idx))
				if len(lines) > maxLines {
					t.Errorf("%q %s 第 2 段第 %d 筆有 %d 行，最多 %d 行——最後一句會被按鈕蓋掉",
						style, lang, idx, len(lines), maxLines)
				}
				for i, ln := range lines {
					if w := halfWidths(ln); w > maxHalfWidths {
						t.Errorf("%q %s 第 2 段第 %d 筆第 %d 行寬 %d，最多 %d：%s",
							style, lang, idx, i+1, w, maxHalfWidths, ln)
					}
				}
			}
		}
	}
}

// halfWidths 用半形格數量一行的寬度。全形判定與 tools/build_font.py 的
// is_wide 同一套範圍——量錯邊會讓這支測試放行真正會溢出的行。
func halfWidths(s string) int {
	n := 0
	for _, r := range s {
		if isWideRune(r) {
			n += 2
		} else {
			n++
		}
	}
	return n
}

func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F,
		r >= 0x2E80 && r <= 0xA4CF,
		r >= 0xAC00 && r <= 0xD7A3,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE30 && r <= 0xFE6F,
		r >= 0xFF00 && r <= 0xFF60,
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x3000 && r <= 0x303F:
		return true
	}
	return false
}

