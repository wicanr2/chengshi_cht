// Package assets 解 DOS 版 1.10 的資料檔。它只做解碼，回傳純資料結構，
// 不認識 Ebiten，也不認識模擬規則。
package assets

import "fmt"

// DOS 版所有資料檔共用一種 LZSS 壓縮。
// 證據：docs/formats/02-dos-lzss.md
//
// 參數是實測反推的（原版執行檔尚未反組譯）：
//
//	環形緩衝 4096 位元組，初值填 0x20（空白）
//	寫入指標起點 4078（＝ 4096 − 18）
//	旗標位元組，LSB 先，位元為 1 代表字面量、0 代表回指
//	回指是兩個位元組：off = b1 | ((b2 & 0xF0) << 4)、len = (b2 & 0x0F) + 3
//
// 驗證方式見同一份文件：`.PPF` 解出來剛好是 112000 位元組
// （640×350 EGA 四平面），`.PSN` 解出來是 27264（144 頭 ＋ 27120 城市檔），
// `DATA/SOUNDDAT.PSF` 解出來的大小與未壓縮的 `SOUNDDAT.V4` 完全相同。
const (
	lzssRingSize = 4096
	lzssRingInit = 0x20
	lzssRingRPos = 4078 // 4096 - 18
	lzssMinLen   = 3    // THRESHOLD 2 → 長度欄位加 3
)

// MaxDecompressed 是一次解壓的上限，防止壞檔造成無界配置。
// 目前最大的資料檔是 CEGADAT.PGF，解出來約 220 KB。
const MaxDecompressed = 4 << 20

// DecompressLZSS 解開一個 DOS 資料檔。
func DecompressLZSS(src []byte) ([]byte, error) {
	out := make([]byte, 0, len(src)*3)
	var win [lzssRingSize]byte
	for i := range win {
		win[i] = lzssRingInit
	}
	r := lzssRingRPos
	i := 0
	for i < len(src) {
		flags := src[i]
		i++
		for b := 0; b < 8; b++ {
			if i >= len(src) {
				break
			}
			if len(out) > MaxDecompressed {
				return nil, fmt.Errorf("解壓超過上限 %d 位元組，來源大概不是 LZSS", MaxDecompressed)
			}
			if flags&(1<<uint(b)) != 0 {
				c := src[i]
				i++
				out = append(out, c)
				win[r] = c
				r = (r + 1) % lzssRingSize
				continue
			}
			if i+1 >= len(src) {
				// 尾端不足兩個位元組的回指：原版靠檔案長度收尾，這裡直接結束。
				return out, nil
			}
			b1, b2 := int(src[i]), int(src[i+1])
			i += 2
			off := b1 | ((b2 & 0xF0) << 4)
			ln := (b2 & 0x0F) + lzssMinLen
			for k := 0; k < ln; k++ {
				c := win[(off+k)%lzssRingSize]
				out = append(out, c)
				win[r] = c
				r = (r + 1) % lzssRingSize
			}
		}
	}
	return out, nil
}
