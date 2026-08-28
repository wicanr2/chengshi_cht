package sim

// 分區處理。證據：docs/re/07-traffic-and-zones.md／規格：docs/spec/traffic-and-zones.md
// 一手出處：s_zone.c
//
// 每個分區中心每一輪掃描做三件事：算自己的人口、跑一次交通、
// 依「需求閥 ＋ 本地評分」擲骰決定長大還是縮小。

// 旗標組合。s_zone.c:118-119
const (
	ascBit = ANIMBIT | CONDBIT | BURNBIT
	regBit = CONDBIT | BURNBIT
)

// DoZone 是分區的總入口。s_zone.c:68
func (w *World) DoZone() {
	zonePwr := w.SetZPower()
	if zonePwr {
		w.PwrdZCnt++
	} else {
		w.UnPwrdZCnt++
	}

	switch {
	case w.CChr9 > PORTBASE:
		w.doSPZone(zonePwr)
	case w.CChr9 < HOSPITAL:
		w.doResidential(zonePwr)
	case w.CChr9 < COMBASE:
		w.doHospChur()
	case w.CChr9 < INDBASE:
		w.doCommercial(zonePwr)
	default:
		w.doIndustrial(zonePwr)
	}
}

// SetZPower 依 PowerMap 設目前這一格的 PWRBIT，並回報有沒有電。s_zone.c:624
func (w *World) SetZPower() bool {
	if w.CChr9 == NUCLEAR || w.CChr9 == POWERPLANT {
		w.Map[w.SMapX][w.SMapY] = w.CChr | PWRBIT
		return true
	}
	word, mask := PowerWord(w.SMapX, w.SMapY)
	if word < PwrMapSize && w.PowerMap[word]&mask != 0 {
		w.Map[w.SMapX][w.SMapY] = w.CChr | PWRBIT
		return true
	}
	w.Map[w.SMapX][w.SMapY] = w.CChr &^ uint16(PWRBIT)
	return false
}

// doHospChur 處理醫院與教堂。s_zone.c:97
//
// NeedHosp／NeedChurch 是三態：−1 代表太多了，這時有二十分之一機率把自己
// 換成住宅區（`ZonePlop(RESBASE)`）——**醫院會自己消失**，這是原版行為。
func (w *World) doHospChur() {
	if w.CChr9 == HOSPITAL {
		w.HospPop++
		if w.CityTime&15 == 0 {
			w.repairZone(HOSPITAL, 3)
		}
		if w.NeedHosp == -1 && w.Rand.Rand(20) == 0 {
			w.zonePlop(RESBASE)
		}
	}
	if w.CChr9 == CHURCH {
		w.ChurchPop++
		if w.CityTime&15 == 0 {
			w.repairZone(CHURCH, 3)
		}
		if w.NeedChurch == -1 && w.Rand.Rand(20) == 0 {
			w.zonePlop(RESBASE)
		}
	}
}

// setSmoke 開關工業區的煙囪動畫。s_zone.c:121
//
// ⚠ 原始碼在這裡對同一格連寫兩次（`Map[xx][yy] = A; Map[xx][yy] = B;`），
// 第一次的值立刻被覆蓋。看起來像編輯殘留，但照抄——最終值只有第二次那個。
func (w *World) setSmoke(zonePower bool) {
	aniThis := [8]bool{true, false, true, true, false, false, true, true}
	dx1 := [8]int{-1, 0, 1, 0, 0, 0, 0, 1}
	dy1 := [8]int{-1, 0, -1, -1, 0, 0, -1, -1}
	aniTabB := [8]int{0, 0, 36, 44, 0, 0, 52, 60}
	aniTabC := [8]int{IND1, 0, IND2, IND4, 0, 0, IND6, IND8}
	aniTabD := [8]int{IND1, 0, IND3, IND5, 0, 0, IND7, IND9}

	if w.CChr9 < IZB {
		return
	}
	z := ((w.CChr9 - IZB) >> 3) & 7
	if !aniThis[z] {
		return
	}
	xx, yy := w.SMapX+dx1[z], w.SMapY+dy1[z]
	if !InBounds(xx, yy) {
		return
	}
	cur := int(w.Map[xx][yy] & LOMASK)
	if zonePower {
		if cur == aniTabC[z] {
			w.Map[xx][yy] = uint16(ascBit | (SMOKEBASE + aniTabB[z]))
		}
	} else {
		if cur > aniTabC[z] {
			w.Map[xx][yy] = uint16(regBit | aniTabD[z])
		}
	}
}

