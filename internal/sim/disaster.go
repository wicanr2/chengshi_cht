package sim

// 災難。證據：docs/re/08-disasters.md／一手出處：s_disast.c
//
// 六種災難：火災、水災、空難、龍捲風、地震、怪獸。
// 劇本另有自己的排程（ScenarioDisaster），與隨機災難是兩條路。

// DoDisasters 是每一刻的災難判定。s_disast.c:74
//
// 三個難度的災難機率：Easy 每 480 刻一次、Medium 每 240 刻、Hard 每 60 刻。
// 一年 48 刻，所以 Easy 約十年一次、Hard 約一年三次。
func (w *World) DoDisasters() {
	disChance := [3]int{10 * 48, 5 * 48, 60}

	if w.FloodCnt != 0 {
		w.FloodCnt--
	}
	if w.DisasterEvent != 0 {
		w.scenarioDisaster()
	}

	x := w.GameLevel
	if x > 2 {
		x = 0
	}
	if w.NoDisasters {
		return
	}
	if w.Rand.Rand(disChance[x]) != 0 {
		return
	}
	// ⚠ Rand(8) 的值域是 0…8（九個值），而 switch 只列到 8——
	// case 7 與 case 8 共用「怪獸」。所以火災與水災各佔 2/9，怪獸也是 2/9。
	switch w.Rand.Rand(8) {
	case 0, 1:
		w.SetFire()
	case 2, 3:
		w.MakeFlood()
	case 4:
		w.makeAirCrash()
	case 5:
		w.makeTornado()
	case 6:
		w.MakeEarthquake()
	case 7, 8:
		// 汙染平均要超過 60 才會出現怪獸。註解裡的舊值是 80。
		if w.PolluteAverage > 60 {
			w.makeMonster()
		}
	}
}

// scenarioDisaster 是劇本自己的災難排程。s_disast.c:117
func (w *World) scenarioDisaster() {
	switch w.DisasterEvent {
	case 2: // 舊金山：等到倒數第一刻才地震
		if w.DisasterWait == 1 {
			w.MakeEarthquake()
		}
	case 3: // 漢堡：持續投彈
		w.dropFireBombs()
	case 5: // 東京：等到倒數第一刻才出怪獸
		if w.DisasterWait == 1 {
			w.makeMonster()
		}
	case 7: // 波士頓：等到倒數第一刻才熔毀
		if w.DisasterWait == 1 {
			w.makeMeltdownSomewhere()
		}
	case 8: // 里約：每 24 刻淹一次
		if w.DisasterWait%24 == 0 {
			w.MakeFlood()
		}
	}
	if w.DisasterWait != 0 {
		w.DisasterWait--
	} else {
		w.DisasterEvent = 0
	}
}

// makeMeltdownSomewhere 找第一座核電廠讓它熔毀。s_disast.c:151
func (w *World) makeMeltdownSomewhere() {
	for x := 0; x < WorldX-1; x++ {
		for y := 0; y < WorldY-1; y++ {
			if int(w.Map[x][y]&LOMASK) == NUCLEAR {
				w.DoMeltdown(x, y)
				return
			}
		}
	}
}

// dropFireBombs 空襲：隨機一點爆炸。s_disast.c:167
func (w *World) dropFireBombs() {
	w.CrashX = w.Rand.Rand(WorldX - 1)
	w.CrashY = w.Rand.Rand(WorldY - 1)
	w.sprites().MakeExplosion(w.CrashX, w.CrashY)
}

// MakeEarthquake 地震。s_disast.c:178
//
// 震動 300…1000 次擲點，每一點若是「可摧毀的」就變廢墟；四分之一機率變火。
func (w *World) MakeEarthquake() {
	t := w.Rand.Rand(700) + 300
	for z := 0; z < t; z++ {
		x := w.Rand.Rand(WorldX - 1)
		y := w.Rand.Rand(WorldY - 1)
		if !InBounds(x, y) {
			continue
		}
		if !vulnerable(int(w.Map[x][y])) {
			continue
		}
		if z&3 != 0 {
			w.Map[x][y] = uint16(RUBBLE + BULLBIT + (w.Rand.Rand16() & 3))
		} else {
			w.Map[x][y] = uint16(FIRE + ANIMBIT + (w.Rand.Rand16() & 7))
		}
	}
}

