package sim

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// 精靈的逐 frame 對拍。
//
// 長版的東京對拍（framescen_test.go）只知道「這個 frame 精靈那一側多抽了
// 幾次」。這一份短得多（400 個 frame），但**每個 frame 都比對每一隻精靈的
// 十八個欄位**——所以分岔的時候知道的是「第 N 個 frame、第幾隻、哪個欄位」。
//
// 資料由 tools/oracle/tcl/frame-parity-tokyo-short.tcl 產生。它用
// `sim RandState` 直接讀亂數狀態，不像長版那樣抽四次來反推——指令數少四倍，
// 而且完全不擾動數列，所以這一份沒有那個「差 4」的簿記。
//
// spriteParityBudget 是這份資料集的長度：**400 個 frame 全部對上**，
// 包含每一隻精靈的十八個欄位、規則與精靈各自的抽樣次數，以及
// **整張地圖的雜湊**（`sim MapHash`）。
//
// ⚠ **這個數字不跨資料集比較。** 原版每次啟動會先產生一座隨機城市，
// 載入劇本時 `InitWillStuff` 又會 `RandomlySeedRand()` 重設種子——所以
// 重跑一次 oracle 就是一條不同的軌跡。重新產生資料集之後要重跑這個測試；
// 掉下來就是程式碼退步了。
const spriteParityBudget = 400

var spriteFieldNames = [18]string{
	"type", "frame", "x", "y", "orig_x", "orig_y", "dest_x", "dest_y",
	"count", "sound_count", "dir", "new_dir", "step", "flag", "control",
	"turn", "accel", "speed",
}

// spriteFields 把一隻精靈攤成原版倒出來的十八個欄位。
func spriteFields(sp *Sprite) [18]int {
	return [18]int{
		sp.Type, sp.Frame, sp.X, sp.Y, sp.OrigX, sp.OrigY,
		sp.DestX, sp.DestY, sp.Count, sp.SoundCount, sp.Dir, sp.NewDir,
		sp.Step, sp.Flag, sp.Control, sp.Turn, sp.Accel, sp.Speed,
	}
}

// loadSpriteFrames 讀「每個 frame 之後的精靈狀態」。
func loadSpriteFrames(t *testing.T, path string) [][][18]int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skip("沒有 " + path)
	}
	var out [][][18]int
	for _, ln := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		f := strings.Split(ln, ",")
		n, err := strconv.Atoi(f[1])
		if err != nil {
			t.Fatalf("sprite-frames.csv 這行讀不了：%q", ln)
		}
		if len(f) != 2+n*18 {
			t.Fatalf("sprite-frames.csv 第 %s 個 frame 說有 %d 隻，欄位卻是 %d 個",
				f[0], n, len(f)-2)
		}
		one := make([][18]int, n)
		for k := 0; k < n; k++ {
			for j := 0; j < 18; j++ {
				v, err := strconv.Atoi(f[2+k*18+j])
				if err != nil {
					t.Fatalf("sprite-frames.csv 這行讀不了：%q", ln)
				}
				one[k][j] = v
			}
		}
		out = append(out, one)
	}
	return out
}

