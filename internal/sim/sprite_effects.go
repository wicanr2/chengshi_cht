package sim

// 精靈造成的破壞、產生與 SpriteHooks 介面實作。
// 一手出處：w_sprite.c:1370 起。

// explodeSprite 讓一隻精靈爆炸。w_sprite.c:1370
func (s *spriteSystem) explodeSprite(sp *Sprite) {
	if sp == nil {
		return
	}
	sp.Frame = 0
	x := sp.X + sp.XHot
	y := sp.Y + sp.YHot
	s.MakeExplosionAt(x, y)
	s.w.CrashX = x >> 4
	s.w.CrashY = y >> 4
	// 原版在這裡依型別送不同的訊息（−24 飛機、−25 船、−26 火車、−27 直昇機）。
	// 訊息系統還沒實作。
}

// checkWet 判斷一格是不是「濕的」基礎建設（毀掉會變回河）。w_sprite.c:1415
func checkWet(x int) bool {
	return x == POWERBASE || x == POWERBASE+1 ||
		x == RAILBASE || x == RAILBASE+1 ||
		x == BRWH || x == BRWV
}

// destroy 摧毀一格。w_sprite.c:1426
//
// ⚠ 下界是 `TREEBASE`（21）——**樹以下（水、空地）不會被摧毀**。
// 不可燃的道路會變回河；橋樑因此被「沖走」。
func (s *spriteSystem) destroy(ox, oy int) {
	x := ox >> 4
	y := oy >> 4
	if !InBounds(x, y) {
		return
	}
	z := s.w.Map[x][y]
	t := int(z & LOMASK)
	if t < TREEBASE {
		return
	}
	if z&BURNBIT == 0 {
		if t >= ROADBASE && t <= LASTROAD {
			s.w.Map[x][y] = RIVER
		}
		return
	}
	if z&ZONEBIT != 0 {
		s.oFireZone(x, y, int(z))
		if t > RZB {
			s.MakeExplosionAt(ox, oy)
		}
	}
	if checkWet(t) {
		s.w.Map[x][y] = RIVER
	} else {
		s.w.Map[x][y] = uint16(TINYEXP | BULLBIT | ANIMBIT)
	}
}

// oFireZone 與 s_sim.c 的 FireZone 幾乎相同，但**少了邊界檢查**。
// w_sprite.c:1459
//
// ⚠ 原版這一份直接寫 `Map[Xtem][Ytem]`，沒有 TestBounds——
// 在地圖邊緣的分區被摧毀時會讀寫到陣列外。Go 版加上邊界檢查，
// 這是刻意不照做的一項（見 docs/re/13-sprites.md §4）。
func (s *spriteSystem) oFireZone(xloc, yloc, ch int) {
	s.w.RateOGMem[xloc>>3][yloc>>3] -= 20
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
			if int(s.w.Map[xt][yt]&LOMASK) >= ROADBASE {
				s.w.Map[xt][yt] |= BULLBIT
			}
		}
	}
}

// startFire 在一格點火。w_sprite.c:1483
func (s *spriteSystem) startFire(x, y int) {
	x >>= 4
	y >>= 4
	if !InBounds(x, y) {
		return
	}
	z := s.w.Map[x][y]
	t := int(z & LOMASK)
	if z&BURNBIT == 0 && t != 0 {
		return
	}
	if z&ZONEBIT != 0 {
		return
	}
	s.w.Map[x][y] = uint16(FIRE + (s.w.Rand.Rand16() & 3) + ANIMBIT)
}

// ---- SpriteHooks 介面 ----

// 火車的擺放偏移。w_sprite.c 的 TRA_GROOVE_X/Y。
const (
	traGrooveX = -39
	traGrooveY = 6
	busGrooveX = -39
	busGrooveY = 6
)

// GenerateTrain。w_sprite.c:1501
//
// 條件：人口超過 20、場上沒有火車、而且 1/26 機率。
func (s *spriteSystem) GenerateTrain(x, y int) {
	if s.w.TotalPop > 20 && s.getSprite(SpriteTrain) == nil && s.w.Rand.Rand(25) == 0 {
		s.makeSprite(SpriteTrain, (x<<4)+traGrooveX, (y<<4)+traGrooveY)
	}
}

// GenerateShip 從地圖四邊的水道生一艘船。w_sprite.c:1520
//
// ⚠ 四個方向各自先擲 `Rand16() & 3`，**四次都會擲**（除非提前 return）。
func (s *spriteSystem) GenerateShip() {
	if s.w.Rand.Rand16()&3 == 0 {
		for x := 4; x < WorldX-2; x++ {
			if s.w.Map[x][0] == CHANNEL {
				s.makeShipHere(x, 0)
				return
			}
		}
	}
	if s.w.Rand.Rand16()&3 == 0 {
		for y := 1; y < WorldY-2; y++ {
			if s.w.Map[0][y] == CHANNEL {
				s.makeShipHere(0, y)
				return
			}
		}
	}
	if s.w.Rand.Rand16()&3 == 0 {
		for x := 4; x < WorldX-2; x++ {
			if s.w.Map[x][WorldY-1] == CHANNEL {
				s.makeShipHere(x, WorldY-1)
				return
			}
		}
	}
	if s.w.Rand.Rand16()&3 == 0 {
		for y := 1; y < WorldY-2; y++ {
			if s.w.Map[WorldX-1][y] == CHANNEL {
				s.makeShipHere(WorldX-1, y)
				return
			}
		}
	}
}

