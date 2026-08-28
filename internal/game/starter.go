package game

import "github.com/wicanr2/chengshi_cht/internal/sim"

// BuildStarterCity 蓋一座能自己長起來的起始城市。
//
// 用途有兩個：`-demo` 讓人一啟動就看到活的城市，以及讓截圖驗收有東西可看。
//
// 佈局不是隨便排的。分區要長起來必須同時滿足三件事，缺一樣就會出現
// 「蓋好了但完全不動」而且沒有任何錯誤訊息：
//
//	1. 通電 —— 有一條 CONDBIT 連通的路徑接到發電廠
//	2. 有路 —— MakeTraf 在分區周長上找得到路
//	3. 有目的地 —— 住宅走得到商業或工業
//
// 在道路上拉電線會得到導電的路面（圖塊 77／78），一條縱列同時解決
// 第一和第二件事。
func BuildStarterCity(w *sim.World) (int, int, bool) {
	ox, oy, ok := findBuildable(w, 20, 16)
	if !ok {
		return 0, 0, false
	}
	x0, y0 := ox+5, oy+2
	const rows = 4

	w.ApplyTool(sim.ToolCoalPower, x0-3, y0+1)
	for y := y0; y < y0+rows*3; y++ {
		w.ApplyTool(sim.ToolWire, x0, y)
		w.ApplyTool(sim.ToolRoad, x0+4, y)
		w.ApplyTool(sim.ToolWire, x0+4, y)
		w.ApplyTool(sim.ToolRoad, x0+8, y)
		w.ApplyTool(sim.ToolWire, x0+8, y)
	}
	for i := 0; i < rows; i++ {
		cy := y0 + 1 + i*3
		w.ApplyTool(sim.ToolResidential, x0+2, cy)
		w.ApplyTool(sim.ToolCommercial, x0+6, cy)
		w.ApplyTool(sim.ToolIndustrial, x0+10, cy)
	}
	return x0, y0, true
}

// findBuildable 找一塊 w×h 的可建地。空地與樹林都算（自動整地會處理樹），
// 水不算。
func findBuildable(world *sim.World, w, h int) (int, int, bool) {
	for y := 2; y < sim.WorldY-h-2; y++ {
		for x := 2; x < sim.WorldX-w-2; x++ {
			if buildable(world, x, y, w, h) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func buildable(world *sim.World, x, y, w, h int) bool {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			t := world.TileNum(x+i, y+j)
			if t == 0 || (t >= sim.TREEBASE && t <= sim.WOODS5) {
				continue
			}
			return false
		}
	}
	return true
}
