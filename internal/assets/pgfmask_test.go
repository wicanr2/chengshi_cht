package assets

import (
	"os"
	"testing"
)

// 遮罩庫：從第 10 庫起兩兩成對，後面那一庫是前一庫的透明遮罩。
//
// 判準不是「旗標是 0x0100」——單色檔每一庫都是單平面。判準是
// **尺寸與張數相同、而且內容只有 0 與 1**，再加上位置從第 10 庫起
// 兩兩成對。⚠ 第 6、7 庫也是兩個同尺寸的單平面庫（20×70），
// 但那是兩張色階圖例，照尺寸配會配錯——所以起點是 10。
func TestSpriteMasksPairUp(t *testing.T) {
	dir := dosDir(t)
	for _, c := range []struct {
		sub, name string
		tile, bpp int
		pairs     int // 預期幾對
	}{
		{"CEGA", "ASIACEGA.PGF", 16, 4, 7},
		{"sega", "asiasega.pgf", 8, 4, 7},
		{"MONO", "ASIAMONO.PGF", 16, 1, 7},
		// ⚠ 256 色**沒有遮罩**：逐位元組比對色號 0 就夠了，不必另存一份。
		{"mcga", "asiamcga.pgf", 8, 8, 0},
	} {
		path := findCase(dir, c.sub, c.name)
		if path == "" {
			t.Logf("%s/%s 不在，跳過", c.sub, c.name)
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		g, err := ParsePGF(raw)
		if err != nil {
			t.Fatalf("%s：%v", c.name, err)
		}
		pairs, blackInside := 0, 0
		for i := 10; i+1 < len(g.Banks); {
			a, m := &g.Banks[i], &g.Banks[i+1]
			if m.Flags&pgfSinglePlane == 0 || a.Width != m.Width ||
				a.Height != m.Height || len(a.Images) != len(m.Images) {
				i++
				continue
			}
			pairs++
			blackInside += checkMaskPair(t, c.name, i, a, m)
			i += 2
		}
		if pairs != c.pairs {
			t.Errorf("%s 配出 %d 對遮罩，應為 %d", c.name, pairs, c.pairs)
		}
		// ⚠ 這一條要對**整個檔案**算，不能逐庫算：有幾組精靈（爆炸、船）
		// 身上本來就沒有黑色，那幾庫的「遮罩 0 而美術 0」是 0，很正常。
		if c.pairs > 0 && blackInside == 0 {
			t.Errorf("%s：整個檔案都沒有「遮罩 0 而美術 0」的像素 —— "+
				"那樣的話色號 0 當透明就夠了，不需要遮罩", c.name)
		}
		if c.pairs > 0 {
			t.Logf("%-14s %d 對遮罩，精靈內部的黑色像素 %d 個",
				c.name, pairs, blackInside)
		}
	}
}

// checkMaskPair 驗一對美術與遮罩的語意：
//
//   - 遮罩是 1 的地方，美術幾乎一定是色號 0 —— 所以 1 是**透明**。
//   - 但**反過來不成立**：有一批像素遮罩是 0（不透明）而美術是色號 0。
//     那就是精靈內部真正的黑色（旋翼、輪廓線）。這一群的存在正是
//     「不能拿色號 0 當透明」的證據，也是原版要另存一份遮罩的理由。
//
// 回傳這一對裡「不透明的黑色」有幾個像素。
func checkMaskPair(t *testing.T, name string, bank int, a, m *PGFBank) int {
	t.Helper()
	var one1, tot, opaqueBlack int
	for k := range a.Images {
		ap, mp := a.Images[k].Pixels, m.Images[k].Pixels
		for j := range ap {
			tot++
			switch {
			case mp[j] != 0 && ap[j] != 0:
				one1++
			case mp[j] == 0 && ap[j] == 0:
				opaqueBlack++
			}
		}
	}
	if one1*20 > tot {
		t.Errorf("%s 第 %d 庫：遮罩是 1 而美術不是 0 的像素有 %.1f%%（應接近 0）——"+
			"遮罩的 1 可能不是透明", name, bank+1, float64(one1)*100/float64(tot))
	}
	return opaqueBlack
}
