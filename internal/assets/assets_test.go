package assets

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// dosDir 回傳解開的 DOS 1.10 資料目錄；不在就跳過（玩家自備）。
func dosDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(thisFile), "..", "..", "workplace", "dos110", "SIMCITY 1.10")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Skip("沒有解開的 DOS 1.10 資料，跳過（玩家自備）")
	}
	return p
}

// LZSS 的正確性錨點：MESSAGE.PTF 的第一筆訊息。
// 這是整條解碼鏈唯一一個「內容」斷言，其餘都驗結構。
func TestLZSSDecodesFirstMessage(t *testing.T) {
	dir := dosDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "DATA", "MESSAGE.PTF"))
	if err != nil {
		t.Skip("沒有 DATA/MESSAGE.PTF")
	}
	secs, err := LoadPTF(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) == 0 {
		t.Fatal("一個段落都沒解出來")
	}
	msgs := TextMessages(secs)
	const want = "More residential zones needed."
	if len(msgs) == 0 || msgs[0].Text != want {
		t.Fatalf("第一筆 = %q，應為 %q", msgs[0].Text, want)
	}

	// 月份是最好的回歸哨兵：把整份檔案當成「文字、屬性」交替去讀，
	// 會得到「一月、三月、五月……」——二月被當成一月的屬性吃掉，
	// 而輸出看起來完全合理，只是少了一半。
	var months *Section
	for i := range secs {
		if len(secs[i].Strings) >= 12 && secs[i].Strings[0] == "Jan" {
			months = &secs[i]
			break
		}
	}
	if months == nil {
		t.Fatal("找不到月份段落 —— 段落切割可能壞了")
	}
	want12 := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	for i, w := range want12 {
		if months.Strings[i] != w {
			t.Errorf("第 %d 個月是 %q，應為 %q", i, months.Strings[i], w)
		}
	}
	if months.Count != 12 {
		t.Errorf("月份段落宣告 %d 筆，應為 12", months.Count)
	}
}

// 七個訊息檔（基本 ＋ 六個圖形集）都要解得開，而且解出來幾乎全是可列印字元。
//
// 「可列印比例」是判斷 LZSS 參數對不對的結構性指標：參數錯一點點，
// 輸出仍然有長度、看起來也像資料，但比例會掉下來。
func TestAllMessageFilesDecode(t *testing.T) {
	dir := dosDir(t)
	entries, err := os.ReadDir(filepath.Join(dir, "DATA"))
	if err != nil {
		t.Skip("沒有 DATA 目錄")
	}
	found := 0
	for _, e := range entries {
		if !strings.EqualFold(filepath.Ext(e.Name()), ".ptf") {
			continue
		}
		found++
		raw, err := os.ReadFile(filepath.Join(dir, "DATA", e.Name()))
		if err != nil {
			t.Errorf("%s：%v", e.Name(), err)
			continue
		}
		d, err := DecompressLZSS(raw)
		if err != nil {
			t.Errorf("%s：解壓失敗 %v", e.Name(), err)
			continue
		}
		printable := 0
		for _, c := range d {
			if (c >= 32 && c < 127) || c == 0 || c == '\n' {
				printable++
			}
		}
		ratio := float64(printable) / float64(len(d))
		if ratio < 0.95 {
			t.Errorf("%s：解出 %d 位元組，可列印比例只有 %.0f%% —— LZSS 參數可能不對",
				e.Name(), len(d), ratio*100)
		}
		secs, err := LoadPTF(raw)
		if err != nil {
			t.Errorf("%s：解析失敗 %v", e.Name(), err)
			continue
		}
		msgs := TextMessages(secs)
		if len(secs) < 20 || len(msgs) < 150 {
			t.Errorf("%s：%d 個段落、%d 筆文字，太少", e.Name(), len(secs), len(msgs))
		}
	}
	if found == 0 {
		t.Skip("DATA 目錄裡沒有 .PTF")
	}
	t.Logf("解開 %d 個訊息檔", found)
}

// 八個 DOS 劇本：解出來必須是 144 位元組檔頭 ＋ 27120 位元組城市資料，
// 檔頭裡有 CITYMCRP 標記，而且長度前綴的城市名讀得出來。
//
// 27120 這個數字與 Micropolis 的 `.cty` 完全相同 —— 這是「DOS 版的地圖也是
// 120×100」最直接的證據。
func TestAllScenariosDecode(t *testing.T) {
	dir := dosDir(t)
	entries, err := os.ReadDir(filepath.Join(dir, "SCENARIO"))
	if err != nil {
		t.Skip("沒有 SCENARIO 目錄")
	}
	names := map[string]bool{}
	for _, e := range entries {
		if !strings.EqualFold(filepath.Ext(e.Name()), ".psn") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, "SCENARIO", e.Name()))
		if err != nil {
			t.Errorf("%s：%v", e.Name(), err)
			continue
		}
		sc, err := LoadPSN(raw)
		if err != nil {
			t.Errorf("%s：%v", e.Name(), err)
			continue
		}
		if len(sc.Body) != PSNBodyLen {
			t.Errorf("%s：城市資料 %d 位元組，應為 %d", e.Name(), len(sc.Body), PSNBodyLen)
		}
		if sc.Name == "" {
			t.Errorf("%s：讀不出城市名", e.Name())
		}
		names[sc.Name] = true
	}
	if len(names) != 8 {
		t.Errorf("解出 %d 個不同的城市名，應為 8：%v", len(names), names)
	}
}

