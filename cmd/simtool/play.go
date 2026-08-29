package main

// 自動玩家的命令列入口。策略在 internal/autoplay。
//
//	tools/go.sh run ./cmd/simtool play 6            # 玩第 6 個劇本
//	tools/go.sh run ./cmd/simtool play all          # 八個都玩
//	tools/go.sh run ./cmd/simtool play -seed 3 all  # 換一顆種子
//	tools/go.sh run ./cmd/simtool play -v -noop all # 不出手，逐年印狀態

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/wicanr2/chengshi_cht/internal/autoplay"
	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

func cmdPlay(args []string) {
	fs := flag.NewFlagSet("play", flag.ExitOnError)
	seed := fs.Uint("seed", 1, "亂數種子")
	noop := fs.Bool("noop", false, "不出手，只跑")
	verbose := fs.Bool("v", false, "每年印一行狀態")
	dbg := fs.Bool("debug", false, "印出每次動作的結果")
	tax := fs.Int("tax", 0, "固定稅率（0 = 用策略）")
	dir := fs.String("data", os.Getenv("SIMCITY_DATA"), "SIMCITY 1.10 目錄")
	_ = fs.Parse(args)
	autoplay.Debug = *dbg
	if *dir == "" {
		*dir = "workplace/dos110/SIMCITY 1.10"
	}
	list := []int{}
	for _, a := range fs.Args() {
		if a == "all" {
			list = []int{1, 2, 3, 4, 5, 6, 7, 8}
			break
		}
		var n int
		if _, err := fmt.Sscanf(a, "%d", &n); err == nil && n >= 1 && n <= 8 {
			list = append(list, n)
		}
	}
	if len(list) == 0 {
		list = []int{1, 2, 3, 4, 5, 6, 7, 8}
	}
	sort.Ints(list)

	win := 0
	for _, n := range list {
		w, err := game.LoadScenarioSeed(*dir, n, uint32(*seed))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		w.AutoBudget = true
		p := autoplay.New(w, autoplay.ScenarioGoal[n])
		p.FixedTax = *tax
		verdict, ticks := 0, 0
		for i := 0; i < (autoplay.ScoreWait[n]+48)*16 && verdict == 0; i++ {
			w.Frame()
			if w.CityTime > ticks && w.CityTime%48 == 0 {
				ticks = w.CityTime
				if !*noop {
					p.Year()
				}
				if *verbose {
					fmt.Printf("  %d 年 資金 %6d 稅 %2d 等級 %d 評分 %4d 犯罪 %3d 汙染 %3d 交通 %3d\n",
						1900+w.CityTime/48, w.TotalFunds, w.CityTax, w.CityClass,
						w.CityScore, w.CrimeAverage, w.PolluteAverage, w.Eval.TrafficAverage)
				}
			}
			switch w.MessagePort {
			case -sim.MsgScenarioWin, -sim.MsgScenarioLose:
				verdict = w.MessagePort
			}
			w.MessagePort = 0
		}
		tag := "未過"
		if verdict == -sim.MsgScenarioWin {
			tag = "通關"
			win++
		}
		fmt.Printf("%d %-14s %s　等級 %d 評分 %d 人口 %d 犯罪 %d 交通 %d 資金 %d\n",
			n, w.CityName, tag, w.CityClass, w.CityScore, w.Eval.CityPop,
			w.CrimeAverage, w.Eval.TrafficAverage, w.TotalFunds)
	}
	fmt.Printf("%d/%d 通關（種子 %d）\n", win, len(list), *seed)
}
