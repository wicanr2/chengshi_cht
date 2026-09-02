// promomusic 產生推廣影片的配樂，輸出 16-bit 立體聲 WAV。
//
// ⚠ **這段音樂是本專案自己合成的，不是原版素材。**
// 原版沒有背景音樂——四條證據互相印證，見 `docs/re/19-no-music.md`
// （手冊製作名單只有 `Sounds:`、`SIMCITY.CFG` 只選得了音效裝置、
// `SOUNDDAT` 裡沒有樂曲結構、X11 版的 `.au` 全是單發音效）。
// 遊戲本體因此不附任何樂曲，只播玩家自己放進 `music/` 的檔案
// （`internal/ui/music.go`）。推廣影片要有聲音，就只能自己作一段。
//
// 合成方式全部是決定性的：同樣的參數每次算出同樣的位元組，
// 沒有亂數，也沒有取樣素材。
//
// 用法：promomusic -sec 15 -o music.wav
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
)

const (
	sampleRate = 44100
	bpm        = 76.0
)

// 半音 → 頻率。0 是中央 C（261.63 Hz），負數往下。
func pitch(semi float64) float64 {
	return 261.625565 * math.Pow(2, semi/12)
}

// 和弦進行：Dm — B♭ — F — C，每個和弦一小節。
// 用級數（半音距中央 C）寫，免得到處是浮點常數。
var chords = [][]float64{
	{2, 5, 9},   // Dm  D  F  A
	{-2, 2, 5},  // B♭  B♭ D  F
	{5, 9, 12},  // F   F  A  C
	{0, 4, 7},   // C   C  E  G
}

// 旋律：D 小調五聲音階（D E F A C）。每個元素是 {半音, 拍長}。
var melody = [][2]float64{
	{14, 1.5}, {17, 0.5}, {19, 1}, {17, 1},
	{14, 1.5}, {12, 0.5}, {9, 2},
	{17, 1.5}, {19, 0.5}, {21, 1}, {19, 1},
	{17, 2}, {14, 2},
}

// adsr 是一段音的音量包絡。t 與各段都用秒。
func adsr(t, dur, a, d, s, r float64) float64 {
	switch {
	case t < 0 || t > dur+r:
		return 0
	case t < a:
		return t / a
	case t < a+d:
		return 1 - (1-s)*(t-a)/d
	case t < dur:
		return s
	default:
		return s * (1 - (t-dur)/r)
	}
}

// voice 是加了泛音與微幅失諧的正弦音，聽起來比純正弦厚一點。
func voice(t, f float64, detune float64) float64 {
	w := math.Sin(2*math.Pi*f*t) * 1.0
	w += math.Sin(2*math.Pi*f*(1+detune)*t) * 0.6
	w += math.Sin(2*math.Pi*f*2*t) * 0.18
	w += math.Sin(2*math.Pi*f*3*t) * 0.07
	return w / 1.85
}

func main() {
	sec := flag.Float64("sec", 15, "長度（秒）")
	out := flag.String("o", "music.wav", "輸出的 WAV")
	flag.Parse()

	beat := 60.0 / bpm       // 一拍幾秒
	bar := beat * 4          // 一小節
	n := int(*sec * sampleRate)
	left := make([]float64, n)
	right := make([]float64, n)

	add := func(start, dur, f, gain, pan, detune float64, a, d, s, r float64) {
		i0 := int(start * sampleRate)
		i1 := int((start + dur + r) * sampleRate)
		for i := i0; i < i1 && i < n; i++ {
			if i < 0 {
				continue
			}
			t := float64(i)/sampleRate - start
			v := voice(float64(i)/sampleRate, f, detune) * adsr(t, dur, a, d, s, r) * gain
			left[i] += v * (1 - pan)
			right[i] += v * pan
		}
	}

	// 襯底和弦：慢起音、長延音，鋪成一片。
	for b := 0; float64(b)*bar < *sec; b++ {
		ch := chords[b%len(chords)]
		st := float64(b) * bar
		for k, semi := range ch {
			pan := 0.5 + (float64(k)-1)*0.22 // 三個音散開
			add(st, bar*0.95, pitch(semi), 0.13, pan, 0.004, 0.5, 0.6, 0.75, 0.7)
		}
		// 低音根音，低八度。
		add(st, bar*0.6, pitch(ch[0]-12), 0.16, 0.5, 0.002, 0.06, 0.4, 0.5, 0.5)
	}

	// 旋律：跑完一輪就從頭再跑，直到填滿。
	{
		t := beat * 4 // 空一小節再進來
		for i := 0; t < *sec; i++ {
			nt := melody[i%len(melody)]
			add(t, beat*nt[1]*0.9, pitch(nt[0]), 0.17, 0.5, 0.006, 0.03, 0.25, 0.6, 0.45)
			t += beat * nt[1]
		}
	}

	// 單抽頭延遲，給一點空間感。附點八分音符，回授兩次。
	dly := int(beat * 0.75 * sampleRate)
	for i := dly; i < n; i++ {
		left[i] += right[i-dly] * 0.26
		right[i] += left[i-dly] * 0.22
	}

	// 頭尾淡入淡出，免得爆音。
	fi, fo := int(1.2*sampleRate), int(1.8*sampleRate)
	for i := 0; i < n; i++ {
		g := 1.0
		if i < fi {
			g = float64(i) / float64(fi)
		}
		if i > n-fo {
			g *= float64(n-i) / float64(fo)
		}
		left[i] *= g
		right[i] *= g
	}

	// 正規化到 -3 dBFS，避免不同長度的音量忽大忽小。
	peak := 0.0
	for i := 0; i < n; i++ {
		peak = math.Max(peak, math.Max(math.Abs(left[i]), math.Abs(right[i])))
	}
	if peak == 0 {
		fmt.Fprintln(os.Stderr, "合出來是靜音")
		os.Exit(1)
	}
	norm := 0.707 / peak

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	data := uint32(n * 4) // 2 聲道 × 2 位元組
	w.WriteString("RIFF")
	binary.Write(w, binary.LittleEndian, uint32(36+data))
	w.WriteString("WAVEfmt ")
	binary.Write(w, binary.LittleEndian, uint32(16))
	binary.Write(w, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(w, binary.LittleEndian, uint16(2)) // 立體聲
	binary.Write(w, binary.LittleEndian, uint32(sampleRate))
	binary.Write(w, binary.LittleEndian, uint32(sampleRate*4))
	binary.Write(w, binary.LittleEndian, uint16(4))
	binary.Write(w, binary.LittleEndian, uint16(16))
	w.WriteString("data")
	binary.Write(w, binary.LittleEndian, data)
	for i := 0; i < n; i++ {
		binary.Write(w, binary.LittleEndian, int16(math.Round(clamp(left[i]*norm)*32767)))
		binary.Write(w, binary.LittleEndian, int16(math.Round(clamp(right[i]*norm)*32767)))
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s：%.1f 秒，%d 位元組\n", *out, *sec, 44+data)
}

func clamp(v float64) float64 {
	return math.Max(-1, math.Min(1, v))
}