// growShrink 是三種分區共用的成長／衰退擲骰。s_zone.c:161、:192、:230
//
// ⚠ 那兩個門檻式子有**明確的 short 轉型**：
//
//	((short)(zscore - 26380)) > ((short)Rand16Signed())
//
// 26380 這個常數會讓 zscore 減出來的值繞回 int16 的另一端——
// 這不是比大小，是**刻意利用溢位**做出的機率曲線。用 int 算會得到完全不同的結果。
func growShrink(zscore int, r *Rand) (grow, shrink bool) {
	if zscore > -350 && int16(zscore-26380) > int16(r.Rand16Signed()) {
		return true, false
	}
	if zscore < 350 && int16(zscore+26380) < int16(r.Rand16Signed()) {
		return false, true
	}
	return false, false
}

// doIndustrial。s_zone.c:161
func (w *World) doIndustrial(zonePwr bool) {
	w.IndZPop++
	w.setSmoke(zonePwr)
	tpop := izPop(w.CChr9)
	w.IndPop += tpop
	trfGood := 1
	if tpop > w.Rand.Rand(5) {
		trfGood = w.MakeTraf(2)
	}
	if trfGood == -1 {
		w.doIndOut(tpop, w.Rand.Rand16()&1)
		return
	}
	if w.Rand.Rand16()&7 != 0 {
		return
	}
	zscore := w.IValve + evalInd(trfGood)
	if !zonePwr {
		zscore = -500
	}
	grow, shrink := growShrink(zscore, w.Rand)
	if grow {
		w.doIndIn(tpop, w.Rand.Rand16()&1)
	} else if shrink {
		w.doIndOut(tpop, w.Rand.Rand16()&1)
	}
}

// doCommercial。s_zone.c:192
func (w *World) doCommercial(zonePwr bool) {
	w.ComZPop++
	tpop := czPop(w.CChr9)
	w.ComPop += tpop
	trfGood := 1
	if tpop > w.Rand.Rand(5) {
		trfGood = w.MakeTraf(1)
	}
	if trfGood == -1 {
		w.doComOut(tpop, w.getCRVal())
		return
	}
	if w.Rand.Rand16()&7 != 0 {
		return
	}
	zscore := w.CValve + w.evalCom(trfGood)
	if !zonePwr {
		zscore = -500
	}
	// ⚠ 商業區的成長多一個條件，而且它排在**最前面**：
	//     if (TrfGood && (zscore > -350) && (…Rand16Signed…))
	// C 的 && 是短路的，所以 TrfGood 為零時**那一次 Rand16Signed 根本不會被呼叫**。
	// 寫成「先算 growShrink 再檢查 trfGood」會多消耗一個亂數，
	// 之後整條數列就錯開了。這是本專案第一個被對拍抓出來的實作 bug。
	if trfGood != 0 && zscore > -350 &&
		int16(zscore-26380) > int16(w.Rand.Rand16Signed()) {
		w.doComIn(tpop, w.getCRVal())
		return
	}
	if zscore < 350 && int16(zscore+26380) < int16(w.Rand.Rand16Signed()) {
		w.doComOut(tpop, w.getCRVal())
	}
}

// doResidential。s_zone.c:230
func (w *World) doResidential(zonePwr bool) {
	w.ResZPop++
	var tpop int
	if w.CChr9 == FREEZ {
		tpop = w.doFreePop(w.SMapX, w.SMapY)
	} else {
		tpop = rzPop(w.CChr9)
	}
	w.ResPop += tpop
	trfGood := 1
	if tpop > w.Rand.Rand(35) {
		trfGood = w.MakeTraf(0)
	}
	if trfGood == -1 {
		w.doResOut(tpop, w.getCRVal())
		return
	}
	// ⚠ 空住宅區（FREEZ）**每一輪都會擲**，其餘只有八分之一機率。
	if w.CChr9 != FREEZ && w.Rand.Rand16()&7 != 0 {
		return
	}
	zscore := w.RValve + w.evalRes(trfGood)
	if !zonePwr {
		zscore = -500
	}
	grow, shrink := growShrink(zscore, w.Rand)
	if grow {
		if tpop == 0 && w.Rand.Rand16()&3 == 0 {
			w.makeHosp()
			return
		}
		w.doResIn(tpop, w.getCRVal())
		return
	}
	if shrink {
		w.doResOut(tpop, w.getCRVal())
	}
}

// makeHosp 在空住宅區蓋醫院或教堂。s_zone.c:272
func (w *World) makeHosp() {
	if w.NeedHosp > 0 {
		w.zonePlop(HOSPITAL - 4)
		w.NeedHosp = 0
		return
	}
	if w.NeedChurch > 0 {
		w.zonePlop(CHURCH - 4)
		w.NeedChurch = 0
	}
}

