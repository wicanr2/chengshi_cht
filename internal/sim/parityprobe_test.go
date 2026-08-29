// 逐次元對拍的探查工具。
//
// 這一組不是回歸測試，是**診斷**：微實驗對不上的時候，用它們一層一層
// 把分歧夾出來。資料由 tools/oracle/ 產生後放在 workplace/（不進版控），
// 檔案不在時全部跳過。
//
// 三層，由粗到細：
//
//	TestMicroIndSegments      逐檢查點追，找出第一個追不到的區間
//	TestMicroIndTrace         在那個區間裡逐次記下抽樣的呼叫點
//	TestMicroIndScycleSearch  把 Scycle 也納入搜尋
//
// 方法本身見 docs/re/12-tick-parity.md §5 與 §6。
package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"testing"
)


// 工業區微實驗的**分段**診斷。
//
// 一次量「960 刻差 33 次抽樣」只知道有錯，不知道錯在哪。把原版跑成
// 十一個檢查點（每個相隔三百多次抽樣），逐段追下去，第一個追不到的
// 區間就是分歧點所在——這是 docs/re/12-tick-parity.md §5 第 4 條
// 「縮到微實驗」的下一層：把時間也切細。
//
// 資料由 tools/oracle/tcl/micro-ind-steps.tcl 產生，放在 workplace/
// （不進版控），所以檔案不在時跳過。
type indCheckpoint struct {
	Tag   string   `json:"tag"`
	Year  int      `json:"year"`
	State uint32   `json:"state"`
	Map   []uint16 `json:"map"`

	// 這幾個是為了往下追「哪個純量先跑掉」加的觀測點。
	// LandValue／Crime 是平均值，CrimeMaxX／Y 是平手擲骰的結果——
	// 犯罪值全是 0 的時候每一格都平手，那個座標等於把擲骰結果攤在外面看。
	LandValue int `json:"landValue"`
	Crime     int `json:"crime"`
	CrimeMaxX int `json:"crimeMaxX"`
	CrimeMaxY int `json:"crimeMaxY"`
}

func TestMicroIndSegments(t *testing.T) {
	raw, err := os.ReadFile("../../workplace/oracle/micro-ind-cps.json")
	if err != nil {
		t.Skip("沒有檢查點資料，跳過（跑 tools/oracle/drive.sh micro-ind-steps.tcl 產生）")
	}
	var cps []indCheckpoint
	if err := json.Unmarshal(raw, &cps); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d 個檢查點，%d–%d 年", len(cps), cps[0].Year, cps[len(cps)-1].Year)

	toMap := func(v []uint16) [WorldX][WorldY]uint16 {
		var m [WorldX][WorldY]uint16
		for y := 0; y < WorldY; y++ {
			for x := 0; x < WorldX; x++ {
				m[x][y] = v[y*WorldX+x]
			}
		}
		return m
	}

	for startPhase := 0; startPhase < 16; startPhase++ {
		w := NewWorld(0)
		w.Map = toMap(cps[0].Map)
		w.CityTime = (cps[0].Year - 1900) * 48
		w.CityTax = 7
		w.SimSpeed = 3
		w.NoDisasters = true
		w.NewPower = true
		w.RValve, w.CValve, w.IValve = 2000, -1500, 1500
		w.EMarket = 6.0
		for i := range w.MoneyHis {
			w.MoneyHis[i] = 128
		}
		w.EvalInit()
		w.EnableSprites()
		w.Fcycle = startPhase
		w.Rand.SetState(cps[0].State)

		reached := 0
		for k := 1; k < len(cps); k++ {
			// 檢查點自己那四次取樣也在同一條數列上，所以要往回捲四次
			// 才是「原版停下來那一刻」的狀態。
			target := rewindRand(cps[k].State, 4)
			hit := false
			for n := 0; n < 40000; n++ {
				if w.Rand.State() == target {
					hit = true
					break
				}
				w.Frame()
			}
			if !hit {
				break
			}
			t.Logf("相位 %2d：追到 %s（%d 年）地價均 %d/%d 犯罪均 %d/%d CrimeMax (%d,%d)/(%d,%d)",
				startPhase, cps[k].Tag, cps[k].Year,
				w.LVAverage, cps[k].LandValue,
				w.CrimeAverage, cps[k].Crime,
				w.CrimeMaxX, w.CrimeMaxY, cps[k].CrimeMaxX, cps[k].CrimeMaxY)
			if len(cps[k].Map) > 0 {
				d := mapDiffM(&w.Map, ptrMap(toMap(cps[k].Map)))
				t.Logf("相位 %2d：追到 %s（%d 年），地圖差 %d 格", startPhase, cps[k].Tag, cps[k].Year, d)
			}
			w.Rand.SetState(cps[k].State) // 跳過那四次取樣
			reached = k
		}
		if reached > 0 {
			t.Logf("相位 %2d：追到第 %d 個檢查點就斷了（共 %d 個）", startPhase, reached, len(cps)-1)
			return
		}
	}
	t.Log("十六個相位都連第一個檢查點都追不到")
}

