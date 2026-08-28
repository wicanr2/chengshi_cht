package sim

import "testing"

// 訊息埠的兩套規則不一樣，這是最容易寫錯的地方。
func TestSendMesPortRules(t *testing.T) {
	w := NewWorld(0)

	// 正數：先到先得。埠上有東西時後來的被丟掉。
	if !w.SendMes(MsgNeedRes) {
		t.Fatal("空埠應該收得下")
	}
	if w.SendMes(MsgNeedCom) {
		t.Fatal("埠上有東西時，正數訊息應該被丟掉")
	}
	if w.MessagePort != MsgNeedRes {
		t.Fatalf("埠上應該還是第一則，得到 %d", w.MessagePort)
	}

	// 負數（有圖）：不管埠上有沒有東西都覆蓋，但同一張圖不會重送。
	if !w.SendMes(-MsgCrimeHigh) {
		t.Fatal("有圖訊息應該覆蓋掉埠上的文字訊息")
	}
	if w.MessagePort != -MsgCrimeHigh {
		t.Fatalf("埠上應該是圖片訊息，得到 %d", w.MessagePort)
	}
	if w.SendMes(-MsgCrimeHigh) {
		t.Fatal("同一張圖不該重送")
	}

	// ClearMes 之後同一張圖才能再送。
	w.ClearMes()
	if !w.SendMes(-MsgCrimeHigh) {
		t.Fatal("ClearMes 之後應該能重送")
	}
}

// SendMesAt 只有在真的送出去時才記座標。
func TestSendMesAtOnlyRecordsWhenSent(t *testing.T) {
	w := NewWorld(0)
	w.SendMesAt(MsgNeedRes, 10, 20)
	if w.MesX != 10 || w.MesY != 20 {
		t.Fatalf("座標應為 (10,20)，得到 (%d,%d)", w.MesX, w.MesY)
	}
	w.SendMesAt(MsgNeedCom, 30, 40) // 埠滿了，應該整則被丟掉
	if w.MesX != 10 || w.MesY != 20 {
		t.Fatalf("沒送出去就不該改座標，得到 (%d,%d)", w.MesX, w.MesY)
	}
}

// 分區上限旗標：條件不成立時要被清掉，不是只在成立時設。
// 漏掉 else 分支的話，蓋好體育場之後住宅區永遠長不起來。
func TestSendMessagesClearsCaps(t *testing.T) {
	w := NewWorld(0)
	w.ResPop = 600
	w.StadiumPop = 0
	w.CityTime = 26
	w.SendMessages()
	if !w.ResCap {
		t.Fatal("人口夠多又沒體育場，ResCap 應該被設起來")
	}

	w.StadiumPop = 1
	w.CityTime = 26 + 64 // 下一輪的同一個 case
	w.SendMessages()
	if w.ResCap {
		t.Fatal("蓋了體育場之後 ResCap 應該被清掉")
	}
}

// CheckGrowth 的人口算法和 TakeCensus 的 TotalPop 不同，而且同一個
// 等級只通知一次。
func TestCheckGrowthMilestones(t *testing.T) {
	w := NewWorld(0)
	w.CityTime = 0
	w.ResPop = 50 // (50 + 0 + 0) * 20 = 1000
	w.CheckGrowth()
	if w.LastCityPop != 1000 {
		t.Fatalf("人口應為 1000，得到 %d", w.LastCityPop)
	}
	if w.MessagePort != 0 {
		t.Fatal("第一次沒有基準，不該發訊息")
	}

	w.ResPop = 150 // 3000，跨過 2000
	w.CheckGrowth()
	if w.MessagePort != -MsgPopTown {
		t.Fatalf("應該發出 2000 人的里程碑，得到 %d", w.MessagePort)
	}

	// 同一個等級不重複通知
	w.ClearMes()
	w.ResPop = 160
	w.CheckGrowth()
	if w.MessagePort != 0 {
		t.Fatalf("同一個等級不該重送，得到 %d", w.MessagePort)
	}

	// 一次跨好幾級只會發最高的那一則（原版是五個獨立 if，後面覆蓋前面）
	w2 := NewWorld(0)
	w2.CityTime = 0 // NewWorld 的初值是 50，50&3 != 0 會直接跳過檢查
	w2.ResPop = 50
	w2.CheckGrowth()
	w2.ResPop = 6000 // 120 000，一口氣跨過四個門檻
	w2.CheckGrowth()
	if w2.MessagePort != -MsgPopMetro {
		t.Fatalf("應該只發最高的那一則（%d），得到 %d", -MsgPopMetro, w2.MessagePort)
	}

	// 每四刻才檢查一次
	w3 := NewWorld(0)
	w3.CityTime = 1
	w3.ResPop = 999
	w3.CheckGrowth()
	if w3.LastCityPop != 0 {
		t.Fatal("CityTime & 3 != 0 時不該檢查")
	}
}

type gameOverSpy struct{ won, lost bool }

func (g *gameOverSpy) OnWin()  { g.won = true }
func (g *gameOverSpy) OnLose() { g.lost = true }

// 劇本判定只看計時器歸零那一刻的狀態。
func TestScenarioScore(t *testing.T) {
	cases := []struct {
		name string
		typ  int
		set  func(*World)
		win  bool
	}{
		{"無聊鎮達標", 1, func(w *World) { w.CityClass = 4 }, true},
		{"無聊鎮沒達標", 1, func(w *World) { w.CityClass = 3 }, false},
		{"伯恩交通夠順", 4, func(w *World) { w.Eval.TrafficAverage = 79 }, true},
		{"伯恩塞車", 4, func(w *World) { w.Eval.TrafficAverage = 80 }, false},
		{"東京評分夠高", 5, func(w *World) { w.CityScore = 501 }, true},
		{"底特律治安改善", 6, func(w *World) { w.CrimeAverage = 59 }, true},
		{"底特律治安沒改善", 6, func(w *World) { w.CrimeAverage = 60 }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := NewWorld(0)
			spy := &gameOverSpy{}
			w.GameOver = spy
			c.set(w)
			w.DoScenarioScore(c.typ)
			wantPort := -MsgScenarioLose
			if c.win {
				wantPort = -MsgScenarioWin
			}
			if w.MessagePort != wantPort {
				t.Fatalf("訊息埠應為 %d，得到 %d", wantPort, w.MessagePort)
			}
			if spy.lost == c.win {
				t.Fatalf("勝敗回呼不對：win=%v lost=%v", c.win, spy.lost)
			}
		})
	}
}

// 倒數計時：ScoreWait 歸零那一刻才判定，中途不判。
func TestScenarioScoreWaitsForTimer(t *testing.T) {
	w := NewWorld(0)
	w.Scenario = ScenarioDullsville
	w.ScoreType = 1
	w.ScoreWait = 3
	w.CityClass = 5
	for i := 0; i < 2; i++ {
		w.CityTime = 100 // 避開會發別的訊息的 case
		w.SendMessages()
		if w.MessagePort != 0 {
			t.Fatalf("第 %d 次就判定了，太早", i+1)
		}
	}
	w.CityTime = 100
	w.SendMessages()
	if w.MessagePort != -MsgScenarioWin {
		t.Fatalf("倒數歸零應該判定過關，得到 %d", w.MessagePort)
	}
}