// `.PPF` 是整幅畫面：解出來剛好 112000 位元組 ＝ 640 × 350 ÷ 8 × 4，
// 也就是 EGA 高解析的四個位元平面。這一條同時證實了 CEGA ＝ 640×350。
func TestIntroScreenIsExactlyEGAHiRes(t *testing.T) {
	dir := dosDir(t)
	const wantLen = 640 * 350 / 8 * 4 // 112000
	for _, name := range []string{
		filepath.Join("CEGA", "CEGANTRO.PPF"),
		filepath.Join("CEGA", "CEGASCEN.PPF"),
	} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		d, err := DecompressLZSS(raw)
		if err != nil {
			t.Errorf("%s：%v", name, err)
			continue
		}
		if len(d) != wantLen {
			t.Errorf("%s：解出 %d 位元組，640×350 的 EGA 四平面應為 %d", name, len(d), wantLen)
		}
	}
}

// 音效：DATA/SOUNDDAT.PSF 解出來的長度必須等於未壓縮的 SOUNDDAT.V4。
// 這是「.PSF 就是壓縮過的 .V4」最直接的證據。
func TestCompressedSoundMatchesUncompressedLength(t *testing.T) {
	dir := dosDir(t)
	comp, err1 := os.ReadFile(filepath.Join(dir, "DATA", "SOUNDDAT.PSF"))
	plain, err2 := os.ReadFile(filepath.Join(dir, "SOUNDDAT.V4"))
	if err1 != nil || err2 != nil {
		t.Skip("缺 SOUNDDAT.PSF 或 SOUNDDAT.V4")
	}
	d, err := DecompressLZSS(comp)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != len(plain) {
		t.Errorf("SOUNDDAT.PSF 解出 %d 位元組，SOUNDDAT.V4 是 %d", len(d), len(plain))
	}
}

// 壞資料不能讓解壓器無界配置。
func TestDecompressRejectsRunaway(t *testing.T) {
	// 病態輸入：旗標 0x00（八個都是回指）＋ 八組最長的回指（0xFF 0xFF → 長度 18）。
	// 每 17 個輸入位元組展開成 144 個，膨脹 8.5 倍；要超過 4 MB 上限約需 50 萬位元組。
	//
	// ⚠ 第一版的測試資料是 `00 FF FF` 重複，看起來很病態，實際上膨脹率不到 1——
	// 因為位移之後的旗標位元組常常是 0xFF（八個都是字面量）。**要真的算過。**
	group := append([]byte{0x00}, bytes.Repeat([]byte{0xFF, 0xFF}, 8)...)
	huge := bytes.Repeat(group, 40000) // 68 萬位元組 → 約 5.8 MB
	if _, err := DecompressLZSS(huge); err == nil {
		t.Error("超過解壓上限卻沒有拒絕")
	}
}

