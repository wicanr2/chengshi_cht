package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
	eaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/wicanr2/chengshi_cht/internal/assets"
	gaudio "github.com/wicanr2/chengshi_cht/internal/audio"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 音效。規格：docs/spec/sound.md
//
// 三個來源，對應原版的三個發聲位置：
//
//   - 精靈（交通壅塞、爆炸、怪獸、船笛）由規則層排進 `World` 的佇列，
//     因為觸發條件（`SoundCount`、亂數）本來就在那裡。
//   - 警笛由訊息類別決定，在訊息**顯示**的那一刻播——原版也是在
//     `doMessage` 的 `firstTime` 分支播的。
//   - 工具音由玩家的點擊結果決定。
//
// ⚠ 音效裝置開不起來就靜默降級。無頭環境（對拍、自動玩家、CI）
// 根本不會呼叫 EnableSound。

// outSampleRate 是輸出取樣率。原版是 5400 Hz（強證據，見 gaudio.DOSSampleRate），
// 但音效裝置多半只吃常見的速率，所以在 internal/audio 先重取樣上來。
const outSampleRate = 48000

type soundSystem struct {
	ctx  *eaudio.Context
	bank *gaudio.Bank
	// last 記住每一段上次播放的畫格，避免同一段在同一秒疊十次。
	// 原版的 DMA 一次只放得了一段，這是最接近的近似。
	last [assets.SoundCount]int
	// frame 是自己數的畫格，不用 Ebiten 的計時器。
	frame int
	// live 是還在播的播放器。**兩個理由都不能省**：
	//
	//  1. Ebiten 的 `audio.Context` 只在 `playingPlayers` 裡抓 `playerImpl`，
	//     不抓 `*Player` 本身（`audio.go:265 updatePlayers`）。`Play()` 之後
	//     把 `*Player` 丟掉，它就是不可達物件，GC 隨時可以執行
	//     `NewPlayer` 註冊的 `runtime.AddCleanup(p, (*playerImpl).finalize, …)`
	//     ——聲音會在播到一半被切掉，而且只在 GC 剛好落在那一刻時才發生。
	//  2. 播完的播放器不 `Close` 就只能等 GC 回收，是一條**沒有上界**的資源。
	//     蓋城市會一直觸發工具音，時間拉長就一直長。
	live []livePlayer
}

// livePlayer 記住一個播放器與它開始播的畫格。
//
// 要記畫格是因為**剛 `Play()` 的那一格 `IsPlaying()` 可能還是 false**
// （music.go 那邊踩過同一件事），立刻回收會把還沒開始的聲音關掉。
type livePlayer struct {
	p     *eaudio.Player
	start int
}

// livePlayerGrace 是新播放器免於回收的畫格數。
const livePlayerGrace = 3

// maxLivePlayers 是同時活著的播放器上限。超過就關掉最舊的——
// 上界是刻意的：沒有上界的資源在長時間遊玩下就是慢性中毒。
const maxLivePlayers = 16

// minGapFrames 是同一段音效的最短間隔（畫格）。
//
// 為什麼要有：爆炸精靈一次災難會生好幾隻，同一畫格可以排進四五聲爆炸，
// 疊起來只會爆音。半秒是聽感上的取捨，不是原版的規則。
const minGapFrames = 30

// EnableSound 載入原版音效並開音效裝置。
//
// dataDir 是解開的 SIMCITY 1.10 目錄，style 是圖形集（base／asia／…）。
// 開不起來回錯誤，呼叫端印一行警告就好，**不要因此中止遊戲**。
func (g *Game) EnableSound(dataDir, style string) error {
	path, err := soundFile(dataDir, style)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	snds, err := assets.LoadPSF(raw)
	if err != nil {
		return fmt.Errorf("%s：%w", filepath.Base(path), err)
	}
	ctx := eaudio.CurrentContext()
	if ctx == nil {
		ctx = eaudio.NewContext(outSampleRate)
	}
	g.snd = &soundSystem{ctx: ctx, bank: gaudio.NewBank(snds, outSampleRate)}
	return nil
}