func ptrMap(m [WorldX][WorldY]uint16) *[WorldX][WorldY]uint16 { return &m }


// 逐次抽樣的呼叫點追蹤：分歧的那一刻，我們比原版多抽的是哪一個？
//
// 前一個診斷（TestMicroIndSegments）把分歧夾到兩個檢查點之間，
// 這一個再往下切一層：從最後一個追得到的檢查點出發，記下每一次抽樣
// 是誰呼叫的，看跨過原版那個位置時多出來的是哪一筆。
func TestMicroIndTrace(t *testing.T) {
	raw, err := os.ReadFile("../../workplace/oracle/micro-ind-cps.json")
	if err != nil {
		t.Skip("沒有檢查點資料，跳過")
	}
	var cps []indCheckpoint
	if err := json.Unmarshal(raw, &cps); err != nil {
		t.Fatal(err)
	}

	// 先找到最後一個追得到的檢查點，並把世界推到那裡。
	w, k := advanceToLastGood(t, cps)
	if k < 1 {
		t.Fatal("連第一個檢查點都追不到")
	}
	t.Logf("最後追得到的是第 %d 個檢查點（%d 年）", k, cps[k].Year)

	// 原版從這裡走到下一個檢查點用了幾次抽樣
	want := lcgDistance(cps[k].State, rewindRand(cps[k+1].State, 4))
	t.Logf("原版走到下一個檢查點用了 %d 次抽樣", want)

	type hit struct {
		n    int
		site string
	}
	var hits []hit
	w.Rand.Watch = func() {
		hits = append(hits, hit{len(hits) + 1, callerSite()})
	}
	for len(hits) < want+40 {
		w.Frame()
	}
	w.Rand.Watch = nil

	// 統計呼叫點
	count := map[string]int{}
	for _, h := range hits[:min(len(hits), want)] {
		count[h.site]++
	}
	var keys []string
	for kk := range count {
		keys = append(keys, kk)
	}
	sort.Slice(keys, func(i, j int) bool { return count[keys[i]] > count[keys[j]] })
	t.Log("前 " + fmt.Sprint(want) + " 次抽樣的呼叫點分布：")
	for _, kk := range keys {
		t.Logf("   %5d  %s", count[kk], kk)
	}
	t.Log("跨過原版終點附近的十次：")
	for i := max(0, want-5); i < min(len(hits), want+5); i++ {
		t.Logf("   #%d  %s", hits[i].n, hits[i].site)
	}

	// 我們的 frame 邊界落在第幾次抽樣？原版的停點一定是某個邊界。
	// ⚠ 要重新推一個世界到同一個檢查點，不能用 newIndMicroWorld(cps[k])
	// ——只有第 0 個檢查點帶地圖，其餘的 Map 是空的，那樣起出來的世界
	// 沒有工業區，抽樣次數會少一個數量級（症狀是「怎麼跑都到不了」）。
	w2, _ := advanceToLastGood(t, cps)
	var bounds []int
	cnt := 0
	w2.Rand.Watch = func() { cnt++ }
	for f := 0; f < 40000 && cnt < want+30; f++ {
		w2.Frame()
		bounds = append(bounds, cnt)
	}
	w2.Rand.Watch = nil
	var near []int
	for _, b := range bounds {
		if b >= want-12 && b <= want+12 {
			near = append(near, b)
		}
	}
	t.Logf("我們在 %d 附近的 frame 邊界：%v", want, near)
	for j := 1; j <= 3 && k+j < len(cps); j++ {
		d := lcgDistance(cps[k].State, rewindRand(cps[k+j].State, 4))
		t.Logf("原版到第 %d 個檢查點的距離：%d", k+j, d)
	}
}

// callerSite 回傳呼叫 Rand16 的那一層的位置。
//
// 0 = runtime.Callers、1 = callerSite、2 = Watch 的閉包、3 = Rand16、
// 4 = Rand／Rand16Signed／ERand 之類的包裝或直接呼叫者。
func callerSite() string {
	var pcs [8]uintptr
	n := runtime.Callers(4, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var last string
	for i := 0; i < 3; i++ {
		f, more := frames.Next()
		name := shortFn(f.Function)
		// 跳過 Rand 自己的包裝
		if name == "(*Rand).Rand" || name == "(*Rand).Rand16Signed" ||
			name == "(*Rand).ERand" || name == "(*Rand).Rand16" {
			last = name
			if !more {
				break
			}
			continue
		}
		return fmt.Sprintf("%s:%d %s", shortFile(f.File), f.Line, name)
	}
	return last
}

func shortFn(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			for j := i - 1; j >= 0; j-- {
				if s[j] == '/' || s[j] == '.' {
					return s[j+1:]
				}
			}
			return s
		}
	}
	return s
}

