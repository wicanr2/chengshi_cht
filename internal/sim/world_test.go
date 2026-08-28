package sim

import "testing"

// SmY 有進位。這一條單獨立一個測試，因為寫成 SimHeight>>3 之後
// 一切照跑，要等城市長到地圖底部才看得出少一列。
// docs/spec/map-and-tiles.md 不變量 3
func TestSmYHasCarry(t *testing.T) {
	if SmY != 13 {
		t.Fatalf("SmY = %d，原版 ((SimHeight+7)>>3) = 13", SmY)
	}
	if SimHeight>>3 == SmY {
		t.Fatal("SmY 寫成 SimHeight>>3 了 —— 少了最外圈那一列")
	}
}

// HISTLEN 的單位是 byte，不是元素數（s_alloc.c 用 short* 指向 NewPtr(HISTLEN)）。
func TestHistoryLengthIsInElementsNotBytes(t *testing.T) {
	if HistLen != 240 {
		t.Errorf("HistLen = %d，HISTLEN(480 bytes) ÷ 2 = 240", HistLen)
	}
	if MiscHistLen != 120 {
		t.Errorf("MiscHistLen = %d，MISCHISTLEN(240 bytes) ÷ 2 = 120", MiscHistLen)
	}
}

// oracle 實測值的解碼：2026-08-29 讀 `sim Tile 63 63` 得到 12322。
// docs/re/03-map-and-tiles.md §2
func TestDecodeObservedOracleTile(t *testing.T) {
	const observed = 12322
	w := NewWorld(1)
	w.SetTile(63, 63, observed)

	if got := w.TileNum(63, 63); got != 34 {
		t.Errorf("圖塊編號 = %d，應為 34（TREEBASE 21 ≤ 34 ≤ LASTTREE 36）", got)
	}
	if !w.HasFlag(63, 63, BULLBIT) {
		t.Error("樹應該可推平（BULLBIT）")
	}
	if !w.HasFlag(63, 63, BURNBIT) {
		t.Error("樹應該可燃（BURNBIT）")
	}
	if w.HasFlag(63, 63, CONDBIT) {
		t.Error("樹不該導電（CONDBIT）")
	}
	if w.HasFlag(63, 63, ZONEBIT) {
		t.Error("樹不是分區中心（ZONEBIT）")
	}
	if w.HasFlag(63, 63, PWRBIT) {
		t.Error("樹不該有電（PWRBIT）")
	}
}

// 換圖塊不能動到旗標，反之亦然。docs/spec/map-and-tiles.md 不變量 2
func TestSetTileNumKeepsFlags(t *testing.T) {
	w := NewWorld(1)
	w.SetTile(1, 1, PWRBIT|CONDBIT|uint16(ROADS))
	w.SetTileNum(1, 1, INTERSECTION)
	if got := w.TileNum(1, 1); got != INTERSECTION {
		t.Errorf("圖塊 = %d，應為 %d", got, INTERSECTION)
	}
	if w.TileFlags(1, 1) != PWRBIT|CONDBIT {
		t.Errorf("旗標被動到了：%d", w.TileFlags(1, 1))
	}
}

// 圖塊編號只有 10 位元，而 TILE_COUNT = 960，所以每個常數都塞得下。
func TestAllTileConstantsFitInLomask(t *testing.T) {
	if TILE_COUNT > LOMASK+1 {
		t.Fatalf("TILE_COUNT %d 超過 LOMASK 容量 %d", TILE_COUNT, LOMASK+1)
	}
	for name, v := range map[string]int{
		"LASTZONE": LASTZONE, "VBRDG3": VBRDG3, "NUCLEAR": NUCLEAR,
		"FOOTBALLGAME2": FOOTBALLGAME2,
	} {
		if v > LOMASK {
			t.Errorf("%s = %d 放不進 10 位元", name, v)
		}
	}
}

// 電力位元圖的索引：原版寫 (x>>4)+(y<<3)，這裡寫乘法，兩者必須相等。
func TestPowerWordMatchesOriginalShift(t *testing.T) {
	for y := 0; y < WorldY; y++ {
		for x := 0; x < WorldX; x++ {
			word, _ := PowerWord(x, y)
			if want := (x >> 4) + (y << 3); word != want {
				t.Fatalf("PowerWord(%d,%d) = %d，原版 (x>>4)+(y<<3) = %d", x, y, word, want)
			}
			if word >= PowerMapRow*WorldY {
				t.Fatalf("PowerWord(%d,%d) = %d 越界", x, y, word)
			}
		}
	}
}

func TestPowerBitRoundTrip(t *testing.T) {
	w := NewWorld(1)
	for _, p := range [][2]int{{0, 0}, {15, 0}, {16, 0}, {119, 99}, {63, 50}} {
		if w.TestPowerBit(p[0], p[1]) {
			t.Fatalf("(%d,%d) 一開始就有電", p[0], p[1])
		}
		w.SetPowerBit(p[0], p[1])
		if !w.TestPowerBit(p[0], p[1]) {
			t.Fatalf("(%d,%d) 設了電卻讀不到", p[0], p[1])
		}
	}
	// 相鄰的格子不該被連帶設起來
	w2 := NewWorld(1)
	w2.SetPowerBit(16, 0)
	if w2.TestPowerBit(15, 0) || w2.TestPowerBit(17, 0) {
		t.Error("設一格的電位元波及了鄰居")
	}
}