// soundFile 挑出這個圖形集要用的音效檔。
//
// ⚠ 基本組是 `DATA/SOUNDDAT.PSF`，**不是根目錄那一份**。根目錄還有一份
// 2012 年重打包時放進去的同名檔，內容第 2 段不一樣；執行檔的字串表裡
// 音效檔名緊鄰 `message.ptf`、`monodat.pgf` 與 `DATA`，指向 `DATA/` 那一份
// （docs/re/16-dos-oracle.md §六）。
func soundFile(dataDir, style string) (string, error) {
	data := filepath.Join(dataDir, "DATA")
	name := "SOUNDDAT.PSF"
	if style != "" && style != "base" {
		name = strings.ToUpper(style) + "_SND.PSF"
	}
	p, err := findFile(data, name)
	if err != nil {
		return "", err
	}
	return p, nil
}

// findFile 在目錄裡找一個不分大小寫的檔名。原版磁片的大小寫不一致
// （`DATA/` 是大寫、`sega/` 是小寫），而 Linux 的檔案系統分大小寫。
func findFile(dir, name string) (string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if strings.EqualFold(e.Name(), name) {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("%s 裡找不到 %s", dir, name)
}

// ProbeAudio 真的開一次音效裝置，開得起來回 nil。
//
// ⚠ **這支只能在拋棄式的子行程裡呼叫**，不能在遊戲行程裡跑：
// `oto.NewContext` 沒有 Close，開起來之後裝置就被這個行程佔住，
// Ebiten 自己那個環境就開不成了。
//
// 為什麼要走到「另開一個行程」這一步：Ebiten 把音效環境的錯誤當成**致命
// 錯誤**，從 `RunGame` 回傳出來——裝置開不起來的機器上，遊戲會在玩家已經
// 看到視窗之後直接結束，而且訊息是一長串 ALSA 的裝置名。實測過兩種情境：
//
//   - 容器裡沒有 `/dev/snd`：oto 立刻失敗。
//   - 容器裡有 `/dev/snd` 但主機的音效伺服器佔著：
//     `snd_pcm_open` 對每一個裝置都回 `Invalid argument`。
//
// 第二種「看得到裝置卻開不起來」用檔案存不存在判斷是判不出來的，
// 所以只能真的開一次。
func ProbeAudio() error {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   outSampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		return err
	}
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		return fmt.Errorf("音效裝置三秒還沒準備好")
	}
	return ctx.Err()
}

// playSound 播一段。沒有音效系統就什麼都不做。
func (s *soundSystem) play(n int) {
	if s == nil || n < 0 || n >= assets.SoundCount {
		return
	}
	if s.frame-s.last[n] < minGapFrames && s.last[n] != 0 {
		return
	}
	pcm := s.bank.Clip(n)
	if len(pcm) == 0 {
		return
	}
	p := s.ctx.NewPlayerFromBytes(pcm)
	p.Play()
	s.live = append(s.live, livePlayer{p: p, start: s.frame})
	if len(s.live) > maxLivePlayers {
		_ = s.live[0].p.Close()
		s.live = s.live[1:]
	}
	s.last[n] = s.frame
}

// reap 關掉播完的播放器。每個畫格呼叫一次。
func (s *soundSystem) reap() {
	if s == nil || len(s.live) == 0 {
		return
	}
	kept := s.live[:0]
	for _, lp := range s.live {
		if s.frame-lp.start < livePlayerGrace || lp.p.IsPlaying() {
			kept = append(kept, lp)
			continue
		}
		_ = lp.p.Close()
	}
	s.live = kept
}

// PlaySoundOnce 直接播一段。給驗收用（`-sound-test`）：
// 遊戲一開起來就放指定的那一段，錄音才有東西可以驗。
func (g *Game) PlaySoundOnce(n int) { g.snd.play(n) }

// pumpSounds 把規則層排出來的音效播掉。Update 每個畫格呼叫一次。
func (g *Game) pumpSounds() {
	queued := g.world.TakeSounds()
	if g.snd == nil || g.soundOff {
		return
	}
	g.snd.frame++
	g.snd.reap()
	for _, n := range queued {
		g.snd.play(n)
	}
}

// toolSound 依工具與結果播 5／6／7。
//
// 分界是**工具編號 4**，不是「大建物／小建物」：原版比的就是工具編號
// （解壓映像 `0x0EFD6`／`0x0EFDE`，`es:2B50h` 與 4 比）。
// 編號順序見 sim.Tool，與 `sim.h:429` 一致。
func (g *Game) toolSound(result int, tool sim.Tool) {
	if g.snd == nil || g.soundOff {
		return
	}
	if result != sim.ToolOK {
		g.snd.play(sim.SoundToolFail)
		return
	}
	if tool <= 4 {
		g.snd.play(sim.SoundToolSmall)
	} else {
		g.snd.play(sim.SoundToolBig)
	}
}
