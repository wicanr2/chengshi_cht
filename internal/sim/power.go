package sim

// 電力傳導。證據：docs/re/05-power-scan.md／規格：docs/spec/power.md
// 一手出處：s_power.c、s_zone.c:624 SetZPower、s_sim.c:1014 DoSPZone
//
// 傳導是一個從電廠出發的堆疊式泛洪：沿著帶 CONDBIT 的格子走，
// 走到分岔（同時有兩個以上未通電的鄰居）就把目前位置壓進堆疊。

// 電力相關的容量常數。headers/sim.h:194-195
const (
	// PwrMapSize 是電力位元圖的 word 數。POWERMAPROW * WORLD_Y
	PwrMapSize = PowerMapRow * WorldY // 800
	// PwrStkSize 是傳導堆疊的容量。(WORLD_X * WORLD_Y) / 4
	PwrStkSize = (WorldX * WorldY) / 4 // 3000
)

// 每座電廠能供應的格數。s_power.c:196
//
// 註解寫 `post release`，也就是發行後才加上的上限。超過就送 40 號訊息
// （「電力不足」）並中止整個掃描——**中止之後剩下的電線一格都不會通電**。
const (
	CoalPowerCapacity    = 700
	NuclearPowerCapacity = 2000
)

// powerScan 是傳導的可重入狀態。原版全部是全域變數（s_power.c:68-70）。
type powerScan struct {
	w          *World
	stackX     [PwrStkSize]int
	stackY     [PwrStkSize]int
	stackNum   int
	sMapX      int
	sMapY      int
	cChr9      int // MapScan 留下的「目前圖塊」，見 docs/re/05-power-scan.md §3
	maxPower   int
	numPower   int
	OutOfPower bool // 對應 SendMes(40)
}

// moveMapSim 把游標往某個方向移一格，撞到邊界回 false。
// s_power.c:74 MoveMapSim(short MDir)
//
// 方向編號：0 上、1 右、2 下、3 左、4 原地不動（回 true）。
// 原版的 case 4 直接回 TRUE 而不移動——DoPowerScan 的第一步就是用它，
// 讓剛從堆疊取出的位置自己先通電。
func (p *powerScan) moveMapSim(dir int) bool {
	switch dir {
	case 0:
		if p.sMapY > 0 {
			p.sMapY--
			return true
		}
		if p.sMapY < 0 {
			p.sMapY = 0
		}
		return false
	case 1:
		if p.sMapX < WorldX-1 {
			p.sMapX++
			return true
		}
		if p.sMapX > WorldX-1 {
			p.sMapX = WorldX - 1
		}
		return false
	case 2:
		if p.sMapY < WorldY-1 {
			p.sMapY++
			return true
		}
		if p.sMapY > WorldY-1 {
			p.sMapY = WorldY - 1
		}
		return false
	case 3:
		if p.sMapX > 0 {
			p.sMapX--
			return true
		}
		if p.sMapX < 0 {
			p.sMapX = 0
		}
		return false
	case 4:
		return true
	}
	return false
}

// testForCond 回報某個方向的鄰居「導電而且還沒通電」。
// s_power.c:149 TestForCond(short TFDir)
//
// ⚠ 那個 `CChr9 != NUCLEAR && CChr9 != POWERPLANT` 的判斷用的是
// **MapScan 留下來的全域**，不是鄰居的圖塊——原版把 TestPowerBit() 內聯進來時
// 把它一起帶進來了。見 docs/re/05-power-scan.md §3。
func (p *powerScan) testForCond(dir int) bool {
	xs, ys := p.sMapX, p.sMapY
	ok := false
	if p.moveMapSim(dir) {
		if p.w.Map[p.sMapX][p.sMapY]&CONDBIT != 0 &&
			p.cChr9 != NUCLEAR && p.cChr9 != POWERPLANT {
			word, mask := PowerWord(p.sMapX, p.sMapY)
			if word > PwrMapSize || p.w.PowerMap[word]&mask == 0 {
				ok = true
			}
		}
	}
	p.sMapX, p.sMapY = xs, ys
	return ok
}

func (p *powerScan) push() {
	if p.stackNum < PwrStkSize-2 {
		p.stackNum++
		p.stackX[p.stackNum] = p.sMapX
		p.stackY[p.stackNum] = p.sMapY
	}
}

