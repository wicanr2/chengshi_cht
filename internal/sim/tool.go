package sim

// 玩家工具：放置、推土、成本。證據：docs/re/15-tools.md／一手出處：w_tool.c
//
// 這是規則層，不是 UI。工具會扣錢、改地圖、觸發自動接線與自動推土；
// UI 只負責把滑鼠座標換算成格子座標再叫這裡。

// Tool 是工具編號。順序與 sim.h:429 的 *State 常數一致——
// CostOf／toolSize／toolOffset 三張表都靠這個順序索引，不能重排。
type Tool int

const (
	ToolResidential Tool = 0
	ToolCommercial  Tool = 1
	ToolIndustrial  Tool = 2
	ToolFireStation Tool = 3
	ToolQuery       Tool = 4
	ToolPolice      Tool = 5
	ToolWire        Tool = 6
	ToolBulldozer   Tool = 7
	ToolRail        Tool = 8
	ToolRoad        Tool = 9
	ToolChalk       Tool = 10
	ToolEraser      Tool = 11
	ToolStadium     Tool = 12
	ToolPark        Tool = 13
	ToolSeaport     Tool = 14
	ToolCoalPower   Tool = 15
	ToolNuclear     Tool = 16
	ToolAirport     Tool = 17
	ToolNetwork     Tool = 18
)

// ToolCost 是各工具的基本造價。w_tool.c:75 CostOf
//
// ⚠ 這是**基本**造價。自動推土開啟時，每推掉一格再加 1 元；
// 橋樑與海底隧道另有更高的價碼（見 connect.go）。
var ToolCost = [19]int{
	100, 100, 100, 500,
	0, 500, 5, 1,
	20, 10, 0, 0,
	5000, 10, 3000, 3000,
	5000, 10000, 100,
}

// ToolSize 是各工具佔的格數（0 代表不是方形建物）。w_tool.c:84
var ToolSize = [19]int{
	3, 3, 3, 3,
	1, 3, 1, 1,
	1, 1, 0, 0,
	4, 1, 4, 4,
	4, 6, 1,
}

// ToolOffset 是點擊點相對於左上角的偏移。w_tool.c:93
var ToolOffset = [19]int{
	1, 1, 1, 1,
	0, 1, 0, 0,
	0, 0, 0, 0,
	1, 0, 1, 1,
	1, 1, 0,
}

// 工具的回傳碼。w_tool.c 各 *_tool 函式。
const (
	ToolOK          = 1  // 做了
	ToolBlocked     = 0  // 這裡放不了（但不是錯誤，例如同一格重複拉電線）
	ToolNeedsClear  = -1 // 出界，或格子沒清乾淨
	ToolNoMoney     = -2
	ToolNeedsPermit = -3 // 多人模式下的昂貴建物要投票
)

// tally 判斷一格能不能被自動推土機清掉。w_tool.c:279
//
// ⚠ 三段範圍：河岸到瓦礫、電線的一小段、以及爆炸動畫。
// 原始碼在第三段旁邊留了 `/* ??? */`——連作者都不確定那個 +2 是什麼。
// 照抄。
func tally(t int) bool {
	return (t >= FIRSTRIVEDGE && t <= LASTRUBBLE) ||
		(t >= POWERBASE+2 && t <= POWERBASE+12) ||
		(t >= TINYEXP && t <= LASTTINYEXP+2)
}

// checkSize 從中心圖塊反推建物大小。w_tool.c:296
func checkSize(t int) int {
	switch {
	case (t >= RESBASE-1 && t <= PORTBASE-1) ||
		(t >= LASTPOWERPLANT+1 && t <= POLICESTATION+4):
		return 3
	case (t >= PORTBASE && t <= LASTPORT) ||
		(t >= COALBASE && t <= LASTPOWERPLANT) ||
		(t >= STADIUMBASE && t <= LASTZONE):
		return 4
	}
	return 0
}

// checkBigZone 從**任何一格**反推它屬於哪個大型建物，以及中心在哪。
// w_tool.c:201
//
// 只認得大型建物的左上四格（4×4）或左上十六格（6×6 機場）。
// 推土機靠它處理「玩家點到建物邊角」的情形。
func checkBigZone(id int) (size, dh, dv int) {
	switch id {
	case POWERPLANT, PORT, NUCLEAR, STADIUM:
		return 4, 0, 0
	case POWERPLANT + 1, COALSMOKE3, COALSMOKE3 + 1, COALSMOKE3 + 2,
		PORT + 1, NUCLEAR + 1, STADIUM + 1:
		return 4, -1, 0
	case POWERPLANT + 4, PORT + 4, NUCLEAR + 4, STADIUM + 4:
		return 4, 0, -1
	case POWERPLANT + 5, PORT + 5, NUCLEAR + 5, STADIUM + 5:
		return 4, -1, -1
	}
	// 機場的十六格：編號在 6 寬的網格裡，列與行各自對應偏移。
	if id >= AIRPORT && id <= AIRPORT+21 {
		d := id - AIRPORT
		col, row := d%6, d/6
		if col <= 3 && row <= 3 {
			return 6, -col, -row
		}
	}
	return 0, 0, 0
}

