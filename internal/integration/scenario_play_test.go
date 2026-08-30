package integration

import (
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/autoplay"
	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 自動玩家把八個劇本各玩一次，看過幾個。
//
// 這一支與 TestAllScenariosReachVerdict 是一對：那一支不出手，證明判定
// 機制會觸發；這一支動手，證明**玩得贏**。差別是實質的——不出手八個
// 只過一個（伯恩，因為不蓋東西交通自然低），動手過五個。
//
// 為什麼要留成測試而不只是一支工具：自動玩家只用 ApplyTool 與稅率，
// 所以它把成長、電力、交通、犯罪、預算、勝敗判定整條路徑串起來跑。
// 規則層任何一處壞掉，通關數會掉——那是單元測試看不到的整體回歸。
//
// ⚠ 門檻寫 5 不寫 6：地圖成長帶亂數。實測種子 1–5 是 6/6/6/6/6
// （漢堡、伯恩、東京、底特律、波士頓、里約五顆種子全過），留一格餘裕。
// 門檻卡在觀測到的最好值就會變成靠運氣過的測試。
func TestAutoPlayerWinsScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("跑八個劇本要一分鐘上下")
	}
	dir := dosDir(t)
	win := 0
	for n := 1; n <= 8; n++ {
		w, err := game.LoadScenarioSeed(dir, n, 1)
		if err != nil {
			t.Fatalf("第 %d 個劇本載入失敗：%v", n, err)
		}
		w.AutoBudget = true
		p := autoplay.New(w, autoplay.ScenarioGoal[n])
		verdict, ticks := 0, 0
		for i := 0; i < (autoplay.ScoreWait[n]+48)*16 && verdict == 0; i++ {
			w.Frame()
			if w.CityTime > ticks && w.CityTime%48 == 0 {
				ticks = w.CityTime
				p.Year()
			}
			switch w.MessagePort {
			case -sim.MsgScenarioWin, -sim.MsgScenarioLose:
				verdict = w.MessagePort
			}
			w.MessagePort = 0
		}
		switch verdict {
		case -sim.MsgScenarioWin:
			win++
			t.Logf("%-14s 通關　等級 %d 評分 %d 人口 %d",
				w.CityName, w.CityClass, w.CityScore, w.Eval.CityPop)
		case -sim.MsgScenarioLose:
			t.Logf("%-14s 未過　等級 %d 評分 %d 人口 %d 犯罪 %d 交通 %d",
				w.CityName, w.CityClass, w.CityScore, w.Eval.CityPop,
				w.CrimeAverage, w.Eval.TrafficAverage)
		default:
			t.Errorf("%s：跑完還是沒有判定", w.CityName)
		}
	}
	t.Logf("自動玩家 %d/8 通關", win)
	if win < 5 {
		t.Errorf("自動玩家只過 %d/8，之前是 6/8 —— 規則層可能有回歸", win)
	}
}
