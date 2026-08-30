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
// 它有兩種用法，測的是**兩件不一樣的事**：
//
//   - **接手一座既有城市能不能救**：八個劇本（`TestAutoPlayerWinsScenarios`）。
//   - **一張白紙能不能長成一座能自我維持的城市**：`bootstrap.go` ＋
//     `TestFreshCityGrowsAndSustains`。五顆種子五十年後都是等級 2、
//     人口兩萬到三萬、資金為正、還在長。
//
// ⚠ 第二種原本**一次都沒被走過**，因為驗收只跑劇本——而劇本一律有現成的
// 路網與電網。真的開一座新城市，自動玩家五十年一格都不會蓋（見 bootstrap.go）。
//
// 它不是 AI，是一份寫成程式的攻略：每年動一次手，照劇本的過關條件分四種
// 打法。種子 1–5 的實測（每個劇本各跑五顆種子）：
//
//	漢堡 5/5　伯恩 5/5　東京 5/5　波士頓 5/5　里約 5/5
//	底特律 2/5
//	達斯維利 0/5　舊金山 0/5
//
// 合計 27/40。不出手是 5/40。
//
// 兩個過不了的劇本各有各的難處：
//
//   - **達斯維利**是 bootstrap 問題。開局 $5 000、三十年、要加權人口
//     100 000（城市等級 4）。準備金改成按開銷算之後它終於出得了手，
//     但三十年只長到兩萬出頭——每年能蓋的格數受限於稅收，而稅收又受限於
//     人口，複利起不來。要贏得換一套「先鋪格狀路網再密集填分區」的打法。
//   - **舊金山**只有五年。地震把電網打斷，290 個分區裡有 100 多個是暗的
//     ——暗的分區不會成長。`connectDark`（wire.go）把暗區接回來之後，
//     五年結束時的人口從 82 960 拉到 92 720，但過關要 100 000，還差一截。
//
// ⚠ **這支程式最容易寫出「合法但自殺」的操作。** 每一步都是玩家做得到的
// 動作、每一步都回傳 `ToolOK`，而城市在十年內死光。已經踩過三種，
// 三種的共同症狀都是「看起來像城市自然萎縮」：
//
//  1. **推掉自己的電線**（`clearable`）。蓋警局清 3×3 的那一下清掉電廠到
//     市區的那一條線，沒電的分區從 6 個跳到 25 個。
//  2. **撞上供電上限**（`powerTight`）。超過上限 `DoPowerScan` 整個中止，
//     後面的電線一格都不通——2 → 25 → 57，城市六年內死光。
//  3. **鋪路穿過市區**。`AutoBulldoze` 打開時鋪路會把擋路的房子推平，
//     一條 L 形路可以把城市攔腰切掉，人口從兩萬多變成 0。
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

	// 白紙一張的時候要先起頭：電廠 ＋ 一條路 ＋ 路上的電線（bootstrap.go）。
	// 沒有這一步，`grow`／`power` 三個條件互相等待，五十年一格都不會蓋。
	if p.zoneCount() == 0 {
		if !p.bootstrap() {
			return
		}
	}

	p.clearRubble(40)
	p.power() // 沒電就不會長，這一項排在服務前面
	// 電網被打斷的暗區接回來（wire.go）。舊金山地震之後 291 個分區裡
	// 103 個是暗的，蓋再多電廠也接不回來——要的是**接線**不是發電量。
	p.connectDark(20)

	// ⚠ **留準備金。** 破產一次，自動預算就撥不出警消經費
	// （`PoliceEffect`／`FireEffect` 掉下來，評分乘 0.9 兩次），
	// 犯罪與火災跟著失控——實測東京存款歸零那一年評分從 654 掉到 155。
	// 蓋東西是投資，但投到見底就不是投資了。
	spare := w.TotalFunds - p.reserve()
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

// reserve 是這一年不能動用的準備金。
//
// ⚠ **不能寫死。** 寫死 6 000 的那一版讓達斯維利**一手都不出**：
// 它開局只有 $5 000，`spare` 永遠是負的，三十年只坐著收稅。
// 那不是「劇本難」，是策略自己把自己鎖死了。
//
// 按**年度開銷**算：撐得過兩年的維護費就夠了。小城開銷近乎零，準備金落到
// 下限，錢就投得出去；大城開銷高，準備金自然跟著長回六千。
func (p *Player) reserve() int {
	w := p.w
	need := 2 * (w.RoadFund + w.PoliceFund + w.FireFund)
	if need < minReserve {
		need = minReserve
	}
	if need > maxReserve {
		need = maxReserve
	}
	return need
}

