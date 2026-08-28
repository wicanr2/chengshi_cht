package sim

// 模擬主迴圈。證據：docs/re/11-simulate-loop.md／一手出處：s_sim.c:113 Simulate
//
// 一「刻」（tick）分成十六個相位，每次呼叫 Simulate 做一個相位。
// 相位 1–8 各掃八分之一張地圖，其餘相位做普查、稅收、四個掃描與災難。
// 所以**掃描全圖要八個相位、也就是半刻**，而且各種掃描的頻率還受速度影響。

// 各種掃描依速度的週期。s_sim.c:114-118
//
// 索引是 SimSpeed（0…3）。速度越快，掃描間隔越長——
// 這不是最佳化，是遊戲設計：快轉時模擬得比較粗。
var (
	spdPwr = [4]int{1, 2, 4, 5}
	spdPtl = [4]int{1, 2, 7, 17}
	spdCri = [4]int{1, 1, 8, 18}
	spdPop = [4]int{1, 1, 9, 19}
	spdFir = [4]int{1, 1, 10, 20}
)

// Simulate 做一個相位。s_sim.c:113
func (w *World) Simulate(mod16 int) {
	x := w.SimSpeed
	if x > 3 {
		x = 3
	}
	if x < 0 {
		x = 0
	}

	switch mod16 {
	case 0:
		w.Scycle++
		if w.Scycle > 1023 {
			w.Scycle = 0 // 原始碼註解：this is cosmic
		}
		if w.DoInitialEval {
			w.DoInitialEval = false
			w.CityEvaluation()
		}
		w.CityTime++
		w.AvCityTax += w.CityTax
		if w.Scycle&1 == 0 {
			w.SetValves()
		}
		w.ClearCensus()
	case 1, 2, 3, 4, 5, 6, 7, 8:
		w.MapScan((mod16-1)*WorldX/8, mod16*WorldX/8)
	case 9:
		if w.CityTime%CensusRate == 0 {
			w.TakeCensus()
		}
		if w.CityTime%(CensusRate*12) == 0 {
			w.Take2Census()
		}
		if w.CityTime%TaxFreq == 0 {
			w.CollectTax()
			w.CityEvaluation()
		}
	case 10:
		if w.Scycle%5 == 0 {
			w.DecROGMem()
		}
		w.DecTrafficMem()
		w.SendMessages()
	case 11:
		if w.Scycle%spdPwr[x] == 0 {
			w.DoPowerScan()
			w.NewPower = true
		}
	case 12:
		if w.Scycle%spdPtl[x] == 0 {
			w.PTLScan()
		}
	case 13:
		if w.Scycle%spdCri[x] == 0 {
			w.CrimeScan()
		}
	case 14:
		if w.Scycle%spdPop[x] == 0 {
			w.PopDenScan()
		}
	case 15:
		if w.Scycle%spdFir[x] == 0 {
			w.FireAnalysis()
		}
		w.DoDisasters()
	}
}

// Tick 跑完整的十六個相位，也就是一刻。
func (w *World) Tick() {
	for i := 0; i < 16; i++ {
		w.Simulate(i)
	}
}

// DecTrafficMem 讓車流密度隨時間衰退。s_sim.c:256
//
// ⚠ 衰退量分段：超過 200 的減 34、24…200 的減 24、24 以下直接歸零。
// 所以壅塞路段退得比較快，而低流量路段是一步清零不是漸退。
func (w *World) DecTrafficMem() {
	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			z := int(w.TrfDensity[x][y])
			if z == 0 {
				continue
			}
			switch {
			case z > 200:
				w.TrfDensity[x][y] = uint8(z - 34)
			case z > 24:
				w.TrfDensity[x][y] = uint8(z - 24)
			default:
				w.TrfDensity[x][y] = 0
			}
		}
	}
}

// DecROGMem 讓成長率隨時間趨近零。s_sim.c:273
//
// ⚠ 夾限在**遞減之後**才做，而且只夾一邊：先 ±1 再看有沒有超過 ±200。
func (w *World) DecROGMem() {
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			z := w.RateOGMem[x][y]
			if z == 0 {
				continue
			}
			if z > 0 {
				w.RateOGMem[x][y]--
				if z > 200 {
					w.RateOGMem[x][y] = 200
				}
				continue
			}
			w.RateOGMem[x][y]++
			if z < -200 {
				w.RateOGMem[x][y] = -200
			}
		}
	}
}

// SendMessages 是訊息系統的入口。s_msg.c:75
//
// **還沒實作。** 訊息與劇本勝敗判定在 s_msg.c，見 docs/re/11 §4。
func (w *World) SendMessages() {}

