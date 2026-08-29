package sim

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// 分段對拍。
//
// 整段對拍（tickparity_test.go）用「地圖差幾格」當指標，太粗：一次分歧
// 之後的差異全部混在一起，看不出是哪裡壞的。這裡改成把同一次原版執行
// 切成 23 段短脈衝（tools/oracle/tcl/tick-parity-seg.tcl），每段只跑
// 十幾刻，並在每個接縫記下亂數狀態與地圖。
//
// 判準是**抽樣次數**：LCG 的狀態由起點與步數唯一決定，所以「亂數狀態
// 對上」等價於「這一段消耗的抽樣次數和原版一模一樣」。再加上地圖零差異，
// 就是這一段逐次元等價。
//
// 起始的 Fcycle、CityTime%48 與 Scycle 三個都不可觀測，要用搜的
// （TestSegmentParityDeep）。搜到的解記在 segSolutions，日常測試只驗不搜。
// 進度與方法記在 docs/re/12-tick-parity.md。
const segParityBudget = 10

// segChainBudget 是接力式對拍（TestSegmentParityChain）目前連續對上的段數。
const segChainBudget = 1

// segSolution 是一段對拍的起始狀態解。
//
// 三個值都**不可觀測**：Fcycle 決定這一刻跑十六段裡的哪一段，
// CityTime%48 決定年內位置，Scycle 決定哪一刻跑那五個週期性掃描
// （PTLScan／CrimeScan／PopDenScan／FireAnalysis／DoPowerScan，
// 各自是 Scycle % 17／18／19／20／5）。原版沒有讀取它們的指令。
type segSolution struct{ Ph, CT, Scycle int }

// segSolutions 是 TestSegmentParityDeep 搜出來的解。
//
// 把搜尋的結果記下來，日常測試就從「搜」變成「驗」——快得多，而且
// 判準更強：解不再驗得過，代表規則層真的動到了，不是搜尋範圍不夠。
// 要重新搜（改過規則層之後）跑：
//
//	SIMCITY_DEEP=1 SIMCITY_SEGS=<段號> tools/go.sh test ./internal/sim/ \
//	    -run SegmentParityDeep -v -timeout 90m
var segSolutions = map[int]segSolution{
	1:  {0, 34, 200},
	3:  {0, 31, 200},
	10: {0, 45, 200},
	11: {0, 1, 200},
	13: {0, 1, 200},
	14: {2, 1, 1020},
	16: {2, 40, 200},
	20: {0, 22, 200},
	21: {0, 36, 200},
	23: {0, 17, 200},
}

func TestSegmentParity(t *testing.T) {
	meta := loadSegMeta(t)
	maps := loadSegMaps(t, len(meta))
	matched, total := 0, 0
	for i := 1; i < len(meta); i++ {
		if meta[i].Draws == nil {
			continue
		}
		total++
		sol, known := segSolutions[i]
		if !known {
			t.Logf("段 %2d（%4d 次抽樣）— 還沒搜到解", i, *meta[i].Draws)
			continue
		}
		if verifySegment(t, meta, maps, i, sol) {
			matched++
			t.Logf("段 %2d（%4d 次抽樣）✓ 相位 %d、CityTime%%48=%d、Scycle %d",
				i, *meta[i].Draws, sol.Ph, sol.CT, sol.Scycle)
		} else {
			t.Errorf("段 %2d 記著的解驗不過了 —— 規則層被動到了，或者解該重搜", i)
		}
	}
	t.Logf("逐次元對上 %d/%d 段", matched, total)
	if matched < segParityBudget {
		t.Errorf("只對上 %d 段，低於現況 %d —— 有東西退步了", matched, segParityBudget)
	}
	if matched > segParityBudget {
		t.Errorf("對上 %d 段，比現況 %d 好 —— 請把 segParityBudget 調到 %d",
			matched, segParityBudget, matched)
	}
}

