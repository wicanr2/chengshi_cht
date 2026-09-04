package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 查詢面板。原版按住 `Q` ＋ **按住**左鍵時顯示，放開就收
// （參考附表寫「Q（配合左滑鼠鈕）」，實測見
// workplace/dosbox/uq4-60-query-held.png）。
//
// ⚠ **點一下是看不到的**：資料只在按住的時候顯示。前兩次對拍都用「點一下」，
// 結果什麼都沒出現，一度以為查詢功能要靠別的操作。
//
// 版面量自原版：面板在編輯視窗左下角，x 8–175、y 208–321，
// 亮青底、藍字、白框。

const (
	queryX, queryY = 8, 208
	queryW, queryH = 168, 114
)

// queryWords 是五個欄位的分級用字，索引就是 sim.QueryBuckets 的編號。
//
// ⚠ **這二十個字不在任何 `.PTF` 裡**，是硬編碼在執行檔中的（解壓映像
// `0x02548D` 起，二十個 NUL 結尾字串）。所以它們屬於 CLAUDE.md §3.2
// 講的「第三個翻譯來源」。原文與分組：
//
//	Sparse, Low,    Medium, High     ← 密度
//	Low,    Medium, High,   High!    ← 地價
//	Little, Some,   Much,   Severe   ← 犯罪
//	Little, Some,   Much,   Severe   ← 汙染
//	Loss,   None,   Some,   Rapid    ← 成長
//
// ⚠ **與 Micropolis 的字串資源 202 用字完全不同**（那邊是
// `Low／Medium／High／Very High`、`Slum／Lower Class／…`、
// `Safe／Light／Moderate／Dangerous`…）。分級的**程式碼**兩邊一樣，
// 但**用字**是 DOS 版自己的。本專案重現的是 DOS 版，所以照 DOS 的。
//
// 對照驗證：原版截圖那一格是 `Sparse／High／Little／Little／None`，
// 用這張表與 sim.QueryBuckets 算出來的編號是 0／6／8／12／17，逐項對得上
// （workplace/dosbox/uq4-60-query-held.png）。
// 譯文在 `internal/i18n/messages/ui.tsv`，這裡只留鍵。
var queryWords = [20]string{
	// 0–3 人口密度：Sparse Low Medium High
	"q_dens0", "q_dens1", "q_dens2", "q_dens3",
	// 4–7 地價：Low Medium High High!
	"q_val0", "q_val1", "q_val2", "q_val3",
	// 8–11 犯罪：Little Some Much Severe
	"q_crime0", "q_crime1", "q_crime2", "q_crime3",
	// 12–15 汙染：Little Some Much Severe
	"q_poll0", "q_poll1", "q_poll2", "q_poll3",
	// 16–19 成長率：Loss None Some Rapid
	"q_grow0", "q_grow1", "q_grow2", "q_grow3",
}

// drawQueryPanel 畫查詢面板。
func (g *Game) drawQueryPanel(dst *ebiten.Image) {
	if !g.querying {
		return
	}
	x, y := g.queryTX, g.queryTY
	if !sim.InBounds(x, y) {
		return
	}
	fill(dst, queryX-2, queryY-2, queryW+4, queryH+4, colInkLight)
	fill(dst, queryX, queryY, queryW, queryH, colMenuBar)

	name := g.txt.S(i18n.SecTileName, tileNameIndex(g.world.TileNum(x, y)))
	if name == "" {
		name = "？？？"
	}
	ink := color.RGBA{0x55, 0x55, 0xff, 0xff}
	g.font.Draw(dst, name,
		(queryX+queryW/2)*UIScale-g.font.Measure(name)/2, (queryY+4)*UIScale, ink)

	b := g.world.QueryBuckets(x, y)
	for i := 0; i < 5; i++ {
		row := (queryY + 22 + i*17) * UIScale
		g.font.Draw(dst, trimMenu(g.txt.S(i18n.SecQuery, i)), (queryX+4)*UIScale, row, ink)
		w := g.txt.UI(queryWords[b[i]])
		g.font.Draw(dst, w, (queryX+queryW-6)*UIScale-g.font.Measure(w), row, ink)
	}
}
