package autoplay

import "github.com/wicanr2/chengshi_cht/internal/sim"

// 開局：一張白紙上怎麼起第一座城。
//
// ⚠ **這件事原本整個沒做。** 自動玩家只跑過劇本，而劇本一律是「接手一座
// 既有城市」——有路、有電、有分區。真的開一座新城市讓它玩五十年，結果是
// **一格都沒蓋、資金原封不動 20 000**：`growSites` 要「挨著路又在電網內」，
// 白紙上兩個條件都不成立；`power()` 要「有分區沒電」才動手，而一個分區都沒有。
// 三個條件互相等待，永遠不會有人先動。
//
// CLAUDE.md §4 把「從零開始蓋到一座能自我維持的城市」列為正常玩家路徑，
// 而那條路徑一次都沒被走過——因為驗收只跑劇本。

// bootstrapRoad 是開局那條路的長度。夠讓兩側各放得下兩三個 3×3 分區。
const bootstrapRoad = 12

// bootstrap 在空白地圖上起頭：電廠 ＋ 一條路 ＋ 路上的電線。
//
// 之後的成長交給 `grow`——它要的「挨著路又有電」這時候成立了。
// 回傳 false 代表找不到夠平的地，下一年再試。
func (p *Player) bootstrap() bool {
	w := p.w
	cx, cy := sim.WorldX/2, sim.WorldY/2
	// 電廠 4×4，點擊點在 (1,1)。從中心往外找一塊空地。
	px, py, ok := p.vacant4x4Near(cx, cy)
	if !ok {
		return false
	}
	if w.ApplyTool(sim.ToolCoalPower, px, py) != sim.ToolOK {
		return false
	}
	// 從電廠右緣往右鋪路，並在同一條路上架電線。
	//
	// ⚠ 電線非架不可。道路本身**不導電**（只有架了電線的路才導電），
	// 只鋪路的話新分區永遠不在電網內，`growSites` 一格都挑不到。
	x, y := px+3, py
	for i := 0; i < bootstrapRoad && w.TotalFunds > 500; i++ {
		if !sim.InBounds(x, y) || !vacant(w.Map[x][y]) {
			break
		}
		if w.ApplyTool(sim.ToolRoad, x, y) != sim.ToolOK {
			break
		}
		w.ApplyTool(sim.ToolWire, x, y)
		x++
	}
	return true
}

// vacant4x4Near 由近而遠找一塊 4×4 的空地。
func (p *Player) vacant4x4Near(cx, cy int) (int, int, bool) {
	for r := 0; r < 40; r++ {
		for i := -r; i <= r; i++ {
			for _, c := range [][2]int{{cx + i, cy - r}, {cx + i, cy + r},
				{cx - r, cy + i}, {cx + r, cy + i}} {
				x, y := c[0], c[1]
				if !sim.InBounds(x-1, y-1) || !sim.InBounds(x+2, y+2) {
					continue
				}
				if p.blockVacant4x4(x, y) {
					return x, y, true
				}
			}
		}
	}
	return 0, 0, false
}

// blockVacant4x4 判斷 4×4 是不是全部空地（土、樹、瓦礫）。
func (p *Player) blockVacant4x4(x, y int) bool {
	for i := -1; i <= 2; i++ {
		for j := -1; j <= 2; j++ {
			if !vacant(p.w.Map[x+i][y+j]) {
				return false
			}
		}
	}
	return true
}

// zoneCount 數地圖上有幾個分區。
func (p *Player) zoneCount() int {
	n := 0
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			if p.w.Map[x][y]&sim.ZONEBIT != 0 {
				n++
			}
		}
	}
	return n
}

// countTile 數地圖上有幾個中心格是指定圖塊（警局、消防隊、電廠…）。
//
// ⚠ 不要用 `w.PolicePop`／`w.FireStPop` 之類的普查計數：那些是
// `ClearCensus` 每輪歸零、`MapScan` 十六個相位累加的中間值，
// 在年界取樣**永遠是 0**——用它來判斷「蓋了幾座」會永遠判成沒蓋。
func (p *Player) countTile(tile int) int {
	n := 0
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			c := p.w.Map[x][y]
			if c&sim.ZONEBIT != 0 && int(c&sim.LOMASK) == tile {
				n++
			}
		}
	}
	return n
}
