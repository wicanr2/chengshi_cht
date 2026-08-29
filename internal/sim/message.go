package sim

// 訊息系統。證據：docs/re/14-messages.md／一手出處：s_msg.c
//
// 訊息不只是提示文字。它同時承擔三件規則層的工作：
//
//   1. `SendMessages` 依 `CityTime & 63` 輪流檢查十七個城市狀況，
//      **順便設定 ResCap／ComCap／IndCap** —— 那三個旗標直接壓住
//      對應分區的成長（見 evalRes／evalCom／evalInd）。
//   2. `CheckGrowth` 判定人口里程碑，決定城市等級的升級訊息。
//   3. `DoScenarioScore` 判定劇本的勝敗。
//
// 所以「還沒做 UI 所以先不做訊息」是錯的：少了它，分區上限永遠不會生效。

// 訊息編號。正數是純文字，負數是「有圖的訊息」（原版會先開圖片視窗，
// 再把同一個編號當文字重送一次）。編號對應 translations/messages/*.toml
// 的索引，文本不寫在程式裡。
const (
	MsgNeedRes         = 1
	MsgNeedCom         = 2
	MsgNeedInd         = 3
	MsgNeedRoads       = 4
	MsgNeedRail        = 5
	MsgNeedPower       = 6
	MsgNeedStadium     = 7
	MsgNeedSeaport     = 8
	MsgNeedAirport     = 9
	MsgPollutionHigh   = 10
	MsgCrimeHigh       = 11
	MsgTrafficHigh     = 12
	MsgNeedFireStation = 13
	MsgNeedPolice      = 14
	MsgBlackout        = 15
	MsgTaxTooHigh      = 16
	MsgRoadsDeteriorat = 17
	MsgFireDeptNeeds   = 18
	MsgPoliceNeeds     = 19

	MsgScenarioWin  = 100 // 送出時取負值
	MsgScenarioLose = 200 // 送出時取負值
)

// 災難、工具與預算訊息。s_disast.c／w_sprite.c／w_tool.c／w_budget.c 的
// `SendMes` 引數，編號與 `.PTF` 第 0 段的索引差 1（索引 19 ＝ 訊息 20）。
//
// ⚠ **DOS 與 Micropolis 的訊息表只在第 30 則不一樣**：Micropolis 是
// `Firebombing reported !`，DOS 1.10 是 `Bulldozing too many trees.`。
// 所以空襲（`dropFireBombs`）不能照 Micropolis 送 −30——那會讓玩家看到
// 「亂砍樹」。DOS 版空襲用哪一則還沒解（docs/re/14-messages.md §5）。
const (
	MsgFire         = 20 // 火警
	MsgMonster      = 21 // 怪獸
	MsgTornado      = 22 // 龍捲風
	MsgEarthquake   = 23 // 大地震
	MsgPlaneCrash   = 24 // 空難
	MsgShipwreck    = 25 // 船難
	MsgTrainCrash   = 26 // 火車事故
	MsgCopterCrash  = 27 // 直升機事故
	MsgBroke        = 29 // 破產
	MsgExplosion    = 32 // 爆炸
	MsgNoMoney      = 33 // 錢不夠
	MsgNeedsClear   = 34 // 要先推平
	MsgBrownout     = 40 // 供電到頂
	MsgHeavyTraffic = 41 // 交通壅塞
	MsgFlood        = 42 // 水災
	MsgMeltdown     = 43 // 爐心熔毀
)

// 人口里程碑訊息（CheckGrowth）。
const (
	MsgPopTown    = 35 // 2 000
	MsgPopCity    = 36 // 10 000
	MsgPopCapital = 37 // 50 000
	MsgPopMetro   = 38 // 100 000
	MsgPopMegalop = 39 // 500 000
)

// GameOverHook 讓呈現層接勝敗。規則層只負責判定，不負責畫面。
type GameOverHook interface {
	OnWin()
	OnLose()
}

