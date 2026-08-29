package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// 逐 frame 對拍。
//
// 早期的分段對拍段界由 oracle 的事件迴圈決定，長度不可控
// （實測 85 到 2039 次抽樣），而且起始的 Fcycle／Scycle 觀察不到，只能搜。
// 給 oracle 加了三個觀測指令之後（`sim Frame N`、`sim Scycle`、`sim Fcycle`，
// 見 tools/oracle/patches/apply.py），這兩個問題一起消失：
//
//   - 單步：每個 frame 都有自己的亂數讀數，分歧點是**查表**不是搜尋。
//   - 起點：Fcycle 與 Scycle 都是 0，不必猜。
//
// 判準仍然是抽樣次數——LCG 的狀態由起點與步數唯一決定。
//
// 資料由 tools/oracle/tcl/frame-parity.tcl 產生、
// tools/oracle/extract_frames.py 轉檔。

// frameParityBudget 是目前逐 frame 完全一致的 frame 數。
const frameParityBudget = 8000

// frameCP 是一個 frame 的原版讀數。
//
// ⚠ 存的是**抽樣次數**不是亂數值：LCG 的狀態由起點與步數唯一決定，
// 所以次數與狀態等價，而次數就是對拍的判準（`docs/re/02-rng.md` §2）。
type frameCP struct {
	I      int
	Scycle int
	Valves [3]int
	Draws  int
	// State 是原版**讀完那四次之後**的亂數狀態。抽樣次數對上等價於
	// 狀態對上，但直接存狀態可以在對拍失敗時分辨「數目對、值錯了」。
	State uint32
	// 以下只有劇本版有：城市評估的分數與問題表。
	HasProb   bool
	CityScore int
	CityYes   int
	CityNo    int
	ProbTable [7]int
	HasVote   bool
	Vote      [4]int // 投票迴圈抽樣、市民投票抽樣、迭代、成功
	// HasFStat／FStat 是把一個 frame 的抽樣拆成「SimFrame（規則）」與
	// 「MoveObjects（精靈）」兩段。少抽一次的時候，第一件事是問它在哪一邊。
	HasFStat bool
	FStat    [2]int
	// SprDraws 是逐「型別」的抽樣次數（索引就是精靈型別編號）。
	// 精靈那一側分岔時，這個直接指出是哪一型多抽的。
	HasSprDraws bool
	SprDraws    [9]int
	// MapHash 是整張地圖的 FNV-1a（原版在 C 裡算）。地圖偏掉但抽樣次數
	// 正常的情況——例如怪獸拆房子（Destroy 不抽亂數）——只有它抓得到。
	HasMapHash bool
	MapHash    uint32
}

type frameMeta struct {
	Init struct {
		Fcycle int   `json:"fcycle"`
		Scycle int   `json:"scycle"`
		Funds  int   `json:"funds"`
		Rands  []int `json:"rands"`
		// PreState 是**載入劇本之前**的亂數狀態。載入時 DoSimInit 自己
		// 會抽亂數，所以要從這裡出發才重建得出同一份起始地圖。
		PreState uint32 `json:"prestate"`
		// R0State 是短版（用 sim RandState 直接讀）的起始狀態。
		// 長版是抽四次反推的，那四次要算進簿記；短版沒有。
		R0State uint32 `json:"r0state"`
	} `json:"init"`
	End struct {
		Fcycle int `json:"fcycle"`
		Scycle int `json:"scycle"`
		Funds  int `json:"funds"`
	} `json:"end"`
	Frames []frameCP
}

func loadFrameMeta(t *testing.T) frameMeta { return loadFrameMetaIn(t, "testdata/frame") }

