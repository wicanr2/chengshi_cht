package sim

// 精靈系統：火車、直昇機、飛機、船、怪獸、龍捲風、爆炸、公車。
// 證據：docs/re/13-sprites.md／一手出處：w_sprite.c
//
// 精靈是**規則**不是呈現層：牠們會摧毀格子、引發火災、觸發爆炸。
// `MoveObjects()` 每個 frame（每個相位）都跑一次，所以精靈的抽樣次數
// 直接影響整個模擬的亂數數列。

// 精靈型別。headers/sim.h:258-266
const (
	SpriteTrain     = 1 // TRA
	SpriteCopter    = 2 // COP
	SpriteAirplane  = 3 // AIR
	SpriteShip      = 4 // SHI
	SpriteMonster   = 5 // GOD
	SpriteTornado   = 6 // TOR
	SpriteExplosion = 7 // EXP
	SpriteBus       = 8 // BUS
	SpriteTypeCount = 9 // OBJN
)

// Sprite 是一隻精靈。欄位順序與名稱照 view.h:274 的 SimSprite。
type Sprite struct {
	Type       int
	Frame      int // 0 代表這隻不在場上
	X, Y       int // 像素座標（一格 16 像素）
	Width      int
	Height     int
	XOffset    int
	YOffset    int
	XHot, YHot int
	OrigX      int
	OrigY      int
	DestX      int
	DestY      int
	Count      int
	SoundCount int
	Dir        int
	NewDir     int
	Step       int
	Flag       int
	Control    int
	Turn       int
	Accel      int
	Speed      int
	// Named 代表這隻是具名精靈（原版靠 name[0] != 0 判斷要不要回收）。
	Named bool
}

// spriteSystem 實作 SpriteHooks，是原版 w_sprite.c 的行為。
type spriteSystem struct {
	w       *World
	list    []*Sprite
	globals [SpriteTypeCount]*Sprite
	cycle   int
	// absDist 是 GetDir 的副作用輸出（w_sprite.c:576 的全域 absDist）。
	absDist int
}

// EnableSprites 把精靈系統掛上去。
func (w *World) EnableSprites() {
	s := &spriteSystem{w: w}
	w.Sprites = s
	w.spriteSys = s
}

// MoveObjects 推進所有精靈一格。w_sprite.c:614
//
// ⚠ 它在 `sim_loop` 裡**每個 frame 都呼叫**，與 `SimFrame` 平行；
// 速度為 0 時直接 return。
func (s *spriteSystem) MoveObjects() {
	if s.w.SimSpeed == 0 {
		return
	}
	s.cycle++
	kept := s.list[:0]
	for _, sp := range s.list {
		if sp.Frame != 0 {
			switch sp.Type {
			case SpriteTrain:
				s.doTrain(sp)
			case SpriteCopter:
				s.doCopter(sp)
			case SpriteAirplane:
				s.doAirplane(sp)
			case SpriteShip:
				s.doShip(sp)
			case SpriteMonster:
				s.doMonster(sp)
			case SpriteTornado:
				s.doTornado(sp)
			case SpriteExplosion:
				s.doExplosion(sp)
			case SpriteBus:
				s.doBus(sp)
			}
			kept = append(kept, sp)
			continue
		}
		// frame 0 且無名的精靈會被回收（w_sprite.c:643 → DestroySprite）。
		// ⚠ 回收時**要把 GlobalSprites 指過來的那一格清掉**（w_sprite.c:403）。
		if sp.Named {
			kept = append(kept, sp)
			continue
		}
		if s.globals[sp.Type] == sp {
			s.globals[sp.Type] = nil
		}
	}
	s.list = kept
}

// getSprite 回傳某型別在場上的精靈；不在場上回 nil。w_sprite.c:425
func (s *spriteSystem) getSprite(t int) *Sprite {
	sp := s.globals[t]
	if sp == nil || sp.Frame == 0 {
		return nil
	}
	return sp
}