func shortFile(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return s[i+1:]
		}
	}
	return s
}

func lcgDistance(from, to uint32) int {
	r := NewRand(0)
	r.SetState(from)
	for i := 1; i <= 4_000_000; i++ {
		r.Rand16()
		if r.State() == to {
			return i
		}
	}
	return -1
}

func advanceToLastGood(t *testing.T, cps []indCheckpoint) (*World, int) {
	t.Helper()
	for startPhase := 0; startPhase < 16; startPhase++ {
		w := newIndMicroWorld(cps[0], startPhase)
		last := 0
		for k := 1; k < len(cps); k++ {
			target := rewindRand(cps[k].State, 4)
			hit := false
			for n := 0; n < 40000; n++ {
				if w.Rand.State() == target {
					hit = true
					break
				}
				w.Frame()
			}
			if !hit {
				break
			}
			w.Rand.SetState(cps[k].State)
			last = k
		}
		if last > 0 {
			return w, last
		}
	}
	return nil, 0
}

func newIndMicroWorld(c indCheckpoint, phase int) *World {
	w := NewWorld(0)
	if len(c.Map) > 0 {
		for y := 0; y < WorldY; y++ {
			for x := 0; x < WorldX; x++ {
				w.Map[x][y] = c.Map[y*WorldX+x]
			}
		}
	}
	w.CityTime = (c.Year - 1900) * 48
	w.CityTax = 7
	w.SimSpeed = 3
	w.NoDisasters = true
	w.NewPower = true
	w.RValve, w.CValve, w.IValve = 2000, -1500, 1500
	w.EMarket = 6.0
	for i := range w.MoneyHis {
		w.MoneyHis[i] = 128
	}
	w.EvalInit()
	w.EnableSprites()
	w.Fcycle = phase
	w.Rand.SetState(c.State)
	return w
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}


// 把 Scycle 也納入搜尋。
//
// 相位（Fcycle）決定這一刻跑哪一段，Scycle 決定「這一刻要不要跑
// PTLScan／CrimeScan／PopDenScan／FireAnalysis／DoPowerScan」——
// 它們各自是 Scycle % 17／18／19／20／5。**Scycle 從外面觀察不到**
// （Tcl 沒有這個存取子），所以微實驗只設相位是不夠的：抽樣總數可以
// 對得上，但那幾個掃描落在不同的刻，吃到的亂數值就不一樣。
//
// 症狀很容易被誤讀成「工業區的成長判斷寫錯了」：抽樣次數對、地圖也對，
// 只有 CrimeMaxX／Y（平手擲骰的結果）偶爾不一樣，然後在某個檢查點突然
// 追不下去。實際上是實驗的起始狀態少設了一個變數。
func TestMicroIndScycleSearch(t *testing.T) {
	raw, err := os.ReadFile("../../workplace/oracle/micro-ind-cps.json")
	if err != nil {
		t.Skip("沒有檢查點資料，跳過")
	}
	var cps []indCheckpoint
	if err := json.Unmarshal(raw, &cps); err != nil {
		t.Fatal(err)
	}

	bestReach, bestPhase, bestScycle := 0, -1, -1
	for phase := 0; phase < 16; phase++ {
		for scycle := 0; scycle < 1024; scycle++ {
			w := newIndMicroWorld(cps[0], phase)
			w.Scycle = scycle
			reach := 0
			for k := 1; k < len(cps); k++ {
				target := rewindRand(cps[k].State, 4)
				hit := false
				for n := 0; n < 6000; n++ {
					if w.Rand.State() == target {
						hit = true
						break
					}
					w.Frame()
				}
				if !hit {
					break
				}
				w.Rand.SetState(cps[k].State)
				reach = k
			}
			if reach > bestReach {
				bestReach, bestPhase, bestScycle = reach, phase, scycle
				t.Logf("相位 %2d、Scycle %4d → 追到第 %d 個檢查點", phase, scycle, reach)
				if reach == len(cps)-1 {
					t.Logf("★ 全部 %d 個檢查點都對上（%d–%d 年）",
						reach, cps[0].Year, cps[len(cps)-1].Year)
					return
				}
			}
		}
	}
	t.Errorf("最好只追到第 %d 個檢查點（共 %d 個），相位 %d、Scycle %d",
		bestReach, len(cps)-1, bestPhase, bestScycle)
}
