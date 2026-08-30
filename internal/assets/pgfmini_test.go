package assets

import (
	"os"
	"testing"
)

// 四個模式的縮圖尺寸都要對得上，而且色號要能解釋原版地圖視窗的畫面。
//
// 判準不是「解得開」——尺寸猜錯一樣解得開，只是畫出來變條紋。真正的
// 判準是**內容**：CEGA 的泥土（圖塊 0）是色號 6、水（2）是 9、樹林（37）
// 是 10，在 EGA 標準調色盤裡剛好是棕、亮藍、亮綠，與原版地圖視窗的畫面
// 一致（workplace/dosbox/ui-04-windows.png）。
//
// 其餘三個模式的色號不同：640×200 的 sega 用較暗的 1／2，單色只有 0／1，
// 256 色是自己的調色盤索引。它們各自寫死在下表，改動任何一個都要重新
// 對原版畫面，不是「調到綠色為止」。
func TestMiniTiles(t *testing.T) {
	dir := dosDir(t)
	cases := []struct {
		sub, name          string
		bpp                int
		w, h               int
		fonts              int
		dirt, water, woods []uint8
	}{
		{"CEGA", "ASIACEGA.PGF", 4, 3, 3, 0,
			rep(6, 9), rep(9, 9), rep(10, 9)},
		{"sega", "asiasega.pgf", 4, 3, 1, 0,
			rep(6, 3), rep(1, 3), rep(2, 3)},
		// 單色只有兩階，所以用網點：泥土實心、水一點、樹林棋盤格。
		// 這三個圖樣只有在寬度＝3 的時候才排得出來，也就是寬度的反證。
		{"MONO", "ASIAMONO.PGF", 1, 3, 3, 2,
			[]uint8{1, 1, 1, 1, 1, 1, 1, 1, 1},
			[]uint8{0, 0, 0, 0, 1, 0, 0, 0, 0},
			[]uint8{1, 0, 1, 0, 1, 0, 1, 0, 1}},
		{"mcga", "asiamcga.pgf", 8, 1, 1, 1,
			[]uint8{0x23}, []uint8{0x30}, []uint8{0x2d}},
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
		g, err := ParsePGF(raw)
		if err != nil {
			t.Errorf("%s：%v", c.name, err)
			continue
		}
		if g.Mini == nil {
			t.Errorf("%s 沒有解出縮圖", c.name)
			continue
		}
		if g.Mini.Width != c.w || g.Mini.Height != c.h {
			t.Errorf("%s 縮圖 %d×%d，應為 %d×%d",
				c.name, g.Mini.Width, g.Mini.Height, c.w, c.h)
		}
		if n := len(g.Mini.Pixels) / (g.Mini.Width * g.Mini.Height); n != tileCount {
			t.Errorf("%s 縮圖 %d 張，應為 %d", c.name, n, tileCount)
		}
		if len(g.Fonts) != c.fonts {
			t.Errorf("%s 自帶字型 %d 份，應為 %d", c.name, len(g.Fonts), c.fonts)
		}
		for _, k := range []struct {
			tile int
			want []uint8
			what string
		}{{0, c.dirt, "泥土"}, {2, c.water, "水"}, {37, c.woods, "樹林"}} {
			px := g.Mini.Tile(k.tile)
			if len(px) != len(k.want) {
				t.Errorf("%s 圖塊 %d 取到 %d 個色號，應為 %d",
					c.name, k.tile, len(px), len(k.want))
				continue
			}
			for i := range k.want {
				if px[i] != k.want[i] {
					t.Errorf("%s %s（圖塊 %d）縮圖是 %v，應為 %v",
						c.name, k.what, k.tile, px, k.want)
					break
				}
			}
		}
		t.Logf("%-12s 縮圖 %d×%d，自帶字型 %d 份", c.name, g.Mini.Width, g.Mini.Height, len(g.Fonts))
	}
}