// 準備金的上下限。下限要低於任何一個劇本的開局資金（最少的是達斯維利的
// $5 000），否則那個劇本會一手都出不了。
var (
	minReserve = 1500
	maxReserve = 6000
)

// power 檢查有沒有分區沒電；超過一成就補一座燃煤電廠並拉線接上。
//
// 為什麼要單獨處理：沒電的分區**不會成長**（`doResidential` 把 zscore
// 直接設成 −500）。自動玩家一直放新分區卻沒發電量的話，放再多都是空的。
func (p *Player) power() {
	// ⚠ **一年可能要蓋不只一座。** 舊金山地震之後有 290 個分區、
	// 2 600 多個導電格，而一座燃煤只供 700 格——差四座。一年一座的話
	// 五年都補不齊，291 個分區裡一直有 100 個是暗的，而畫面上看起來
	// 只是「城市長不回來」。
	// ⚠ 第一座可以動用準備金（沒電會死人），**第二座起不行**。
	// 不設這一條的話一年蓋四座、$12 000 一次花光，舊金山第三年就破產，
	// 分區從 294 個掉到 259 個——暗區是變少了，城市也一起縮了。
	for i := 0; i < 4; i++ {
		if i > 0 && p.w.TotalFunds < p.reserve()+3300 {
			return
		}
		if !p.powerOnce() {
			return
		}
	}
}

// powerOnce 需要的話蓋一座電廠，回傳有沒有蓋。
func (p *Player) powerOnce() bool {
	w := p.w
	// ⚠ 電廠 $3 000，門檻就抓 $3 300，**不受準備金限制**。
	// 寫 9 000 的那一版讓達斯維利一輩子沒蓋過電廠（它的存款從沒到過九千），
	// 八十個分區裡三十七個是暗的——而畫面上只看得到「城市長不大」。
	// 沒電是會死人的，缺警力只是難看。
	if w.TotalFunds < 3300 {
		return false
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
	if zones == 0 || dark == 0 {
		return false
	}
	// ⚠ **要在供電撞上限之前買，不是撞了才買。** 供電一超過上限，
	// `DoPowerScan` 整個中止，後面的電線一格都不通（`internal/sim/power.go`）
	// ——所以沒電的分區不是慢慢爬，是某一年從 2 個跳到 25 個再跳到 57 個，
	// 城市六年內死光。實測（達斯維利，種子 1）人口從兩萬多變成 2 660。
	//
	// ⚠ **暗區多不代表發電量不夠。** 舊金山地震之後 294 個分區裡 106 個是
	// 暗的，而導電格只有 2 879、上限 4 900——電夠得很，斷的是**線**。
	// 按暗區比例蓋電廠的那一版一年蓋四座、$12 000 一次花光，第三年就破產，
	// 分區從 294 個掉到 259 個：暗區是變少了，城市也一起縮了。
	//
	// 所以蓋電廠只看一件事：**供電是不是快到上限**。接線是 `connectDark` 的事。
	if !p.powerTight() {
		return false
	}
	// 電廠 4×4，點擊點在 (1,1)。找離暗區最近的一塊**空地**。
	//
	// ⚠ **判準是空地，不是「推得掉」。** 用 `clearable` 找的那一版在密集
	// 城市裡幾乎永遠找不到位置——4×4 裡只要有一格是分區的非中心格就不算，
	// 而舊金山滿地都是。實測：地震之後 290 個分區裡 102 個是暗的，
	// 而自動玩家五年**一座電廠都沒蓋**，錢全花在警消上。
	//
	// 找得遠沒關係：接線交給 `connectDark` 的 BFS 走，不是拉直線。
	x, y, ok := p.vacant4x4Near(dx, dy)
	if !ok {
		return false
	}
	p.makeRoom4x4(x, y)
	if w.ApplyTool(sim.ToolCoalPower, x, y) != sim.ToolOK {
		return false
	}
	if Debug {
		fmt.Printf("    電廠 (%d,%d)，暗區 %d/%d\n", x, y, dark, zones)
	}
	return true
}

// powerTight 判斷供電是不是快到上限（導電格數 ≥ 上限的九成）。
// 上限是 `coalPop*700 + nuclearPop*2000`（`internal/sim/power.go`），
// 負載是導電格的總數——分區、電線、架了電線的路都算。
func (p *Player) powerTight() bool {
	load, coal, nuke := 0, 0, 0
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			c := p.w.Map[x][y]
			if c&sim.CONDBIT != 0 {
				load++
			}
			if c&sim.ZONEBIT == 0 {
				continue
			}
			switch int(c & sim.LOMASK) {
			case sim.POWERPLANT:
				coal++
			case sim.NUCLEAR:
				nuke++
			}
		}
	}
	capacity := coal*sim.CoalPowerCapacity + nuke*sim.NuclearPowerCapacity
	return load*10 >= capacity*9
}

