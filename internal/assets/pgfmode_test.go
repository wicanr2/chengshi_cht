package assets

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// modeDir 是兩片資料片解開之後放六種顯示模式圖形檔的地方。
// 玩家自備，不入版控——沒有就跳過。
func modeDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(thisFile), "..", "..", "workplace", "modes")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Skip("沒有解開的資料片圖形檔（workplace/modes），跳過（玩家自備）")
	}
	return p
}

func loadPGFFile(t *testing.T, p string) *PGF {
	t.Helper()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Skipf("讀不到 %s：%v", p, err)
	}
	g, err := ParsePGF(raw)
	if err != nil {
		t.Fatalf("%s 解析失敗：%v", p, err)
	}
	return g
}

// Tandy 的 16 色是**封裝式** 4bpp（一個位元組兩個像素、高四位元在前），
// 不是 EGA 的平面式。
//
// 判準是**與 sega 逐格比色號陣列**，不是「畫出來像地形」：
// 兩者是同一份美術的兩種編碼（EGA 低解析平面式 vs Tandy 封裝式），
// 圖塊尺寸都是 8×8、位元深度都是 4，所以色號陣列本來就該一模一樣。
//
// 為什麼要用這個判準：拿原版截圖做樣板比對只涵蓋得到**畫面上出現過的**
// 圖塊——實測跑 Tandy 原版，空地與樹林那幾段命中一百多個逐像素完全相同的
// 位置，而水域那一段（圖塊 1–20）從頭到尾沒有出現在編輯視窗裡，
// 於是「找不到」既可能是解錯也可能是沒涵蓋，分不出來。逐格比色號陣列
// 一次涵蓋全部 960 張。
//
// ⚠ 兩種讀法用掉的位元組數**一模一樣**（8×8 都是 32 個位元組），
// 所以長度檢查抓不到；讀錯畫出來是一片彩色雜訊，看起來像檔案壞了。
func TestTandyIsPacked4BPP(t *testing.T) {
	dir := modeDir(t)
	for _, style := range []string{"ASIA", "MEDI", "WEST", "FUSA", "FEUR", "MOON"} {
		tdy := loadPGFFile(t, filepath.Join(dir, style+"TDY.PGF"))
		sega := loadPGFFile(t, filepath.Join(dir, style+"SEGA.PGF"))
		if tdy.Mode != 'T' {
			t.Fatalf("%sTDY 的模式碼 = %q，要 'T'", style, tdy.Mode)
		}
		a, b := tdy.Banks[0], sega.Banks[0]
		if a.Width != 8 || a.Height != 8 || a.Width != b.Width || a.Height != b.Height {
			t.Fatalf("%s 圖塊尺寸 Tandy %dx%d／sega %dx%d，都該是 8x8",
				style, a.Width, a.Height, b.Width, b.Height)
		}
		if len(a.Images) != 960 || len(b.Images) != 960 {
			t.Fatalf("%s 第 0 庫張數 Tandy %d／sega %d，都該是 960",
				style, len(a.Images), len(b.Images))
		}
		for i := range a.Images {
			for j, v := range a.Images[i].Pixels {
				if v != b.Images[i].Pixels[j] {
					t.Fatalf("%s 圖塊 %d 的第 %d 個像素：Tandy %d、sega %d —— "+
						"Tandy 的像素排列讀錯了", style, i, j, v, b.Images[i].Pixels[j])
				}
			}
		}
	}
}

// CGA 那一份不是四色 320×200，是 `SIMCITY.CFG` 表上寫的 **CGA Mono**：
// 640×200 兩色，圖塊 16×8、單平面。判準是尺寸與位元深度——
// 猜成 8×8 兩位元封裝式的話尺寸會對不上（16 個位元組兩種讀法都成立）。
func TestCGAIsMono16x8(t *testing.T) {
	dir := modeDir(t)
	g := loadPGFFile(t, filepath.Join(dir, "ASIACGA.PGF"))
	if g.Mode != 'C' {
		t.Fatalf("模式碼 = %q，要 'C'", g.Mode)
	}
	if g.BitsPerPixel != 1 {
		t.Fatalf("位元深度 = %d，要 1", g.BitsPerPixel)
	}
	b := g.Banks[0]
	if b.Width != 16 || b.Height != 8 {
		t.Fatalf("圖塊 %dx%d，要 16x8", b.Width, b.Height)
	}
	// 兩色：色號只會是 0 或 1。
	for i, im := range b.Images {
		for _, v := range im.Pixels {
			if v > 1 {
				t.Fatalf("圖塊 %d 出現色號 %d —— 單色只該有 0 與 1", i, v)
			}
		}
	}
}

// 六種顯示模式的第 0 庫都是 960 張，與 Micropolis 的 `TILE_COUNT` 對得上。
// 尺寸與位元深度各自不同，全部量自檔案本身。
func TestAllSixDisplayModesDecode(t *testing.T) {
	dir := modeDir(t)
	want := []struct {
		suffix string
		mode   byte
		bpp    int
		w, h   int
	}{
		{"CEGA", 'E', 4, 16, 16},
		{"SEGA", 'e', 4, 8, 8},
		{"TDY", 'T', 4, 8, 8},
		{"MCGA", '2', 8, 8, 8},
		{"MONO", 'V', 1, 16, 16},
		{"CGA", 'C', 1, 16, 8},
	}
	for _, w := range want {
		g := loadPGFFile(t, filepath.Join(dir, "ASIA"+w.suffix+".PGF"))
		b := g.Banks[0]
		if g.Mode != w.mode || g.BitsPerPixel != w.bpp ||
			b.Width != w.w || b.Height != w.h || len(b.Images) != 960 {
			t.Errorf("ASIA%s：mode=%q bpp=%d %dx%d %d 張，要 mode=%q bpp=%d %dx%d 960 張",
				w.suffix, g.Mode, g.BitsPerPixel, b.Width, b.Height, len(b.Images),
				w.mode, w.bpp, w.w, w.h)
		}
	}
}
