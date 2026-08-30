package sim

// 查詢工具：一格的五項狀態。一手出處：w_tool.c:732 getDensityStr、
// w_tool.c:769 doZoneStatus。
//
// 原版把結果做成一個小面板（人口密度、地價、犯罪、汙染、成長率五列），
// 按住 `Q` ＋ 左鍵時顯示，放開就收——版面量測見 docs/spec/ui-layout.md。
//
// ⚠ 五項的取樣解析度**各不相同**：人口密度、地價、犯罪、汙染是半解析
// （`[x>>1][y>>1]`），成長率是八分之一（`[x>>3][y>>3]`）。
// 全部用同一個除法會讓成長率整片相同，而畫面上看起來完全正常。

// 查詢面板的五個欄位。順序即原版的列序。
const (
	QueryDensity   = 0 // 人口密度
	QueryValue     = 1 // 地價
	QueryCrime     = 2 // 犯罪
	QueryPollution = 3 // 汙染
	QueryGrowth    = 4 // 成長率
)

// QueryBuckets 回傳一格五個欄位的**分級編號**（0–19）。
//
// 分級表是連號的：0–3 密度、4–7 地價、8–11 犯罪、12–15 汙染、16–19 成長。
// 這個編號直接對應原版的字串資源（Micropolis 的 `stri.202`），
// 呈現層照編號查字。
func (w *World) QueryBuckets(x, y int) [5]int {
	var out [5]int
	if !InBounds(x, y) {
		return out
	}
	hx, hy := x>>1, y>>1

	// 密度：取高兩位。w_tool.c:737
	out[QueryDensity] = int(w.PopDensity[hx][hy]>>6) & 3

	// 地價：三個門檻切四級。w_tool.c:742
	switch v := int(w.LandValueMem[hx][hy]); {
	case v < 30:
		out[QueryValue] = 4
	case v < 80:
		out[QueryValue] = 5
	case v < 150:
		out[QueryValue] = 6
	default:
		out[QueryValue] = 7
	}

	// 犯罪：同密度的取法。w_tool.c:748
	out[QueryCrime] = (int(w.CrimeMem[hx][hy]>>6) & 3) + 8

	// 汙染：**有一個特例**——`0 < z < 64` 回 13（第二級）而不是 12。
	// 所以「完全沒有汙染」與「一點點汙染」分得出來。w_tool.c:752
	z := int(w.PollutionMem[hx][hy])
	if z < 64 && z > 0 {
		out[QueryPollution] = 13
	} else {
		out[QueryPollution] = ((z >> 6) & 3) + 12
	}

	// 成長率：八分之一解析度，四級。w_tool.c:759
	switch r := int(w.RateOGMem[x>>3][y>>3]); {
	case r < 0:
		out[QueryGrowth] = 16
	case r == 0:
		out[QueryGrowth] = 17
	case r > 100:
		out[QueryGrowth] = 19
	default:
		out[QueryGrowth] = 18
	}
	return out
}
