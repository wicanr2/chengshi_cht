package sim

// 逐格掃描與各種圖塊的每一輪處理。
// 證據：docs/re/07-traffic-and-zones.md／一手出處：s_sim.c:695 起
//
// MapScan 是模擬的主力：`Simulate` 的第 1–8 相位各掃八分之一張地圖，
// 每一格依圖塊種類分派。分派順序本身就是規則的一部分——
// 例如「先看是不是火」比「是不是分區」早，所以燒起來的分區走的是火的路徑。

// SpriteHooks 是精靈系統的介面。
//
// **精靈（w_sprite.c）還沒實作。** 它包含怪獸、龍捲風、飛機、船、火車、
// 直昇機、爆炸與起火點——這些是**規則**而不是呈現層，儘管檔名前綴是 w_
//（機制筆記的原始碼地圖已更正過那個誤判）。
// 在它實作出來之前，這個介面讓規則層可以照原版的位置呼叫，
// 而預設實作什麼都不做——**這是已知差異，記在 docs/re/07 §5。**
type SpriteHooks interface {
	GenerateShip()
	GeneratePlane(x, y int)
	GenerateCopter(x, y int)
	GenerateTrain(x, y int)
	MakeExplosionAt(x, y int)
	MakeExplosion(x, y int)
	// HasShip 回報場上有沒有船（原版是 GetSprite(SHI) != NULL）。
	HasShip() bool
	// BoatDistance 回傳最近的船到 (x,y) 的距離；沒有船時原版回 99999。
	BoatDistance(x, y int) int
}

// noSprites 是預設的空實作。
type noSprites struct{}

func (noSprites) GenerateShip()               {}
func (noSprites) GeneratePlane(x, y int)      {}
func (noSprites) GenerateCopter(x, y int)     {}
func (noSprites) GenerateTrain(x, y int)      {}
func (noSprites) MakeExplosionAt(x, y int)    {}
func (noSprites) MakeExplosion(x, y int)      {}
func (noSprites) HasShip() bool               { return false }
func (noSprites) BoatDistance(x, y int) int   { return 99999 }

func (w *World) sprites() SpriteHooks {
	if w.Sprites == nil {
		return noSprites{}
	}
	return w.Sprites
}

// MapScan 掃描 [x1, x2) 這幾欄。s_sim.c:695
func (w *World) MapScan(x1, x2 int) {
	for x := x1; x < x2; x++ {
		for y := 0; y < WorldY; y++ {
			w.CChr = w.Map[x][y]
			if w.CChr == 0 {
				continue
			}
			w.CChr9 = int(w.CChr & LOMASK)
			if w.CChr9 < FLOOD {
				continue
			}
			w.SMapX, w.SMapY = x, y

			if w.CChr9 < ROADBASE {
				if w.CChr9 >= FIREBASE {
					w.FirePop++
					// 四分之一機率讓火蔓延。
					if w.Rand.Rand16()&3 == 0 {
						w.doFire()
					}
					continue
				}
				if w.CChr9 < RADTILE {
					w.doFlood()
				} else {
					w.doRadTile()
				}
				continue
			}

			if w.NewPower && w.CChr&CONDBIT != 0 {
				w.SetZPower()
			}

			if w.CChr9 >= ROADBASE && w.CChr9 < POWERBASE {
				w.doRoad()
				continue
			}
			if w.CChr&ZONEBIT != 0 {
				w.DoZone()
				continue
			}
			if w.CChr9 >= RAILBASE && w.CChr9 < RESBASE {
				w.doRail()
				continue
			}
			if w.CChr9 >= SOMETINYEXP && w.CChr9 <= LASTTINYEXP {
				w.Map[x][y] = uint16(RUBBLE + (w.Rand.Rand16() & 3) + BULLBIT)
			}
		}
	}
}

