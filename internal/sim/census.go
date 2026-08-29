package sim

// 普查、需求閥、稅收與預算。
// 證據：docs/re/09-census-valves-budget.md
// 一手出處：s_sim.c:414 SetValves、:524 ClearCensus、:559 TakeCensus、
//           :641 CollectTax、:670 UpdateFundEffects、w_budget.c:105 DoBudgetNow

// CensusRate／TaxFreq 是普查與收稅的週期。headers/sim.h
const (
	CensusRate = 4  // 每 4 刻做一次短期普查
	TaxFreq    = 48 // 每 48 刻（一年）收一次稅
)

// ClearCensus 把每輪要重數的計數歸零。s_sim.c:524
//
// ⚠ 它同時清掉 `PoliceMap` 與 `FireStMap`——那兩張圖是 `MapScan` 期間由
// 各個警消局「加上去」的，不是持久狀態。清錯地方會讓涵蓋率一路累積。
func (w *World) ClearCensus() {
	w.PwrdZCnt, w.UnPwrdZCnt = 0, 0
	w.FirePop = 0
	w.RoadTotal, w.RailTotal = 0, 0
	w.ResPop, w.ComPop, w.IndPop = 0, 0, 0
	w.ResZPop, w.ComZPop, w.IndZPop = 0, 0, 0
	w.HospPop, w.ChurchPop = 0, 0
	w.PolicePop, w.FireStPop = 0, 0
	w.StadiumPop = 0
	w.CoalPop, w.NuclearPop = 0, 0
	w.PortPop, w.APortPop = 0, 0
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			w.FireStMap[x][y] = 0
			w.PoliceMap[x][y] = 0
		}
	}
}

// taxTable 是稅率對成長的影響。s_sim.c:415
//
// 索引是 `CityTax + GameLevel`（上限 20）。0% 稅給 +200 的成長推力，
// 7% 是 0，20% 是 −600。**手冊建議的 5–7% 正好落在由正轉負的交界。**
var taxTable = [21]int{
	200, 150, 120, 100, 80, 50, 30, 0, -10, -40, -100,
	-150, -200, -250, -300, -350, -400, -450, -500, -550, -600,
}

