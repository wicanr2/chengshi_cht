package sim

// 各型精靈的每格行為。一手出處：w_sprite.c:663 起。

// doTrain。w_sprite.c:663
//
// ⚠ 只有 `Cycle & 3 == 0`（每四個 frame）才重新找軌道，其餘 frame 只是移動。
func (s *spriteSystem) doTrain(sp *Sprite) {
	cx := [4]int{0, 16, 0, -16}
	cy := [4]int{-16, 0, 16, 0}
	dx := [5]int{0, 4, 0, -4, 0}
	dy := [5]int{-4, 0, 4, 0, 0}
	trainPic2 := [5]int{1, 2, 1, 2, 5}

	if sp.Frame == 3 || sp.Frame == 4 {
		sp.Frame = trainPic2[sp.Dir]
	}
	sp.X += dx[sp.Dir]
	sp.Y += dy[sp.Dir]
	if s.cycle&3 != 0 {
		return
	}
	dir := s.w.Rand.Rand16() & 3
	for z := dir; z < dir+4; z++ {
		dir2 := z & 3
		if sp.Dir != 4 && dir2 == ((sp.Dir+2)&3) {
			continue
		}
		c := s.getChar(sp.X+cx[dir2]+48, sp.Y+cy[dir2])
		if (c >= RAILBASE && c <= LASTRAIL) || c == RAILVPOWERH || c == RAILHPOWERV {
			if sp.Dir != dir2 && sp.Dir != 4 {
				if sp.Dir+dir2 == 3 {
					sp.Frame = 3
				} else {
					sp.Frame = 4
				}
			} else {
				sp.Frame = trainPic2[dir2]
			}
			if c == RAILBASE || c == RAILBASE+1 {
				sp.Frame = 5
			}
			sp.Dir = dir2
			return
		}
	}
	if sp.Dir == 4 {
		sp.Frame = 0
		return
	}
	sp.Dir = 4
}

// doCopter。w_sprite.c:712
//
// ⚠ 直昇機會**被怪獸與龍捲風吸引**（原始碼註解：so it blows up more often）。
func (s *spriteSystem) doCopter(sp *Sprite) {
	cdx := [9]int{0, 0, 3, 5, 3, 0, -3, -5, -3}
	cdy := [9]int{0, -5, -3, 0, 3, 5, 3, 0, -3}

	if sp.SoundCount > 0 {
		sp.SoundCount--
	}
	if sp.Control < 0 {
		if sp.Count > 0 {
			sp.Count--
		}
		if sp.Count == 0 {
			if m := s.getSprite(SpriteMonster); m != nil {
				sp.DestX, sp.DestY = m.X, m.Y
			} else if t := s.getSprite(SpriteTornado); t != nil {
				sp.DestX, sp.DestY = t.X, t.Y
			} else {
				sp.DestX, sp.DestY = sp.OrigX, sp.OrigY
			}
		}
		if sp.Count == 0 { // 降落
			s.getDir(sp.X, sp.Y, sp.OrigX, sp.OrigY)
			if s.absDist < 30 {
				sp.Frame = 0
				return
			}
		}
	} else {
		s.getDir(sp.X, sp.Y, sp.DestX, sp.DestY)
		if s.absDist < 16 {
			sp.DestX, sp.DestY = sp.OrigX, sp.OrigY
			sp.Control = -1
		}
	}

	if sp.SoundCount == 0 { // 回報壅塞
		x := (sp.X + 48) >> 5
		y := sp.Y >> 5
		if x >= 0 && x < WorldX>>1 && y >= 0 && y < WorldY>>1 {
			if s.w.TrfDensity[x][y] > 170 && s.w.Rand.Rand16()&7 == 0 {
				s.w.SendMesAt(-MsgHeavyTraffic, (x<<1)+1, (y<<1)+1)
				s.w.playSound(SoundHeavyTraffic)
				sp.SoundCount = 200
			}
		}
	}
	z := sp.Frame
	if s.cycle&3 == 0 {
		z = turnTo(z, s.getDir(sp.X, sp.Y, sp.DestX, sp.DestY))
		sp.Frame = z
	}
	sp.X += cdx[z]
	sp.Y += cdy[z]
}

