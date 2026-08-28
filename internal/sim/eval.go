package sim

// 城市評分與市民投票。證據：docs/re/10-evaluation.md／一手出處：s_eval.c

// ProbNum 是問題表的長度。headers/sim.h:178 #define PROBNUM 10
const ProbNum = 10

// Evaluation 是一次評分的結果。
type Evaluation struct {
	CityYes, CityNo int // 市民投票：滿意 / 不滿意（各 100 票）
	CityPop         int // 城市人口
	DeltaCityPop    int // 與上次的差
	CityAssValue    int // 資產價值
	CityClass       int // 0…5：村莊 / 小鎮 / 城市 / 首府 / 大都會 / 巨型都會
	CityScore       int // 0…1000
	DeltaCityScore  int //
	ProblemTable    [ProbNum]int
	ProblemVotes    [ProbNum]int
	ProblemOrder    [4]int // 前四大問題的索引；7 代表「沒問題」
	TrafficAverage  int
}

// 問題索引。s_eval.c:163-169
const (
	ProbCrime = iota
	ProbPollution
	ProbHousing
	ProbTaxes
	ProbTraffic
	ProbUnemployment
	ProbFire
	ProbNone = 7
)

// CityEvaluation 跑一次完整評分。s_eval.c:82
func (w *World) CityEvaluation() {
	if w.TotalPop == 0 {
		w.EvalInit()
		return
	}
	w.getAssValue()
	w.doPopNum()
	w.doProblems()
	w.getScore()
	w.doVotes()
}

// EvalInit 把評分重設成初值。s_eval.c:101
//
// ⚠ 起始分數是 **500**，不是 0。空城的分數不會掉到 0。
func (w *World) EvalInit() {
	w.Eval = Evaluation{CityScore: 500}
	w.CityClass = 0
	w.CityScore = 500
}

// getAssValue 由基礎建設算資產價值。s_eval.c:123
func (w *World) getAssValue() {
	z := w.RoadTotal * 5
	z += w.RailTotal * 10
	z += w.PolicePop * 1000
	z += w.FireStPop * 1000
	z += w.HospPop * 400
	z += w.StadiumPop * 3000
	z += w.PortPop * 5000
	z += w.APortPop * 10000
	z += w.CoalPop * 3000
	z += w.NuclearPop * 6000
	w.Eval.CityAssValue = z * 1000
}

// doPopNum 算城市人口與等級。s_eval.c:142
//
// ⚠ 商業與工業人口各乘 8，再整體乘 20。所以「城市人口」與住宅人口不是同一件事。
func (w *World) doPopNum() {
	old := w.Eval.CityPop
	w.Eval.CityPop = (w.ResPop + w.ComPop*8 + w.IndPop*8) * 20
	w.Eval.DeltaCityPop = w.Eval.CityPop - old

	c := 0
	for _, threshold := range []int{2000, 10000, 50000, 100000, 500000} {
		if w.Eval.CityPop > threshold {
			c++
		}
	}
	w.Eval.CityClass = c
	w.CityClass = c
}

// doProblems 填問題表、投票、排出前四大問題。s_eval.c:163
func (w *World) doProblems() {
	for z := range w.Eval.ProblemTable {
		w.Eval.ProblemTable[z] = 0
	}
	w.Eval.ProblemTable[ProbCrime] = w.CrimeAverage
	w.Eval.ProblemTable[ProbPollution] = w.PolluteAverage
	w.Eval.ProblemTable[ProbHousing] = int(float64(w.LVAverage) * 0.7)
	w.Eval.ProblemTable[ProbTaxes] = w.CityTax * 10
	w.Eval.ProblemTable[ProbTraffic] = w.averageTrf()
	w.Eval.ProblemTable[ProbUnemployment] = w.getUnemployment()
	w.Eval.ProblemTable[ProbFire] = w.getFire()

	w.voteProblems()

	taken := [ProbNum]bool{}
	for z := 0; z < 4; z++ {
		max, thisProb := 0, 0
		for x := 0; x < 7; x++ {
			if w.Eval.ProblemVotes[x] > max && !taken[x] {
				thisProb = x
				max = w.Eval.ProblemVotes[x]
			}
		}
		if max != 0 {
			taken[thisProb] = true
			w.Eval.ProblemOrder[z] = thisProb
		} else {
			w.Eval.ProblemOrder[z] = ProbNone
			w.Eval.ProblemTable[7] = 0
		}
	}
}

