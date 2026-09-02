package game

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 三種來源的切法只有一個決定：**檔頭 128**。這條把它釘住。
//
// ⚠ 這個決定錯過一次，而且錯得沒有症狀：先前 `.PSN` 跳 144、DOS 存檔把地圖
// 當成從 3264 開始，兩者都讓地圖晚 16 個位元組 ＝ 8 格被讀到。地圖是欄優先
// 存的，所以那是整張圖往下平移 8 列。純量讀得對、地物格數也對、連 remake 與
// DOS 存檔的逐格對拍都對（兩邊同樣偏移，互相抵銷）——只有跟**原版畫面**
// 對拍才看得出來。量法寫在 normalizeCityBytes 的說明與
// docs/formats/01-city-file.md。
func TestNormalizeSkips128ByteHeader(t *testing.T) {
	for _, c := range []struct {
		name string
		size int
	}{
		{"DOS 存檔", dosSaveSize},
		{"解壓後的 .PSN", psnSize},
	} {
		raw := make([]byte, c.size)
		for i := range raw {
			raw[i] = byte(i % 251)
		}
		body, err := normalizeCityBytes(raw)
		if err != nil {
			t.Fatalf("%s：%v", c.name, err)
		}
		if len(body) != sim.CityFileSize1x1 {
			t.Errorf("%s：切出 %d 位元組，應為 %d", c.name, len(body), sim.CityFileSize1x1)
			continue
		}
		if !bytes.Equal(body, raw[128:128+sim.CityFileSize1x1]) {
			t.Errorf("%s：切出來的不是 raw[128:128+27120] —— 檔頭長度是 128，"+
				"不是用檔案大小相減推出來的", c.name)
		}
	}
}

// 城市名藏在檔頭裡，而**長度前綴不可盡信**。
//
// 下面的位元組取自軟體世界 1990 年那片地形編輯器磁片上的真實檔案
// （`E220-1.ZIP`／`E220-2.ZIP`，玩家自備）。同一批檔案裡，
// `BadNews.cty` 的前綴 11 剛好等於字串長度，而 `TAIWAN`（6 字）、
// `KAOHSIUN`（8 字）與 `Joffebrg.cty`（12 字）的前綴一律是 13。
// 只信前綴就會把後面的補零一起讀成名字。
func TestCityHeaderNameCutsAtNUL(t *testing.T) {
	mk := func(prefix uint16, name string) []byte {
		h := make([]byte, cityHeaderLen)
		h[0] = byte(prefix >> 8)
		h[1] = byte(prefix)
		copy(h[2:], name)
		return h
	}
	for _, c := range []struct {
		what   string
		header []byte
		want   string
	}{
		{"前綴等於字串長度", mk(11, "BadNews.cty"), "BadNews.cty"},
		{"前綴大於字串長度（TAIWAN）", mk(13, "TAIWAN"), "TAIWAN"},
		{"前綴大於字串長度（KAOHSIUN）", mk(13, "KAOHSIUN"), "KAOHSIUN"},
		{"前綴多一（Joffebrg.cty）", mk(13, "Joffebrg.cty"), "Joffebrg.cty"},
		{"前綴為零", mk(0, "Linear.cty"), "Linear.cty"},
		{"前綴超出檔頭", mk(9999, "Meddeve.cty"), "Meddeve.cty"},
		{"沒有檔頭", nil, ""},
		{"檔頭太短", []byte{0x00}, ""},
	} {
		if got := cityHeaderName(c.header); got != c.want {
			t.Errorf("%s：得到 %q，應為 %q", c.what, got, c.want)
		}
	}
}

// 沒有檔頭的裸檔身（Micropolis 那種 27120）不該生出名字。
func TestBareBodyHasNoCityName(t *testing.T) {
	raw := make([]byte, sim.CityFileSize1x1)
	_, header, err := splitCityBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	if header != nil {
		t.Errorf("裸檔身不該有檔頭，卻切出 %d 位元組", len(header))
	}
}

// 別人做的城市檔會帶越界值，載入不能因此壞掉。
//
// 素材是軟體世界 1990 年那片地形編輯器磁片（玩家自備，解到
// `workplace/e220/`）。十一個檔裡有五個的純量超出遊戲自己會產生的範圍：
// `BADNEWS` 稅率 230、`FREDSVIL` 與 `HAPPY_IL` 稅率 −1、`JOFFEBRG` 年份 4930、
// `BIG_CITY` 資金 1 138 401 675（Maxis 自己的作弊示範城）。
//
// 這個測試要的不是「數字對」，是**載得進來、夾得住、跑得動**。
func TestLoadForeignCityFiles(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "workplace", "e220")
	files, _ := filepath.Glob(filepath.Join(dir, "*.CTY"))
	if len(files) == 0 {
		t.Skip("沒有 workplace/e220 的城市檔，跳過（玩家自備）")
	}
	for _, f := range files {
		name := filepath.Base(f)
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("%s：%v", name, err)
			continue
		}
		w, err := LoadCitySeed(f, 12345)
		if err != nil {
			t.Errorf("%s：載入失敗 %v", name, err)
			continue
		}
		if w.CityTax < 0 || w.CityTax > 20 {
			t.Errorf("%s：稅率 %d 沒有被夾進 0–20", name, w.CityTax)
		}
		if w.SimSpeed < 0 || w.SimSpeed > 3 {
			t.Errorf("%s：速度 %d 沒有被夾進 0–3", name, w.SimSpeed)
		}
		// 檔頭有名字就要讀出來，不能留在預設的 HERESVILLE。
		if len(raw) > sim.CityFileSize1x1 {
			if want := cityHeaderName(raw[:cityHeaderLen]); want != "" && w.CityName != want {
				t.Errorf("%s：城市名讀成 %q，檔頭寫的是 %q", name, w.CityName, want)
			}
		}
		// 跑一段模擬，越界值不能在這裡才炸開。
		w.SimSpeed = 3
		for i := 0; i < 200; i++ {
			w.Frame()
		}
	}
}