// 單色模式自帶的兩份字型是 CP437 排序的 8×8 與 8×14，各 128 字。
// 判準用 'A'（0x41）——它的形狀在兩份裡都認得出來。
func TestMonoFonts(t *testing.T) {
	dir := dosDir(t)
	path := findCase(dir, "MONO", "ASIAMONO.PGF")
	if path == "" {
		t.Skip("MONO/ASIAMONO.PGF 不在")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	g, err := ParsePGF(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Fonts) != 2 {
		t.Fatalf("字型 %d 份，應為 2", len(g.Fonts))
	}
	for i, want := range []struct{ w, h int }{{8, 8}, {8, 14}} {
		f := g.Fonts[i]
		if f.Width != want.w || f.Height != want.h {
			t.Errorf("第 %d 份 %d×%d，應為 %d×%d", i, f.Width, f.Height, want.w, want.h)
		}
		if f.Count() != fontGlyphs {
			t.Errorf("第 %d 份 %d 字，應為 %d", i, f.Count(), fontGlyphs)
		}
		// 'A' 的上緣那一列是 0x38（..###...）——這個位元組同時證明字序是
		// CP437、位元順序是最高位在左、列高沒有算錯。
		per := f.Width * f.Height
		a := f.Pixels[0x41*per : 0x41*per+per]
		top := 0
		if f.Height == 14 {
			top = 2 // 8×14 的上面留兩列空白
		}
		want := []uint8{0, 0, 1, 1, 1, 0, 0, 0}
		got := a[top*f.Width : (top+1)*f.Width]
		for x := range want {
			if got[x] != want[x] {
				t.Errorf("第 %d 份 'A' 第 %d 列是 %v，應為 %v", i, top, got, want)
				break
			}
		}
		// 空白（0x20）整格全空。
		sp := f.Pixels[0x20*per : 0x20*per+per]
		if sum(sp) != 0 {
			t.Errorf("第 %d 份空白不是空的", i)
		}
	}
}

// rep 是「整片同一個色號」。
func rep(v uint8, n int) []uint8 {
	b := make([]uint8, n)
	for i := range b {
		b[i] = v
	}
	return b
}

func sum(b []uint8) int {
	n := 0
	for _, v := range b {
		n += int(v)
	}
	return n
}

// 縮圖在**基本檔與風格檔之間共用**——除了 MOON。
//
// 這一條原本寫成「風格檔與基本檔逐位元組相同」，是只比對 ASIA 得到的
// 結論。月球殖民地自己一份：960 張裡有 396 張不一樣。
func TestMiniTilesSharedExceptMoon(t *testing.T) {
	dir := dosDir(t)
	base := findCase(dir, "CEGA", "CEGADAT.PGF")
	if base == "" {
		t.Skip("CEGA/CEGADAT.PGF 不在")
	}
	raw, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadPGFBase(raw, 16, 4)
	if err != nil {
		t.Fatal(err)
	}
	if b.Mini == nil {
		t.Fatal("基本檔沒有解出縮圖")
	}
	for _, st := range []struct {
		name string
		same bool
	}{
		{"ASIACEGA.PGF", true}, {"MEDICEGA.PGF", true}, {"WESTCEGA.PGF", true},
		{"FUSACEGA.PGF", true}, {"FEURCEGA.PGF", true}, {"MOONCEGA.PGF", false},
	} {
		p := findCase(dir, "CEGA", st.name)
		if p == "" {
			t.Logf("%s 不在，跳過", st.name)
			continue
		}
		r, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		g, err := ParsePGF(r)
		if err != nil {
			t.Errorf("%s：%v", st.name, err)
			continue
		}
		if g.Mini == nil {
			t.Errorf("%s 沒有解出縮圖", st.name)
			continue
		}
		diff := 0
		for i := range b.Mini.Pixels {
			if g.Mini.Pixels[i] != b.Mini.Pixels[i] {
				diff++
			}
		}
		if st.same && diff != 0 {
			t.Errorf("%s 的縮圖與基本檔差 %d 個像素，應相同", st.name, diff)
		}
		if !st.same && diff == 0 {
			t.Errorf("%s 的縮圖與基本檔相同，應該自己一份", st.name)
		}
	}
}
