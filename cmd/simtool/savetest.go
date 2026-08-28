package main

import (
	"fmt"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// cmdSave 產生一個測試用的城市檔，給 oracle 交叉驗證用。
func cmdSave(args []string) {
	out := "workplace/oracle/roundtrip.cty"
	if len(args) > 0 {
		out = args[0]
	}
	w := sim.NewWorld(4242)
	w.GenerateMap(4242, sim.DefaultTerrainParams())
	w.DoSimInit()
	if _, _, ok := game.BuildStarterCity(w); !ok {
		fmt.Fprintln(os.Stderr, "找不到可建地")
		os.Exit(1)
	}
	for i := 0; i < 20*48*16; i++ {
		w.Frame()
	}
	if err := game.SaveCity(out, w); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// 同時倒出我們自己看到的地圖，給比對用
	f, _ := os.Create(out + ".csv")
	for y := 0; y < sim.WorldY; y++ {
		for x := 0; x < sim.WorldX; x++ {
			if x > 0 {
				fmt.Fprint(f, ",")
			}
			fmt.Fprint(f, w.Map[x][y])
		}
		fmt.Fprintln(f)
	}
	f.Close()
	fmt.Printf("寫入 %s（%d 年、資金 %d）\n", out, w.CityTime/48, w.TotalFunds)
}
