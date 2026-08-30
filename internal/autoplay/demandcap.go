package autoplay

import (
	"fmt"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 需求封頂：模擬層自己會把三種需求壓成 0。
//
// `internal/sim/message.go` 的第 26／28／30 個訊息相位，除了送訊息還設三個旗標：
//
//	ResPop > 500  且沒有體育場 → ResCap
//	IndPop > 70   且沒有海港   → IndCap
//	ComPop > 100  且沒有機場   → ComCap
//
// 而 `SetValves` 的最後三行是「旗標成立就把對應的正需求歸零」
// （`internal/sim/census.go`）。所以那不是建議，是**成長的硬上限**。
//
// ⚠ 這件事在資金與評分上完全看不出來。達斯維利給無限資金、蓋到 385 個分區、
// 跑滿一百二十年，人口一樣在兩萬多震盪——因為 `RValve` 長期被壓在 0。
// 先前把它當成「策略還不夠好」調了好幾輪參數，調的是錯的那一層。
//
// 判準用旗標本身，不用 `StadiumPop`／`PortPop`／`APortPop`：那三個是
// `ClearCensus` 每輪歸零的普查中間值，在年界取樣永遠是 0（見 countTile 的註解）。
// 旗標是模擬層算完之後留下的結論，而且正是玩家在畫面上看到的那三則訊息。
func (p *Player) demandCaps() {
	w := p.w
	// ⚠ **暗區沒控制住就不要解封頂。** 沒電的分區本來就不成長，這時候解開
	// 需求上限買不到任何東西，而錢一旦花掉就補不了電。
	// 實測舊金山種子 1：地震之後 290 個分區裡 102 個沒電，
	// 玩家把 $10 000 拿去蓋機場，電廠就補不下去了，
	// 人口從 92 880 一路掉到 75 980。
	if zones, dark := p.zoneDark(); zones > 0 && dark*10 >= zones {
		return
	}
	// 體育場 $3000／4×4；海港 $5000／4×4；機場 $10000／6×6。
	// 由便宜到貴，錢不夠時先解開最便宜的那道封頂。
	type want struct {
		capped bool
		have   int
		tool   sim.Tool
		size   int
	}
	for _, it := range []want{
		{w.ResCap, p.countTile(sim.STADIUM) + p.countTile(sim.FULLSTADIUM), sim.ToolStadium, 4},
		{w.IndCap, p.countTile(sim.PORT), sim.ToolSeaport, 4},
		{w.ComCap, p.countTile(sim.AIRPORT), sim.ToolAirport, 6},
	} {
		if !it.capped || it.have > 0 {
			continue
		}
		// 這一項是**投資不是開銷**：解不開封頂，後面所有的分區都是白蓋的。
		// 所以它跟第一座電廠一樣可以動用準備金，只留下限——照 `reserve()`
		// 算的話大城市永遠付不起：舊金山的準備金頂到 $6 000，而它的存款
		// 一輩子在 $3 000 上下，`spare` 是負的，`year` 早就 return 了。
		if w.TotalFunds < sim.ToolCost[it.tool]+minReserve {
			continue
		}
		p.buildBig(it.tool, it.size)
	}
}

// buildBig 蓋一座 size×size 的大型建物：由城市重心往外找一塊放得下的地。
//
// ⚠ 不能沿用 `bestSites`＋`build`：那一組寫死 3×3。4×4 與 6×6 的
// 佔地與 `ToolOffset` 都不同，用 3×3 的檢查會挑到放不下的點，
// 而 `PlaceZone` 只回 `ToolNeedsClear`，畫面上沒有任何提示。
func (p *Player) buildBig(tool sim.Tool, size int) bool {
	w := p.w
	for r := 0; r < sim.WorldX; r++ {
		for i := -r; i <= r; i++ {
			for _, c := range [][2]int{{w.CCx + i, w.CCy - r}, {w.CCx + i, w.CCy + r},
				{w.CCx - r, w.CCy + i}, {w.CCx + r, w.CCy + i}} {
				x, y := c[0], c[1]
				if !p.roomFor(x, y, size) {
					continue
				}
				p.makeRoomN(x, y, size)
				if w.ApplyTool(tool, x, y) == sim.ToolOK {
					if Debug {
						fmt.Printf("    蓋了 %d 號工具（%d×%d）於 (%d,%d)\n", tool, size, size, x, y)
					}
					return true
				}
			}
		}
	}
	return false
}

// roomFor 判斷以 (x,y) 為基準點的 size×size 佔地推不推得平。
//
// 佔地的位移照 `sim.ToolOffset`：4×4 與 6×6 的點擊點都在左上角 +1，
// 所以範圍是 −1 … size−2。
func (p *Player) roomFor(x, y, size int) bool {
	for i := -1; i <= size-2; i++ {
		for j := -1; j <= size-2; j++ {
			if !sim.InBounds(x+i, y+j) || !clearable(p.w.Map[x+i][y+j]) {
				return false
			}
		}
	}
	return true
}

// makeRoomN 把 size×size 的佔地推平到見土為止。要推三遍，理由同 makeRoom。
func (p *Player) makeRoomN(x, y, size int) {
	for pass := 0; pass < 3; pass++ {
		for i := -1; i <= size-2; i++ {
			for j := -1; j <= size-2; j++ {
				if int(p.w.Map[x+i][y+j]&sim.LOMASK) != sim.DIRT {
					p.w.ApplyTool(sim.ToolBulldozer, x+i, y+j)
				}
			}
		}
	}
}

// savingForCap 回答「這一年該不該停下來存錢」。
//
// 三樣解封頂的建物裡最貴的機場是 $10 000，而劇本的年結餘常常只有幾百塊。
// 照平常那樣把餘額分給公園與分區，存款永遠摸不到門檻——實測舊金山三十年
// 一座都沒蓋成，而它的商業需求正被 `ComCap` 壓著。
//
// 封頂還在、又付不起的時候，多蓋的分區是空的（需求已經被壓成 0），
// 那些錢等於丟掉。所以停掉公園與新分區，把錢留給封頂。
//
// ⚠ 存款一旦跨過門檻就不再回報 true，所以放不下建物的時候不會卡死：
// `demandCaps` 試過一次、沒放成，下一年存款仍在門檻之上，這裡回 false，
// 支出恢復正常。
func (p *Player) savingForCap() bool {
	w := p.w
	if zones, dark := p.zoneDark(); zones > 0 && dark*10 >= zones {
		return false
	}
	for _, it := range []struct {
		capped bool
		have   int
		tool   sim.Tool
	}{
		{w.ResCap, p.countTile(sim.STADIUM) + p.countTile(sim.FULLSTADIUM), sim.ToolStadium},
		{w.IndCap, p.countTile(sim.PORT), sim.ToolSeaport},
		{w.ComCap, p.countTile(sim.AIRPORT), sim.ToolAirport},
	} {
		if it.capped && it.have == 0 && w.TotalFunds < sim.ToolCost[it.tool]+minReserve {
			return true
		}
	}
	return false
}

// zoneDark 數分區總數與其中沒電的個數。
func (p *Player) zoneDark() (zones, dark int) {
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			c := p.w.Map[x][y]
			if c&sim.ZONEBIT == 0 {
				continue
			}
			zones++
			if c&sim.PWRBIT == 0 {
				dark++
			}
		}
	}
	return
}
