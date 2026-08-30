package ui

// 背景音樂。
//
// ⚠ **原版沒有音樂，這是 remake 加的。** 1989 年的 DOS 版只有八段數位
// 音效，證據四條互相印證（`docs/re/19-no-music.md`）：官方手冊的製作名單
// 只有 `Sounds:` 沒有 `Music:`、`SIMCITY.CFG` 只選得了音效裝置、
// 執行檔的字串表裡沒有任何音樂相關字串、而 148 秒的實機錄音裡
// **只有 1 秒不是靜音**（那 1 秒是對話框的提示音，正好當正對照）。
//
// 所以這裡做的是**播放玩家自己準備的音樂**：掃一個 `music/` 目錄，
// 依檔名排序當播放清單。本專案不散布任何音樂。

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// musicExt 是認得的副檔名。`.ogg` 與 `.wav` 的解碼器都在 Ebiten 自己的
// 子套件裡，不必為了音樂多牽一個播放框架進來。
var musicExt = map[string]bool{".ogg": true, ".wav": true}

type musicPlayer struct {
	ctx    *eaudio.Context
	dir    string
	tracks []string
	cur    int
	p      *eaudio.Player
	on      bool
	started bool
	vol     float64
}

// FindMusicDir 決定音樂目錄：命令列指定的優先，其次是存檔目錄底下的
// `music/`，最後是執行時的工作目錄。回空字串代表沒有可用的目錄。
func FindMusicDir(flagDir, savePath string) string {
	cands := []string{flagDir}
	if savePath != "" {
		cands = append(cands, filepath.Join(filepath.Dir(savePath), "music"))
	}
	cands = append(cands, "music")
	for _, d := range cands {
		if d == "" {
			continue
		}
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	return ""
}

// EnableMusic 掃描音樂目錄並準備播放器。目錄裡一首都沒有時回 nil 錯誤，
// 但播放器不會有曲目——**沒有音樂不是錯誤**，原版本來就沒有。
func (g *Game) EnableMusic(dir string) error {
	if dir == "" {
		return nil
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var tracks []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if musicExt[strings.ToLower(filepath.Ext(e.Name()))] {
			tracks = append(tracks, e.Name())
		}
	}
	sort.Strings(tracks)
	ctx := eaudio.CurrentContext()
	if ctx == nil {
		ctx = eaudio.NewContext(outSampleRate)
	}
	g.music = &musicPlayer{ctx: ctx, dir: dir, tracks: tracks, cur: 0, vol: 0.6}
	return nil
}

// MusicTracks 回傳播放清單，給選單用。
func (g *Game) MusicTracks() []string {
	if g.music == nil {
		return nil
	}
	return g.music.tracks
}

// musicOn 回報現在有沒有在放。
func (g *Game) musicOn() bool { return g.music != nil && g.music.on }

// toggleMusic 開關背景音樂。
func (g *Game) toggleMusic() {
	if g.music == nil || len(g.music.tracks) == 0 {
		g.setMessage(g.txt.UI("music_none"))
		return
	}
	if g.music.on {
		g.music.stop()
		g.setMessage(fmt.Sprintf(g.txt.UI("music_toggle"), g.txt.UI("off")))
		return
	}
	if err := g.music.play(g.music.cur); err != nil {
		g.setMessage("音樂放不出來：" + err.Error())
		return
	}
	g.setMessage(fmt.Sprintf(g.txt.UI("music_now"), g.music.tracks[g.music.cur]))
}

// stepTrack 換上一首／下一首。播放中才會立刻換。
func (g *Game) stepTrack(d int) {
	m := g.music
	if m == nil || len(m.tracks) == 0 {
		return
	}
	m.cur = (m.cur + d + len(m.tracks)) % len(m.tracks)
	if m.on {
		if err := m.play(m.cur); err != nil {
			g.setMessage("音樂放不出來：" + err.Error())
			return
		}
	}
	g.setMessage(fmt.Sprintf(g.txt.UI("music_now"), m.tracks[m.cur]))
}

// updateMusic 每個畫格看一次：一首放完就接下一首。
//
// ⚠ Ebiten 的播放器沒有「放完了」的回呼，只能問 `IsPlaying()`。
// 剛按下 `Play()` 的那一格它可能還是 false，所以要等真的開始放過
// 之後才判斷結束（`started`）。
func (g *Game) updateMusic() {
	m := g.music
	if m == nil || !m.on || m.p == nil {
		return
	}
	if m.p.IsPlaying() {
		m.started = true
		return
	}
	if !m.started {
		return
	}
	m.cur = (m.cur + 1) % len(m.tracks)
	_ = m.play(m.cur)
}

func (m *musicPlayer) stop() {
	if m.p != nil {
		_ = m.p.Close()
		m.p = nil
	}
	m.on = false
	m.started = false
}

// play 開一首。串流播放，不整首讀進記憶體——音樂檔可能有幾十 MB。
func (m *musicPlayer) play(i int) error {
	m.stop()
	if i < 0 || i >= len(m.tracks) {
		return nil
	}
	f, err := os.Open(filepath.Join(m.dir, m.tracks[i]))
	if err != nil {
		return err
	}
	// ⚠ 兩個解碼器回傳的是各自的 `*Stream`，型別不同——要接成同一個變數
	// 得先攤成介面。`io.Reader` 就夠了：`NewPlayerF32` 只要讀得到位元組。
	var st io.Reader
	switch strings.ToLower(filepath.Ext(m.tracks[i])) {
	case ".ogg":
		st, err = vorbis.DecodeF32(f)
	default:
		st, err = wav.DecodeF32(f)
	}
	if err != nil {
		_ = f.Close()
		return err
	}
	p, err := m.ctx.NewPlayerF32(st)
	if err != nil {
		_ = f.Close()
		return err
	}
	p.SetVolume(m.vol)
	p.Play()
	m.p = p
	m.on = true
	m.started = false
	return nil
}