// doAirplane。w_sprite.c:786
func (s *spriteSystem) doAirplane(sp *Sprite) {
	cdx := [12]int{0, 0, 6, 8, 6, 0, -6, -8, -6, 8, 8, 8}
	cdy := [12]int{0, -8, -6, 0, 6, 8, 6, 0, -6, 0, 0, 0}

	z := sp.Frame
	if s.cycle%5 == 0 {
		if z > 8 { // 起飛
			z--
			if z < 9 {
				z = 3
			}
			sp.Frame = z
		} else {
			z = turnTo(z, s.getDir(sp.X, sp.Y, sp.DestX, sp.DestY))
			sp.Frame = z
		}
	}
	// ⚠ absDist 是上一次 getDir 的殘留值——如果這個 frame 沒有呼叫 getDir，
	// 用的就是更早以前算出來的距離。照抄。
	if s.absDist < 50 {
		sp.DestX = s.w.Rand.Rand(WorldX*16+100) - 50
		sp.DestY = s.w.Rand.Rand(WorldY*16+100) - 50
	}

	if !s.w.NoDisasters {
		explode := false
		for _, o := range s.list {
			if o.Frame != 0 &&
				(o.Type == SpriteCopter || (o != sp && o.Type == SpriteAirplane)) &&
				checkSpriteCollision(sp, o) {
				s.explodeSprite(o)
				explode = true
			}
		}
		if explode {
			s.explodeSprite(sp)
		}
	}

	sp.X += cdx[z]
	sp.Y += cdy[z]
	if spriteNotInBounds(sp) {
		sp.Frame = 0
	}
}

// doShip。w_sprite.c:835
func (s *spriteSystem) doShip(sp *Sprite) {
	bdx := [9]int{0, 0, 1, 1, 1, 0, -1, -1, -1}
	bdy := [9]int{0, -1, -1, 0, 1, 1, 1, 0, -1}
	bpx := [9]int{0, 0, 2, 2, 2, 0, -2, -2, -2}
	bpy := [9]int{0, -2, -2, 0, 2, 2, 2, 0, -2}
	btClrTab := [8]int{RIVER, CHANNEL, POWERBASE, POWERBASE + 1,
		RAILBASE, RAILBASE + 1, BRWH, BRWV}

	t := RIVER

	if sp.SoundCount > 0 {
		sp.SoundCount--
	}
	if sp.SoundCount == 0 {
		// ⚠ **不管有沒有發聲都抽樣**——條件不成立時那一次 Rand 照樣消耗掉。
		if s.w.Rand.Rand16()&3 == 1 {
			if s.w.Scenario == ScenarioSanFrancisco {
				s.w.Rand.Rand(10) // 舊金山的汽笛有兩種，多抽一次
			}
			s.w.playSound(SoundShipHorn)
		}
		sp.SoundCount = 200
	}

	if sp.Count > 0 {
		sp.Count--
	}
	if sp.Count == 0 {
		sp.Count = 9
		if sp.Frame != sp.NewDir {
			sp.Frame = turnTo(sp.Frame, sp.NewDir)
			return
		}
		tem := s.w.Rand.Rand16() & 7
		pem := tem
		for ; pem < tem+8; pem++ {
			z := (pem & 7) + 1
			if z == sp.Dir {
				continue
			}
			x := ((sp.X + 47) >> 4) + bdx[z]
			y := (sp.Y >> 4) + bdy[z]
			if !InBounds(x, y) {
				continue
			}
			t = int(s.w.Map[x][y] & LOMASK)
			if t == CHANNEL || t == BRWH || t == BRWV || tryOther(t, sp.Dir, z) {
				sp.NewDir = z
				sp.Frame = turnTo(sp.Frame, sp.NewDir)
				sp.Dir = z + 4
				if sp.Dir > 8 {
					sp.Dir -= 8
				}
				break
			}
		}
		if pem == tem+8 {
			sp.Dir = 10
			sp.NewDir = (s.w.Rand.Rand16() & 7) + 1
		}
	} else {
		z := sp.Frame
		if z == sp.NewDir {
			sp.X += bpx[z]
			sp.Y += bpy[z]
		}
	}
	if spriteNotInBounds(sp) {
		sp.Frame = 0
		return
	}
	if !s.w.NoDisasters {
		// ⚠ 這個迴圈的寫法很怪：只有在 z 走到 7 而且前面都沒 break 時才爆炸，
		// 所以「船開到不該開的地方」的判定其實是「t 不在白名單裡」。照抄。
		for z := 0; z < 8; z++ {
			if t == btClrTab[z] {
				break
			}
			if z == 7 {
				s.explodeSprite(sp)
				s.destroy(sp.X+48, sp.Y)
			}
		}
	}
}

