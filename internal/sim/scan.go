package sim

// 四個逐格掃描：人口密度、汙染／地形／地價、犯罪、消防涵蓋。
// 證據：docs/re/06-scans.md／規格：docs/spec/scans.md
// 一手出處：s_scan.c、s_zone.c:428-460（分區人口）
//
// 這四個掃描互相回饋：PTLScan 的地價要用上一輪的 PollutionMem 與 CrimeMem，
// CrimeScan 的犯罪要用這一輪的 LandValueMem。地圖不變時會收斂到固定點。

// ClrTemArray 清空半解析度暫存。s_scan.c:474
func (w *World) clrTemArray() {
	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			w.Tem[x][y] = 0
		}
	}
}

// doSmooth 把 Tem 平滑進 Tem2。s_scan.c:407（非 dither 分支）
//
// ⚠ 核是「四鄰居和 ＋ 自己」再右移 2 位，也就是除以 4 而不是除以 5。
// 這不是筆誤：邊界上少算的鄰居也不補，所以邊緣會偏低。照抄。
func (w *World) doSmooth() {
	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			z := 0
			if x > 0 {
				z += int(w.Tem[x-1][y])
			}
			if x < HWldX-1 {
				z += int(w.Tem[x+1][y])
			}
			if y > 0 {
				z += int(w.Tem[x][y-1])
			}
			if y < HWldY-1 {
				z += int(w.Tem[x][y+1])
			}
			z = (z + int(w.Tem[x][y])) >> 2
			if z > 255 {
				z = 255
			}
			w.Tem2[x][y] = uint8(z)
		}
	}
}

// doSmooth2 是反向：Tem2 平滑進 Tem。s_scan.c:455（非 dither 分支）
func (w *World) doSmooth2() {
	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			z := 0
			if x > 0 {
				z += int(w.Tem2[x-1][y])
			}
			if x < HWldX-1 {
				z += int(w.Tem2[x+1][y])
			}
			if y > 0 {
				z += int(w.Tem2[x][y-1])
			}
			if y < HWldY-1 {
				z += int(w.Tem2[x][y+1])
			}
			z = (z + int(w.Tem2[x][y])) >> 2
			if z > 255 {
				z = 255
			}
			w.Tem[x][y] = uint8(z)
		}
	}
}

// smoothFSMap 平滑消防局涵蓋圖。s_scan.c:486
//
// 核與 doSmooth 不同：`(四鄰居和 >> 2 + 自己) >> 1`。兩者不可互換。
func (w *World) smoothFSMap() {
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			edge := 0
			if x > 0 {
				edge += int(w.FireStMap[x-1][y])
			}
			if x < SmX-1 {
				edge += int(w.FireStMap[x+1][y])
			}
			if y > 0 {
				edge += int(w.FireStMap[x][y-1])
			}
			if y < SmY-1 {
				edge += int(w.FireStMap[x][y+1])
			}
			edge = (edge >> 2) + int(w.FireStMap[x][y])
			w.STem[x][y] = int16(edge >> 1)
		}
	}
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			w.FireStMap[x][y] = w.STem[x][y]
		}
	}
}

// smoothPSMap 平滑警察局涵蓋圖。s_scan.c:507。核與 smoothFSMap 相同。
func (w *World) smoothPSMap() {
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			edge := 0
			if x > 0 {
				edge += int(w.PoliceMap[x-1][y])
			}
			if x < SmX-1 {
				edge += int(w.PoliceMap[x+1][y])
			}
			if y > 0 {
				edge += int(w.PoliceMap[x][y-1])
			}
			if y < SmY-1 {
				edge += int(w.PoliceMap[x][y+1])
			}
			edge = (edge >> 2) + int(w.PoliceMap[x][y])
			w.STem[x][y] = int16(edge >> 1)
		}
	}
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			w.PoliceMap[x][y] = w.STem[x][y]
		}
	}
}

// smoothTerrain 把 Qtem 平滑進 TerrainMem。s_scan.c:365（非 dither 分支）
//
// ⚠ 這一支的截斷點在括號外面：`(unsigned char)((z>>2) + Qtem[x][y]) >> 1`
// —— 先加總、先轉成 8 位元（**這裡會截斷**）、再右移 1。
// 寫成 `((z>>2 + Qtem) >> 1)` 而不先截斷，超過 255 的格子就會不一樣。
func (w *World) smoothTerrain() {
	for x := 0; x < QWX; x++ {
		for y := 0; y < QWY; y++ {
			z := 0
			if x > 0 {
				z += int(w.Qtem[x-1][y])
			}
			if x < QWX-1 {
				z += int(w.Qtem[x+1][y])
			}
			if y > 0 {
				z += int(w.Qtem[x][y-1])
			}
			if y < QWY-1 {
				z += int(w.Qtem[x][y+1])
			}
			w.TerrainMem[x][y] = uint8((z>>2)+int(w.Qtem[x][y])) >> 1
		}
	}
}

