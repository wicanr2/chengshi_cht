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
	fresh := fs.Bool("new", false, "不玩劇本，開一座新城市從零蓋")
	years := fs.Int("years", 50, "-new 時要玩幾年")
	dir := fs.String("data", os.Getenv("SIMCITY_DATA"), "SIMCITY 1.10 目錄")
	_ = fs.Parse(args)
	autoplay.Debug = *dbg
	if *dir == "" {
		*dir = "workplace/dos110/SIMCITY 1.10"
	}
	if *fresh {
		playFresh(uint32(*seed), *years, *verbose)
		return
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
					// ⚠ 逐年要印**分區數與沒電數**。只印資金與評分的話，
					// 「城市為什麼長不大」看不出來——實測達斯維利卡住的原因是
					// 84 個分區裡 29 個沒電，而那在資金與評分上完全看不出來。
					//
					// ⚠ 不要印 `ResPop`／`PwrdZCnt`：那些是 `ClearCensus` 每一輪
					// 歸零、`MapScan` 十六個相位累加的中間值，在年界取樣**永遠是 0**。
					zones, dark := 0, 0
					for x := 0; x < sim.WorldX; x++ {
						for y := 0; y < sim.WorldY; y++ {
							if w.Map[x][y]&sim.ZONEBIT != 0 {
								zones++
								if w.Map[x][y]&sim.PWRBIT == 0 {
									dark++
								}
							}
						}
					}
					fmt.Printf("  %d 年 資金 %6d 稅 %2d 等級 %d 評分 %4d 犯罪 %3d 汙染 %3d 交通 %3d "+
						"人口 %6d 地價 %3d 閥 %5d/%5d/%5d 分區 %3d 沒電 %3d\n",
						1900+w.CityTime/48, w.TotalFunds, w.CityTax, w.CityClass,
						w.CityScore, w.CrimeAverage, w.PolluteAverage, w.Eval.TrafficAverage,
						w.LastCityPop, w.LVAverage, w.RValve, w.CValve, w.IValve,
						zones, dark)
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

// playFresh 從零開一座新城市，讓自動玩家玩指定的年數。
//
// 這是 CLAUDE.md §4 另一條驗收：「從零開始蓋到一座能自我維持的城市」。
// 劇本測的是「接手一座既有城市能不能救」，這一支測的是「白紙一張能不能長」
// ——兩件事的失敗模式不一樣：劇本有現成的路網與電網，新城市什麼都沒有。
func playFresh(seed uint32, years int, verbose bool) {
	w := sim.NewWorld(seed)
	w.GenerateMap(seed, sim.DefaultTerrainParams())
	w.DoSimInit()
	w.AutoBudget = true
	p := autoplay.New(w, 0) // 追人口
	ticks := 0
	for w.CityTime < years*48 {
		w.Frame()
		if w.CityTime > ticks && w.CityTime%48 == 0 {
			ticks = w.CityTime
			p.Year()
			if verbose {
				zones, dark := 0, 0
				for x := 0; x < sim.WorldX; x++ {
					for y := 0; y < sim.WorldY; y++ {
						if w.Map[x][y]&sim.ZONEBIT != 0 {
							zones++
							if w.Map[x][y]&sim.PWRBIT == 0 {
								dark++
							}
						}
					}
				}
				fmt.Printf("  %d 年 資金 %7d 等級 %d 評分 %4d 人口 %7d 分區 %3d 沒電 %3d 犯罪 %3d\n",
					1900+w.CityTime/48, w.TotalFunds, w.CityClass, w.CityScore,
					w.LastCityPop, zones, dark, w.CrimeAverage)
			}
		}
		w.MessagePort = 0
	}
	fmt.Printf("種子 %d：%d 年後 等級 %d 人口 %d 評分 %d 資金 %d\n",
		seed, years, w.CityClass, w.LastCityPop, w.CityScore, w.TotalFunds)
}