// doMonster。w_sprite.c:930
func (s *spriteSystem) doMonster(sp *Sprite) {
	gx := [5]int{2, 2, -2, -2, 0}
	gy := [5]int{-2, 2, 2, -2, 0}
	nd1 := [4]int{0, 1, 2, 3}
	nd2 := [4]int{1, 2, 3, 0}
	nn1 := [4]int{2, 5, 8, 11}
	nn2 := [4]int{11, 2, 5, 8}

	if sp.SoundCount > 0 {
		sp.SoundCount--
	}

	var d, z, c int
	if sp.Control < 0 {
		if sp.Control == -2 {
			d = (sp.Frame - 1) / 3
			z = (sp.Frame - 1) % 3
			if z == 2 {
				sp.Step = 0
			}
			if z == 0 {
				sp.Step = 1
			}
			if sp.Step != 0 {
				z++
			} else {
				z--
			}
			c = s.getDir(sp.X, sp.Y, sp.DestX, sp.DestY)
			if s.absDist < 18 {
				sp.Control = -1
				sp.Count = 1000
				sp.Flag = 1
				sp.DestX, sp.DestY = sp.OrigX, sp.OrigY
			} else {
				c = (c - 1) / 2
				if (c != d && s.w.Rand.Rand(5) == 0) || s.w.Rand.Rand(20) == 0 {
					diff := (c - d) & 3
					if diff == 1 || diff == 3 {
						d = c
					} else {
						if s.w.Rand.Rand16()&1 != 0 {
							d++
						} else {
							d--
						}
						d &= 3
					}
				} else if s.w.Rand.Rand(20) == 0 {
					if s.w.Rand.Rand16()&1 != 0 {
						d++
					} else {
						d--
					}
					d &= 3
				}
			}
		} else {
			d = (sp.Frame - 1) / 3
			if d < 4 { // 轉向
				z = (sp.Frame - 1) % 3
				if z == 2 {
					sp.Step = 0
				}
				if z == 0 {
					sp.Step = 1
				}
				if sp.Step != 0 {
					z++
				} else {
					z--
				}
				s.getDir(sp.X, sp.Y, sp.DestX, sp.DestY)
				if s.absDist < 60 {
					if sp.Flag == 0 {
						sp.Flag = 1
						sp.DestX, sp.DestY = sp.OrigX, sp.OrigY
					} else {
						sp.Frame = 0
						return
					}
				}
				c = s.getDir(sp.X, sp.Y, sp.DestX, sp.DestY)
				c = (c - 1) / 2
				if c != d && s.w.Rand.Rand(10) == 0 {
					if s.w.Rand.Rand16()&1 != 0 {
						z = nd1[d]
					} else {
						z = nd2[d]
					}
					d = 4
					if sp.SoundCount == 0 {
						s.w.playSound(SoundMonster)
						sp.SoundCount = 50 + s.w.Rand.Rand(100)
					}
				}
			} else {
				d = 4
				c = sp.Frame
				z = (c - 13) & 3
				if s.w.Rand.Rand16()&3 == 0 {
					if s.w.Rand.Rand16()&1 != 0 {
						z = nn1[z]
					} else {
						z = nn2[z]
					}
					d = (z - 1) / 3
					z = (z - 1) % 3
				}
			}
		}
	} else {
		d = sp.Control
		z = (sp.Frame - 1) % 3
		if z == 2 {
			sp.Step = 0
		}
		if z == 0 {
			sp.Step = 1
		}
		if sp.Step != 0 {
			z++
		} else {
			z--
		}
	}

	z = d*3 + z + 1
	if z > 16 {
		z = 16
	}
	sp.Frame = z

	if d < 0 {
		d = 0
	}
	if d > 4 {
		d = 4
	}
	sp.X += gx[d]
	sp.Y += gy[d]

	if sp.Count > 0 {
		sp.Count--
	}
	c = s.getChar(sp.X+sp.XHot, sp.Y+sp.YHot)
	if c == -1 || (c == RIVER && sp.Count != 0 && sp.Control == -1) {
		sp.Frame = 0 // 回到水裡就消失
	}

	for _, o := range s.list {
		if o.Frame != 0 &&
			(o.Type == SpriteAirplane || o.Type == SpriteCopter ||
				o.Type == SpriteShip || o.Type == SpriteTrain) &&
			checkSpriteCollision(sp, o) {
			s.explodeSprite(o)
		}
	}
	s.destroy(sp.X+48, sp.Y+16)
}