// doRail 處理鐵路。s_sim.c:748
func (w *World) doRail() {
	w.RailTotal++
	w.sprites().GenerateTrain(w.SMapX, w.SMapY)
	if w.RoadEffect >= 30 {
		return
	}
	if w.Rand.Rand16()&511 != 0 {
		return
	}
	if w.CChr&CONDBIT != 0 {
		return
	}
	if w.RoadEffect < w.Rand.Rand16()&31 {
		if w.CChr9 < RAILBASE+2 {
			w.Map[w.SMapX][w.SMapY] = RIVER
		} else {
			w.Map[w.SMapX][w.SMapY] = uint16(RUBBLE + (w.Rand.Rand16() & 3) + BULLBIT)
		}
	}
}

// doRadTile 是輻射衰變：每格每輪 1/4096 機率消失。s_sim.c:765
func (w *World) doRadTile() {
	if w.Rand.Rand16()&4095 == 0 {
		w.Map[w.SMapX][w.SMapY] = 0
	}
}

// doRoad 處理道路：失修、開橋、車流動畫。s_sim.c:772
func (w *World) doRoad() {
	denTab := [3]int{ROADBASE, LTRFBASE, HTRFBASE}
	w.RoadTotal++

	if w.RoadEffect < 30 && w.Rand.Rand16()&511 == 0 && w.CChr&CONDBIT == 0 &&
		w.RoadEffect < w.Rand.Rand16()&31 {
		// 失修：低編號的道路變回河，其餘變廢墟。
		if (w.CChr9&15) < 2 || (w.CChr9&15) == 15 {
			w.Map[w.SMapX][w.SMapY] = RIVER
		} else {
			w.Map[w.SMapX][w.SMapY] = uint16(RUBBLE + (w.Rand.Rand16() & 3) + BULLBIT)
		}
		return
	}

	if w.CChr&BURNBIT == 0 { // 不可燃 ＝ 這是橋
		w.RoadTotal += 4
		if w.doBridge() {
			return
		}
	}

	var tden int
	switch {
	case w.CChr9 < LTRFBASE:
		tden = 0
	case w.CChr9 < HTRFBASE:
		tden = 1
	default:
		w.RoadTotal++
		tden = 2
	}

	density := int(w.TrfDensity[w.SMapX>>1][w.SMapY>>1]) >> 6
	if density > 1 {
		density--
	}
	if tden != density {
		z := ((w.CChr9 - ROADBASE) & 15) + denTab[density]
		z += int(w.CChr) & (ALLBITS - ANIMBIT)
		if density != 0 {
			z += ANIMBIT
		}
		w.Map[w.SMapX][w.SMapY] = uint16(z)
	}
}

