package integration

import (
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 八個悲情城市各跑到它自己的判定時限，記錄勝敗。
//
// **這是「不出手」的跑法**：載進來之後一步都不操作，就讓它跑到 ScoreWait
// 歸零。所以它證明的不是「這個 remake 玩得贏」，而是**判定機制真的會在
// 該觸發的那一刻觸發，而且勝敗兩種結果都出得來**——劇本災難、分區成長、
// 普查、評估、訊息與勝敗判定要全部串起來，這條路徑才走得完。
//
// 玩家真的動手時結果不同：`TestAutoPlayerWinsScenarios` 用同一組劇本跑
// 自動玩家，過五個。兩支一起看才完整——這一支說「判定會觸發」，
// 那一支說「玩得贏」。
func TestAllScenariosReachVerdict(t *testing.T) {
	dir := dosDir(t)
	// 每個劇本的判定等待刻數。s_sim.c 的 scoreWaitTab。
	wait := [9]int{0, 30 * 48, 5 * 48, 5 * 48, 10 * 48, 5 * 48, 10 * 48, 5 * 48, 10 * 48}

	win := 0
	for n := 1; n <= 8; n++ {
		// ⚠ 固定種子。載入路徑會擲亂數（DoSimInit 的 MapScan），
		// 用時鐘播種的話這支測試的通關數會在 1/8 與 2/8 之間跳。
		w, err := game.LoadScenarioSeed(dir, n, 1)
		if err != nil {
			t.Fatalf("第 %d 個劇本載入失敗：%v", n, err)
		}
		if w.ScoreType != n {
			t.Errorf("%s：ScoreType = %d，應該是 %d", w.CityName, w.ScoreType, n)
		}
		if w.ScoreWait != wait[n] {
			t.Errorf("%s：ScoreWait = %d，應該是 %d", w.CityName, w.ScoreWait, wait[n])
		}

		var verdict int
		// 多跑一年當緩衝，確認判定確實在時限內送出。
		for i := 0; i < (wait[n]+48)*16 && verdict == 0; i++ {
			w.Frame()
			switch w.MessagePort {
			case -sim.MsgScenarioWin, -sim.MsgScenarioLose:
				verdict = w.MessagePort
			}
			w.MessagePort = 0
		}
		switch verdict {
		case -sim.MsgScenarioWin:
			win++
			t.Logf("%-14s 通關（%d 年）人口 %d 評分 %d 等級 %d",
				w.CityName, wait[n]/48, w.TotalPop, w.CityScore, w.CityClass)
		case -sim.MsgScenarioLose:
			t.Logf("%-14s 未過（%d 年）人口 %d 評分 %d 等級 %d",
				w.CityName, wait[n]/48, w.TotalPop, w.CityScore, w.CityClass)
		default:
			t.Errorf("%s：跑完 %d 年還是沒有判定 —— 勝敗判定沒有觸發",
				w.CityName, wait[n]/48+1)
		}
	}
	t.Logf("不出手的情況下 %d/8 通關", win)
}
