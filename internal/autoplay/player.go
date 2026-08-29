// Package autoplay 是自動玩家：把劇本從頭玩到勝敗判定。
//
// 為什麼要有它：`TestAllScenariosReachVerdict` 證明的是「判定機制會觸發」，
// **不出手**的跑法只有 1/8 過。那回答不了「這個 remake 玩得贏嗎」——
// 而玩得贏才是 remake 的驗收（CLAUDE.md §4：正常玩家路徑是「從零蓋到
// 一座能自我維持的城市」與「通關八個劇本」）。
//
// 這支自動玩家只用 `sim.ApplyTool` 與稅率欄位，也就是**玩家在畫面上做得到
// 的事**。它不呼叫任何規則層的內部捷徑，不改參數，不給錢。
//
// 它不是 AI，是一份寫成程式的攻略：每年動一次手，照劇本的過關條件分四種
// 打法。種子 1–5 的實測（每個劇本各跑五顆種子）：
//
//	漢堡 5/5　伯恩 5/5　東京 5/5　波士頓 5/5
//	底特律 3/5　里約 2/5
//	達斯維利 0/5　舊金山 0/5
//
// 每顆種子平均過五個，不出手是一個。
//
// 兩個過不了的劇本各有各的難處：
//
//   - **達斯維利**是 bootstrap 問題。開局 $5 000，存款三十年都在
//     3 000–6 000 之間打轉，累積不出資本；三十年只長到 26 740，
//     而過關要加權人口 100 000。
//   - **舊金山**只有五年，而地震把路網打斷了。現在的鋪路策略是
//     「從最近的路拉一條 L 形」，接不回被震斷的主幹道。
package autoplay