// SendMes 把一個訊息放進訊息埠。s_msg.c:243
//
// 兩種編號的規則不一樣：
//
//   - 負數（有圖）：只要和上一張圖不同就覆蓋，**不管埠上有沒有東西**。
//   - 正數（純文字）：埠上有東西就丟掉，先到先得。
//
// 回傳有沒有真的送出去（SendMesAt 要靠它決定要不要記座標）。
func (w *World) SendMes(n int) bool {
	if n < 0 {
		if n != w.LastPicNum {
			w.MessagePort = n
			w.MesX, w.MesY = 0, 0
			w.LastPicNum = n
			return true
		}
		return false
	}
	if w.MessagePort == 0 {
		w.MessagePort = n
		w.MesX, w.MesY = 0, 0
		return true
	}
	return false
}

// SendMesAt 送一個帶座標的訊息（玩家可以按「前往」跳過去）。s_msg.c:265
func (w *World) SendMesAt(n, x, y int) {
	if w.SendMes(n) {
		w.MesX, w.MesY = x, y
	}
}

// ClearMes 清掉訊息埠。s_msg.c:234
func (w *World) ClearMes() {
	w.MessagePort = 0
	w.MesX, w.MesY = 0, 0
	w.LastPicNum = 0
}

// SendMessages 是十六相位主迴圈的相位 10。s_msg.c:170
//
// ⚠ 這個函式在原版**不消耗亂數**（唯一的 Rand 在 doMessage 的音效分支，
// 那屬於呈現層）。但它會改 ResCap／ComCap／IndCap，那會改變分區成長，
// 進而改變抽樣次數——所以它不是「純顯示」。
func (w *World) SendMessages() {
	if w.Scenario != 0 && w.ScoreType != 0 && w.ScoreWait != 0 {
		w.ScoreWait--
		if w.ScoreWait == 0 {
			w.DoScenarioScore(w.ScoreType)
		}
	}

	w.CheckGrowth()

	totalZPop := w.ResZPop + w.ComZPop + w.IndZPop
	powerPop := w.NuclearPop + w.CoalPop

	switch w.CityTime & 63 {
	case 1:
		if totalZPop>>2 >= w.ResZPop {
			w.SendMes(MsgNeedRes)
		}
	case 5:
		if totalZPop>>3 >= w.ComZPop {
			w.SendMes(MsgNeedCom)
		}
	case 10:
		if totalZPop>>3 >= w.IndZPop {
			w.SendMes(MsgNeedInd)
		}
	case 14:
		if totalZPop > 10 && totalZPop<<1 > w.RoadTotal {
			w.SendMes(MsgNeedRoads)
		}
	case 18:
		if totalZPop > 50 && totalZPop > w.RailTotal {
			w.SendMes(MsgNeedRail)
		}
	case 22:
		if totalZPop > 10 && powerPop == 0 {
			w.SendMes(MsgNeedPower)
		}
	case 26:
		// ⚠ 這三個 case 除了送訊息，還設分區上限。
		// 「人口夠多但沒有體育場」就把住宅成長壓住，商業要機場、
		// 工業要港口。條件不成立時**要清掉旗標**，不是只在成立時設。
		if w.ResPop > 500 && w.StadiumPop == 0 {
			w.SendMes(MsgNeedStadium)
			w.ResCap = true
		} else {
			w.ResCap = false
		}
	case 28:
		if w.IndPop > 70 && w.PortPop == 0 {
			w.SendMes(MsgNeedSeaport)
			w.IndCap = true
		} else {
			w.IndCap = false
		}
	case 30:
		if w.ComPop > 100 && w.APortPop == 0 {
			w.SendMes(MsgNeedAirport)
			w.ComCap = true
		} else {
			w.ComCap = false
		}
	case 32:
		if tm := w.UnPwrdZCnt + w.PwrdZCnt; tm != 0 {
			if float64(w.PwrdZCnt)/float64(tm) < 0.7 {
				w.SendMes(MsgBlackout)
			}
		}
	case 35:
		// ⚠ 原始碼裡這個門檻寫的是 60，旁邊留著 `/* 80 */` 的註解——
		// 官方改過。以程式碼為準。
		if w.PolluteAverage > 60 {
			w.SendMes(-MsgPollutionHigh)
		}
	case 42:
		if w.CrimeAverage > 100 {
			w.SendMes(-MsgCrimeHigh)
		}
	case 45:
		if w.TotalPop > 60 && w.FireStPop == 0 {
			w.SendMes(MsgNeedFireStation)
		}
	case 48:
		if w.TotalPop > 60 && w.PolicePop == 0 {
			w.SendMes(MsgNeedPolice)
		}
	case 51:
		if w.CityTax > 12 {
			w.SendMes(MsgTaxTooHigh)
		}
	case 54:
		if w.RoadEffect < 20 && w.RoadTotal > 30 {
			w.SendMes(MsgRoadsDeteriorat)
		}
	case 57:
		if w.FireEffect < 700 && w.TotalPop > 20 {
			w.SendMes(MsgFireDeptNeeds)
		}
	case 60:
		if w.PoliceEffect < 700 && w.TotalPop > 20 {
			w.SendMes(MsgPoliceNeeds)
		}
	case 63:
		if w.Eval.TrafficAverage > 60 {
			w.SendMes(-MsgTrafficHigh)
		}
	}
}

