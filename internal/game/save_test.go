package game

import (
	"bytes"
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