// DoSimInit 是載入或新開城市之後的初始化。s_sim.c:207
func (w *World) DoSimInit() {
	w.Fcycle = 0
	w.Scycle = 0

	switch w.InitSimLoad {
	case 2:
		w.initSimMemory()
	case 1:
		w.simLoadInit()
	}

	w.SetValves()
	w.ClearCensus()
	w.MapScan(0, WorldX)
	w.DoPowerScan()
	w.NewPower = true
	w.PTLScan()
	w.CrimeScan()
	w.PopDenScan()
	w.FireAnalysis()
	w.TotalPop = 1
	w.DoInitialEval = true
}

// initSimMemory 是「新城市」的初始化。s_sim.c:295
func (w *World) initSimMemory() {
	w.setCommonInits()
	for x := 0; x < HistLen; x++ {
		w.ResHis[x] = 0
		w.ComHis[x] = 0
		w.IndHis[x] = 0
		w.MoneyHis[x] = 128 // ⚠ 資金圖的基準是 128（0 收支），不是 0
		w.CrimeHis[x] = 0
		w.PollutionHis[x] = 0
	}
	w.CrimeRamp, w.PolluteRamp = 0, 0
	w.TotalPop = 0
	w.RValve, w.CValve, w.IValve = 0, 0, 0
	w.ResCap, w.ComCap, w.IndCap = false, false, false
	w.EMarket = 6.0
	w.DisasterEvent = 0
	w.ScoreType = 0
	w.DoPowerScan()
	w.NewPower = true
	w.InitSimLoad = 0
}

// simLoadInit 是「載入城市」的初始化。s_sim.c:333
//
// ⚠ 它把 `PowerMap` **整張設成全 1**，然後 `DoNilPower()` 讓所有分區都帶電。
// 那是暫時狀態：`DoSimInit` 隨後的 `DoPowerScan` 會算出正確的電力網。
// 少了這一步，載入後的第一輪 `MapScan` 會把所有分區判成沒電。
func (w *World) simLoadInit() {
	disTab := [9]int{0, 2, 10, 5, 20, 3, 5, 5, 2 * 48}
	scoreWaitTab := [9]int{0, 30 * 48, 5 * 48, 5 * 48, 10 * 48, 5 * 48, 10 * 48, 5 * 48, 10 * 48}

	w.EMarket = float64(w.MiscHis[1])
	w.ResPop = int(w.MiscHis[2])
	w.ComPop = int(w.MiscHis[3])
	w.IndPop = int(w.MiscHis[4])
	w.RValve = int(w.MiscHis[5])
	w.CValve = int(w.MiscHis[6])
	w.IValve = int(w.MiscHis[7])
	w.CrimeRamp = int(w.MiscHis[10])
	w.PolluteRamp = int(w.MiscHis[11])
	w.LVAverage = int(w.MiscHis[12])
	w.CrimeAverage = int(w.MiscHis[13])
	w.PolluteAverage = int(w.MiscHis[14])
	w.GameLevel = int(w.MiscHis[15])

	if w.CityTime < 0 {
		w.CityTime = 0
	}
	if w.EMarket == 0 {
		w.EMarket = 4.0
	}
	if w.GameLevel > 2 || w.GameLevel < 0 {
		w.GameLevel = 0
	}

	w.setCommonInits()

	w.CityClass = int(w.MiscHis[16])
	w.CityScore = int(w.MiscHis[17])
	if w.CityClass > 5 || w.CityClass < 0 {
		w.CityClass = 0
	}
	if w.CityScore > 999 || w.CityScore < 1 {
		w.CityScore = 500
	}
	w.Eval.CityScore = w.CityScore
	w.Eval.CityClass = w.CityClass

	w.ResCap, w.ComCap, w.IndCap = false, false, false
	w.AvCityTax = (w.CityTime % 48) * 7

	for i := range w.PowerMap {
		w.PowerMap[i] = 0xFFFF
	}
	w.doNilPower()

	if w.Scenario > 8 {
		w.Scenario = 0
	}
	if w.Scenario != 0 {
		w.DisasterEvent = int(w.Scenario)
		w.DisasterWait = disTab[w.DisasterEvent]
		w.ScoreType = w.DisasterEvent
		w.ScoreWait = scoreWaitTab[w.DisasterEvent]
	} else {
		w.DisasterEvent = 0
		w.ScoreType = 0
	}

	w.RoadEffect = 32
	w.PoliceEffect = 1000
	w.FireEffect = 1000
	w.InitSimLoad = 0
}

// setCommonInits。s_sim.c:398
func (w *World) setCommonInits() {
	w.EvalInit()
	w.RoadEffect = 32
	w.PoliceEffect = 1000
	w.FireEffect = 1000
	w.TaxFlag = false
	w.TaxFund = 0
}

// doNilPower 讓所有分區帶電。s_sim.c:237
func (w *World) doNilPower() {
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			z := w.Map[x][y]
			if z&ZONEBIT == 0 {
				continue
			}
			w.SMapX, w.SMapY = x, y
			w.CChr = z
			w.CChr9 = int(z & LOMASK)
			w.SetZPower()
		}
	}
}