// CheckGrowth 判定人口里程碑。s_msg.c:277
//
// ⚠ 這裡的人口算法和 TakeCensus 的 TotalPop **不一樣**：
// `(ResPop + ComPop*8 + IndPop*8) * 20`。同一款遊戲裡有好幾個
// 「人口」，數字對不上是正常的，不要拿其中一個去校正另一個。
//
// ⚠ 每四刻才檢查一次（`CityTime & 3`），而且靠 LastCategory 去重——
// 同一個等級只會通知一次，即使人口在門檻附近來回震盪。
func (w *World) CheckGrowth() {
	if w.CityTime&3 != 0 {
		return
	}
	thisPop := (w.ResPop + w.ComPop*8 + w.IndPop*8) * 20
	z := 0
	if w.LastCityPop != 0 {
		switch {
		case w.LastCityPop < 2000 && thisPop >= 2000:
			z = MsgPopTown
		}
		// 原版是五個獨立的 if，後面的會覆蓋前面的——人口一次跨過
		// 好幾個門檻時，只會發出**最高**的那一則。照抄。
		if w.LastCityPop < 10000 && thisPop >= 10000 {
			z = MsgPopCity
		}
		if w.LastCityPop < 50000 && thisPop >= 50000 {
			z = MsgPopCapital
		}
		if w.LastCityPop < 100000 && thisPop >= 100000 {
			z = MsgPopMetro
		}
		if w.LastCityPop < 500000 && thisPop >= 500000 {
			z = MsgPopMegalop
		}
	}
	if z != 0 && z != w.LastCategory {
		w.SendMes(-z)
		w.LastCategory = z
	}
	w.LastCityPop = thisPop
}

// DoScenarioScore 判定劇本勝敗。s_msg.c:302
//
// 八個劇本各有各的過關條件，而且**只在災難計時器歸零的那一刻判一次**
// （ScoreWait 由 s_sim.c:384 的 ScoreWaitTab 設定）。中途達標沒有用，
// 中途失守也沒有關係——只看那一刻的狀態。
func (w *World) DoScenarioScore(typ int) {
	z := -MsgScenarioLose
	switch typ {
	case 1, 2, 3: // 無聊鎮、舊金山、漢堡：看城市等級
		if w.CityClass >= 4 {
			z = -MsgScenarioWin
		}
	case 4: // 伯恩：看交通
		if w.Eval.TrafficAverage < 80 {
			z = -MsgScenarioWin
		}
	case 5: // 東京：看城市評分
		if w.CityScore > 500 {
			z = -MsgScenarioWin
		}
	case 6: // 底特律：看犯罪
		if w.CrimeAverage < 60 {
			z = -MsgScenarioWin
		}
	case 7, 8: // 波士頓、里約：看城市評分
		if w.CityScore > 500 {
			z = -MsgScenarioWin
		}
	}
	w.ClearMes()
	w.SendMes(z)
	if z == -MsgScenarioLose && w.GameOver != nil {
		w.GameOver.OnLose()
	}
}