// 28 個 .PGF 全部要解得開，而且長度要對得上宣告。
//
// 長度公式是這個格式最強的自證：
// 資料長度 = 寬 × 高 × 張數 × 位元深度 ÷ 8 ＋ 4 × 張數（旗標 bit0 時）。
// 調色盤長度、位元深度、平面式排列任何一個弄錯，走到第二個圖形庫就會爆。
func TestAllGraphicsFilesDecode(t *testing.T) {
	dir := dosDir(t)
	cases := []struct {
		sub   string
		mode  byte
		bpp   int
		tile  int
		banks int
	}{
		{"mcga", '2', 8, 8, 17},  // MCGA 320×200 256 色
		{"CEGA", 'E', 4, 16, 24}, // EGA 640×350 十六色
		{"MONO", 'V', 1, 16, 24}, // 單色 640×350
		{"sega", 'e', 4, 8, 24},  // EGA 320×200 十六色
	}
	styles := map[string]bool{}
	total := 0
	for _, c := range cases {
		files, err := filepath.Glob(filepath.Join(dir, c.sub, "*"))
		if err != nil || len(files) == 0 {
			t.Fatalf("%s 底下找不到圖形檔", c.sub)
		}
		for _, f := range files {
			if !strings.EqualFold(filepath.Ext(f), ".pgf") {
				continue
			}
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Errorf("%s：讀不到 %v", filepath.Base(f), err)
				continue
			}
			g, err := ParsePGF(raw)
			if err != nil {
				// 基本檔沒有那五個位元組的檔頭，是另一種版面。
				if strings.Contains(err.Error(), "基本檔") {
					continue
				}
				t.Errorf("%s：解析失敗 %v", filepath.Base(f), err)
				continue
			}
			total++
			styles[g.Name] = true
			if g.Mode != c.mode {
				t.Errorf("%s：模式碼 %q，應為 %q", filepath.Base(f), g.Mode, c.mode)
			}
			if g.BitsPerPixel != c.bpp {
				t.Errorf("%s：位元深度 %d，應為 %d", filepath.Base(f), g.BitsPerPixel, c.bpp)
			}
			if len(g.Banks) != c.banks {
				t.Errorf("%s：%d 個圖形庫，應為 %d", filepath.Base(f), len(g.Banks), c.banks)
			}
			// 第 0 庫一定是 960 張地圖圖塊。960 = Micropolis 的 TILE_COUNT，
			// 是圖塊編號的獨立佐證。
			b0 := g.Banks[0]
			if len(b0.Images) != 960 {
				t.Errorf("%s：第 0 庫有 %d 張，應為 960", filepath.Base(f), len(b0.Images))
			}
			if b0.Width != c.tile || b0.Height != c.tile {
				t.Errorf("%s：圖塊是 %d×%d，應為 %d×%d",
					filepath.Base(f), b0.Width, b0.Height, c.tile, c.tile)
			}
			if len(b0.Images[0].Pixels) != c.tile*c.tile {
				t.Errorf("%s：一張圖塊解出 %d 個像素，應為 %d",
					filepath.Base(f), len(b0.Images[0].Pixels), c.tile*c.tile)
			}
		}
	}
	if total != 24 {
		t.Errorf("解開 %d 個風格圖形檔，應為 24（4 個模式 × 6 種風格）", total)
	}
	want := []string{"Ancient Asia", "Future Europe", "Future USA",
		"Medieval Times", "Moon Colony", "Wild West"}
	for _, w := range want {
		if !styles[w] {
			t.Errorf("少了風格 %q", w)
		}
	}
	if len(styles) != 6 {
		t.Errorf("解出 %d 種風格名稱，應為 6：%v", len(styles), styles)
	}
}

// 調色盤前 16 色一定是標準 EGA 十六色。這條是「調色盤起點對不對」的
// 獨立檢查——起點差一個位元組，顏色會整組錯位而且看起來仍然像調色盤。
func TestPaletteIsStandardEGA(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(dosDir(t), "mcga", "asiamcga.pgf"))
	if err != nil {
		t.Skip("找不到 asiamcga.pgf")
	}
	g, err := ParsePGF(raw)
	if err != nil {
		t.Fatal(err)
	}
	lo, mid, hi := uint8(0x00), uint8(0x57), uint8(0xab)
	full := uint8(0xff)
	want := []PGFColor{
		{lo, lo, lo}, {lo, lo, hi}, {lo, hi, lo}, {lo, hi, hi},
		{hi, lo, lo}, {hi, lo, hi}, {hi, mid, lo}, {hi, hi, hi},
		{mid, mid, mid}, {mid, mid, full}, {mid, full, mid}, {mid, full, full},
		{full, mid, mid}, {full, mid, full}, {full, full, mid}, {full, full, full},
	}
	for i, w := range want {
		if g.Palette[i] != w {
			t.Errorf("第 %d 色 = %v，應為 %v —— 調色盤起點可能差了幾個位元組",
				i, g.Palette[i], w)
		}
	}
}

// TestMessageClassAnchors 釘住段落 0 的訊息類別，順便釘住「類別屬於前一筆」
// 這個對齊。判準取自原版的訊息派送常式（docs/re/16-dos-oracle.md §五之四）：
// 十則災難訊息是類別 6、大地震是 7、四則工具錯誤是 9。
//
// 這是回歸哨兵：對齊差一筆的話，爆炸會變成類別 3，而整份檔案仍然「解得出來」。
func TestMessageClassAnchors(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(dosDir(t), "DATA", "MESSAGE.PTF"))
	if err != nil {
		t.Fatal(err)
	}
	secs, err := LoadPTF(raw)
	if err != nil {
		t.Fatal(err)
	}
	sec := secs[0]

	want := map[int]int{
		19: MsgClassDisaster, // Fire reported !
		20: MsgClassDisaster, // A Monster has been sighted !!
		22: MsgClassQuake,    // Major earthquake reported !!!
		31: MsgClassDisaster, // Explosion detected !（派送常式特例排除的那一則）
		40: MsgClassTraffic,  // Heavy Traffic reported.
		32: MsgClassToolFail, // Insufficient funds to build that.
		44: MsgClassToolFail, // Cannot build that here.
	}
	for i, w := range want {
		if got := MessageClass(sec, i); got != w {
			t.Errorf("第 %d 筆（%q）類別 %d，要 %d", i, TrimPrefix(sec.Strings[2*i]), got, w)
		}
	}

	// 每一筆都要查得到類別，最後一筆（終止記錄）除外。
	for i := 0; i < 49; i++ {
		if MessageClass(sec, i) == 0 {
			t.Errorf("第 %d 筆查不到類別", i)
		}
	}
}
