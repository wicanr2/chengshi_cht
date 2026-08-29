package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/assets"
	"github.com/wicanr2/chengshi_cht/internal/audio"
)

// cmdSound 把一份 DOS 音效檔的八段倒成 WAV。
//
// 八段各是什麼事件已經解出來了（0 交通壅塞、1 爆炸、2 怪獸、3 警笛、
// 4 船笛、5／6 工具成功、7 工具失敗，見 docs/re/16-dos-oracle.md §五之四）。
//
// ⚠ 取樣率**只到強證據**：預設 5400 Hz 是量出來的區間 5300–5450 的中值
// （同文件 §五之五），不是從程式碼直讀的常數。倒出來的檔案拿去聽或比對
// 可以，但不要拿這個數字當結論。
func cmdSound(args []string) {
	fs := flag.NewFlagSet("sound", flag.ExitOnError)
	in := fs.String("file", "", "音效檔（.PSF 或未壓縮的 .V4）")
	out := fs.String("out", "sound", "輸出目錄")
	rate := fs.Int("rate", audio.DOSSampleRate, "取樣率（預設是量出來的 5400 Hz，強證據）")
	fs.Parse(args)
	if *in == "" {
		fs.Usage()
		os.Exit(2)
	}
	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var snds []assets.Sound
	if strings.EqualFold(filepath.Ext(*in), ".v4") {
		snds, err = assets.ParsePSF(raw)
	} else {
		snds, err = assets.LoadPSF(raw)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i, s := range snds {
		pcm := s.Samples()
		p := filepath.Join(*out, fmt.Sprintf("%d.wav", i))
		if err := os.WriteFile(p, wav8(pcm, *rate), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%d  %6d 位元組 → %6d 取樣  %.2f 秒  %s\n",
			i, len(s.Raw), len(pcm), float64(len(pcm))/float64(*rate), p)
	}
}

// wav8 包一個 8 位元單聲道的 WAV。
func wav8(pcm []byte, rate int) []byte {
	b := make([]byte, 44+len(pcm))
	copy(b[0:], "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+len(pcm)))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1) // PCM
	binary.LittleEndian.PutUint16(b[22:], 1) // 單聲道
	binary.LittleEndian.PutUint32(b[24:], uint32(rate))
	binary.LittleEndian.PutUint32(b[28:], uint32(rate))
	binary.LittleEndian.PutUint16(b[32:], 1)
	binary.LittleEndian.PutUint16(b[34:], 8)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(pcm)))
	copy(b[44:], pcm)
	return b
}