// loadFrameMetaIn 讀一份逐 frame 對拍資料（meta.json ＋ frames.csv）。
func loadFrameMetaIn(t *testing.T, dir string) frameMeta {
	t.Helper()
	b, err := os.ReadFile(dir + "/meta.json")
	if err != nil {
		t.Fatal(err)
	}
	var m frameMeta
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	csv, err := os.ReadFile(dir + "/frames.csv")
	if err != nil {
		t.Fatal(err)
	}
	for _, ln := range strings.Split(strings.TrimSpace(string(csv)), "\n") {
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		var f frameCP
		cols := strings.Split(ln, ",")
		nums := make([]int, len(cols))
		for i, c := range cols {
			v, err := strconv.Atoi(strings.TrimSpace(c))
			if err != nil {
				t.Fatalf("frames.csv 這行讀不了：%q（%v）", ln, err)
			}
			nums[i] = v
		}
		// 版面用欄數分辨。基本七欄之後可以接：
		//   ＋2  FrameStats（規則／精靈各抽幾次）
		//   ＋11 FrameStats ＋ 逐型精靈的抽樣次數
		//   ＋14 城市評估 ＋ 投票計數
		// 最後**可以再多一欄地圖雜湊**，所以每種版面都有 +1 的變體。
		if n := len(nums); n == 8 || n == 10 || n == 19 || n == 22 {
			f.HasMapHash = true
			f.MapHash = uint32(nums[n-1])
			nums = nums[:n-1]
		}
		if len(nums) != 7 && len(nums) != 9 && len(nums) != 18 && len(nums) != 21 {
			t.Fatalf("frames.csv 的欄數 %d 不認得：%q", len(cols), ln)
		}
		f.I, f.Scycle = nums[0], nums[1]
		f.Valves = [3]int{nums[2], nums[3], nums[4]}
		f.Draws, f.State = nums[5], uint32(nums[6])
		switch len(nums) {
		case 9:
			f.HasFStat = true
			f.FStat = [2]int{nums[7], nums[8]}
		case 18:
			f.HasFStat = true
			f.FStat = [2]int{nums[7], nums[8]}
			f.HasSprDraws = true
			copy(f.SprDraws[:], nums[9:18])
		case 21:
			f.HasProb = true
			f.CityScore, f.CityYes, f.CityNo = nums[7], nums[8], nums[9]
			copy(f.ProbTable[:], nums[10:17])
			f.HasVote = true
			copy(f.Vote[:], nums[17:21])
		}
		m.Frames = append(m.Frames, f)
	}
	return m
}

// newFrameParityWorld 重建逐 frame 實驗的起始狀態。
// 順序與 newTickParityWorld 相同（先裸地形跑 DoSimInit，再蓋上建物）。
func newFrameParityWorld(t *testing.T, m frameMeta) *World {
	t.Helper()
	start := loadGoldenMap(t, "testdata/frame/cp0.csv")
	w := newTickParityWorld(start, 0, m.Init.Funds, m.Init.Scycle, true)
	w.Fcycle = m.Init.Fcycle
	w.Rand.SetState(mustRec(m.Init.Rands))
	return w
}

// loadFrameEndMap 讀最後一個 frame 之後的地圖（存成相對 cp0 的差異）。
func loadFrameEndMap(t *testing.T) [WorldX][WorldY]uint16 {
	return loadFrameEndMapIn(t, "testdata/frame")
}

func loadFrameEndMapIn(t *testing.T, dir string) [WorldX][WorldY]uint16 {
	t.Helper()
	m := loadGoldenMap(t, dir+"/cp0.csv")
	b, err := os.ReadFile(dir + "/cpend.diff")
	if err != nil {
		t.Fatal(err)
	}
	for _, ln := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if ln == "" {
			continue
		}
		var x, y, v int
		if _, err := fmt.Sscanf(ln, "%d,%d,%d", &x, &y, &v); err != nil {
			t.Fatalf("cpend.diff 這行讀不了：%q", ln)
		}
		m[x][y] = uint16(v)
	}
	return m
}

