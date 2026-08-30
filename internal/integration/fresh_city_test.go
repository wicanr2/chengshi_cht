package integration

import (
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/autoplay"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 從一張白紙蓋到一座能自我維持的城市。
//
// CLAUDE.md §4 把這條與「通關八個劇本」並列為正常玩家路徑，而它原本
// **一次都沒被走過**——驗收只跑劇本，而劇本一律是接手一座既有城市
// （有路、有電、有分區）。真的開一座新城市讓自動玩家玩五十年，結果是
// 一格都沒蓋、資金原封不動 20 000：`growSites` 要「挨著路又在電網內」，
// 白紙上兩個條件都不成立；`power()` 要「有分區沒電」才動手，而一個分區
// 都沒有。三個條件互相等待，永遠不會有人先動。
//
// 這一支測的是「能不能長」，`TestAutoPlayerWinsScenarios` 測的是
// 「接手能不能救」。兩者的失敗模式不一樣，缺一個就會漏掉一整類問題。
func TestFreshCityGrowsAndSustains(t *testing.T) {
	if testing.Short() {
		t.Skip("五十個遊戲年要幾秒")
	}
	const years = 50
	for _, seed := range []uint32{1, 2, 3} {
		w := sim.NewWorld(seed)
		w.GenerateMap(seed, sim.DefaultTerrainParams())
		w.DoSimInit()
		w.AutoBudget = true
		p := autoplay.New(w, 0) // 追人口
		ticks, popAt30 := 0, 0
		for w.CityTime < years*48 {
			w.Frame()
			if w.CityTime > ticks && w.CityTime%48 == 0 {
				ticks = w.CityTime
				p.Year()
				if w.CityTime/48 == 30 {
					popAt30 = w.LastCityPop
				}
			}
			w.MessagePort = 0
		}
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
		// 門檻取自實測（種子 1–5：等級都是 2，人口 21 940–30 440，
		// 資金 5 490–5 788，分區 120 上下）。留餘裕，不卡在觀測到的最好值。
		if w.CityClass < 2 {
			t.Errorf("種子 %d：五十年後只有等級 %d，應至少 2（城市）", seed, w.CityClass)
		}
		if w.LastCityPop < 15000 {
			t.Errorf("種子 %d：五十年後人口 %d，應超過 15 000", seed, w.LastCityPop)
		}
		if w.TotalFunds <= 0 {
			t.Errorf("種子 %d：五十年後資金 %d —— 破產的城市不算自我維持",
				seed, w.TotalFunds)
		}
		if zones < 60 {
			t.Errorf("種子 %d：五十年只蓋了 %d 個分區", seed, zones)
		}
		// ⚠ 沒電的分區要少。它們不會成長，多了代表城市在空轉——
		// 而空轉在人口與資金上要好幾年才看得出來。
		if dark*5 > zones {
			t.Errorf("種子 %d：%d/%d 個分區沒電（超過兩成）", seed, dark, zones)
		}
		// 還在長，不是停在高點慢慢掉。
		if w.LastCityPop < popAt30 {
			t.Errorf("種子 %d：第 50 年人口 %d 比第 30 年的 %d 還少",
				seed, w.LastCityPop, popAt30)
		}
		t.Logf("種子 %d：等級 %d 人口 %d 分區 %d（沒電 %d）資金 %d",
			seed, w.CityClass, w.LastCityPop, zones, dark, w.TotalFunds)
	}
}