// getDisCC 回傳 (x,y) 到城市重心的曼哈頓距離，上限 32。s_scan.c:277
func (w *World) getDisCC(x, y int) int {
	xdis := x - w.CCx2
	if xdis < 0 {
		xdis = -xdis
	}
	ydis := y - w.CCy2
	if ydis < 0 {
		ydis = -ydis
	}
	z := xdis + ydis
	if z > 32 {
		return 32
	}
	return z
}

// rzPop／czPop／izPop 從圖塊編號反推分區人口。s_zone.c:428-455
func rzPop(ch9 int) int { return ((ch9-RZB)/9%4)*8 + 16 }

func czPop(ch9 int) int {
	if ch9 == COMCLR {
		return 0
	}
	return (ch9-CZB)/9%5 + 1
}

func izPop(ch9 int) int {
	if ch9 == INDCLR {
		return 0
	}
	return (ch9-IZB)/9%4 + 1
}

// doFreePop 數 3×3 範圍內的房舍。s_zone.c:605
func (w *World) doFreePop(sx, sy int) int {
	count := 0
	for x := sx - 1; x <= sx+1; x++ {
		for y := sy - 1; y <= sy+1; y++ {
			if !InBounds(x, y) {
				continue
			}
			loc := int(w.Map[x][y] & LOMASK)
			if loc >= LHTHR && loc <= HHTHR {
				count++
			}
		}
	}
	return count
}

// getPDen 回傳一個分區中心的人口密度貢獻。s_scan.c:161
func (w *World) getPDen(ch9, sx, sy int) int {
	switch {
	case ch9 == FREEZ:
		return w.doFreePop(sx, sy)
	case ch9 < COMBASE:
		return rzPop(ch9)
	case ch9 < INDBASE:
		return czPop(ch9) << 3
	case ch9 < PORTBASE:
		return izPop(ch9) << 3
	}
	return 0
}

// getPValue 回傳一個圖塊的汙染貢獻。s_scan.c:257
//
// 原始碼註解裡有一行 `XXX: Why negative pollution from radiation?`——
// 輻射圖塊回 255 而不是註解掉的 −40。照抄現行值。
func getPValue(loc int) int {
	if loc < POWERBASE {
		if loc >= HTRFBASE {
			return 75 // 壅塞車流
		}
		if loc >= LTRFBASE {
			return 50 // 稀疏車流
		}
		if loc < ROADBASE {
			if loc > FIREBASE {
				return 90
			}
			if loc >= RADTILE {
				return 255 // 輻射
			}
		}
		return 0
	}
	if loc <= LASTIND {
		return 0
	}
	if loc < PORTBASE {
		return 50 // 工業
	}
	if loc <= LASTPOWERPLANT {
		return 100 // 海港、機場、電廠
	}
	return 0
}

// PopDenScan 算人口密度、城市重心與商業比率。s_scan.c:93
func (w *World) PopDenScan() {
	w.clrTemArray()
	var xtot, ytot, ztot int
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			z := int(w.Map[x][y])
			if z&ZONEBIT == 0 {
				continue
			}
			z &= LOMASK
			z = w.getPDen(z, x, y) << 3
			if z > 254 {
				z = 254
			}
			w.Tem[x>>1][y>>1] = uint8(z)
			xtot += x
			ytot += y
			ztot++
		}
	}
	w.doSmooth()  // Tem  -> Tem2
	w.doSmooth2() // Tem2 -> Tem
	w.doSmooth()  // Tem  -> Tem2

	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			w.PopDensity[x][y] = w.Tem2[x][y] << 1
		}
	}
	w.distIntMarket()

	if ztot != 0 {
		w.CCx = xtot / ztot
		w.CCy = ytot / ztot
	} else {
		// 人口為零時重心是地圖中心——注意用的是**半解析度**的 HWLDX/HWLDY，
		// 而 CCx/CCy 是全解析度座標。原版就是這樣寫的（s_scan.c:130）。
		w.CCx = HWldX
		w.CCy = HWldY
	}
	w.CCx2 = w.CCx >> 1
	w.CCy2 = w.CCy >> 1
}

// distIntMarket 依離重心的距離設商業比率。s_scan.c:529
func (w *World) distIntMarket() {
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			z := w.getDisCC(x<<2, y<<2)
			z <<= 2
			z = 64 - z
			w.ComRate[x][y] = int16(z)
		}
	}
}