// voteProblems 讓市民對七個問題投票。s_eval.c:201
//
// ⚠ 迴圈的索引 `x` 遞增到 **> PROBNUM（10）** 才歸零，
// 但問題只有 0…6 有值——所以 7、8、9、10 這四輪一定不會投票，
// 而且 `ProblemTable[10]` 會**越界讀取**（原版陣列長度就是 10）。
// Go 版把索引夾在陣列內，行為差異記在 docs/re/10-evaluation.md §3。
func (w *World) voteProblems() {
	for z := range w.Eval.ProblemVotes {
		w.Eval.ProblemVotes[z] = 0
	}
	x, z, count := 0, 0, 0
	for z < 100 && count < 600 {
		// ⚠ 迴圈尾是 `if x > PROBNUM { x = 0 }`，不是 `>=`，
		// 所以 x 會走到 10，而 ProblemTable 只有 0..9 —— 原版每 11 次
		// 就讀一次越界。**那一次的 Rand(300) 照樣被呼叫**，只是拿去和
		// 相鄰記憶體比較（s_eval.c 的宣告順序是 ProblemTable、
		// ProblemTaken、ProblemVotes，所以讀到的多半是 ProblemTaken[0]，
		// 值域 {0,1}，幾乎永遠比不過）。
		//
		// 抽樣次數才是關鍵：跳過這一次會讓每次評估少抽約 54 次，
		// 亂數數列從此整條錯開。這裡照抽不誤，比較對象取 0。
		v := 0
		if x < len(w.Eval.ProblemTable) {
			v = w.Eval.ProblemTable[x]
		}
		if w.Rand.Rand(300) < v {
			w.Eval.ProblemVotes[x]++
			z++
		}
		x++
		if x > ProbNum {
			x = 0
		}
		count++
	}
}

// averageTrf 算平均車流。s_eval.c:224
//
// 只統計「有地價」的格子——空地不算。分母從 1 起算避免除以零。
func (w *World) averageTrf() int {
	total, count := 0, 1
	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			if w.LandValueMem[x][y] != 0 {
				total += int(w.TrfDensity[x][y])
				count++
			}
		}
	}
	w.Eval.TrafficAverage = int(float64(total/count) * 2.4)
	return w.Eval.TrafficAverage
}

// getUnemployment 由住宅人口與工作機會算失業率。s_eval.c:244
func (w *World) getUnemployment() int {
	b := (w.ComPop + w.IndPop) << 3
	if b == 0 {
		return 0
	}
	r := float64(w.ResPop) / float64(b)
	v := int((r - 1) * 255)
	if v > 255 {
		v = 255
	}
	return v
}

// getFire 由火災數算火災問題分數。s_eval.c:263
func (w *World) getFire() int {
	z := w.FirePop * 5
	if z > 255 {
		return 255
	}
	return z
}

// getScore 算城市分數。s_eval.c:276
//
// ⚠ 最後一行是 `CityScore = (CityScore + z) / 2`——**新分數只佔一半**，
// 所以分數是一階平滑的，不會跳。
func (w *World) getScore() {
	old := w.Eval.CityScore

	x := 0
	for z := 0; z < 7; z++ {
		x += w.Eval.ProblemTable[z]
	}
	x /= 3 // 原始碼註解寫 "7 + 2 average"
	if x > 256 {
		x = 256
	}
	z := (256 - x) * 4
	z = clampInt(z, 0, 1000)

	zf := float64(z)
	if w.ResCap {
		zf *= 0.85
	}
	if w.ComCap {
		zf *= 0.85
	}
	if w.IndCap {
		zf *= 0.85
	}
	if w.RoadEffect < 32 {
		zf -= float64(32 - w.RoadEffect)
	}
	if w.PoliceEffect < 1000 {
		zf *= 0.9 + float64(w.PoliceEffect)/10000.1
	}
	if w.FireEffect < 1000 {
		zf *= 0.9 + float64(w.FireEffect)/10000.1
	}
	if w.RValve < -1000 {
		zf *= 0.85
	}
	if w.CValve < -1000 {
		zf *= 0.85
	}
	if w.IValve < -1000 {
		zf *= 0.85
	}

	// 人口成長率乘數。
	sm := 1.0
	pop, delta := w.Eval.CityPop, w.Eval.DeltaCityPop
	switch {
	case pop == 0 || delta == 0 || delta == pop:
		sm = 1.0
	case delta > 0:
		sm = float64(delta)/float64(pop) + 1.0
	case delta < 0:
		sm = 0.95 + float64(delta)/float64(pop-delta)
	}
	zf *= sm
	zf -= float64(w.getFire())
	zf -= float64(w.CityTax)

	// 沒電的分區按比例扣分。
	tm := w.UnPwrdZCnt + w.PwrdZCnt
	if tm != 0 {
		zf *= float64(w.PwrdZCnt) / float64(tm)
	}

	z = clampInt(int(zf), 0, 1000)
	w.Eval.CityScore = (w.Eval.CityScore + z) / 2
	w.CityScore = w.Eval.CityScore
	w.Eval.DeltaCityScore = w.Eval.CityScore - old
}

// doVotes 一百位市民依分數投票。s_eval.c:332
func (w *World) doVotes() {
	w.Eval.CityYes, w.Eval.CityNo = 0, 0
	for z := 0; z < 100; z++ {
		if w.Rand.Rand(1000) < w.Eval.CityScore {
			w.Eval.CityYes++
		} else {
			w.Eval.CityNo++
		}
	}
}