// PlaceZone 放一個 n×n 的建物。w_tool.c:351／:483／:613
//
// 三個尺寸的邏輯一模一樣，只有邊長、動畫格與收尾的邊界修整不同，
// 所以合成一支。原版拆成 check3x3／check4x4／check6x6 是因為 C 沒有
// 方便的方式參數化，不是因為行為不同。
//
// ⚠ 點擊點不是左上角。原版一律先 `mapH--; mapV--`，也就是**點擊點是
// 左上角往右下一格**。3×3 時那正好是中心，4×4 與 6×6 時不是。
func (w *World) PlaceZone(mapH, mapV, base int, anim bool, tool Tool, n int) int {
	mapH--
	mapV--
	if mapH < 0 || mapH > WorldX-n || mapV < 0 || mapV > WorldY-n {
		return ToolNeedsClear
	}

	cost := 0
	clear := true
	for row := 0; row < n; row++ {
		for col := 0; col < n; col++ {
			t := int(w.Map[mapH+col][mapV+row]) & LOMASK
			if t == 0 {
				continue
			}
			if w.AutoBulldoze {
				if tally(t) {
					cost++ // 自動推土每格 1 元
				} else {
					clear = false
				}
			} else {
				clear = false
			}
		}
	}
	if !clear {
		return ToolNeedsClear
	}

	cost += ToolCost[tool]
	if w.TotalFunds-cost < 0 {
		return ToolNoMoney
	}
	w.spend(cost)

	// 圖塊編號從 base 開始，**逐格遞增**（列優先）。中心格加 ZONEBIT；
	// 4×4 的發電廠與核電廠在 (1,2) 那一格加 ANIMBIT（冒煙／冷卻塔）。
	b := base
	for row := 0; row < n; row++ {
		for col := 0; col < n; col++ {
			switch {
			case col == 1 && row == 1:
				w.Map[mapH+col][mapV+row] = uint16(b + BNCNBIT + ZONEBIT)
			case col == 1 && row == 2 && anim:
				w.Map[mapH+col][mapV+row] = uint16(b + BNCNBIT + ANIMBIT)
			default:
				w.Map[mapH+col][mapV+row] = uint16(b + BNCNBIT)
			}
			b++
		}
	}
	w.fixZoneBorder(mapH, mapV, n)
	return ToolOK
}

// fixZoneBorder 對新建物外圍一圈跑 ConnecTile(…, 0)，讓周邊的路與電線
// 重新接上。w_tool.c:315／:442／:577
//
// ⚠ 四條邊各跑 n 格，**四個角落不跑**。所以蓋在轉角的路不會被修——
// 那是原版行為，不是漏寫。
func (w *World) fixZoneBorder(x, y, n int) {
	for i := 0; i < n; i++ {
		w.ConnecTile(x+i, y-1, ConnFixZone) // 上
		w.ConnecTile(x-1, y+i, ConnFixZone) // 左
		w.ConnecTile(x+i, y+n, ConnFixZone) // 下
		w.ConnecTile(x+n, y+i, ConnFixZone) // 右
	}
}

// putRubble 把一個 n×n 區域炸成瓦礫。w_tool.c:821／:843／:865
//
// ⚠ 起點是 (x-1, y-1)，範圍是 n×n——所以「4×4 的建物」實際上是從
// 中心往左上一格開始的 4×4，不是以中心為準的對稱範圍。
// ⚠ 輻射瓦礫（RADTILE）與空地不會被覆蓋。
func (w *World) putRubble(x, y, n int) {
	for xx := x - 1; xx < x-1+n; xx++ {
		for yy := y - 1; yy < y-1+n; yy++ {
			if !InBounds(xx, yy) {
				continue
			}
			z := int(w.Map[xx][yy]) & LOMASK
			if z == RADTILE || z == 0 {
				continue
			}
			if w.DoAnimation {
				w.Map[xx][yy] = uint16(TINYEXP + w.Rand.Rand(2) + ANIMBIT + BULLBIT)
			} else {
				w.Map[xx][yy] = uint16(SOMETINYEXP + ANIMBIT + BULLBIT)
			}
		}
	}
}

