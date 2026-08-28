package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// cmdFlat 在指定種子的地圖上找一塊可以直接蓋東西的方形空地。
//
// 用途是產生**可重跑的**試玩腳本與截圖：地形是隨機的，把座標寫死在
// 腳本裡，換一顆種子就會莫名其妙點到水裡。讓腳本先問一次，滑鼠座標
// 才有依據。
//
// 預設把搜尋範圍限制在遊戲開場的視野裡（相機一開始置中，見
// internal/ui 的 centerCamera），這樣腳本不必先捲動畫面。
func cmdFlat(args []string) {
	fs := flag.NewFlagSet("flat", flag.ExitOnError)
	seed := fs.Int("seed", 1, "地形種子")
	w := fs.Int("w", 12, "要找的寬度（格）")
	h := fs.Int("h", 10, "要找的高度（格）")
	x0 := fs.Int("x0", 44, "搜尋範圍左界")
	y0 := fs.Int("y0", 38, "搜尋範圍上界")
	x1 := fs.Int("x1", 76, "搜尋範圍右界")
	y1 := fs.Int("y1", 62, "搜尋範圍下界")
	fs.Parse(args)

	world := sim.NewWorld(uint32(*seed))
	world.GenerateMap(uint32(*seed), sim.DefaultTerrainParams())
	for y := *y0; y+*h <= *y1; y++ {
		for x := *x0; x+*w <= *x1; x++ {
			if flatArea(world, x, y, *w, *h) {
				fmt.Printf("%d %d\n", x, y)
				return
			}
		}
	}
	fmt.Fprintf(os.Stderr, "種子 %d 的開場視野裡沒有 %d×%d 的空地\n", *seed, *w, *h)
	os.Exit(1)
}

func flatArea(world *sim.World, x, y, w, h int) bool {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			if world.Map[i][j]&sim.LOMASK != sim.DIRT {
				return false
			}
		}
	}
	return true
}