func (s *spriteSystem) makeShipHere(x, y int) {
	s.makeSprite(SpriteShip, (x<<4)-47, y<<4)
}

// GeneratePlane。w_sprite.c 的 GeneratePlane（在 g_ 或別處，行為同 MakeSprite）。
func (s *spriteSystem) GeneratePlane(x, y int) {
	if s.getSprite(SpriteAirplane) != nil {
		return
	}
	s.makeSprite(SpriteAirplane, (x<<4)+48, (y<<4)+12)
}

// GenerateCopter。w_sprite.c:1596
func (s *spriteSystem) GenerateCopter(x, y int) {
	if s.getSprite(SpriteCopter) != nil {
		return
	}
	s.makeSprite(SpriteCopter, x<<4, (y<<4)+30)
}

// MakeExplosionAt 在像素座標處生一團爆炸。
func (s *spriteSystem) MakeExplosionAt(x, y int) {
	sp := &Sprite{Type: SpriteExplosion}
	s.list = append(s.list, sp)
	s.initSprite(sp, x-40, y-16)
}

// MakeExplosion 在格子座標處生一團爆炸。
func (s *spriteSystem) MakeExplosion(x, y int) {
	if InBounds(x, y) {
		s.MakeExplosionAt((x<<4)+8, (y<<4)+8)
	}
}

// HasShip 回報場上有沒有船。
func (s *spriteSystem) HasShip() bool { return s.getSprite(SpriteShip) != nil }

// BoatDistance 回傳最近的船到 (x,y) 的曼哈頓距離。s_sim.c:898 GetBoatDis
//
// 沒有船時回 99999——這正是吊橋一直關著的原因。
func (s *spriteSystem) BoatDistance(mx, my int) int {
	dist := 99999
	px := (mx << 4) + 8
	py := (my << 4) + 8
	for _, sp := range s.list {
		if sp.Type != SpriteShip || sp.Frame == 0 {
			continue
		}
		d := getDis(sp.X+sp.XHot, sp.Y+sp.YHot, px, py)
		if d < dist {
			dist = d
		}
	}
	return dist
}

// MakeMonster 生一隻怪獸。w_sprite.c:1557
//
// ⚠ 收尾那一行原版寫的是 `if (!done == 0) MonsterHere(60, 50);`
// —— `!done == 0` 等價於 `done != 0`，所以**找到河之後還會再生一隻在 (60,50)**，
// 而 `MakeSprite` 會重用同一隻精靈，結果就是**怪獸永遠出現在 (60,50)**。
// 反過來，三百次都沒找到河時反而不會有怪獸。這是原版的邏輯反了，照抄。
func (s *spriteSystem) MakeMonster() {
	if sp := s.getSprite(SpriteMonster); sp != nil {
		sp.SoundCount = 1
		sp.Count = 1000
		sp.DestX = s.w.PolMaxX << 4
		sp.DestY = s.w.PolMaxY << 4
		return
	}
	done := false
	for z := 0; z < 300; z++ {
		x := s.w.Rand.Rand(WorldX-20) + 10
		y := s.w.Rand.Rand(WorldY-10) + 5
		if s.w.Map[x][y] == RIVER || s.w.Map[x][y] == RIVER+BULLBIT {
			s.monsterHere(x, y)
			done = true
			break
		}
	}
	if done {
		s.monsterHere(60, 50)
	}
}

func (s *spriteSystem) monsterHere(x, y int) {
	s.makeSprite(SpriteMonster, (x<<4)+48, y<<4)
}

// MakeTornado 生一個龍捲風。
func (s *spriteSystem) MakeTornado() {
	if sp := s.getSprite(SpriteTornado); sp != nil {
		sp.Count = 200
		return
	}
	x := s.w.Rand.Rand((WorldX<<4)-800) + 400
	y := s.w.Rand.Rand((WorldY<<4)-200) + 100
	s.makeSprite(SpriteTornado, x, y)
}

// MakeAirCrash 讓一架飛機墜毀。w_sprite.c:1052
//
// ⚠ 原版的建置旗標是 `-DNO_AIRCRASH`（`micropolis-activity/src/sim/makefile`），
// 所以**這個函式整段被編譯掉了**——隨機空難在官方建置裡不會發生。
// Go 版照原始碼實作，但預設關閉，由 `World.HasAirCrash` 控制。
func (s *spriteSystem) MakeAirCrash() {
	if !s.w.HasAirCrash {
		return
	}
	if s.getSprite(SpriteAirplane) == nil {
		x := s.w.Rand.Rand(WorldX-20) + 10
		y := s.w.Rand.Rand(WorldY-10) + 5
		s.GeneratePlane(x, y)
	}
	s.explodeSprite(s.getSprite(SpriteAirplane))
}