// verifySegment 用一個已知的解跑一段，回報是否逐次元一致。
func verifySegment(t *testing.T, meta []segCP, maps [][WorldX][WorldY]uint16,
	i int, sol segSolution) bool {
	t.Helper()
	want := *meta[i].Draws
	mA, mB := maps[i-1], maps[i]
	s := recoverOrDie(t, meta[i-1].Rands)

	w := newTickParityWorld(mA, 0, meta[i-1].Funds, 0, false)
	for k := 0; k < 200*16; k++ {
		w.Frame()
	}
	w.Map = mA
	w.TotalFunds = meta[i-1].Funds
	w.CityTime = (w.CityTime/48)*48 + sol.CT
	w.Fcycle = sol.Ph
	w.Scycle = sol.Scycle
	w.Rand.SetState(advanceRand(s, 4))

	got := 0
	for n := 0; n < 4000 && got <= want; n++ {
		if got == want && mapDiffM(&w.Map, &mB) == 0 {
			return true
		}
		b := w.Rand.State()
		w.Frame()
		got += drawsBetween(b, w.Rand.State())
	}
	return false
}

// cloneWorld 複製一份世界。
//
// World 全是值型別（陣列、純量）＋ 一個 *Rand 與一個介面。介面在對拍
// 用的世界裡是零大小的 noSprites，複製安全；*Rand 要另外配一份，
// 否則所有候選會共用同一個亂數狀態——症狀是搜尋結果隨迴圈順序改變。
func cloneWorld(src *World) *World {
	w := *src
	r := *src.Rand
	w.Rand = &r
	return &w
}

// TestSegmentParityDeep 把 Scycle 也納入搜尋。
//
// 微實驗那邊已經證實 Scycle 設錯會讓對拍在完全無關的地方失敗
// （docs/re/12-tick-parity.md §6）。分段對拍目前只搜相位與
// CityTime%48（16×48＝768 個候選），沒搜 Scycle。
//
// 成本：加上 Scycle 是 768×1024 ≈ 79 萬個候選，一段約八分半。
// 所以預設跳過，要跑就設 SIMCITY_DEEP=1，並用 -run 指定段號範圍：
//
//	SIMCITY_DEEP=1 SIMCITY_SEGS=3,5 tools/go.sh test ./internal/sim/ \
//	    -run SegmentParityDeep -v -timeout 60m
func TestSegmentParityDeep(t *testing.T) {
	if os.Getenv("SIMCITY_DEEP") == "" {
		t.Skip("預設跳過；設 SIMCITY_DEEP=1 才跑（一段約八分半）")
	}
	want := map[int]bool{}
	for _, f := range strings.Split(os.Getenv("SIMCITY_SEGS"), ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(f)); err == nil {
			want[n] = true
		}
	}
	meta := loadSegMeta(t)
	maps := loadSegMaps(t, len(meta))
	for i := 1; i < len(meta); i++ {
		if meta[i].Draws == nil || (len(want) > 0 && !want[i]) {
			continue
		}
		target := *meta[i].Draws
		mA, mB := maps[i-1], maps[i]
		s := recoverOrDie(t, meta[i-1].Rands)

		settled := newTickParityWorld(mA, 0, meta[i-1].Funds, 0, false)
		for k := 0; k < 200*16; k++ {
			settled.Frame()
		}

		hit := false
		for sc := 0; sc < 1024 && !hit; sc++ {
			for ph := 0; ph < 16 && !hit; ph++ {
				for ct := 0; ct < 48 && !hit; ct++ {
					w := cloneWorld(settled)
					w.Map = mA
					w.TotalFunds = meta[i-1].Funds
					w.CityTime = (w.CityTime/48)*48 + ct
					w.Fcycle = ph
					w.Scycle = sc
					w.Rand.SetState(advanceRand(s, 4))

					got := 0
					for n := 0; n < 4000 && got <= target; n++ {
						if got == target && mapDiffM(&w.Map, &mB) == 0 {
							t.Logf("段 %2d（%4d 次抽樣）✓ 相位 %d、CityTime%%48=%d、Scycle %d",
								i, target, ph, ct, sc)
							hit = true
							break
						}
						b := w.Rand.State()
						w.Frame()
						got += drawsBetween(b, w.Rand.State())
					}
				}
			}
		}
		if !hit {
			t.Logf("段 %2d（%4d 次抽樣）✗ 連 Scycle 都搜過了還是對不上", i, target)
		}
	}
}