func TestSpriteParity(t *testing.T) {
	dir := "testdata/frame-tokyo-short"
	raw, err := os.ReadFile("../../workplace/ref/micropolis/micropolis-activity/res/snro.555")
	if err != nil {
		t.Skip("封存裡沒有 snro.555")
	}
	cf, err := ParseCityFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	m := loadFrameMetaIn(t, dir)
	want := loadSpriteFrames(t, dir+"/sprite-frames.csv")

	w := NewWorld(1)
	w.EnableSprites()
	loadPreState(t, w, dir+"/prestate.csv")
	w.Rand.SetState(m.Init.PreState)
	w.LoadScenarioFile(cf, ScenarioTokyo)
	w.InitSimLoad = 1
	w.NewPower = true
	w.DoSimInit()
	w.Map = loadGoldenMap(t, dir+"/cp0.csv")
	loadPreState(t, w, dir+"/poststate.csv")
	w.spriteSys.list = w.spriteSys.list[:0]
	w.spriteSys.globals = [SpriteTypeCount]*Sprite{}
	loadSprites(t, w, dir+"/sprites.csv")
	// ⚠ 這一份沒有那四次 sim Rand，起始狀態直接是 RandState 讀到的值。
	w.Rand.SetState(m.Init.R0State)

	matched := 0
	for i, f := range m.Frames {
		before := w.Rand.State()
		w.SimFrame()
		sf := drawsBetween(before, w.Rand.State())
		mid := w.Rand.State()
		sites := map[string]int{}
		w.Rand.Watch = func() { sites[callerSite()]++ }
		w.spriteSys.MoveObjects()
		w.Rand.Watch = nil
		mo := drawsBetween(mid, w.Rand.State())

		if f.HasMapHash && mapHash(w) != f.MapHash {
			t.Logf("第 %d 個 frame 的**地圖**對不上：我們 %d、原版 %d",
				f.I, mapHash(w), f.MapHash)
			t.Logf("  這個 frame：規則抽 %d（原版 %d）、精靈抽 %d（原版 %d）",
				sf, f.FStat[0], mo, f.FStat[1])
			t.Logf("  精靈欄位的比對結果：%q（空字串代表精靈本身沒問題）",
				spriteMismatch(w, want[i]))
			break
		}
		if bad := spriteMismatch(w, want[i]); bad != "" {
			t.Logf("第 %d 個 frame 的精靈狀態對不上：%s", f.I, bad)
			t.Logf("  這個 frame：規則抽 %d（原版 %d）、精靈抽 %d（原版 %d）",
				sf, f.FStat[0], mo, f.FStat[1])
			if i > 0 {
				t.Logf("  上一個 frame 原版的精靈：%v", want[i-1])
			}
			t.Logf("  這個 frame 原版的精靈：%v", want[i])
			var live [][18]int
			for _, sp := range w.spriteSys.list {
				if sp.Frame != 0 {
					live = append(live, spriteFields(sp))
				}
			}
			t.Logf("  這個 frame 我們的精靈：%v", live)
			for k, v := range sites {
				t.Logf("  精靈那一側是誰抽的：%s ×%d", k, v)
			}
			if f.HasSprDraws {
				t.Logf("  原版逐型的抽樣：%v（索引就是型別編號）", f.SprDraws)
			}
			t.Logf("  我們的 cycle = %d", w.spriteSys.cycle)
			break
		}
		if sf != f.FStat[0] || mo != f.FStat[1] {
			for k, v := range sites {
				t.Logf("  精靈那一側是誰抽的：%s ×%d", k, v)
			}
			if f.HasSprDraws {
				t.Logf("  原版逐型的抽樣：%v", f.SprDraws)
			}
			t.Logf("第 %d 個 frame 分岔：規則抽 %d（原版 %d）、精靈抽 %d（原版 %d）",
				f.I, sf, f.FStat[0], mo, f.FStat[1])
			break
		}
		matched++
	}

	t.Logf("含精靈全欄位的逐 frame 對拍：%d/%d 個", matched, len(m.Frames))
	if matched < spriteParityBudget {
		t.Errorf("只對上 %d 個 frame，低於現況 %d —— 有東西退步了", matched, spriteParityBudget)
	}
	if matched > spriteParityBudget {
		t.Errorf("對上 %d 個 frame，比現況 %d 好 —— 請把門檻調到 %d",
			matched, spriteParityBudget, matched)
	}
}

// spriteMismatch 比對場上所有精靈的十八個欄位；相同回空字串。
func spriteMismatch(w *World, want [][18]int) string {
	var live []*Sprite
	for _, sp := range w.spriteSys.list {
		if sp.Frame != 0 {
			live = append(live, sp)
		}
	}
	if len(live) != len(want) {
		return fmt.Sprintf("我們場上 %d 隻，原版 %d 隻", len(live), len(want))
	}
	for k, sp := range live {
		got := spriteFields(sp)
		for j := range got {
			if got[j] != want[k][j] {
				return fmt.Sprintf("第 %d 隻（type %d）的 %s：我們 %d、原版 %d",
					k, sp.Type, spriteFieldNames[j], got[j], want[k][j])
			}
		}
	}
	return ""
}

// mapHash 與 oracle 的 `sim MapHash` 同一套 FNV-1a：x 外層、y 內層，
// 每格先低位元組再高位元組。實作在 tools/oracle/patches/apply.py。
func mapHash(w *World) uint32 {
	h := uint32(2166136261)
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			v := w.Map[x][y]
			h = (h ^ uint32(v&0xFF)) * 16777619
			h = (h ^ uint32((v>>8)&0xFF)) * 16777619
		}
	}
	return h
}
