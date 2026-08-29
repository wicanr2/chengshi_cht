// Package audio 把原版的 4 位元音效重取樣成播放層要的格式。
//
// 這一層**不認識 Ebiten**，與 internal/assets 同一條界線：解碼與重取樣是
// 純函式，可以在無頭環境測。真正開音效裝置的是 internal/ui。
//
// 容器格式與 4 位元編碼：docs/formats/05-psf-sound.md
// 事件對應與取樣率：docs/re/16-dos-oracle.md §五之四、§五之五
// 規格：docs/spec/sound.md
package audio

import (
	"encoding/binary"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// DOSSampleRate 是原版八段音效的取樣率。
//
// ⚠ **推論等級：強證據，不是已確認。** 這是量出來的區間 5300–5450 Hz
// 取中間值，量法見 docs/re/16-dos-oracle.md §五之五：段 0／1／4 對 X11 版
// 同名 `.au`（8000 Hz）的取樣數比值是 0.678／0.677／0.673。
//
// 真值還沒從程式碼直讀——音效卡驅動的初始化只送出一個 16 位元參數（值 20），
// 而那張卡的命令集沒有規格可查。解出來就改這一個常數，別的地方不會有第二處。
//
// 誤差的後果：±1.4% ＝ ±0.24 個半音。
const DOSSampleRate = 5400

// 事件編號。與 `.PSF` 的段序號一致——原版的 `PlaySound(n)` 的 n 就是段編號。
// 對應關係的證據見 docs/re/16-dos-oracle.md §五之四。
const (
	HeavyTraffic = 0 // 直升機回報交通壅塞
	Explosion    = 1 // 爆炸
	Monster      = 2 // 怪獸吼叫
	Siren        = 3 // 警笛（災難訊息）
	ShipHorn     = 4 // 船笛
	ToolSmall    = 5 // 工具成功，工具編號 ≤ 4
	ToolBig      = 6 // 工具成功，工具編號 > 4
	ToolFail     = 7 // 工具失敗
)

// Bank 是八段已經轉成播放格式的音效。
type Bank struct {
	rate  int
	clips [assets.SoundCount][]byte
}

// NewBank 把八段 4 位元音效重取樣成 outRate 的 16 位元小端立體聲。
//
// 線性內插就夠了：來源是 4 位元 5.4 kHz，本來就粗，
// 用更高階的濾波器只會讓它聽起來不像原版。
func NewBank(snds []assets.Sound, outRate int) *Bank {
	b := &Bank{rate: outRate}
	for i := 0; i < assets.SoundCount && i < len(snds); i++ {
		b.clips[i] = resample(snds[i].Samples(), DOSSampleRate, outRate)
	}
	return b
}

// Rate 回傳這一組音效的輸出取樣率。
func (b *Bank) Rate() int { return b.rate }

// Clip 回傳第 n 段的 PCM。編號超出範圍或那一段是空的就回 nil。
func (b *Bank) Clip(n int) []byte {
	if b == nil || n < 0 || n >= assets.SoundCount {
		return nil
	}
	return b.clips[n]
}

// resample 把 8 位元無號單聲道（中心 128）線性內插成 16 位元小端立體聲。
//
// ⚠ 中心值要對。原版的 4 位元是 0–15、中心 8，`assets.Sound.Samples` 乘 17
// 攤成 0–255、中心 136——不是 128。差的那 8 是**直流偏移**，
// 播出來會在每一段的頭尾聽到「啵」的一聲，而波形本身完全正確。
//
// ⚠ 增益也不能取 256。攤開之後中心以下有 136 階、以上只有 119 階
// （乘 17 把 15 對到 255、8 對到 136，本來就不對稱），
// 乘 256 會讓最低的取樣算成 −34816 —— **超出 int16 就繞回正數**，
// 於是最安靜的那一段變成滿格的反相噪音。取 240（＝ 32767 ÷ 136 取整）
// 讓兩側都放得下。
const (
	dcCenter = 136 // 4 位元的中心 8 乘 17
	gain     = 240 // 32767 ÷ 136，兩側都不溢位
)

func resample(src []byte, inRate, outRate int) []byte {
	if len(src) == 0 || inRate <= 0 || outRate <= 0 {
		return nil
	}
	n := len(src) * outRate / inRate
	if n < 1 {
		return nil
	}
	out := make([]byte, n*4)
	step := float64(inRate) / float64(outRate)
	for i := 0; i < n; i++ {
		p := float64(i) * step
		j := int(p)
		frac := p - float64(j)
		a := float64(src[j])
		bb := a
		if j+1 < len(src) {
			bb = float64(src[j+1])
		}
		// 8 位元無號（中心 136）→ 16 位元有號。
		v := int16(((a + (bb-a)*frac) - dcCenter) * gain)
		u := uint16(v)
		binary.LittleEndian.PutUint16(out[i*4:], u)
		binary.LittleEndian.PutUint16(out[i*4+2:], u)
	}
	return out
}