// TestSegmentParityChain 接力式對拍。
//
// 分段對拍每一段都用「200 刻收斂」重建內部狀態，那是虛構的：真正的
// 閥門、成長記憶、交通密度是從上一段延續下來的，不是從那一段的起始
// 地圖重新算出來的。而 tools/oracle/tcl/tick-parity-seg.tcl 顯示
// **段與段之間沒有玩家動作**（只有 Speed 3 → Speed 0 → 讀狀態），
// 所以合法的做法是接力：只在第一段搜起點，之後一路跑下去，中途不重設
// 任何東西（地圖也不重設）。
//
// 這是比逐段獨立搜更強的判準——它證明的是「連續 N 段逐次元一致」，
// 而不是「N 段各自在某個湊出來的起點下一致」。
func TestSegmentParityChain(t *testing.T) {
	meta := loadSegMeta(t)
	maps := loadSegMaps(t, len(meta))
	sol, ok := segSolutions[1]
	if !ok {
		t.Skip("第 1 段還沒有解，接力無從起頭")
	}
	s0 := recoverOrDie(t, meta[0].Rands)

	settled := newTickParityWorld(maps[0], 0, meta[0].Funds, 0, false)
	for k := 0; k < 200*16; k++ {
		settled.Frame()
	}

	// 第 1 段可能不只一個解（Scycle 觀察不到），能接得最長的那個才是對的。
	best, bestSc := 0, -1
	for sc := 0; sc < 1024; sc++ {
		w := cloneWorld(settled)
		w.Map = maps[0]
		w.TotalFunds = meta[0].Funds
		w.CityTime = (w.CityTime/48)*48 + sol.CT
		w.Fcycle = sol.Ph
		w.Scycle = sc
		w.Rand.SetState(advanceRand(s0, 4))
		if n := runChain(w, meta, maps); n > best {
			best, bestSc = n, sc
		}
	}
	if bestSc >= 0 {
		t.Logf("接力連續對上 %d 段（相位 %d、CityTime%%48=%d、Scycle %d）",
			best, sol.Ph, sol.CT, bestSc)
	} else {
		t.Logf("接力一段都沒對上")
	}
	if best < segChainBudget {
		t.Errorf("接力只對上 %d 段，低於現況 %d —— 有東西退步了", best, segChainBudget)
	}
	if best > segChainBudget {
		t.Errorf("接力對上 %d 段，比現況 %d 好 —— 請把 segChainBudget 調到 %d",
			best, segChainBudget, best)
	}
}

// runChain 從 w 一路跑完所有檢查點，回傳連續對上幾段。
// 中途不重設任何東西——地圖也不重設，這正是接力比逐段獨立搜強的地方。
func runChain(w *World, meta []segCP, maps [][WorldX][WorldY]uint16) int {
	chain := 0
	for i := 1; i < len(meta); i++ {
		if meta[i].Draws == nil {
			break
		}
		want := *meta[i].Draws
		got, hit := 0, false
		for n := 0; n < 20000 && got <= want; n++ {
			if got == want {
				hit = mapDiffM(&w.Map, &maps[i]) == 0
				break
			}
			b := w.Rand.State()
			w.Frame()
			got += drawsBetween(b, w.Rand.State())
		}
		if !hit {
			break
		}
		chain++
		// 原版在每個檢查點自己抽了四次（sim Rand ×4）。
		w.Rand.SetState(advanceRand(w.Rand.State(), 4))
	}
	return chain
}
