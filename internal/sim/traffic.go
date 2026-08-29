package sim

// 交通生成。證據：docs/re/07-traffic-and-zones.md／規格：docs/spec/traffic-and-zones.md
// 一手出處：s_traf.c
//
// 這不是尋路，是**隨機遊走**：從分區邊緣找一條路，沿著路隨機走最多 30 步，
// 每一步看四周有沒有目的地。走到就算通、走不到就算塞。
// 「交通壅塞」在這款遊戲裡是機率現象，不是流量計算。

// maxTrafDis 是一次嘗試的最大步數。s_traf.c:67 #define MAXDIS 30
const maxTrafDis = 30

// MakeTraf 從目前的掃描游標出發跑一次交通。s_traf.c:77
//
// 回傳值有三種，呼叫端要分開處理：
//
//	 1  找到路而且走到目的地（通）
//	 0  找到路但走不到（塞）
//	-1  分區邊緣根本沒有路
func (w *World) MakeTraf(zt int) int {
	xtem, ytem := w.SMapX, w.SMapY
	w.zSource = zt
	w.posStackN = 0

	if w.findPRoad() {
		if w.tryDrive() {
			w.setTrafMem()
			w.SMapX, w.SMapY = xtem, ytem
			return 1
		}
		w.SMapX, w.SMapY = xtem, ytem
		return 0
	}
	return -1
}

// setTrafMem 把走過的路加上車流密度。s_traf.c:108
//
// ⚠ 每經過一次就 +50，上限 240。超過 240 而且擲骰命中時會記下 TrafMaxX/Y
// （警車的目標），**而且把值夾回 240**——所以密度不會無限成長。
func (w *World) setTrafMem() {
	for x := w.posStackN; x > 0; x-- {
		w.pullPos()
		if !InBounds(w.SMapX, w.SMapY) {
			continue
		}
		z := int(w.Map[w.SMapX][w.SMapY] & LOMASK)
		if z < ROADBASE || z >= POWERBASE {
			continue
		}
		d := int(w.TrfDensity[w.SMapX>>1][w.SMapY>>1]) + 50
		if d > 240 && w.Rand.Rand(5) == 0 {
			d = 240
			w.TrafMaxX = w.SMapX << 4
			w.TrafMaxY = w.SMapY << 4
			w.sprites().SetCopterDest(w.TrafMaxX, w.TrafMaxY)
		}
		w.TrfDensity[w.SMapX>>1][w.SMapY>>1] = uint8(d)
	}
}

func (w *World) pushPos() {
	w.posStackN++
	if w.posStackN < len(w.posStack) {
		w.posStack[w.posStackN] = [2]int{w.SMapX, w.SMapY}
	}
}

func (w *World) pullPos() {
	if w.posStackN >= 0 && w.posStackN < len(w.posStack) {
		w.SMapX = w.posStack[w.posStackN][0]
		w.SMapY = w.posStack[w.posStackN][1]
	}
	w.posStackN--
}

// perimX／perimY 是分區周長的 12 個位置（3×3 分區外面一圈）。s_traf.c:162
var perimX = [12]int{-1, 0, 1, 2, 2, 2, 1, 0, -1, -2, -2, -2}
var perimY = [12]int{-2, -2, -2, -1, 0, 1, 2, 2, 2, 1, 0, -1}

// findPRoad 在分區周長上找路。找到就把游標移過去。s_traf.c:160
func (w *World) findPRoad() bool {
	for z := 0; z < 12; z++ {
		tx, ty := w.SMapX+perimX[z], w.SMapY+perimY[z]
		if InBounds(tx, ty) && roadTest(int(w.Map[tx][ty])) {
			w.SMapX, w.SMapY = tx, ty
			return true
		}
	}
	return false
}

// tryDrive 沿著路走最多 30 步。s_traf.c:202
//
// ⚠ 走進死路時的回溯是 `PosStackN--; z += 3`——**退一格、但步數多算三步**。
// 看起來不對稱，是原版的行為。
func (w *World) tryDrive() bool {
	w.lDir = 5
	for z := 0; z < maxTrafDis; z++ {
		if w.tryGo(z) {
			if w.driveDone() {
				return true
			}
			continue
		}
		if w.posStackN != 0 {
			w.posStackN--
			z += 3
			continue
		}
		return false
	}
	return false
}