// doBridge 開關吊橋。s_sim.c:813
//
// ⚠ 開關的條件靠 `GetBoatDis()`——場上沒有船時原版回 99999，
// 所以橋會一直保持關閉。精靈還沒實作，行為等同「永遠沒有船」。
func (w *World) doBridge() bool {
	hDx := [7]int{-2, 2, -2, -1, 0, 1, 2}
	hDy := [7]int{-1, -1, 0, 0, 0, 0, 0}
	hbrTab := [7]int{HBRDG1 | BULLBIT, HBRDG3 | BULLBIT, HBRDG0 | BULLBIT,
		RIVER, BRWH | BULLBIT, RIVER, HBRDG2 | BULLBIT}
	hbrTab2 := [7]int{RIVER, RIVER, HBRIDGE | BULLBIT, HBRIDGE | BULLBIT,
		HBRIDGE | BULLBIT, HBRIDGE | BULLBIT, HBRIDGE | BULLBIT}
	vDx := [7]int{0, 1, 0, 0, 0, 0, 1}
	vDy := [7]int{-2, -2, -1, 0, 1, 2, 2}
	vbrTab := [7]int{VBRDG0 | BULLBIT, VBRDG1 | BULLBIT, RIVER, BRWV | BULLBIT,
		RIVER, VBRDG2 | BULLBIT, VBRDG3 | BULLBIT}
	vbrTab2 := [7]int{VBRIDGE | BULLBIT, RIVER, VBRIDGE | BULLBIT, VBRIDGE | BULLBIT,
		VBRIDGE | BULLBIT, VBRIDGE | BULLBIT, RIVER}

	boatDis := w.sprites().BoatDistance(w.SMapX, w.SMapY)

	if w.CChr9 == BRWV { // 直向橋關閉
		if w.Rand.Rand16()&3 == 0 && boatDis > 340 {
			for z := 0; z < 7; z++ {
				x, y := w.SMapX+vDx[z], w.SMapY+vDy[z]
				if InBounds(x, y) && int(w.Map[x][y]&LOMASK) == (vbrTab[z]&LOMASK) {
					w.Map[x][y] = uint16(vbrTab2[z])
				}
			}
		}
		return true
	}
	if w.CChr9 == BRWH { // 橫向橋關閉
		if w.Rand.Rand16()&3 == 0 && boatDis > 340 {
			for z := 0; z < 7; z++ {
				x, y := w.SMapX+hDx[z], w.SMapY+hDy[z]
				if InBounds(x, y) && int(w.Map[x][y]&LOMASK) == (hbrTab[z]&LOMASK) {
					w.Map[x][y] = uint16(hbrTab2[z])
				}
			}
		}
		return true
	}

	if boatDis >= 300 && w.Rand.Rand16()&7 != 0 {
		return false
	}
	if w.CChr9&1 != 0 { // 直向開啟
		if w.SMapX < WorldX-1 && w.Map[w.SMapX+1][w.SMapY] == CHANNEL {
			for z := 0; z < 7; z++ {
				x, y := w.SMapX+vDx[z], w.SMapY+vDy[z]
				if !InBounds(x, y) {
					continue
				}
				mp := int(w.Map[x][y])
				if mp == CHANNEL || (mp&15) == (vbrTab2[z]&15) {
					w.Map[x][y] = uint16(vbrTab[z])
				}
			}
			return true
		}
		return false
	}
	// 橫向開啟
	if w.SMapY > 0 && w.Map[w.SMapX][w.SMapY-1] == CHANNEL {
		for z := 0; z < 7; z++ {
			x, y := w.SMapX+hDx[z], w.SMapY+hDy[z]
			if !InBounds(x, y) {
				continue
			}
			mp := int(w.Map[x][y])
			if (mp&15) == (hbrTab2[z]&15) || mp == CHANNEL {
				w.Map[x][y] = uint16(hbrTab[z])
			}
		}
		return true
	}
	return false
}

// doFire 讓火向四周蔓延，並依消防涵蓋率決定燒完的速度。s_sim.c:921
func (w *World) doFire() {
	dx := [4]int{-1, 0, 1, 0}
	dy := [4]int{0, -1, 0, 1}
	for z := 0; z < 4; z++ {
		if w.Rand.Rand16()&7 != 0 {
			continue
		}
		xt, yt := w.SMapX+dx[z], w.SMapY+dy[z]
		if !InBounds(xt, yt) {
			continue
		}
		c := w.Map[xt][yt]
		if c&BURNBIT == 0 {
			continue
		}
		if c&ZONEBIT != 0 {
			w.fireZone(xt, yt, int(c))
			if int(c&LOMASK) > IZB {
				w.sprites().MakeExplosionAt((xt<<4)+8, (yt<<4)+8)
			}
		}
		w.Map[xt][yt] = uint16(FIRE + (w.Rand.Rand16() & 3) + ANIMBIT)
	}
	// 消防涵蓋率越高，燒完（變廢墟）的機率越大。
	z := w.FireRate[w.SMapX>>3][w.SMapY>>3]
	rate := 10
	if z != 0 {
		rate = 3
		if z > 20 {
			rate = 2
		}
		if z > 100 {
			rate = 1
		}
	}
	if w.Rand.Rand(rate) == 0 {
		w.Map[w.SMapX][w.SMapY] = uint16(RUBBLE + (w.Rand.Rand16() & 3) + BULLBIT)
	}
}

// fireZone 把燒到的分區周邊標成可推平，並扣成長率。s_sim.c:958
func (w *World) fireZone(xloc, yloc, ch int) {
	w.RateOGMem[xloc>>3][yloc>>3] -= 20
	ch &= LOMASK
	xyMax := 2
	if ch >= PORTBASE {
		if ch == AIRPORT {
			xyMax = 5
		} else {
			xyMax = 4
		}
	}
	for x := -1; x < xyMax; x++ {
		for y := -1; y < xyMax; y++ {
			xt, yt := xloc+x, yloc+y
			if !InBounds(xt, yt) {
				continue
			}
			if int(w.Map[xt][yt]&LOMASK) >= ROADBASE {
				w.Map[xt][yt] |= BULLBIT
			}
		}
	}
}