// SetValves 更新住宅／商業／工業的需求閥。s_sim.c:414
//
// 這是整個模擬的核心回饋：由就業率、勞動力基礎、內部市場與稅率算出三個
// −2000…2000（商業與工業是 ±1500）的「速度」，分區成長時把它當基礎分數。
func (w *World) SetValves() {
	w.MiscHis[1] = int16(w.EMarket)
	w.MiscHis[2] = int16(w.ResPop)
	w.MiscHis[3] = int16(w.ComPop)
	w.MiscHis[4] = int16(w.IndPop)
	w.MiscHis[5] = int16(w.RValve)
	w.MiscHis[6] = int16(w.CValve)
	w.MiscHis[7] = int16(w.IValve)
	w.MiscHis[10] = int16(w.CrimeRamp)
	w.MiscHis[11] = int16(w.PolluteRamp)
	w.MiscHis[12] = int16(w.LVAverage)
	w.MiscHis[13] = int16(w.CrimeAverage)
	w.MiscHis[14] = int16(w.PolluteAverage)
	w.MiscHis[15] = int16(w.GameLevel)
	w.MiscHis[16] = int16(w.CityClass)
	w.MiscHis[17] = int16(w.CityScore)

	// ⚠ 這一整段是**單精度**（C 的 float 是 32 位元），而且
	// `NormResPop = ResPop / 8` 是**整數除法**——`ResPop` 是 short，
	// 8 是 int，所以先截斷再存進 float。用 float64 或浮點除法都會
	// 讓閥門差個一兩點；那個差要跑一百多個 frame 才會變成看得見的
	// 行為差異（`docs/re/12-tick-parity.md` §6之三）。
	//
	// 幾個 double 字面值（.02、3.7、1.3、1.2…）會把該次運算提升到
	// 雙精度再存回 float，下面照抄那個順序。s_sim.c:280
	normResPop := float32(w.ResPop / 8)
	w.LastTotalPop = w.TotalPop
	w.TotalPop = int(normResPop) + w.ComPop + w.IndPop

	employment := float32(1)
	if normResPop != 0 {
		employment = float32(int(w.ComHis[1])+int(w.IndHis[1])) / normResPop
	}
	migration := normResPop * (employment - 1)
	births := float32(float64(normResPop) * 0.02) // .02 是 double 字面值
	pjResPop := normResPop + migration + births

	laborBase := float32(1)
	if t := float32(int(w.ComHis[1]) + int(w.IndHis[1])); t != 0 {
		laborBase = float32(w.ResHis[1]) / t
	}
	if float64(laborBase) > 1.3 {
		laborBase = 1.3
	}
	if laborBase < 0 {
		laborBase = 0
	}

	// ⚠ 原版這裡有一個沒有用到的迴圈：
	//     for (z = 0; z < 2; z++) temp = ResHis[z] + ComHis[z] + IndHis[z];
	// 算出來的 temp 立刻被下一行覆蓋。不照抄（沒有可觀察的效果）。

	intMarket := float32(float64(normResPop+float32(w.ComPop)+float32(w.IndPop)) / 3.7)
	pjComPop := intMarket * laborBase

	extMarket := float32(1)
	switch w.GameLevel {
	case 0:
		extMarket = 1.2
	case 1:
		extMarket = 1.1
	case 2:
		extMarket = 0.98
	}
	pjIndPop := float32(w.IndPop) * laborBase * extMarket
	if pjIndPop < 5 {
		pjIndPop = 5
	}

	rratio := float32(1.3)
	if normResPop != 0 {
		rratio = pjResPop / normResPop
	}
	cratio := pjComPop
	if w.ComPop != 0 {
		cratio = pjComPop / float32(w.ComPop)
	}
	iratio := pjIndPop
	if w.IndPop != 0 {
		iratio = pjIndPop / float32(w.IndPop)
	}
	if rratio > 2 {
		rratio = 2
	}
	if cratio > 2 {
		cratio = 2
	}
	if iratio > 2 {
		iratio = 2
	}

	z := w.CityTax + w.GameLevel
	if z > 20 {
		z = 20
	}
	if z < 0 {
		z = 0 // 原版沒有這個下限；這裡是 Go 的索引保護，實際玩不到
	}
	rr := (rratio-1)*600 + float32(taxTable[z])
	cr := (cratio-1)*600 + float32(taxTable[z])
	ir := (iratio-1)*600 + float32(taxTable[z])

	// ⚠ 這六個 if 是「先看方向，再看有沒有到上限」——到了上限就整個不加，
	// 不是加了再夾。所以閥值可能停在 1999 而不是 2000。
	//
	// ⚠ `RValve += Rratio` 的 RValve 是 short：先提升成 float 相加，
	// 再**向零截斷**存回去。判斷正負看的是浮點值，不是截斷後的值。
	if rr > 0 && w.RValve < 2000 {
		w.RValve = int(float32(w.RValve) + rr)
	}
	if rr < 0 && w.RValve > -2000 {
		w.RValve = int(float32(w.RValve) + rr)
	}
	if cr > 0 && w.CValve < 1500 {
		w.CValve = int(float32(w.CValve) + cr)
	}
	if cr < 0 && w.CValve > -1500 {
		w.CValve = int(float32(w.CValve) + cr)
	}
	if ir > 0 && w.IValve < 1500 {
		w.IValve = int(float32(w.IValve) + ir)
	}
	if ir < 0 && w.IValve > -1500 {
		w.IValve = int(float32(w.IValve) + ir)
	}

	w.RValve = clampInt(w.RValve, -2000, 2000)
	w.CValve = clampInt(w.CValve, -1500, 1500)
	w.IValve = clampInt(w.IValve, -1500, 1500)

	// 體育場／海港／機場沒蓋時，對應的需求被封頂。
	if w.ResCap && w.RValve > 0 {
		w.RValve = 0
	}
	if w.ComCap && w.CValve > 0 {
		w.CValve = 0
	}
	if w.IndCap && w.IValve > 0 {
		w.IValve = 0
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// TakeCensus 把當期數字推進短期歷史圖（前 120 格）。s_sim.c:559
func (w *World) TakeCensus() {
	for x := 118; x >= 0; x-- {
		w.ResHis[x+1] = w.ResHis[x]
		w.ComHis[x+1] = w.ComHis[x]
		w.IndHis[x+1] = w.IndHis[x]
		w.CrimeHis[x+1] = w.CrimeHis[x]
		w.PollutionHis[x+1] = w.PollutionHis[x]
		w.MoneyHis[x+1] = w.MoneyHis[x]
	}
	w.ResHis[0] = int16(w.ResPop / 8)
	w.ComHis[0] = int16(w.ComPop)
	w.IndHis[0] = int16(w.IndPop)

	// ⚠ 犯罪與汙染進圖表前先做一階平滑（每次補四分之一的差距），
	// 所以圖表比實際值慢。
	w.CrimeRamp += (w.CrimeAverage - w.CrimeRamp) / 4
	w.CrimeHis[0] = int16(w.CrimeRamp)
	w.PolluteRamp += (w.PolluteAverage - w.PolluteRamp) / 4
	w.PollutionHis[0] = int16(w.PolluteRamp)

	x := w.CashFlow/20 + 128 // 收支縮到 0…255
	x = clampInt(x, 0, 255)
	w.MoneyHis[0] = int16(x)
	if w.CrimeHis[0] > 255 {
		w.CrimeHis[0] = 255
	}
	if w.PollutionHis[0] > 255 {
		w.PollutionHis[0] = 255
	}

	// 醫院與教堂的需求：每 256 個住宅人口要一座。三態。
	w.NeedHosp = compareNeed(w.HospPop, w.ResPop>>8)
	w.NeedChurch = compareNeed(w.ChurchPop, w.ResPop>>8)
}

// compareNeed 回 1 需要、0 剛好、−1 太多。s_sim.c:598
func compareNeed(have, want int) int {
	switch {
	case have < want:
		return 1
	case have > want:
		return -1
	}
	return 0
}

// Take2Census 推進長期歷史圖（後 120 格）。s_sim.c:611
func (w *World) Take2Census() {
	for x := 238; x >= 120; x-- {
		w.ResHis[x+1] = w.ResHis[x]
		w.ComHis[x+1] = w.ComHis[x]
		w.IndHis[x+1] = w.IndHis[x]
		w.CrimeHis[x+1] = w.CrimeHis[x]
		w.PollutionHis[x+1] = w.PollutionHis[x]
		w.MoneyHis[x+1] = w.MoneyHis[x]
	}
	w.ResHis[120] = int16(w.ResPop / 8)
	w.ComHis[120] = int16(w.ComPop)
	w.IndHis[120] = int16(w.IndPop)
	w.CrimeHis[120] = w.CrimeHis[0]
	w.PollutionHis[120] = w.PollutionHis[0]
	w.MoneyHis[120] = w.MoneyHis[0]
}

// CollectTax 收稅並跑預算。s_sim.c:641
//
// 三個難度的道路維護係數與稅收係數：**難度越高，路更貴、稅更少**。
func (w *World) CollectTax() {
	rLevels := [3]float64{0.7, 0.9, 1.2}
	fLevels := [3]float64{1.4, 1.2, 0.8}

	w.CashFlow = 0
	if w.TaxFlag {
		return
	}
	w.AvCityTax = 0
	w.PoliceFund = w.PolicePop * 100
	w.FireFund = w.FireStPop * 100
	w.RoadFund = int(float64(w.RoadTotal+w.RailTotal*2) * rLevels[w.GameLevel])
	w.TaxFund = int(float64(w.TotalPop*w.LVAverage/120) * float64(w.CityTax) * fLevels[w.GameLevel])

	if w.TotalPop != 0 {
		w.CashFlow = w.TaxFund - (w.PoliceFund + w.FireFund + w.RoadFund)
		w.DoBudget()
		return
	}
	w.RoadEffect = 32
	w.PoliceEffect = 1000
	w.FireEffect = 1000
}

// DoBudget 分配預算。w_budget.c:105 DoBudgetNow(0) 的自動路徑。
//
// **只實作自動預算。** 手動預算要開對話框等玩家決定，那是呈現層；
// 錢不夠時原版會強制關掉自動預算並跳出視窗（w_budget.c:207），
// 這裡改成「按比例配到用完」，差異記在 docs/re/09 §4。
func (w *World) DoBudget() {
	fireInt := int(float64(w.FireFund) * w.FirePercent)
	policeInt := int(float64(w.PoliceFund) * w.PolicePercent)
	roadInt := int(float64(w.RoadFund) * w.RoadPercent)
	total := fireInt + policeInt + roadInt
	yum := w.TaxFund + w.TotalFunds

	var fireValue, policeValue, roadValue int
	broke := yum <= total && total > 0
	switch {
	case yum > total:
		fireValue, policeValue, roadValue = fireInt, policeInt, roadInt
	case total > 0:
		// 優先序：道路 → 消防 → 警察。
		if yum > roadInt {
			roadValue = roadInt
			yum -= roadInt
			if yum > fireInt {
				fireValue = fireInt
				yum -= fireInt
				if yum > policeInt {
					policeValue = policeInt
				} else {
					policeValue = yum
					w.PolicePercent = ratioOrZero(yum, w.PoliceFund)
				}
			} else {
				fireValue = yum
				policeValue = 0
				w.PolicePercent = 0
				w.FirePercent = ratioOrZero(yum, w.FireFund)
			}
		} else {
			roadValue = yum
			w.RoadPercent = ratioOrZero(yum, w.RoadFund)
			w.FirePercent, w.PolicePercent = 0, 0
		}
	default:
		w.FirePercent, w.PolicePercent, w.RoadPercent = 1, 1, 1
	}

	w.FireSpend, w.PoliceSpend, w.RoadSpend = fireValue, policeValue, roadValue
	spent := fireValue + policeValue + roadValue
	w.TotalFunds += w.TaxFund - spent
	if broke {
		// 稅收加存款付不起編列的預算。w_budget.c:214 走的是同一個分支
		// （原版還會強制關掉自動預算並跳視窗，那是呈現層）。
		w.ClearMes()
		w.SendMes(MsgBroke)
	}
}

func ratioOrZero(part, whole int) float64 {
	if part > 0 && whole != 0 {
		return float64(part) / float64(whole)
	}
	return 0
}

// UpdateFundEffects 由實際支出算出三種服務的效果。s_sim.c:670
func (w *World) UpdateFundEffects() {
	if w.RoadFund != 0 {
		w.RoadEffect = int(float64(w.RoadSpend) / float64(w.RoadFund) * 32.0)
	} else {
		w.RoadEffect = 32
	}
	if w.PoliceFund != 0 {
		w.PoliceEffect = int(float64(w.PoliceSpend) / float64(w.PoliceFund) * 1000.0)
	} else {
		w.PoliceEffect = 1000
	}
	if w.FireFund != 0 {
		w.FireEffect = int(float64(w.FireSpend) / float64(w.FireFund) * 1000.0)
	} else {
		w.FireEffect = 1000
	}
}