// getCRVal 由地價減汙染算出建物等級 0…3。s_zone.c:287
func (w *World) getCRVal() int {
	lval := int(w.LandValueMem[w.SMapX>>1][w.SMapY>>1]) -
		int(w.PollutionMem[w.SMapX>>1][w.SMapY>>1])
	switch {
	case lval < 30:
		return 0
	case lval < 80:
		return 1
	case lval < 150:
		return 2
	}
	return 3
}

// doResIn。s_zone.c:300
func (w *World) doResIn(pop, value int) {
	if w.PollutionMem[w.SMapX>>1][w.SMapY>>1] > 128 {
		return // 汙染太重就不長
	}
	if w.CChr9 == FREEZ {
		if pop < 8 {
			w.buildHouse(value)
			w.incROG(1)
			return
		}
		if w.PopDensity[w.SMapX>>1][w.SMapY>>1] > 64 {
			w.resPlop(0, value)
			w.incROG(8)
		}
		return
	}
	if pop < 40 {
		w.resPlop(pop/8-1, value)
		w.incROG(8)
	}
}

// doComIn。s_zone.c:327
func (w *World) doComIn(pop, value int) {
	z := int(w.LandValueMem[w.SMapX>>1][w.SMapY>>1]) >> 5
	if pop > z {
		return
	}
	if pop < 5 {
		w.comPlop(pop, value)
		w.incROG(8)
	}
}

// doIndIn。s_zone.c:342
func (w *World) doIndIn(pop, value int) {
	if pop < 4 {
		w.indPlop(pop, value)
		w.incROG(8)
	}
}

// incROG 累計成長率。s_zone.c:351
func (w *World) incROG(amount int) {
	w.RateOGMem[w.SMapX>>3][w.SMapY>>3] += int16(amount << 2)
}

// doResOut。s_zone.c:357
//
// ⚠ `pop == 16` 那一段的內層迴圈把整個 3×3 換成房舍，**沒有 return**，
// 所以接著會掉進 `pop < 16` 的判斷（此時 pop 仍是 16，不成立）。
// 三段是 `>16`／`==16`／`<16` 的獨立 if，不是 else if。照抄。
func (w *World) doResOut(pop, value int) {
	brdr := [9]int{0, 3, 6, 1, 4, 7, 2, 5, 8}
	if pop == 0 {
		return
	}
	if pop > 16 {
		w.resPlop((pop-24)/8, value)
		w.incROG(-8)
		return
	}
	if pop == 16 {
		w.incROG(-8)
		w.Map[w.SMapX][w.SMapY] = uint16(FREEZ | BLBNCNBIT | ZONEBIT)
		for x := w.SMapX - 1; x <= w.SMapX+1; x++ {
			for y := w.SMapY - 1; y <= w.SMapY+1; y++ {
				if !InBounds(x, y) {
					continue
				}
				if int(w.Map[x][y]&LOMASK) != FREEZ {
					w.Map[x][y] = uint16(LHTHR + value + w.Rand.Rand(2) + BLBNCNBIT)
				}
			}
		}
	}
	if pop < 16 {
		w.incROG(-1)
		z := 0
		for x := w.SMapX - 1; x <= w.SMapX+1; x++ {
			for y := w.SMapY - 1; y <= w.SMapY+1; y++ {
				if InBounds(x, y) {
					loc := int(w.Map[x][y] & LOMASK)
					if loc >= LHTHR && loc <= HHTHR {
						w.Map[x][y] = uint16(brdr[z] + BLBNCNBIT + FREEZ - 4)
						return
					}
				}
				// ⚠ z++ 在**內層**迴圈的尾巴，但只在沒有 return 時才走到；
				// 而且它在 InBounds 判斷的外面。照抄。
				z++
			}
		}
	}
}

// doComOut。s_zone.c:400
func (w *World) doComOut(pop, value int) {
	if pop > 1 {
		w.comPlop(pop-2, value)
		w.incROG(-8)
		return
	}
	if pop == 1 {
		w.zonePlop(COMBASE)
		w.incROG(-8)
	}
}

// doIndOut。s_zone.c:414
func (w *World) doIndOut(pop, value int) {
	if pop > 1 {
		w.indPlop(pop-2, value)
		w.incROG(-8)
		return
	}
	if pop == 1 {
		w.zonePlop(INDCLR - 4)
		w.incROG(-8)
	}
}