// isRoad 判斷這一格是不是路面（含上面架了電線的路）。
func isRoad(cell uint16) bool {
	t := int(cell & sim.LOMASK)
	return t >= sim.ROADBASE && t <= sim.LASTROAD
}

// vacant 只認土、樹與瓦礫。水域、分區、道路、電線、鐵軌一律不算。
func vacant(cell uint16) bool {
	t := int(cell & sim.LOMASK)
	return t == sim.DIRT ||
		(t >= sim.TREEBASE && t <= sim.WOODS) ||
		(t >= sim.RUBBLE && t <= sim.LASTRUBBLE)
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
	// ⚠ **警消要配著城市大小蓋，不能一年一座蓋下去。**
	// 一座 $500，維護費更是**每年都要付**——小城市會被自己的警消拖垮：
	// 實測從零開的新城市在第 25 年資金歸零、分區卡在 33 個不動，
	// 錢全花在十四座警局與十四座消防隊上。
	// 一座管得到八分之一解析度的一格（八格見方），二十個分區配一座夠了。
	zones := p.zoneCount()
	if p.countTile(sim.POLICESTATION)*20 >= zones {
		stations = 0
	}
	for i := 0; i < stations && w.TotalFunds > 6500; i++ {
		p.build(sim.ToolPolice, p.bestSites(func(hx, hy int) int {
			return int(w.CrimeMem[hx][hy])*4 - int(w.PoliceMap[hx>>2][hy>>2])
		}, 40))
	}
	if w.TotalFunds > 6500 && p.countTile(sim.FIRESTATION)*20 < zones {
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
	// ⚠ **電線不能推。** 它推得掉，但推掉會把電網切斷——症狀完全不像
	// 「自己拆的」：某一年沒電的分區從 6 個跳到 25 個，人口一路跌到剩幾百，
	// 看起來像「城市自然萎縮」。實測（達斯維利，種子 1，1911 年）：
	// 蓋警局清出 3×3 的那一下，剛好清掉電廠到市區的那一條線。
	//
	// ⚠ 但**路可以推**。連路一起保護的話，密集城市裡幾乎每一塊 3×3 都碰得到
	// 路，`buildable3x3` 一個候選點都給不出來——警局蓋不成，追犯罪與追評分的
	// 劇本全部掉出通關名單（實測 15 個過變成 10 個）。
	if t >= sim.POWERBASE && t <= sim.LASTPOWER ||
		t == sim.HROADPOWER || t == sim.VROADPOWER {
		return false
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
		// ⚠ 這一格的下限也要跟著準備金走。寫死 5 500 的那一版讓達斯維利
		// 幾乎放不下任何分區——它的存款一輩子沒穩定超過 5 500。
		if w.TotalFunds < p.reserve()+300 {
			return
		}
		if i >= len(sites) {
			p.extendRoad()
			sites = p.growSites()
			if i >= len(sites) {
				return
			}
		}
		w.ApplyTool(p.zoneTool(), sites[i][0], sites[i][1])
	}
}

// zoneTool 決定這一格要蓋哪一種分區。
//
// 挑「最渴」的那一種，平手時偏商業。
//
// ⚠ 看過城市等級的公式會很想改成「先蓋商業與工業」：
//
//	CityPop = (ResPop + ComPop×8 + IndPop×8) × 20   （eval.go:83 doPopNum）
//
// 商業與工業在裡面**一格抵住宅八格**，追人口的劇本看的正是這個等級。
// **試過，更差**（達斯維利三顆種子 20 280／17 960／20 420 →
// 14 780／15 040／25 280）：商業與工業要有居民去上班才長得起來，
// 沒有住宅撐著就停在第一級甚至倒退。需求閥本來就把這件事算進去了——
// `RValve` 在達斯維利長期頂在 2000，那是城市在說「我缺房子」。
// **會綁住的是需求，不是選擇**，所以照閥值走才對。
func (p *Player) zoneTool() sim.Tool {
	w := p.w
	switch {
	case w.CValve >= w.RValve && w.CValve >= w.IValve:
		return sim.ToolCommercial
	case w.IValve >= w.RValve && w.IValve >= w.CValve:
		return sim.ToolIndustrial
	}
	return sim.ToolResidential
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
