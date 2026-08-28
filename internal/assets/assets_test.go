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
	msgs, err := LoadPTF(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 {
		t.Fatal("一筆訊息都沒解出來")
	}
	const want = "More residential zones needed."
	if msgs[0].Text != want {
		t.Fatalf("第 0 筆 = %q，應為 %q", msgs[0].Text, want)
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
		msgs, err := LoadPTF(raw)
		if err != nil || len(msgs) < 40 {
			t.Errorf("%s：只解出 %d 筆訊息（err=%v）", e.Name(), len(msgs), err)
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