import (
	"fmt"
	"sort"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// ScoreWait 是八個劇本的判定等待刻數。s_sim.c:384 的 ScoreWaitTab。
var ScoreWait = [9]int{0, 30 * 48, 5 * 48, 5 * 48, 10 * 48, 5 * 48, 10 * 48, 5 * 48, 10 * 48}

// Debug 打開之後每個動作都會印一行。
var Debug bool

// goal 是這個劇本的過關條件在策略上的分類。`DoScenarioScore` 判的東西
// 不一樣，玩法就不一樣——人類玩家也是照著目標玩，不是照著一套通用手法。
type Goal int

const (
	goalPop     Goal = iota // 1／2／3：城市等級 ≥ 4，也就是加權人口 > 100 000
	goalTraffic             // 4：交通均值 < 80
	goalScore               // 5／7／8：城市評分 > 500
	goalCrime               // 6：犯罪均值 < 60
)

// ScenarioGoal 是八個劇本的目標分類。
var ScenarioGoal = [9]Goal{0, goalPop, goalPop, goalPop, goalTraffic, goalScore, goalCrime, goalScore, goalScore}

// player 是一年動一次手的簡單策略。
// Player 是一年動一次手的簡單策略。
type Player struct {
	w    *sim.World
	goal Goal
	// FixedTax 不為 0 時蓋掉稅率策略，用來掃「這個劇本到底該收幾趴」。
	FixedTax int
}

// New 建一個自動玩家。
func New(w *sim.World, g Goal) *Player { return &Player{w: w, goal: g} }

// Year 是每年的例行動作。呼叫端負責在 CityTime 每跨 48 刻時叫一次。
func (p *Player) Year() { p.year() }

// 每年的例行動作。順序就是優先序：**先顧現金流，再止血，最後才擴張。**
//
// 第一版沒有顧現金流，八個劇本裡有三個把錢花到 0——而錢一沒，自動預算
// 就撥不出警消經費，犯罪從 60 一路衝到 120，分數反而比不出手還低。
// 這一版的規則：稅率隨存款調整，存款不夠就不蓋。
func (p *Player) year() {
	w := p.w
	// 稅率是固定的，只分兩檔。
	//
	// 這一格試過四種寫法，掃了 4%–13% 的實測結果才定下來：
	//   - 「錢少就加稅」的保險條款（<3000 → 12%）**是有害的**。稅率在
	//     評分公式裡是 `ProblemTable[ProbTaxes] = CityTax * 10`，
	//     而評分是 `(256 − 問題總和/3) × 4`——12% 比 6% 直接少 80 分。
	//     里約實測：固定 6% 通關（評分 547），加了保險條款只有 226。
	//   - 看存款變化或 `CashFlow` 的控制器都更差，理由記在 WORKLOG。
	// 掃出來的最佳點很平：4%–9% 之間差不多，10% 以上急速崩壞
	// （城市萎縮），所以取中間的 6%。
	//
	// 追犯罪的底特律例外：它要養得起警力，收 9% 換得起。
	if p.FixedTax != 0 {
		w.CityTax = p.FixedTax
	} else if p.goal == goalCrime {
		w.CityTax = 9
	} else {
		w.CityTax = 6
	}

	p.clearRubble(40)
	p.power() // 沒電就不會長，這一項排在服務前面

	// ⚠ **留準備金。** 破產一次，自動預算就撥不出警消經費
	// （`PoliceEffect`／`FireEffect` 掉下來，評分乘 0.9 兩次），
	// 犯罪與火災跟著失控——實測東京存款歸零那一年評分從 654 掉到 155。
	// 蓋東西是投資，但投到見底就不是投資了。
	const reserve = 6000
	spare := w.TotalFunds - reserve
	if spare <= 0 {
		return
	}
	p.services()
	p.parks(4)
	// 一格分區 100 元，加上整地大約 150。用可動用的錢決定這一年蓋幾格，
	// 而不是固定值——小城蓋不動大城蓋不夠。
	n := spare / 300
	if p.goal != goalPop && n > 6 {
		n = 6
	}
	if n > 20 {
		n = 20
	}
	p.grow(n)
}

// power 檢查有沒有分區沒電；超過一成就補一座燃煤電廠並拉線接上。
//
// 為什麼要單獨處理：沒電的分區**不會成長**（`doResidential` 把 zscore
// 直接設成 −500）。自動玩家一直放新分區卻沒發電量的話，放再多都是空的。
func (p *Player) power() {
	w := p.w
	if w.TotalFunds < 9000 {
		return
	}
	zones, dark := 0, 0
	var dx, dy int
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			c := w.Map[x][y]
			if c&sim.ZONEBIT == 0 {
				continue
			}
			zones++
			if c&sim.PWRBIT == 0 {
				dark++
				dx, dy = x, y
			}
		}
	}
	if zones == 0 || dark*10 < zones {
		return
	}
	// 電廠是 4×4，點擊點在 (1,1)。找沒電那一區附近的空地。
	for r := 3; r < 20 && w.TotalFunds > 3000; r++ {
		for _, d := range [][2]int{{r, 0}, {-r, 0}, {0, r}, {0, -r}, {r, r}, {-r, -r}} {
			x, y := dx+d[0], dy+d[1]
			if !sim.InBounds(x+2, y+2) || !sim.InBounds(x-1, y-1) {
				continue
			}
			if !p.buildable4x4(x, y) {
				continue
			}
			p.makeRoom4x4(x, y)
			if w.ApplyTool(sim.ToolCoalPower, x, y) == sim.ToolOK {
				p.wireTo(x, y, dx, dy)
				return
			}
		}
	}
}

func (p *Player) buildable4x4(x, y int) bool {
	for i := -1; i <= 2; i++ {
		for j := -1; j <= 2; j++ {
			if !sim.InBounds(x+i, y+j) || !clearable(p.w.Map[x+i][y+j]) {
				return false
			}
		}
	}
	return true
}

func (p *Player) makeRoom4x4(x, y int) {
	for pass := 0; pass < 3; pass++ {
		for i := -1; i <= 2; i++ {
			for j := -1; j <= 2; j++ {
				if int(p.w.Map[x+i][y+j]&sim.LOMASK) != sim.DIRT {
					p.w.ApplyTool(sim.ToolBulldozer, x+i, y+j)
				}
			}
		}
	}
}