// PTLScan 算汙染、地形與地價。s_scan.c:167
func (w *World) PTLScan() {
	for x := 0; x < QWX; x++ {
		for y := 0; y < QWY; y++ {
			w.Qtem[x][y] = 0
		}
	}
	lvtot, lvnum := 0, 0
	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			plevel, lvflag := 0, 0
			zx, zy := x<<1, y<<1
			for mx := zx; mx <= zx+1; mx++ {
				for my := zy; my <= zy+1; my++ {
					loc := int(w.Map[mx][my] & LOMASK)
					if loc == 0 {
						continue
					}
					if loc < RUBBLE {
						// 自然地形：只加地形分，不算汙染。
						w.Qtem[x>>1][y>>1] += 15
						continue
					}
					plevel += getPValue(loc)
					if loc >= ROADBASE {
						lvflag++
					}
				}
			}
			if plevel > 255 {
				plevel = 255
			}
			w.Tem[x][y] = uint8(plevel)

			if lvflag != 0 {
				// 地價公式。s_scan.c:200
				dis := 34 - w.getDisCC(x, y)
				dis <<= 2
				dis += int(w.TerrainMem[x>>1][y>>1])
				dis -= int(w.PollutionMem[x][y])
				if w.CrimeMem[x][y] > 190 {
					dis -= 20
				}
				if dis > 250 {
					dis = 250
				}
				if dis < 1 {
					dis = 1
				}
				w.LandValueMem[x][y] = uint8(dis)
				lvtot += dis
				lvnum++
			} else {
				w.LandValueMem[x][y] = 0
			}
		}
	}
	if lvnum != 0 {
		w.LVAverage = lvtot / lvnum
	} else {
		w.LVAverage = 0
	}

	w.doSmooth()
	w.doSmooth2()

	pmax, pnum, ptot := 0, 0, 0
	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			z := int(w.Tem[x][y])
			w.PollutionMem[x][y] = uint8(z)
			if z == 0 {
				continue
			}
			pnum++
			ptot += z
			// ⚠ 平手時擲骰決定要不要換位置——所以 PolMaxX/Y **不是決定性的**。
			if z > pmax || (z == pmax && w.Rand.Rand16()&3 == 0) {
				pmax = z
				w.PolMaxX = x << 1
				w.PolMaxY = y << 1
			}
		}
	}
	if pnum != 0 {
		w.PolluteAverage = ptot / pnum
	} else {
		w.PolluteAverage = 0
	}
	w.smoothTerrain()
}

// CrimeScan 算犯罪。s_scan.c:300
func (w *World) CrimeScan() {
	w.smoothPSMap()
	w.smoothPSMap()
	w.smoothPSMap()

	totz, numz, cmax := 0, 0, 0
	for x := 0; x < HWldX; x++ {
		for y := 0; y < HWldY; y++ {
			z := int(w.LandValueMem[x][y])
			if z == 0 {
				w.CrimeMem[x][y] = 0
				continue
			}
			numz++
			z = 128 - z
			z += int(w.PopDensity[x][y])
			if z > 300 {
				z = 300
			}
			z -= int(w.PoliceMap[x>>2][y>>2])
			if z > 250 {
				z = 250
			}
			if z < 0 {
				z = 0
			}
			w.CrimeMem[x][y] = uint8(z)
			totz += z
			// 同樣有平手擲骰，CrimeMaxX/Y 不是決定性的。
			if z > cmax || (z == cmax && w.Rand.Rand16()&3 == 0) {
				cmax = z
				w.CrimeMaxX = x << 1
				w.CrimeMaxY = y << 1
			}
		}
	}
	if numz != 0 {
		w.CrimeAverage = totz / numz
	} else {
		w.CrimeAverage = 0
	}
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			w.PoliceMapEffect[x][y] = w.PoliceMap[x][y]
		}
	}
}

// FireAnalysis 由消防局涵蓋圖算出滅火率。s_scan.c:77
func (w *World) FireAnalysis() {
	w.smoothFSMap()
	w.smoothFSMap()
	w.smoothFSMap()
	for x := 0; x < SmX; x++ {
		for y := 0; y < SmY; y++ {
			w.FireRate[x][y] = w.FireStMap[x][y]
		}
	}
}

// CountPops 從地圖直接數三種人口，**不動任何狀態、不擲亂數**。
//
// 為什麼需要它：`ResPop` 等欄位是 `MapScan` 逐段累加出來的，在一段掃描
// 中途讀到的是半份census。逐 tick 對拍讀得到 Scycle，知道自己在哪一段；
// 拿原版存檔當取樣點時讀不到，所以要一個與掃描相位無關的量。
//
// 演算法與 s_zone.c 的 doResidential／doCommercial／doIndustrial 同一組
// 取值函式（rzPop／czPop／izPop／doFreePop），差別只在不做成長判定。
//
// 為什麼是這個量法：docs/re/18-dos-parity.md §五之二。
func (w *World) CountPops() (res, com, ind int) {
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			t := w.Map[x][y]
			if t&ZONEBIT == 0 {
				continue
			}
			ch9 := int(t & LOMASK)
			switch {
			case ch9 > PORTBASE, ch9 >= HOSPITAL && ch9 < COMBASE:
				// 特殊區與醫院／教堂不計入三種人口。
			case ch9 < HOSPITAL:
				if ch9 == FREEZ {
					res += w.doFreePop(x, y)
				} else {
					res += rzPop(ch9)
				}
			case ch9 < INDBASE:
				com += czPop(ch9)
			default:
				ind += izPop(ch9)
			}
		}
	}
	return
}
