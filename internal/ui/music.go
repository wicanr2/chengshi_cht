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

	"github.com/wicanr2/chengshi_cht/internal/sim"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// musicExt 是認得的副檔名。`.ogg` 與 `.wav` 的解碼器都在 Ebiten 自己的
// 子套件裡，不必為了音樂多牽一個播放框架進來。
var musicExt = map[string]bool{".ogg": true, ".wav": true}

type musicPlayer struct {
	ctx     *eaudio.Context
	dir     string
	tracks  []string
	cur     int
	p       *eaudio.Player
	source  io.Closer
	on      bool
	started bool
	vol     float64
	mode    musicMode
	// startHook 只供純狀態測試攔截播放；正式執行永遠為 nil。
	startHook func(int) error
}

type musicMode uint8

const (
	musicAmbient musicMode = iota
	musicManual
	musicDisaster
)

// disasterMusicCue 是呈現層的選曲語意，不是原版規則或存檔狀態。
type disasterMusicCue uint8

const (
	musicCueNone disasterMusicCue = iota
	musicCueFire
	musicCueFlood
	musicCueAirCrash
	musicCueTornado
	musicCueEarthquake
	musicCueMonster
	musicCueAirRaid
	musicCueMeltdown
)

// 曲目映射是使用者確認的 remake 美化。只有 10004 在外部 SC2000 曲目表中
// 明載「災害發生時」；其餘不可冒稱原作事件映射。
var disasterMusicStem = map[disasterMusicCue]string{
	musicCueFire:       "sc2000-10004",
	musicCueFlood:      "sc2000-10000",
	musicCueAirCrash:   "sc2000-10002",
	musicCueTornado:    "sc2000-10003",
	musicCueEarthquake: "sc2000-10007",
	musicCueMonster:    "sc2000-10013",
	musicCueAirRaid:    "sc2000-10012",
	musicCueMeltdown:   "sc2000-10008",
}

// ambientMusicStems 直接沿用模擬層 CityClass：0…3 各一池，4／5 共池。
var ambientMusicStems = [][]string{
	{"sc2000-10010", "sc2000-10011"},
	{"sc2000-10005", "sc2000-10016"},
	{"sc2000-10006", "sc2000-10014"},
	{"sc2000-10009", "sc2000-10015"},
	{"sc2000-10001", "sc2000-10018"},
}

