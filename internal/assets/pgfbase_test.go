package assets

import (
	"os"
	"path/filepath"
	"testing"
)

// 四個顯示模式的基本圖形檔都要解得開，而且第 0 庫一定是 960 張圖塊。
//
// 「走得到檔尾」本身就是判準：基本檔沒有圖形庫表，靠的是每個庫前面
// 三個位元組的行內表頭一路串下去。串錯一個位元組，後面二十幾個庫
// 不可能全部對齊。
func TestPGFBaseParses(t *testing.T) {
	dir := dosDir(t)
	cases := []struct {
		sub, name string
		tile, bpp int
	}{
		{"CEGA", "CEGADAT.PGF", 16, 4},
		{"sega", "segadat.pgf", 8, 4},
		{"MONO", "MONODAT.PGF", 16, 1},
		{"mcga", "mcgadat.pgf", 8, 8},
	}
	for _, c := range cases {
		path := findCase(dir, c.sub, c.name)
		if path == "" {
			t.Logf("%s/%s 不在，跳過", c.sub, c.name)
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		g, err := LoadPGFBase(raw, c.tile, c.bpp)
		if err != nil {
			t.Errorf("%s：%v", c.name, err)
			continue
		}
		if len(g.Banks[0].Images) != tileCount {
			t.Errorf("%s 第 0 庫 %d 張，應為 %d", c.name, len(g.Banks[0].Images), tileCount)
		}
		if g.Banks[0].Width != c.tile {
			t.Errorf("%s 圖塊邊長 %d，應為 %d", c.name, g.Banks[0].Width, c.tile)
		}
		t.Logf("%-12s %d 個庫，第 0 庫 %d 張 %d×%d，調色盤 %d 色",
			c.name, len(g.Banks), len(g.Banks[0].Images),
			g.Banks[0].Width, g.Banks[0].Height, len(g.Palette))
	}
}

// 泥土（圖塊 0）在基本檔裡是純棕色（EGA 色號 6）。
//
// 這是「第 0 庫真的從位移 0 開始」的獨立判準：位移錯一個位元組，
// 平面就會錯開，解出來不會是單一顏色。
func TestPGFBaseDirtIsBrown(t *testing.T) {
	dir := dosDir(t)
	path := findCase(dir, "CEGA", "CEGADAT.PGF")
	if path == "" {
		t.Skip("沒有 CEGADAT.PGF，跳過")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	g, err := LoadPGFBase(raw, 16, 4)
	if err != nil {
		t.Fatal(err)
	}
	px := g.Banks[0].Images[0].Pixels
	for i, v := range px {
		if v != 6 {
			t.Fatalf("圖塊 0 第 %d 個像素是色號 %d，應該整格都是 6（棕）", i, v)
		}
	}
}

func findCase(dir, sub, name string) string {
	ents, err := os.ReadDir(filepath.Join(dir, sub))
	if err != nil {
		return ""
	}
	for _, e := range ents {
		if equalFold(e.Name(), name) {
			return filepath.Join(dir, sub, e.Name())
		}
	}
	return ""
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 32
		}
		if 'A' <= y && y <= 'Z' {
			y += 32
		}
		if x != y {
			return false
		}
	}
	return true
}
