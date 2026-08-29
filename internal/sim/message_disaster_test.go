package sim

import "testing"

// 災難訊息是「解出來了卻沒接上」的典型：規則層自己會跑災難，但玩家
// 什麼都看不到。這支測試逐項確認每一種災難都真的把訊息放進了訊息埠。
//
// 判準是**訊息編號**不是有沒有字：編號錯的話玩家會看到不相干的一行
// （DOS 與 Micropolis 的訊息表在第 30 則就不一樣），而畫面照樣有東西。
func TestDisastersSendMessages(t *testing.T) {
	cases := []struct {
		name string
		want int
		do   func(w *World)
	}{
		{"隨機火災", -MsgFire, func(w *World) {
			// SetFire 只試一次，所以直接把整張圖鋪成燒得著的圖塊。
			fill(w, uint16(LHTHR+1))
			w.SetFire()
		}},
		{"玩家放火", MsgFire, func(w *World) {
			fill(w, uint16(TREEBASE+1)|BURNBIT)
			w.MakeFire()
		}},
		{"大地震", -MsgEarthquake, func(w *World) { w.MakeEarthquake() }},
		{"水災", -MsgFlood, func(w *World) {
			// 河岸的判準是 4 < 圖塊 < 21，而且要有一格空地或可推可燒的鄰居。
			// 隔行鋪，任何一格河岸左右都是空地。
			for x := 0; x < WorldX; x++ {
				for y := 0; y < WorldY; y++ {
					if x%2 == 0 {
						w.Map[x][y] = uint16(FIRSTRIVEDGE)
					} else {
						w.Map[x][y] = 0
					}
				}
			}
			w.MakeFlood()
		}},
		{"爐心熔毀", -MsgMeltdown, func(w *World) { w.DoMeltdown(40, 40) }},
		{"怪獸", -MsgMonster, func(w *World) {
			fill(w, RIVER)
			w.MakeMonster()
		}},
		{"龍捲風", -MsgTornado, func(w *World) { w.MakeTornado() }},
	}
	for _, c := range cases {
		w := NewWorld(1)
		w.EnableSprites()
		w.ClearMes()
		c.do(w)
		if w.MessagePort != c.want {
			t.Errorf("%s：訊息埠 %d，要 %d", c.name, w.MessagePort, c.want)
		}
	}
}

// 墜毀訊息依精靈型別分四種。w_sprite.c:1383
func TestExplodeSpriteSendsCrashMessage(t *testing.T) {
	cases := []struct {
		typ  int
		want int
	}{
		{SpriteAirplane, -MsgPlaneCrash},
		{SpriteShip, -MsgShipwreck},
		{SpriteTrain, -MsgTrainCrash},
		{SpriteCopter, -MsgCopterCrash},
	}
	for _, c := range cases {
		w := NewWorld(1)
		w.EnableSprites()
		s := w.spriteSys
		sp := s.makeSprite(c.typ, 400, 400)
		w.ClearMes()
		s.explodeSprite(sp)
		if w.MessagePort != c.want {
			t.Errorf("型別 %d：訊息埠 %d，要 %d", c.typ, w.MessagePort, c.want)
		}
	}
}

// 爆炸精靈在第 1 格動畫送訊息 32（正數，只寫訊息欄）。w_sprite.c:1123
func TestExplosionSpriteSendsMessage(t *testing.T) {
	w := NewWorld(1)
	w.EnableSprites()
	s := w.spriteSys
	sp := s.makeSprite(SpriteExplosion, 400, 400)
	sp.Frame = 1
	w.ClearMes()
	s.cycle = 0
	s.doExplosion(sp)
	if w.MessagePort != MsgExplosion {
		t.Fatalf("訊息埠 %d，要 %d", w.MessagePort, MsgExplosion)
	}
}

func fill(w *World, v uint16) {
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			w.Map[x][y] = v
		}
	}
}
