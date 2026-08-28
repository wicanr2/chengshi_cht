package main

import (
	"fmt"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// cmdInspect 讀一個 .cty 並印出摘要，給試玩腳本當機械判準。
//
// 截圖只能給人看；「玩家蓋的東西真的進到存檔裡了嗎」要有一個
// grep 得到的數字才判得了。
func cmdInspect(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法：simtool inspect <城市檔.cty>")
		os.Exit(2)
	}
	w, err := game.LoadCity(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	n := map[string]int{}
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			t := w.Map[x][y] & sim.LOMASK
			switch {
			case t >= sim.ROADBASE && t <= sim.LASTROAD:
				n["road"]++
			case t >= sim.POWERBASE && t <= sim.LASTPOWER:
				n["wire"]++
			case t > sim.LASTPOWER && t <= sim.LASTRAIL:
				n["rail"]++
			case t >= sim.COALBASE && t <= sim.LASTPOWERPLANT:
				n["coal"]++
			// 火力發電廠的四根煙囪在地圖上是動畫圖塊（916 起），
			// 不在 745–760 的範圍裡。只數本體會得到 12 而不是 16。
			case t >= sim.COALSMOKE1 && t < sim.COALSMOKE4+4:
				n["coal"]++
			case t >= sim.RESBASE && t < sim.COMBASE:
				n["res"]++
			case t >= sim.COMBASE && t < sim.INDBASE:
				n["com"]++
			case t >= sim.INDBASE && t < sim.PORTBASE:
				n["ind"]++
			}
		}
	}
	fmt.Printf("year=%d funds=%d pop=%d res=%d com=%d ind=%d road=%d wire=%d rail=%d coal=%d\n",
		1900+w.CityTime/48, w.TotalFunds, w.Eval.CityPop,
		n["res"], n["com"], n["ind"], n["road"], n["wire"], n["rail"], n["coal"])
}