// wireTo 從電廠拉一條 L 形電線到目標格。
func (p *Player) wireTo(x0, y0, x1, y1 int) {
	step := func(a, b int) int {
		if a < b {
			return 1
		}
		return -1
	}
	x, y := x0, y0
	for x != x1 && p.w.TotalFunds > 100 {
		x += step(x, x1)
		p.w.ApplyTool(sim.ToolWire, x, y)
	}
	for y != y1 && p.w.TotalFunds > 100 {
		y += step(y, y1)
		p.w.ApplyTool(sim.ToolWire, x, y)
	}
}

// clearRubble 推平瓦礫。瓦礫不長東西，而且拉低地價。
func (p *Player) clearRubble(budget int) {
	w := p.w
	for x := 0; x < sim.WorldX && budget > 0; x++ {
		for y := 0; y < sim.WorldY && budget > 0; y++ {
			t := int(w.Map[x][y] & sim.LOMASK)
			if t >= sim.RUBBLE && t <= sim.LASTRUBBLE {
				if w.ApplyTool(sim.ToolBulldozer, x, y) == sim.ToolOK {
					budget--
				}
			}
		}
	}
}

// services 在犯罪最高、又沒有警局覆蓋的地方蓋警局；火災風險最高的地方蓋消防隊。
func (p *Player) services() {
	w := p.w
	// 追犯罪的劇本一年蓋三座警局；其餘一座就好。底特律的犯罪均值要從
	// 120 壓到 60 以下，一年一座十年也追不上。
	stations := 1
	if p.goal == goalCrime {
		stations = 6
	}
	for i := 0; i < stations && w.TotalFunds > 6500; i++ {
		p.build(sim.ToolPolice, p.bestSites(func(hx, hy int) int {
			return int(w.CrimeMem[hx][hy])*4 - int(w.PoliceMap[hx>>2][hy>>2])
		}, 40))
	}
	if w.TotalFunds > 6500 {
		// 有東西可燒、消防覆蓋低的地方。用人口密度當「有東西可燒」的代理。
		p.build(sim.ToolFireStation, p.bestSites(func(hx, hy int) int {
			return int(w.PopDensity[hx][hy])*2 - int(w.FireStMap[hx>>2][hy>>2])
		}, 40))
	}
}

// worstCell 找 score 最高、又放得下 3×3 的那一格，回傳全解析度座標。
//
// score 的座標是半解析度（60×50）；服務覆蓋圖是八分之一解析度（15×13），
// 所以呼叫端要自己 `>>2`。
func (p *Player) bestSites(score func(hx, hy int) int, k int) [][2]int {
	type cand struct {
		s, x, y int
	}
	var all []cand
	for x := 0; x < sim.HWldX; x++ {
		for y := 0; y < sim.HWldY; y++ {
			v := score(x, y)
			if v <= 0 {
				continue
			}
			mx, my := x<<1, y<<1
			if !p.buildable3x3(mx, my) {
				continue
			}
			all = append(all, cand{v, mx, my})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].s > all[j].s })
	out := [][2]int{}
	for i := 0; i < len(all) && i < k; i++ {
		out = append(out, [2]int{all[i].x, all[i].y})
	}
	return out
}

// build 在候選點裡挑第一個真的蓋得起來的。整地要先做，而且要檢查有沒有
// 真的整乾淨——makeRoom 不保證成功（見 clearable 的註解）。
func (p *Player) build(tool sim.Tool, sites [][2]int) bool {
	for _, s := range sites {
		x, y := s[0], s[1]
		p.makeRoom(x, y)
		if p.w.ApplyTool(tool, x, y) == sim.ToolOK {
			if Debug {
				fmt.Printf("    蓋了 %d 號工具於 (%d,%d)\n", tool, x, y)
			}
			return true
		}
	}
	if Debug {
		fmt.Printf("    %d 號工具：%d 個候選點全部蓋不起來\n", tool, len(sites))
	}
	return false
}

// clear3x3 檢查以 (x,y) 為中心的 3×3 是不是全部可蓋（空地或樹）。
func (p *Player) clear3x3(x, y int) bool {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			mx, my := x+dx, y+dy
			if !sim.InBounds(mx, my) {
				return false
			}
			if !cheapToClear(int(p.w.Map[mx][my] & sim.LOMASK)) {
				return false
			}
		}
	}
	return true
}