func TestFrameParity(t *testing.T) {
	m := loadFrameMeta(t)
	w := newFrameParityWorld(t, m)
	wantEnd := loadFrameEndMap(t)

	// ⚠ 起始狀態就是 R0 那四次讀完之後的值，**不要再往前推四次**。
	// 推了的話抽樣次數還是對得上（差是常數 4），但每個 frame 吃到的亂數值
	// 整整晚四個，分支結果會在某個看起來無關的地方才爆掉——那個坑花了
	// 一整輪才找到（`docs/re/12-tick-parity.md` §6之四）。
	matched, total := 0, 0
	for _, f := range m.Frames {
		before := w.Rand.State()
		w.Frame()
		got := drawsBetween(before, w.Rand.State())
		if got != f.Draws || w.Scycle != f.Scycle || !valvesEqual(w, f) {
			t.Logf("第 %d 個 frame 分岔：抽樣 %d 次（原版 %d 次）、Scycle %d（原版 %d）、"+
				"閥門 %d/%d/%d（原版 %d/%d/%d）",
				f.I, got, f.Draws, w.Scycle, f.Scycle,
				w.RValve, w.CValve, w.IValve, f.Valves[0], f.Valves[1], f.Valves[2])
			break
		}
		// 原版每個 frame 之後自己抽了四次（sim Rand ×4）。
		w.Rand.SetState(advanceRand(w.Rand.State(), 4))
		// 地圖雜湊放最後：抽樣次數對上了還差，就是「不抽亂數的那條路徑」偏掉。
		if f.HasMapHash && mapHash(w) != f.MapHash {
			t.Logf("第 %d 個 frame 的**地圖**對不上：我們 %d、原版 %d（抽樣次數是對的）",
				f.I, mapHash(w), f.MapHash)
			break
		}
		matched++
		total += f.Draws
	}

	t.Logf("逐 frame 完全一致 %d/%d 個（共 %d 次抽樣）", matched, len(m.Frames), total)
	if matched == len(m.Frames) {
		if d := mapDiffM(&w.Map, &wantEnd); d != 0 {
			t.Errorf("抽樣全部對上，但終點地圖差 %d 格", d)
		}
		if w.TotalFunds != m.End.Funds {
			t.Errorf("終點資金 %d，原版 %d", w.TotalFunds, m.End.Funds)
		}
	}
	if matched < frameParityBudget {
		t.Errorf("只對上 %d 個 frame，低於現況 %d —— 有東西退步了", matched, frameParityBudget)
	}
	if matched > frameParityBudget {
		t.Errorf("對上 %d 個 frame，比現況 %d 好 —— 請把 frameParityBudget 調到 %d",
			matched, frameParityBudget, matched)
	}
}

// TestFrameParityTrace 印出分岔那個 frame 是誰在抽樣。只在設了
// SIMCITY_TRACE 時跑。
func TestFrameParityTrace(t *testing.T) {
	if os.Getenv("SIMCITY_TRACE") == "" {
		t.Skip("設 SIMCITY_TRACE=1 才跑（只印診斷）")
	}
	m := loadFrameMeta(t)
	w := newFrameParityWorld(t, m)

	for _, f := range m.Frames {
		before := w.Rand.State()
		sites := map[string]int{}
		w.Rand.Watch = func() { sites[callerSite()]++ }
		ph, sc := (w.Fcycle+1)&15, w.Scycle
		w.Frame()
		w.Rand.Watch = nil
		got := drawsBetween(before, w.Rand.State())
		want := f.Draws
		if got == want && w.Scycle == f.Scycle && valvesEqual(w, f) {
			w.Rand.SetState(advanceRand(w.Rand.State(), 4))
			continue
		}
		t.Logf("第 %d 個 frame 分岔（相位 %d、Scycle %d）：我們抽 %d 次、原版 %d 次；"+
			"閥門 %d/%d/%d（原版 %d/%d/%d）",
			f.I, ph, sc, got, want,
			w.RValve, w.CValve, w.IValve, f.Valves[0], f.Valves[1], f.Valves[2])
		for k, v := range sites {
			t.Logf("  %s ×%d", k, v)
		}
		return
	}
	t.Logf("八百個 frame 全部一致")
}

// valvesEqual 比對三個需求閥門。它們決定分區長不長，是分岔時第一個
// 要看的量——抽樣次數不同通常是因為某個分區的 zscore 不同。
func valvesEqual(w *World, f frameCP) bool {
	return w.RValve == f.Valves[0] && w.CValve == f.Valves[1] &&
		w.IValve == f.Valves[2]
}