func (p *powerScan) pull() {
	if p.stackNum > 0 {
		p.sMapX = p.stackX[p.stackNum]
		p.sMapY = p.stackY[p.stackNum]
		p.stackNum--
	}
}

// run 是 s_power.c:186 DoPowerScan()。
func (p *powerScan) run(coalPop, nuclearPop int) {
	for i := range p.w.PowerMap {
		p.w.PowerMap[i] = 0
	}
	p.maxPower = coalPop*CoalPowerCapacity + nuclearPop*NuclearPowerCapacity
	p.numPower = 0

	for p.stackNum != 0 {
		p.pull()
		aDir := 4
		for {
			p.numPower++
			if p.numPower > p.maxPower {
				// SendMes(40)：電力不足。整個掃描中止，剩下的電線一格都不通。
				p.OutOfPower = true
				return
			}
			p.moveMapSim(aDir)
			word, mask := PowerWord(p.sMapX, p.sMapY)
			p.w.PowerMap[word] |= mask

			conNum, dir := 0, 0
			for dir < 4 && conNum < 2 {
				if p.testForCond(dir) {
					conNum++
					aDir = dir
				}
				dir++
			}
			if conNum > 1 {
				p.push()
			}
			if conNum == 0 {
				break
			}
		}
	}
}

// PowerScanResult 是一次電力傳導的統計。
type PowerScanResult struct {
	CoalPop    int  // 燃煤電廠座數
	NuclearPop int  // 核電廠座數
	Powered    int  // 帶 CONDBIT 且通電的格數
	OutOfPower bool // 是否因為超過供電上限而中止
}

// DoPowerScan 重算電力網並把結果寫回每一格的 PWRBIT。
//
// 這一支把原版分散在三處的行為收攏成一次可驗證的呼叫：
//
//  1. MapScan 掃過全圖時，DoSPZone 為每座電廠 PushPowerStack（s_sim.c:1021、:1031），
//     同時累計 CoalPop／NuclearPop，並在 CChr9 留下最後一格非零圖塊的編號。
//  2. DoPowerScan 從堆疊出發做泛洪，填 PowerMap（s_power.c:186）。
//  3. 下一輪 MapScan 對每個帶 CONDBIT 的格子呼叫 SetZPower，依 PowerMap 設或清 PWRBIT
//     （s_sim.c:713、s_zone.c:624）。
//
// 收攏是安全的，因為原版在地圖不變時會收斂到同一個狀態——驗收就是拿收斂後的
// 原版地圖來比對（docs/re/05-power-scan.md §5）。
func (w *World) DoPowerScan() PowerScanResult {
	p := &powerScan{w: w}
	var res PowerScanResult

	// 第 1 步：照 MapScan 的順序（x 外層、y 內層）掃一遍。
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			cChr := w.Map[x][y]
			if cChr == 0 {
				continue
			}
			cChr9 := int(cChr & LOMASK)
			p.cChr9 = cChr9 // MapScan 每遇到一格非零就更新，掃完留下最後一格
			if cChr9 < FLOOD {
				continue
			}
			if cChr&ZONEBIT == 0 {
				continue
			}
			switch cChr9 {
			case POWERPLANT:
				res.CoalPop++
				p.sMapX, p.sMapY = x, y
				p.push()
			case NUCLEAR:
				res.NuclearPop++
				p.sMapX, p.sMapY = x, y
				p.push()
			}
		}
	}

	// 第 2 步：泛洪。
	p.run(res.CoalPop, res.NuclearPop)
	res.OutOfPower = p.OutOfPower

	// 第 3 步：依 PowerMap 設每一格的 PWRBIT。s_zone.c:624 SetZPower
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			cChr := w.Map[x][y]
			if cChr&CONDBIT == 0 {
				continue
			}
			cChr9 := int(cChr & LOMASK)
			word, mask := PowerWord(x, y)
			if cChr9 == NUCLEAR || cChr9 == POWERPLANT ||
				(word < PwrMapSize && w.PowerMap[word]&mask != 0) {
				w.Map[x][y] = cChr | PWRBIT
				res.Powered++
			} else {
				w.Map[x][y] = cChr &^ uint16(PWRBIT)
			}
		}
	}
	return res
}