// tryGo 往某個方向走一格。s_traf.c:224
//
// 起始方向是隨機的（`Rand16() & 3`，不是 `Rand(3)`——原始碼裡有一行註解說
// 那是**亂數用量最大的地方**，所以改成位元遮罩），四個方向依序試，
// 跳過剛才來的方向。每走兩步存一次位置。
func (w *World) tryGo(z int) bool {
	rdir := w.Rand.Rand16() & 3
	for x := rdir; x < rdir+4; x++ {
		realdir := x & 3
		if realdir == w.lDir {
			continue
		}
		if !roadTest(w.getFromMap(realdir)) {
			continue
		}
		w.moveMapSimCursor(realdir)
		w.lDir = (realdir + 2) & 3
		if z&1 != 0 {
			w.pushPos()
		}
		return true
	}
	return false
}

// getFromMap 讀某個方向的鄰居圖塊編號；出界回 0。s_traf.c:249
func (w *World) getFromMap(dir int) int {
	switch dir {
	case 0:
		if w.SMapY > 0 {
			return int(w.Map[w.SMapX][w.SMapY-1] & LOMASK)
		}
	case 1:
		if w.SMapX < WorldX-1 {
			return int(w.Map[w.SMapX+1][w.SMapY] & LOMASK)
		}
	case 2:
		if w.SMapY < WorldY-1 {
			return int(w.Map[w.SMapX][w.SMapY+1] & LOMASK)
		}
	case 3:
		if w.SMapX > 0 {
			return int(w.Map[w.SMapX-1][w.SMapY] & LOMASK)
		}
	}
	return 0
}

// moveMapSimCursor 移動掃描游標。與電力用的 MoveMapSim 同一套方向編號。
func (w *World) moveMapSimCursor(dir int) {
	switch dir {
	case 0:
		if w.SMapY > 0 {
			w.SMapY--
		}
	case 1:
		if w.SMapX < WorldX-1 {
			w.SMapX++
		}
	case 2:
		if w.SMapY < WorldY-1 {
			w.SMapY++
		}
	case 3:
		if w.SMapX > 0 {
			w.SMapX--
		}
	}
}

// targLow／targHigh 是三種來源分區各自的目的地圖塊範圍。s_traf.c:274
//
//	0 住宅 → 商業或工業（COMBASE … NUCLEAR）
//	1 商業 → 工業或港口（LHTHR … PORT）
//	2 工業 → 住宅（LHTHR … COMBASE）
var targLow = [3]int{COMBASE, LHTHR, LHTHR}
var targHigh = [3]int{NUCLEAR, PORT, COMBASE}

// driveDone 看四周有沒有目的地。s_traf.c:275
func (w *World) driveDone() bool {
	l, h := targLow[w.zSource], targHigh[w.zSource]
	check := func(x, y int) bool {
		z := int(w.Map[x][y] & LOMASK)
		return z >= l && z <= h
	}
	if w.SMapY > 0 && check(w.SMapX, w.SMapY-1) {
		return true
	}
	if w.SMapX < WorldX-1 && check(w.SMapX+1, w.SMapY) {
		return true
	}
	if w.SMapY < WorldY-1 && check(w.SMapX, w.SMapY+1) {
		return true
	}
	if w.SMapX > 0 && check(w.SMapX-1, w.SMapY) {
		return true
	}
	return false
}

// roadTest 判斷一格能不能走。s_traf.c:319
//
// 道路、橋、鐵路都算；電線不算，**但 RAILHPOWERV 以上的鐵路電線交會算**。
func roadTest(x int) bool {
	x &= LOMASK
	if x < ROADBASE || x > LASTRAIL {
		return false
	}
	if x >= POWERBASE && x < RAILHPOWERV {
		return false
	}
	return true
}