// repairZone 把特殊建築被破壞的格子補回來。s_sim.c:988
//
// ⚠ 計數器 `cnt` 在 InBounds 判斷**外面**遞增，所以貼邊的建築補出來的
// 圖塊編號會偏移。照抄。
func (w *World) repairZone(zCent, zsize int) {
	zsize--
	cnt := 0
	for y := -1; y < zsize; y++ {
		for x := -1; x < zsize; x++ {
			xx, yy := w.SMapX+x, w.SMapY+y
			cnt++
			if !InBounds(xx, yy) {
				continue
			}
			thCh := w.Map[xx][yy]
			if thCh&ZONEBIT != 0 || thCh&ANIMBIT != 0 {
				continue
			}
			t := int(thCh & LOMASK)
			if t < RUBBLE || t >= ROADBASE {
				w.Map[xx][yy] = uint16(zCent - 3 - zsize + cnt + CONDBIT + BURNBIT)
			}
		}
	}
}

// doSPZone 處理特殊建築。s_sim.c:1014
func (w *World) doSPZone(pwrOn bool) {
	mltdwnTab := [3]int{30000, 20000, 10000}

	switch w.CChr9 {
	case POWERPLANT:
		w.CoalPop++
		if w.CityTime&7 == 0 {
			w.repairZone(POWERPLANT, 4)
		}
		w.pushPowerPlant()
		w.coalSmoke(w.SMapX, w.SMapY)

	case NUCLEAR:
		if !w.NoDisasters && w.Rand.Rand(mltdwnTab[w.GameLevel]) == 0 {
			w.DoMeltdown(w.SMapX, w.SMapY)
			return
		}
		w.NuclearPop++
		if w.CityTime&7 == 0 {
			w.repairZone(NUCLEAR, 4)
		}
		w.pushPowerPlant()

	case FIRESTATION:
		w.FireStPop++
		if w.CityTime&7 == 0 {
			w.repairZone(FIRESTATION, 3)
		}
		z := w.FireEffect
		if !pwrOn {
			z >>= 1
		}
		if !w.findPRoadHere() {
			z >>= 1
		}
		w.FireStMap[w.SMapX>>3][w.SMapY>>3] += int16(z)

	case POLICESTATION:
		w.PolicePop++
		if w.CityTime&7 == 0 {
			w.repairZone(POLICESTATION, 3)
		}
		z := w.PoliceEffect
		if !pwrOn {
			z >>= 1
		}
		if !w.findPRoadHere() {
			z >>= 1
		}
		w.PoliceMap[w.SMapX>>3][w.SMapY>>3] += int16(z)

	case STADIUM:
		w.StadiumPop++
		if w.CityTime&15 == 0 {
			w.repairZone(STADIUM, 4)
		}
		if pwrOn && (w.CityTime+w.SMapX+w.SMapY)&31 == 0 {
			w.drawStadium(FULLSTADIUM)
			if InBounds(w.SMapX+1, w.SMapY) {
				w.Map[w.SMapX+1][w.SMapY] = uint16(FOOTBALLGAME1 + ANIMBIT)
			}
			if InBounds(w.SMapX+1, w.SMapY+1) {
				w.Map[w.SMapX+1][w.SMapY+1] = uint16(FOOTBALLGAME2 + ANIMBIT)
			}
		}

	case FULLSTADIUM:
		w.StadiumPop++
		if (w.CityTime+w.SMapX+w.SMapY)&7 == 0 {
			w.drawStadium(STADIUM)
		}

	case AIRPORT:
		w.APortPop++
		if w.CityTime&7 == 0 {
			w.repairZone(AIRPORT, 6)
		}
		if InBounds(w.SMapX+1, w.SMapY-1) {
			if pwrOn {
				if int(w.Map[w.SMapX+1][w.SMapY-1]&LOMASK) == RADAR {
					w.Map[w.SMapX+1][w.SMapY-1] = uint16(RADAR + ANIMBIT + CONDBIT + BURNBIT)
				}
			} else {
				w.Map[w.SMapX+1][w.SMapY-1] = uint16(RADAR + CONDBIT + BURNBIT)
			}
		}
		if pwrOn {
			w.doAirport()
		}

	case PORT:
		w.PortPop++
		if w.CityTime&15 == 0 {
			w.repairZone(PORT, 4)
		}
		if pwrOn && !w.sprites().HasShip() {
			w.sprites().GenerateShip()
		}
	}
}