// cheapToClear：空地、樹、瓦礫。**不含水、道路、分區**——推平那些不是
// 「整地」，是拆別人的城市。
//
// 第一版只認空地與樹，結果在底特律一座警局都蓋不出來（十年下來
// `警局 0`，犯罪從 125 一路留在 120 以上）。密集城市裡沒有那麼多空地，
// 瓦礫倒是滿地都是。
func cheapToClear(t int) bool {
	switch {
	case t == sim.DIRT:
		return true
	case t >= sim.TREEBASE && t <= sim.WOODS5:
		return true
	case t >= sim.RUBBLE && t <= sim.LASTRUBBLE:
		return true
	}
	return false
}

// buildable3x3：九格都推得平就算數（水域不算）。
//
// 這一層比 clear3x3 寬：密集城市裡「乾淨的空地」幾乎不存在，
// 玩家真的要蓋警局時就是拆幾棟房子。第一版只找乾淨空地，
// 底特律十年一座警局都蓋不出來。
func (p *Player) buildable3x3(x, y int) bool {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			mx, my := x+dx, y+dy
			if !sim.InBounds(mx, my) || !clearable(p.w.Map[mx][my]) {
				return false
			}
		}
	}
	return true
}

// clearable：這一格推土機推得掉嗎。
//
// ⚠ **分區的非中心格推不掉。** `Bulldoze` 對 3×3 分區的邊格走
// `checkBigZone` 那一支，而原版在 size == 3 時**只播音效、不放瓦礫**
// （`internal/sim/tool.go` 的註解），所以推了等於沒推。要拆一座分區
// 得點它的中心格（帶 `ZONEBIT` 那格）。
//
// 不知道這件事的話，自動玩家會挑一塊「看起來推得平」的地，推三遍還是
// 有幾格沒清掉，`PlaceZone` 回 `ToolNeedsClear`——而畫面上沒有任何提示，
// 症狀是「底特律十年一座警局都沒蓋成」。
func clearable(cell uint16) bool {
	t := int(cell & sim.LOMASK)
	if t >= sim.RIVER && t <= sim.LASTRIVEDGE {
		return false // 水域
	}
	if t >= sim.RESBASE && t <= sim.LASTZONE {
		return cell&sim.ZONEBIT != 0 // 只有中心格拆得掉
	}
	return true
}

// makeRoom 把以 (x,y) 為中心的 3×3 推平到見土為止。
//
// ⚠ **要推兩遍。** 推平一格分區留下的是瓦礫（`RUBBLE`），不是空地，
// 而 `PlaceZone` 要的是空地——只推一遍的話後面那個 ApplyTool 一定回
// `ToolNeedsClear`，而且沒有任何錯誤訊息，症狀是「十年下來一座警局都沒有」。
func (p *Player) makeRoom(x, y int) {
	for pass := 0; pass < 3; pass++ {
		dirty := false
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if t := int(p.w.Map[x+dx][y+dy] & sim.LOMASK); t != sim.DIRT {
					p.w.ApplyTool(sim.ToolBulldozer, x+dx, y+dy)
					dirty = true
				}
			}
		}
		if !dirty {
			return
		}
	}
}

// parks 在已開發區旁邊的空地種公園：拉地價、壓犯罪，一格 10 塊很便宜。
func (p *Player) parks(n int) {
	w := p.w
	for i := 0; i < n; i++ {
		x, y, ok := p.emptyNearDevelopment()
		if !ok || w.TotalFunds < 6000 {
			return
		}
		w.ApplyTool(sim.ToolPark, x, y)
	}
}

// grow 在有路又有電的地方補分區。三種需求裡挑最渴的那一種。
//
// 蓋不下去的時候會先鋪路——沒有路就沒有可蓋的地，達斯維利只有 119 格路，
// 光靠原有的路面積不夠長到十萬人。
func (p *Player) grow(n int) {
	w := p.w
	sites := p.growSites()
	for i := 0; i < n; i++ {
		if w.TotalFunds < 5500 {
			return
		}
		if i >= len(sites) {
			p.extendRoad()
			sites = p.growSites()
			if i >= len(sites) {
				return
			}
		}
		tool := sim.ToolResidential
		switch {
		case w.CValve >= w.RValve && w.CValve >= w.IValve:
			tool = sim.ToolCommercial
		case w.IValve >= w.RValve && w.IValve >= w.CValve:
			tool = sim.ToolIndustrial
		}
		w.ApplyTool(tool, sites[i][0], sites[i][1])
	}
}