// makeSprite 生一隻精靈。w_sprite.c:437 MakeSprite
//
// ⚠ 判斷是 `GlobalSprites[type] == NULL`，**不是**「那一隻還活著嗎」——
// 同型別已經死掉（`frame == 0`）但節點還在時，原版是**原地重新初始化**，
// 不生新節點。（`GetSprite` 才是看 `frame`，那是給「場上有沒有船」用的。）
//
// ⚠ 新節點加在**尾端**。原始碼的 `NewSprite` 寫的是前插
// （`sprite->next = sim->sprite; sim->sprite = sprite`），但逐 frame 逐欄位
// 對拍量出來的順序是**先舊後新**：把新節點放前面的話，第 2 個 frame 就對
// 不上（船會排到火車前面）。順序有觀察得到的效果——`absDist` 是所有精靈
// 共用的一個全域，飛機拿「上一次算出來的距離」判斷到了沒，而那一次可能
// 是別隻精靈算的。**推論等級：強證據（實測），與原始碼的字面讀法衝突，
// 原因未解**，記在 CONTEXT 的未解表。
func (s *spriteSystem) makeSprite(t, x, y int) *Sprite {
	sp := s.globals[t]
	if sp == nil {
		sp = &Sprite{Type: t}
		s.list = append(s.list, sp)
	}
	s.initSprite(sp, x, y)
	return sp
}

// initSprite 設定一隻精靈的初值。w_sprite.c:272
func (s *spriteSystem) initSprite(sp *Sprite, x, y int) {
	sp.X, sp.Y = x, y
	sp.Frame = 0
	sp.OrigX, sp.OrigY = 0, 0
	sp.DestX, sp.DestY = 0, 0
	sp.Count, sp.SoundCount = 0, 0
	sp.Dir, sp.NewDir = 0, 0
	sp.Step, sp.Flag = 0, 0
	sp.Control = -1
	sp.Turn, sp.Accel = 0, 0
	sp.Speed = 100

	if s.globals[sp.Type] == nil {
		s.globals[sp.Type] = sp
	}

	switch sp.Type {
	case SpriteTrain:
		sp.Width, sp.Height = 32, 32
		sp.XOffset, sp.YOffset = 32, -16
		sp.XHot, sp.YHot = 40, -8
		sp.Frame = 1
		sp.Dir = 4
	case SpriteShip:
		sp.Width, sp.Height = 48, 48
		sp.XOffset, sp.YOffset = 32, -16
		sp.XHot, sp.YHot = 48, 0
		switch {
		case x < 4<<4:
			sp.Frame = 3
		case x >= (WorldX-4)<<4:
			sp.Frame = 7
		case y < 4<<4:
			sp.Frame = 5
		case y >= (WorldY-4)<<4:
			sp.Frame = 1
		default:
			sp.Frame = 3
		}
		sp.NewDir = sp.Frame
		sp.Dir = 10
		sp.Count = 1
	case SpriteMonster:
		sp.Width, sp.Height = 48, 48
		sp.XOffset, sp.YOffset = 24, 0
		sp.XHot, sp.YHot = 40, 16
		if x > (WorldX<<4)/2 {
			if y > (WorldY<<4)/2 {
				sp.Frame = 10
			} else {
				sp.Frame = 7
			}
		} else if y > (WorldY<<4)/2 {
			sp.Frame = 1
		} else {
			sp.Frame = 4
		}
		sp.Count = 1000
		sp.DestX = s.w.PolMaxX << 4
		sp.DestY = s.w.PolMaxY << 4
		sp.OrigX, sp.OrigY = sp.X, sp.Y
	case SpriteCopter:
		sp.Width, sp.Height = 32, 32
		sp.XOffset, sp.YOffset = 32, -16
		sp.XHot, sp.YHot = 40, -8
		sp.Frame = 5
		sp.Count = 1500
		sp.DestX = s.w.Rand.Rand((WorldX << 4) - 1)
		sp.DestY = s.w.Rand.Rand((WorldY << 4) - 1)
		sp.OrigX, sp.OrigY = x-30, y
	case SpriteAirplane:
		sp.Width, sp.Height = 48, 48
		sp.XOffset, sp.YOffset = 24, 0
		sp.XHot, sp.YHot = 48, 16
		if x > (WorldX-20)<<4 {
			sp.X -= 100 + 48
			sp.DestX = sp.X - 200
			sp.Frame = 7
		} else {
			sp.DestX = sp.X + 200
			sp.Frame = 11
		}
		sp.DestY = sp.Y
	case SpriteTornado:
		sp.Width, sp.Height = 48, 48
		sp.XOffset, sp.YOffset = 24, 0
		sp.XHot, sp.YHot = 40, 36
		sp.Frame = 1
		sp.Count = 200
	case SpriteExplosion:
		sp.Width, sp.Height = 48, 48
		sp.XOffset, sp.YOffset = 24, 0
		sp.XHot, sp.YHot = 40, 16
		sp.Frame = 1
	case SpriteBus:
		sp.Width, sp.Height = 32, 32
		sp.XOffset, sp.YOffset = 30, -18
		sp.XHot, sp.YHot = 40, -8
		sp.Frame = 1
		sp.Dir = 1
	}
}