// pushPowerPlant 把目前這座電廠壓進電力堆疊。
//
// 原版是 DoSPZone 直接呼叫 PushPowerStack；本專案的電力傳導收攏成
// World.DoPowerScan（自己重新找電廠），所以這裡只留一個空掛勾，
// 讓分派流程與原版對得起來。見 docs/re/05-power-scan.md §2。
func (w *World) pushPowerPlant() {}

// findPRoadHere 是 FindPRoad 的唯讀版：只回報周長上有沒有路，不移動游標。
// 原版的 FindPRoad 會移動 SMapX/SMapY，但 DoSPZone 呼叫它之後就不再用游標了
// （s_sim.c:1050、:1069）——**除了它會影響後面的 FireStMap 索引**。
// 所以這裡照原版移動游標。
func (w *World) findPRoadHere() bool { return w.findPRoad() }

// drawStadium 重畫體育場的 4×4。s_sim.c:1120
func (w *World) drawStadium(z int) {
	z -= 5
	for y := w.SMapY - 1; y < w.SMapY+3; y++ {
		for x := w.SMapX - 1; x < w.SMapX+3; x++ {
			if InBounds(x, y) {
				w.Map[x][y] = uint16(z | BNCNBIT)
			}
			z++
		}
	}
	w.Map[w.SMapX][w.SMapY] |= ZONEBIT | PWRBIT
}

// doAirport 生飛機或直昇機。s_sim.c:1133
func (w *World) doAirport() {
	if w.Rand.Rand(5) == 0 {
		w.sprites().GeneratePlane(w.SMapX, w.SMapY)
		return
	}
	if w.Rand.Rand(12) == 0 {
		w.sprites().GenerateCopter(w.SMapX, w.SMapY)
	}
}

// coalSmoke 在燃煤電廠上畫煙。s_sim.c:1145
func (w *World) coalSmoke(mx, my int) {
	smTb := [4]int{COALSMOKE1, COALSMOKE2, COALSMOKE3, COALSMOKE4}
	dx := [4]int{1, 2, 1, 2}
	dy := [4]int{-1, -1, 0, 0}
	for x := 0; x < 4; x++ {
		xx, yy := mx+dx[x], my+dy[x]
		if InBounds(xx, yy) {
			w.Map[xx][yy] = uint16(smTb[x] | ANIMBIT | CONDBIT | PWRBIT | BURNBIT)
		}
	}
}

// DoMeltdown 核電廠熔毀。s_sim.c:1159
func (w *World) DoMeltdown(sx, sy int) {
	w.MeltX, w.MeltY = sx, sy

	w.sprites().MakeExplosion(sx-1, sy-1)
	w.sprites().MakeExplosion(sx-1, sy+2)
	w.sprites().MakeExplosion(sx+2, sy-1)
	w.sprites().MakeExplosion(sx+2, sy+2)

	for x := sx - 1; x < sx+3; x++ {
		for y := sy - 1; y < sy+3; y++ {
			if InBounds(x, y) {
				w.Map[x][y] = uint16(FIRE + (w.Rand.Rand16() & 3) + ANIMBIT)
			}
		}
	}
	// 兩百次擲點，把周圍變成輻射區。
	for z := 0; z < 200; z++ {
		x := sx - 20 + w.Rand.Rand(40)
		y := sy - 15 + w.Rand.Rand(30)
		if !InBounds(x, y) {
			continue
		}
		t := w.Map[x][y]
		if t&ZONEBIT != 0 {
			continue
		}
		if t&BURNBIT != 0 || t == 0 {
			w.Map[x][y] = RADTILE
		}
	}
}