// SetFire 放一把火。s_disast.c:205
//
// 只試**一次**：隨機一點，不是分區中心而且圖塊在 LHTHR…LASTZONE 之間才點得著。
// 試不中就這一刻沒有火——與 MakeFire 的四十次重試不同。
func (w *World) SetFire() {
	x := w.Rand.Rand(WorldX - 1)
	y := w.Rand.Rand(WorldY - 1)
	z := w.Map[x][y]
	if z&ZONEBIT != 0 {
		return
	}
	t := int(z & LOMASK)
	if t > LHTHR && t < LASTZONE {
		w.Map[x][y] = uint16(FIRE + ANIMBIT + (w.Rand.Rand16() & 7))
		w.CrashX, w.CrashY = x, y
	}
}

// MakeFire 玩家主動放火：最多試四十次。s_disast.c:225
//
// ⚠ 條件與 SetFire 不同：這裡要求 `z & BURNBIT`，而且下界是 21（TREEBASE）
// 而不是 LHTHR。所以玩家放火燒得到樹，隨機火災燒不到。
func (w *World) MakeFire() {
	for t := 0; t < 40; t++ {
		x := w.Rand.Rand(WorldX - 1)
		y := w.Rand.Rand(WorldY - 1)
		z := w.Map[x][y]
		if z&ZONEBIT != 0 || z&BURNBIT == 0 {
			continue
		}
		v := int(z & LOMASK)
		if v > 21 && v < LASTZONE {
			w.Map[x][y] = uint16(FIRE + ANIMBIT + (w.Rand.Rand16() & 7))
			return
		}
	}
}

// vulnerable 判斷一格會不會被地震摧毀。s_disast.c:246
//
// 只有「建物本體」會壞：分區中心（帶 ZONEBIT）不會，道路與自然地形也不會。
func vulnerable(tem int) bool {
	t := tem & LOMASK
	if t < RESBASE || t > LASTZONE || tem&ZONEBIT != 0 {
		return false
	}
	return true
}

// MakeFlood 引發水災。s_disast.c:260
//
// 三百次擲點找河岸，找到就把相鄰的一格淹掉並把水災計時器設成 30。
func (w *World) MakeFlood() {
	dx := [4]int{0, 1, 0, -1}
	dy := [4]int{-1, 0, 1, 0}
	for z := 0; z < 300; z++ {
		x := w.Rand.Rand(WorldX - 1)
		y := w.Rand.Rand(WorldY - 1)
		c := int(w.Map[x][y] & LOMASK)
		if c <= 4 || c >= 21 {
			continue // 不是河岸
		}
		for t := 0; t < 4; t++ {
			xx, yy := x+dx[t], y+dy[t]
			if !InBounds(xx, yy) {
				continue
			}
			cc := w.Map[xx][yy]
			if cc == 0 || (cc&BULLBIT != 0 && cc&BURNBIT != 0) {
				w.Map[xx][yy] = FLOOD
				w.FloodCnt = 30
				w.FloodX, w.FloodY = xx, yy
				return
			}
		}
	}
}

// doFlood 是水災格每一輪的行為。s_disast.c:293
//
// 計時器還在就往四周蔓延；計時器歸零之後，每格每輪 1/16 機率退去。
func (w *World) doFlood() {
	dx := [4]int{0, 1, 0, -1}
	dy := [4]int{-1, 0, 1, 0}
	if w.FloodCnt != 0 {
		for z := 0; z < 4; z++ {
			if w.Rand.Rand16()&7 != 0 {
				continue
			}
			xx, yy := w.SMapX+dx[z], w.SMapY+dy[z]
			if !InBounds(xx, yy) {
				continue
			}
			c := w.Map[xx][yy]
			t := int(c & LOMASK)
			if c&BURNBIT != 0 || c == 0 || (t >= WOODS5 && t < FLOOD) {
				if c&ZONEBIT != 0 {
					w.fireZone(xx, yy, int(c))
				}
				w.Map[xx][yy] = uint16(FLOOD + w.Rand.Rand(2))
			}
		}
		return
	}
	if w.Rand.Rand16()&15 == 0 {
		w.Map[w.SMapX][w.SMapY] = 0
	}
}

// 這三種災難靠精靈系統。沒掛精靈時什麼都不做。
func (w *World) makeAirCrash() {
	if w.spriteSys != nil {
		w.spriteSys.MakeAirCrash()
	}
}

func (w *World) makeTornado() {
	if w.spriteSys != nil {
		w.spriteSys.MakeTornado()
	}
}

func (w *World) makeMonster() {
	if w.spriteSys != nil {
		w.spriteSys.MakeMonster()
	}
}
