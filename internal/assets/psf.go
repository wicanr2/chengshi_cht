package assets

import (
	"encoding/binary"
	"fmt"
)

// Sound 是一段解開的音效。
//
// 原版存的是 4 位元無號 PCM，一個位元組兩個取樣，**高的 nibble 在前**，
// 中心值 8。判準是把兩種 nibble 順序都攤開來比相鄰取樣的平均絕對差：
// 九個檔案的每一段都是高位在前比較平滑（差距 1.5–2 倍），沒有例外。
type Sound struct {
	Raw []byte // 原始的 4 位元封裝位元組
}

// Samples 把 4 位元攤成 8 位元無號（0–15 乘 17 攤到 0–255）。
func (s Sound) Samples() []byte {
	out := make([]byte, 0, len(s.Raw)*2)
	for _, b := range s.Raw {
		out = append(out, (b>>4)*17, (b&0x0F)*17)
	}
	return out
}

// SoundCount 是每一份音效檔固定的段數。
//
// 九份檔案（基本檔 ＋ 六個風格 ＋ 兩份 SOUNDDAT）解出來都是八段，
// 而且最後一段的結尾剛好是檔尾——長度鏈自己就把段數釘死了，
// 不必另外找一張表。
const SoundCount = 8

// ParsePSF 解一份已經解壓的音效檔（`.PSF` 解壓後，或未壓縮的 `.V4`）。
//
// 版面：
//
//	u16  之後的位元組數（＝檔案大小 − 2）
//	重複：
//	    u16  這一段的位元組數
//	    位元組
func ParsePSF(data []byte) ([]Sound, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("只有 %d 位元組", len(data))
	}
	total := int(binary.LittleEndian.Uint16(data))
	if total != len(data)-2 {
		return nil, fmt.Errorf("檔頭宣告 %d 位元組，實際 %d —— "+
			"這可能還是壓縮狀態的 .PSF，要先過 LZSS", total, len(data)-2)
	}
	var out []Sound
	off := 2
	for off+2 <= len(data) {
		n := int(binary.LittleEndian.Uint16(data[off:]))
		if n == 0 || off+2+n > len(data) {
			break
		}
		out = append(out, Sound{Raw: data[off+2 : off+2+n]})
		off += 2 + n
	}
	if off != len(data) {
		return nil, fmt.Errorf("走到第 %d 位元組就對不上了，檔案有 %d", off, len(data))
	}
	if len(out) != SoundCount {
		return nil, fmt.Errorf("解出 %d 段，應為 %d", len(out), SoundCount)
	}
	return out, nil
}

// LoadPSF 讀一份壓縮的 `.PSF`。
func LoadPSF(raw []byte) ([]Sound, error) {
	d, err := DecompressLZSS(raw)
	if err != nil {
		return nil, fmt.Errorf("解壓失敗：%w", err)
	}
	return ParsePSF(d)
}