// getChar 讀像素座標處的圖塊編號；出界回 −1。w_sprite.c:510
func (s *spriteSystem) getChar(x, y int) int {
	x >>= 4
	y >>= 4
	if !InBounds(x, y) {
		return -1
	}
	return int(s.w.Map[x][y] & LOMASK)
}

// turnTo 讓方向朝目標轉一步（八方向，1…8）。w_sprite.c:521
func turnTo(p, d int) int {
	if p == d {
		return p
	}
	if p < d {
		if d-p < 4 {
			p++
		} else {
			p--
		}
	} else {
		if p-d < 4 {
			p--
		} else {
			p++
		}
	}
	if p > 8 {
		p = 1
	}
	if p < 1 {
		p = 8
	}
	return p
}

// tryOther 判斷船能不能穿過去。w_sprite.c:536
func tryOther(tpoo, told, tnew int) bool {
	z := told + 4
	if z > 8 {
		z -= 8
	}
	if tnew != z {
		return false
	}
	return tpoo == POWERBASE || tpoo == POWERBASE+1 ||
		tpoo == RAILBASE || tpoo == RAILBASE+1
}

func spriteNotInBounds(sp *Sprite) bool {
	x := sp.X + sp.XHot
	y := sp.Y + sp.YHot
	return x < 0 || y < 0 || x >= WorldX<<4 || y >= WorldY<<4
}

// gdtab 是 GetDir 的方向查表。w_sprite.c:565
var gdtab = [13]int{0, 3, 2, 1, 3, 4, 5, 7, 6, 5, 7, 8, 1}

// getDir 回傳從起點指向終點的八方向，並把曼哈頓距離寫進 absDist。
// w_sprite.c:564
//
// ⚠ 第二個修正分支寫的是 `else if ((dispY << 1) < dispY) z--;`
// —— `dispY` 在這裡已經取過絕對值，所以 `dispY*2 < dispY` **永遠是假**，
// 那個 `z--` 一次都不會執行。看起來是把 `dispX` 打成 `dispY` 的筆誤，
// 但它是原版行為，照抄。
func (s *spriteSystem) getDir(orgX, orgY, desX, desY int) int {
	dispX := desX - orgX
	dispY := desY - orgY
	var z int
	if dispX < 0 {
		if dispY < 0 {
			z = 11
		} else {
			z = 8
		}
	} else {
		if dispY < 0 {
			z = 2
		} else {
			z = 5
		}
	}
	if dispX < 0 {
		dispX = -dispX
	}
	if dispY < 0 {
		dispY = -dispY
	}
	s.absDist = dispX + dispY

	if dispX<<1 < dispY {
		z++
	} else if dispY<<1 < dispY { // 永遠為假，照抄
		z--
	}
	if z < 0 || z > 12 {
		z = 0
	}
	return gdtab[z]
}

func getDis(x1, y1, x2, y2 int) int {
	dx := x1 - x2
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y2
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// checkSpriteCollision 兩隻精靈的熱點距離小於 30 就算撞上。w_sprite.c:604
func checkSpriteCollision(a, b *Sprite) bool {
	return a.Frame != 0 && b.Frame != 0 &&
		getDis(a.X+a.XHot, a.Y+a.YHot, b.X+b.XHot, b.Y+b.YHot) < 30
}

// DestroyAll 把場上的精靈全部收掉。w_sprite.c:384 DestroyAllSprites
//
// 原版是把每一個 sprite 的 frame 設成 0（frame 0 代表「不存在」），
// 節點本身留在串列裡等著被重用。這裡照做。
func (s *spriteSystem) DestroyAll() {
	for _, sp := range s.list {
		sp.Frame = 0
	}
}