// buildHouse 在 3×3 裡挑一格蓋房子。s_zone.c:457
//
// 挑法是「分數最高者」，但平手時有八分之一機率換人——所以不是決定性的挑選。
func (w *World) buildHouse(value int) {
	zeX := [9]int{0, -1, 0, 1, -1, 1, -1, 0, 1}
	zeY := [9]int{0, -1, -1, -1, 0, 0, 1, 1, 1}
	bestLoc, hscore := 0, 0
	for z := 1; z < 9; z++ {
		xx, yy := w.SMapX+zeX[z], w.SMapY+zeY[z]
		if !InBounds(xx, yy) {
			continue
		}
		score := w.evalLot(xx, yy)
		if score == 0 {
			continue
		}
		if score > hscore {
			hscore = score
			bestLoc = z
		}
		if score == hscore && w.Rand.Rand16()&7 == 0 {
			bestLoc = z
		}
	}
	if bestLoc != 0 {
		xx, yy := w.SMapX+zeX[bestLoc], w.SMapY+zeY[bestLoc]
		if InBounds(xx, yy) {
			w.Map[xx][yy] = uint16(HOUSE + BLBNCNBIT + w.Rand.Rand(2) + value*3)
		}
	}
}

// resPlop／comPlop／indPlop 換上對應密度與等級的 3×3 建物。s_zone.c:490-508
func (w *World) resPlop(den, value int) { w.zonePlop(((value*4+den)*9 + RZB - 4)) }
func (w *World) comPlop(den, value int) { w.zonePlop(((value*5+den)*9 + CZB - 4)) }
func (w *World) indPlop(den, value int) { w.zonePlop(((value*4+den)*9 + IZB - 4)) }

// evalLot 給一格空地打分：有沒有被佔用、旁邊有幾條路。s_zone.c:517
func (w *World) evalLot(x, y int) int {
	dx := [4]int{0, 1, 0, -1}
	dy := [4]int{-1, 0, 1, 0}
	z := int(w.Map[x][y] & LOMASK)
	if z != 0 && (z < RESBASE || z > RESBASE+8) {
		return -1
	}
	score := 1
	for i := 0; i < 4; i++ {
		xx, yy := x+dx[i], y+dy[i]
		if InBounds(xx, yy) && w.Map[xx][yy] != 0 &&
			int(w.Map[xx][yy]&LOMASK) <= LASTROAD {
			score++
		}
	}
	return score
}

// zonePlop 把 3×3 換成從 base 起算的九個圖塊。s_zone.c:541
//
// ⚠ 前置檢查是「九格裡有沒有火或水災」——有的話整個放棄（回 false）。
// 而且檢查用的是 `>= FLOOD && < ROADBASE`，涵蓋水災、輻射與火。
func (w *World) zonePlop(base int) bool {
	zx := [9]int{-1, 0, 1, -1, 0, 1, -1, 0, 1}
	zy := [9]int{-1, -1, -1, 0, 0, 0, 1, 1, 1}
	for z := 0; z < 9; z++ {
		xx, yy := w.SMapX+zx[z], w.SMapY+zy[z]
		if !InBounds(xx, yy) {
			continue
		}
		x := int(w.Map[xx][yy] & LOMASK)
		if x >= FLOOD && x < ROADBASE {
			return false
		}
	}
	for z := 0; z < 9; z++ {
		xx, yy := w.SMapX+zx[z], w.SMapY+zy[z]
		if InBounds(xx, yy) {
			w.Map[xx][yy] = uint16(base + BNCNBIT)
		}
		// ⚠ base++ 在 InBounds 判斷**外面**：出界的格子照樣把 base 推進去，
		// 所以貼在地圖邊緣的分區，圖塊編號會跟中間的不一樣。照抄。
		base++
	}
	w.CChr = w.Map[w.SMapX][w.SMapY]
	w.SetZPower()
	w.Map[w.SMapX][w.SMapY] |= ZONEBIT + BULLBIT
	return true
}

// evalRes 由地價減汙染算住宅吸引力。s_zone.c:569
func (w *World) evalRes(traf int) int {
	if traf < 0 {
		return -3000
	}
	value := int(w.LandValueMem[w.SMapX>>1][w.SMapY>>1]) -
		int(w.PollutionMem[w.SMapX>>1][w.SMapY>>1])
	if value < 0 {
		value = 0
	} else {
		value <<= 5
	}
	if value > 6000 {
		value = 6000
	}
	return value - 3000
}

// evalCom 直接讀商業比率。s_zone.c:588
func (w *World) evalCom(traf int) int {
	if traf < 0 {
		return -3000
	}
	return int(w.ComRate[w.SMapX>>3][w.SMapY>>3])
}

// evalInd 工業只看有沒有路。s_zone.c:598
func evalInd(traf int) int {
	if traf < 0 {
		return -1000
	}
	return 0
}