// FindMusicDir 決定音樂目錄：命令列指定的優先，其次是存檔目錄底下的
// `music/`，最後是執行時的工作目錄。回空字串代表沒有可用的目錄。
func FindMusicDir(flagDir, savePath string) string {
	cands := []string{flagDir}
	if savePath != "" {
		cands = append(cands, filepath.Join(filepath.Dir(savePath), "music"))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		cands = append(cands,
			filepath.Join(dir, "music"),
			// 城市.app/Contents/MacOS/chengshi → .app 旁邊的 music/。
			filepath.Join(dir, "..", "..", "..", "music"))
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
	tracks, err := scanMusicTracks(dir)
	if err != nil {
		return err
	}
	if len(tracks) == 0 {
		return nil
	}
	ctx := eaudio.CurrentContext()
	if ctx == nil {
		ctx = eaudio.NewContext(outSampleRate)
	}
	g.music = &musicPlayer{ctx: ctx, dir: dir, tracks: tracks, cur: -1, vol: 0.6,
		mode: musicAmbient}
	i := g.music.nextAmbientTrack(g.world.CityClass)
	if i < 0 {
		return nil
	}
	g.music.cur = i
	return g.music.start(i)
}

func scanMusicTracks(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
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
	return tracks, nil
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
	if err := g.music.start(g.music.cur); err != nil {
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
	m.mode = musicManual
	if m.on {
		if err := m.start(m.cur); err != nil {
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
	g.advanceMusicAfterFinish()
}

// cueDisasterMusic 由已被 UI 消費的模擬訊息觸發。重複同一事件不重開曲目；
// 找不到精確 SC2000 檔名時保留目前曲目，遊戲與音訊都不中止。
func (g *Game) cueDisasterMusic(message int) {
	m := g.music
	if m == nil || !m.on || len(m.tracks) == 0 {
		return
	}
	cue := disasterCueForMessage(message, g.world.DisasterEvent)
	stem := disasterMusicStem[cue]
	if stem == "" {
		return
	}
	i := m.trackByStem(stem)
	if i < 0 || (m.mode == musicDisaster && m.cur == i) {
		return
	}
	m.mode = musicDisaster
	m.cur = i
	if err := m.start(i); err != nil {
		fmt.Fprintf(os.Stderr, "情境配樂切換失敗（事件 %d，遊戲照跑）：%v\n", cue, err)
		return
	}
	fmt.Fprintf(os.Stderr, "情境配樂：事件 %d → %s\n", cue, m.tracks[i])
}

func (g *Game) advanceMusicAfterFinish() {
	m := g.music
	if m == nil {
		return
	}
	m.mode = musicAmbient
	m.cur = m.nextAmbientTrack(g.world.CityClass)
	if m.cur >= 0 {
		_ = m.start(m.cur)
	}
}

func disasterCueForMessage(message, scenarioEvent int) disasterMusicCue {
	if message < 0 {
		message = -message
	}
	switch message {
	case sim.MsgFire:
		return musicCueFire
	case sim.MsgFlood:
		return musicCueFlood
	case sim.MsgPlaneCrash, sim.MsgShipwreck, sim.MsgTrainCrash, sim.MsgCopterCrash:
		return musicCueAirCrash
	case sim.MsgTornado:
		return musicCueTornado
	case sim.MsgEarthquake:
		return musicCueEarthquake
	case sim.MsgMonster:
		return musicCueMonster
	case sim.MsgMeltdown:
		return musicCueMeltdown
	case sim.MsgExplosion:
		// 漢堡空襲沒有可用的 DOS 訊息文字；劇本事件 3 的爆炸是可靠替代入口。
		if scenarioEvent == 3 {
			return musicCueAirRaid
		}
	}
	return musicCueNone
}

func (m *musicPlayer) trackByStem(stem string) int {
	for i, name := range m.tracks {
		if strings.EqualFold(strings.TrimSuffix(name, filepath.Ext(name)), stem) {
			return i
		}
	}
	return -1
}

func (m *musicPlayer) nextAmbientTrack(cityClass int) int {
	if len(m.tracks) == 0 {
		return -1
	}
	pool := cityClass
	if pool < 0 {
		pool = 0
	}
	if pool >= len(ambientMusicStems) {
		pool = len(ambientMusicStems) - 1
	}
	var candidates []int
	for _, stem := range ambientMusicStems[pool] {
		if i := m.trackByStem(stem); i >= 0 {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		// 自備播放清單的安全退路：已知災難主題不進平時池；若沒有其他曲目，
		// 才讓整份清單可用，避免「有音樂卻永遠靜音」。
		disaster := make(map[string]bool, len(disasterMusicStem))
		for _, stem := range disasterMusicStem {
			disaster[stem] = true
		}
		for i, name := range m.tracks {
			stem := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
			if !disaster[stem] {
				candidates = append(candidates, i)
			}
		}
	}
	if len(candidates) == 0 {
		for i := range m.tracks {
			candidates = append(candidates, i)
		}
	}
	for i, candidate := range candidates {
		if candidate == m.cur {
			return candidates[(i+1)%len(candidates)]
		}
	}
	return candidates[0]
}

func (m *musicPlayer) stop() {
	if m.p != nil {
		_ = m.p.Close()
		m.p = nil
	}
	if m.source != nil {
		_ = m.source.Close()
		m.source = nil
	}
	m.on = false
	m.started = false
}

func (m *musicPlayer) start(i int) error {
	if m.startHook != nil {
		return m.startHook(i)
	}
	return m.play(i)
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
	m.source = f
	m.on = true
	m.started = false
	return nil
}