// Bulldoze 推土機。w_tool.c:926
//
// 三條路徑：分區中心（整棟炸掉）、大型建物的其他格（找到中心再炸）、
// 一般格子（交給 ConnecTile）。
//
// ⚠ 推水（河、河岸、水道）要 6 元才動手，但只扣 5 元。原版就是這樣。
func (w *World) Bulldoze(x, y int) int {
	if !InBounds(x, y) {
		return ToolNeedsClear
	}
	cur := w.Map[x][y]
	t := int(cur) & LOMASK
	result := ToolOK

	switch {
	case cur&ZONEBIT != 0:
		if w.TotalFunds > 0 {
			w.spend(1)
			switch checkSize(t) {
			case 3:
				w.putRubble(x, y, 3)
			case 4:
				w.putRubble(x, y, 4)
			case 6:
				w.putRubble(x, y, 6)
			}
		}
	default:
		if size, dh, dv := checkBigZone(t); size != 0 {
			if w.TotalFunds > 0 {
				w.spend(1)
				// ⚠ size == 3 時原版只播音效，**不放瓦礫**。
				switch size {
				case 4:
					w.putRubble(x+dh, y+dv, 4)
				case 6:
					w.putRubble(x+dh, y+dv, 6)
				}
			}
		} else if t == RIVER || t == REDGE || t == CHANNEL {
			if w.TotalFunds >= 6 {
				result = w.ConnecTile(x, y, ConnDoze)
				if t != int(w.Map[x][y])&LOMASK {
					w.spend(5)
				}
			} else {
				result = ToolBlocked
			}
		} else {
			result = w.ConnecTile(x, y, ConnDoze)
		}
	}
	return result
}

// PutPark 蓋公園。w_tool.c:151
//
// ⚠ `Rand(4)` 回傳 0..4（**五個值**，見 docs/spec/rng.md），
// 所以 `value == 4` 的那一支是噴泉，機率五分之一。
// 把 Rand(4) 讀成「0..3」的話噴泉永遠不會出現。
func (w *World) PutPark(x, y int) int {
	if !InBounds(x, y) {
		return ToolNeedsClear
	}
	if w.TotalFunds-ToolCost[ToolPark] < 0 {
		return ToolNoMoney
	}
	v := w.Rand.Rand(4)
	var tile int
	if v == 4 {
		tile = FOUNTAIN | BURNBIT | BULLBIT | ANIMBIT
	} else {
		tile = (v + WOODS2) | BURNBIT | BULLBIT
	}
	if w.Map[x][y] != 0 {
		return ToolNeedsClear
	}
	w.spend(ToolCost[ToolPark])
	w.Map[x][y] = uint16(tile)
	return ToolOK
}

// PutNetwork 拉網路線（Micropolis 才有，原版 SimCity 1 沒有）。w_tool.c:176
func (w *World) PutNetwork(x, y int) int {
	if !InBounds(x, y) {
		return ToolNeedsClear
	}
	t := int(w.Map[x][y]) & LOMASK
	if w.TotalFunds > 0 && tally(t) {
		w.Map[x][y] = 0
		t = 0
		w.spend(1)
	}
	if t != 0 {
		return ToolBlocked
	}
	if w.TotalFunds-ToolCost[ToolNetwork] < 0 {
		return ToolNoMoney
	}
	w.Map[x][y] = TELEBASE | CONDBIT | BURNBIT | BULLBIT | ANIMBIT
	w.spend(ToolCost[ToolNetwork])
	return ToolOK
}

// ApplyTool 是給 UI 的單一入口：把工具編號與格子座標換成一次操作。
//
// 回傳碼見 ToolOK 等常數。查詢工具不改地圖，回 ToolOK。
func (w *World) ApplyTool(tool Tool, x, y int) int {
	if !InBounds(x, y) {
		return ToolNeedsClear
	}
	switch tool {
	case ToolResidential:
		return w.PlaceZone(x, y, RESBASE, false, tool, 3)
	case ToolCommercial:
		return w.PlaceZone(x, y, COMBASE, false, tool, 3)
	case ToolIndustrial:
		return w.PlaceZone(x, y, INDBASE, false, tool, 3)
	case ToolFireStation:
		return w.PlaceZone(x, y, FIRESTBASE, false, tool, 3)
	case ToolPolice:
		return w.PlaceZone(x, y, POLICESTBASE, false, tool, 3)
	case ToolStadium:
		return w.PlaceZone(x, y, STADIUMBASE, false, tool, 4)
	case ToolCoalPower:
		return w.PlaceZone(x, y, COALBASE, true, tool, 4)
	case ToolNuclear:
		return w.PlaceZone(x, y, NUCLEARBASE, true, tool, 4)
	case ToolSeaport:
		return w.PlaceZone(x, y, PORTBASE, false, tool, 4)
	case ToolAirport:
		return w.PlaceZone(x, y, AIRPORTBASE, false, tool, 6)
	case ToolRoad:
		return w.ConnecTile(x, y, ConnRoad)
	case ToolRail:
		return w.ConnecTile(x, y, ConnRail)
	case ToolWire:
		return w.ConnecTile(x, y, ConnWire)
	case ToolBulldozer:
		return w.Bulldoze(x, y)
	case ToolPark:
		return w.PutPark(x, y)
	case ToolNetwork:
		return w.PutNetwork(x, y)
	case ToolQuery, ToolChalk, ToolEraser:
		return ToolOK
	}
	return ToolBlocked
}