// growSites 列出「挨著路、又在電網覆蓋範圍內」的 3×3 空地，
// **由近而遠依離城市重心排序**。
//
// 為什麼要排序：第一版從 (2,2) 開始掃、取第一個，於是新分區全部塞在
// 地圖左上角——離城市重心遠、地價低、長不起來。城市是從中心長出去的。
func (p *Player) growSites() [][2]int {
	w := p.w
	type cand struct{ d, x, y int }
	var all []cand
	for x := 2; x < sim.WorldX-2; x++ {
		for y := 2; y < sim.WorldY-2; y++ {
			if !p.clear3x3(x, y) || !p.nearRoad(x, y, 2) || !p.nearPower(x, y) {
				continue
			}
			dx, dy := x-w.CCx, y-w.CCy
			all = append(all, cand{dx*dx + dy*dy, x, y})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].d < all[j].d })
	out := [][2]int{}
	for _, c := range all {
		out = append(out, [2]int{c.x, c.y})
	}
	return out
}

// extendRoad 從離重心最近的一塊空地拉一段路接上既有路網。
func (p *Player) extendRoad() {
	w := p.w
	best, bx, by := 1<<30, -1, -1
	for x := 2; x < sim.WorldX-2; x++ {
		for y := 2; y < sim.WorldY-2; y++ {
			if !p.clear3x3(x, y) || p.nearRoad(x, y, 2) || !p.nearPower(x, y) {
				continue
			}
			dx, dy := x-w.CCx, y-w.CCy
			if d := dx*dx + dy*dy; d < best {
				best, bx, by = d, x, y
			}
		}
	}
	if bx < 0 {
		return
	}
	// 往最近的路鋪過去。找最近的路面格子，走 L 形。
	rx, ry, rd := -1, -1, 1<<30
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			t := int(w.Map[x][y] & sim.LOMASK)
			if t < sim.ROADBASE || t > sim.LASTROAD {
				continue
			}
			dx, dy := x-bx, y-by
			if d := dx*dx + dy*dy; d < rd {
				rd, rx, ry = d, x, y
			}
		}
	}
	if rx < 0 {
		return
	}
	step := func(a, b int) int {
		if a < b {
			return 1
		}
		return -1
	}
	x, y := rx, ry
	for x != bx && w.TotalFunds > 5500 {
		x += step(x, bx)
		w.ApplyTool(sim.ToolRoad, x, y)
	}
	for y != by && w.TotalFunds > 5500 {
		y += step(y, by)
		w.ApplyTool(sim.ToolRoad, x, y)
	}
}

func (p *Player) nearRoad(x, y, r int) bool {
	for dx := -r - 1; dx <= r+1; dx++ {
		for dy := -r - 1; dy <= r+1; dy++ {
			mx, my := x+dx, y+dy
			if !sim.InBounds(mx, my) {
				continue
			}
			t := int(p.w.Map[mx][my] & sim.LOMASK)
			if t >= sim.ROADBASE && t <= sim.LASTROAD {
				return true
			}
		}
	}
	return false
}

func (p *Player) nearPower(x, y int) bool {
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			mx, my := x+dx, y+dy
			if !sim.InBounds(mx, my) {
				continue
			}
			if p.w.Map[mx][my]&sim.PWRBIT != 0 {
				return true
			}
		}
	}
	return false
}

// emptyNearDevelopment 找一格挨著已開發地物的空地。
func (p *Player) emptyNearDevelopment() (int, int, bool) {
	w := p.w
	for x := 1; x < sim.WorldX-1; x++ {
		for y := 1; y < sim.WorldY-1; y++ {
			if int(w.Map[x][y]&sim.LOMASK) != sim.DIRT {
				continue
			}
			if p.nearRoad(x, y, 1) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}
