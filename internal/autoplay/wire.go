package autoplay

import "github.com/wicanr2/chengshi_cht/internal/sim"

// 拉電線：從電廠接到沒電的分區。
//
// ⚠ **L 形直線接不通。** 第一版是「往目標走一條 L 形」，實測 24 格的線裡
// 有 7 格放不下去——穿過分區、穿過房子——而斷一格整條就不通。舊金山地震
// 之後 291 個分區裡有 103 個是暗的，蓋了幾座電廠一個都沒接回來。
//
// 改成 BFS 找路。走得過的格子：空地、樹、瓦礫（推平再拉）、路面（電線可以
// 疊在路上）、既有電線（本來就通）、水面（海底電纜，$25）。繞不過去的只有
// 分區本身。

// wireStep 判斷這一格能不能讓電線通過。
func wireStep(cell uint16) bool {
	t := int(cell & sim.LOMASK)
	switch {
	case vacant(cell): // 空地、樹、瓦礫
		return true
	case isRoad(cell): // 電線疊在路上（connect.go 的 case 66）
		return true
	case t >= sim.POWERBASE && t <= sim.LASTPOWER: // 既有電線
		return true
	case t >= sim.RIVER && t <= sim.LASTRIVEDGE: // 海底電纜
		return true
	}
	return false
}

// wirePath 從 (x0,y0) BFS 到「碰得到 (x1,y1) 那座分區」的地方，
// 回傳沿路要拉電線的格子。找不到路回 nil。
//
// ⚠ **目標不能是分區的中心格。** 3×3 分區的中心被自己的八格圍住，
// 四個正交鄰居全是分區格，而分區格不能拉電線——BFS 永遠走不到，
// 每一條線都回「找不到路」。實測舊金山：`connectDark` 一條都接不上，
// 而外面看起來只是「暗區沒有減少」。
// 正確的目標是**貼著那座分區外緣的可走格**：電線拉到那裡就通了。
func (p *Player) wirePath(x0, y0, x1, y1 int) [][2]int {
	const w, h = sim.WorldX, sim.WorldY
	// 目標：分區 3×3 外圈那一圈裡，可以拉電線的格子。
	goal := map[int]bool{}
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			if dx > -2 && dx < 2 && dy > -2 && dy < 2 {
				continue // 3×3 本身
			}
			nx, ny := x1+dx, y1+dy
			if !sim.InBounds(nx, ny) {
				continue
			}
			// 只取正交貼著 3×3 的那十二格（四個角碰不到分區）。
			if (dx == -2 || dx == 2) && (dy == -2 || dy == 2) {
				continue
			}
			c := p.w.Map[nx][ny]
			if c&sim.CONDBIT != 0 || wireStep(c) {
				goal[nx*h+ny] = true
			}
		}
	}
	if len(goal) == 0 {
		return nil
	}
	prev := make([]int32, w*h)
	for i := range prev {
		prev[i] = -1
	}
	start := x0*h + y0
	prev[start] = int32(start)
	queue := []int{start}
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	found := -1
	if goal[start] {
		found = start
	}
	for qi := 0; qi < len(queue) && found < 0; qi++ {
		cur := queue[qi]
		cx, cy := cur/h, cur%h
		for _, d := range dirs {
			nx, ny := cx+d[0], cy+d[1]
			if !sim.InBounds(nx, ny) {
				continue
			}
			n := nx*h + ny
			if prev[n] >= 0 || !wireStep(p.w.Map[nx][ny]) {
				continue
			}
			prev[n] = int32(cur)
			if goal[n] {
				found = n
				break
			}
			queue = append(queue, n)
		}
	}
	if found < 0 {
		return nil
	}
	var path [][2]int
	for n := found; n != start; n = int(prev[n]) {
		path = append(path, [2]int{n / h, n % h})
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// layWirePath 沿著路徑拉電線，擋路的樹與瓦礫先推平。
// 回傳拉通了沒有。
func (p *Player) layWirePath(path [][2]int) bool {
	w := p.w
	for _, c := range path {
		x, y := c[0], c[1]
		cell := w.Map[x][y]
		if cell&sim.CONDBIT != 0 {
			continue // 已經是導電格
		}
		if w.TotalFunds < 100 {
			return false
		}
		if w.ApplyTool(sim.ToolWire, x, y) == sim.ToolOK {
			continue
		}
		// 樹與瓦礫推平再拉。`layWire` 只認乾淨的土（connect.go 的 case 0）。
		if !vacant(cell) {
			return false
		}
		w.ApplyTool(sim.ToolBulldozer, x, y)
		if w.ApplyTool(sim.ToolWire, x, y) != sim.ToolOK {
			return false
		}
	}
	return true
}

// connectDark 把還沒接上的暗區接回電網。
//
// 每年最多接 maxLinks 條，而且吃 budget 的額度——一條線可能要拉幾十格，
// 全部一次接完會把錢花光（budget.go）。
func (p *Player) connectDark(maxLinks int, budget *purse, reserve int) {
	w := p.w
	done := 0
	for x := 0; x < sim.WorldX && done < maxLinks; x++ {
		for y := 0; y < sim.WorldY && done < maxLinks; y++ {
			c := w.Map[x][y]
			if c&sim.ZONEBIT == 0 || c&sim.PWRBIT != 0 {
				continue
			}
			if !budget.ok(reserve) {
				return
			}
			sx, sy, ok := p.nearestPowered(x, y)
			if !ok {
				return
			}
			path := p.wirePath(sx, sy, x, y)
			if path == nil {
				continue
			}
			if p.layWirePath(path) {
				done++
			}
		}
	}
}

// nearestPowered 找離 (x,y) 最近的**已通電的導電格**。
func (p *Player) nearestPowered(x, y int) (int, int, bool) {
	best, bx, by := 1<<30, -1, -1
	for i := 0; i < sim.WorldX; i++ {
		for j := 0; j < sim.WorldY; j++ {
			c := p.w.Map[i][j]
			if c&sim.CONDBIT == 0 || c&sim.PWRBIT == 0 {
				continue
			}
			dx, dy := i-x, j-y
			if d := dx*dx + dy*dy; d < best {
				best, bx, by = d, i, j
			}
		}
	}
	return bx, by, bx >= 0
}
