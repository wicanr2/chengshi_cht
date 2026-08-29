package audio

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// 靜音進去要靜音出來。原版的靜音是一整片 0x88（4 位元的 8，中心值），
// 攤成 8 位元是 136——**不是 128**。這支測試是那個直流偏移的哨兵：
// 中心值寫錯的話波形完全正確，只是每一段頭尾多一聲「啵」。
func TestSilenceStaysSilent(t *testing.T) {
	s := assets.Sound{Raw: make([]byte, 256)}
	for i := range s.Raw {
		s.Raw[i] = 0x88
	}
	pcm := resample(s.Samples(), DOSSampleRate, 48000)
	if len(pcm) == 0 {
		t.Fatal("沒有輸出")
	}
	for i := 0; i+1 < len(pcm); i += 2 {
		if v := int16(binary.LittleEndian.Uint16(pcm[i:])); v != 0 {
			t.Fatalf("第 %d 個取樣是 %d，靜音應該是 0", i/2, v)
		}
	}
}

// 重取樣要拉長，而且是立體聲（一個取樣四個位元組）。
func TestResampleLengthAndChannels(t *testing.T) {
	src := make([]byte, 1000) // 1000 個 8 位元取樣
	pcm := resample(src, 5400, 48000)
	want := 1000 * 48000 / 5400
	if len(pcm) != want*4 {
		t.Fatalf("輸出 %d 位元組，要 %d（%d 個取樣 × 4）", len(pcm), want*4, want)
	}
}

// 極值不能溢位成反相。4 位元的 15 攤成 255，減中心 136 是 119，
// ×256 ＝ 30464，還在 int16 裡。
func TestExtremesDoNotWrap(t *testing.T) {
	s := assets.Sound{Raw: []byte{0xFF, 0x00, 0xFF, 0x00}}
	pcm := resample(s.Samples(), DOSSampleRate, DOSSampleRate)
	var maxV, minV int16
	for i := 0; i+1 < len(pcm); i += 4 {
		v := int16(binary.LittleEndian.Uint16(pcm[i:]))
		if v > maxV {
			maxV = v
		}
		if v < minV {
			minV = v
		}
	}
	if maxV < 20000 || minV > -20000 {
		t.Fatalf("振幅不對：max %d min %d", maxV, minV)
	}
}

func TestBankKeepsEightSegments(t *testing.T) {
	snds := make([]assets.Sound, assets.SoundCount)
	for i := range snds {
		snds[i] = assets.Sound{Raw: make([]byte, 64)}
	}
	b := NewBank(snds, 48000)
	for i := 0; i < assets.SoundCount; i++ {
		if b.Clip(i) == nil {
			t.Errorf("第 %d 段沒有 PCM", i)
		}
	}
	if b.Clip(-1) != nil || b.Clip(assets.SoundCount) != nil {
		t.Error("越界應該回 nil")
	}
}

// 拿原版檔案跑完整條鏈：LZSS → 長度鏈 → 4 位元 → 重取樣。
// 判準是**每一段的長度落在合理範圍**——取樣率錯一個數量級的話，
// 爆炸會變成 0.16 秒或 16 秒，兩種都不像。
func TestRealSoundsHaveSaneDurations(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(thisFile), "..", "..",
		"workplace", "dos110", "SIMCITY 1.10", "DATA", "SOUNDDAT.PSF")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Skip("沒有解開的 DOS 1.10 資料，跳過（玩家自備）")
	}
	snds, err := assets.LoadPSF(raw)
	if err != nil {
		t.Fatal(err)
	}
	b := NewBank(snds, 48000)
	names := []string{"交通壅塞", "爆炸", "怪獸", "警笛", "船笛",
		"工具成功小", "工具成功大", "工具失敗"}
	for i := 0; i < assets.SoundCount; i++ {
		clip := b.Clip(i)
		sec := float64(len(clip)) / 4 / 48000
		if sec < 0.03 || sec > 3 {
			t.Errorf("段 %d（%s）長 %.3f 秒，不合理", i, names[i], sec)
		}
		t.Logf("段 %d %-10s %.3f 秒", i, names[i], sec)
	}
}