// doTornado。w_sprite.c:1066
func (s *spriteSystem) doTornado(sp *Sprite) {
	cdx := [6]int{2, 3, 2, 0, -2, -3}
	cdy := [6]int{-2, 0, 2, 3, 2, 0}

	z := sp.Frame
	if z == 2 {
		if sp.Flag != 0 {
			z = 3
		} else {
			z = 1
		}
	} else {
		if z == 1 {
			sp.Flag = 1
		} else {
			sp.Flag = 0
		}
		z = 2
	}
	if sp.Count > 0 {
		sp.Count--
	}
	sp.Frame = z

	for _, o := range s.list {
		if o.Frame != 0 &&
			(o.Type == SpriteAirplane || o.Type == SpriteCopter ||
				o.Type == SpriteShip || o.Type == SpriteTrain) &&
			checkSpriteCollision(sp, o) {
			s.explodeSprite(o)
		}
	}

	z = s.w.Rand.Rand(5)
	sp.X += cdx[z]
	sp.Y += cdy[z]
	if spriteNotInBounds(sp) {
		sp.Frame = 0
	}
	// ⚠ 只有 count 已經歸零**以外**的時候才有機會消失：
	// 條件是 `count != 0 && !Rand(500)`。所以龍捲風的壽命是機率性的。
	if sp.Count != 0 && s.w.Rand.Rand(500) == 0 {
		sp.Frame = 0
	}
	s.destroy(sp.X+48, sp.Y+40)
}

// doExplosion。w_sprite.c:1117
//
// 爆炸播六格動畫，結束時在五個位置點火。
func (s *spriteSystem) doExplosion(sp *Sprite) {
	if s.cycle&1 == 0 {
		if sp.Frame == 1 {
			s.w.playSound(SoundExplosion)
			s.w.SendMesAt(MsgExplosion, (sp.X>>4)+3, sp.Y>>4)
		}
		sp.Frame++
	}
	if sp.Frame > 6 {
		sp.Frame = 0
		s.startFire(sp.X+48-8, sp.Y+16)
		s.startFire(sp.X+48-24, sp.Y)
		s.startFire(sp.X+48+8, sp.Y)
		s.startFire(sp.X+48-24, sp.Y+32)
		s.startFire(sp.X+48+8, sp.Y+32)
	}
}

// doBus。w_sprite.c:1144
//
// **公車永遠不會出現**：唯一的產生點 `GenerateBus` 在 `DoRoad` 裡被註解掉了
// （`s_sim.c:776` 的 `/* GenerateBus(SMapX, SMapY); */`）。
// 這裡留一個不做事的實作，並在筆記記下這件事。
func (s *spriteSystem) doBus(sp *Sprite) {}
