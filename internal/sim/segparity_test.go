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
// 起始的 Fcycle 與 CityTime%48 不可觀測（原版沒有對應的讀取指令），
// 所以每段對 16×48 個候選各試一次。
//
// 現況：23 段裡有 9 段完全對上，包含好幾段含完整城市評估（單次約 700 次
// 抽樣）的段落。對不上的多半是十幾刻的短段——那種長度裡，重建不出來的
// 內部狀態（Scycle、閥門、交通密度、成長率記憶）還來不及收斂。
// 進度與方法記在 docs/re/12-tick-parity.md。
const segParityBudget = 9

func TestSegmentParity(t *testing.T) {
	if testing.Short() {
		t.Skip("分段對拍要跑 16×48 個候選 × 23 段，很慢")
	}
	meta := loadSegMeta(t)
	maps := loadSegMaps(t, len(meta))
	matched := 0
	for i := 1; i < len(meta); i++ {
		if meta[i].Draws == nil {
			continue
		}
		want := *meta[i].Draws
		mA, mB := maps[i-1], maps[i]
		s := recoverOrDie(t, meta[i-1].Rands)

		// 收斂只做一次。它與候選的相位／CityTime 無關，放在迴圈裡等於
		// 每個候選都多跑 3200 個 frame——那是整個搜尋六成的成本。
		settled := newTickParityWorld(mA, 0, meta[i-1].Funds, 0, false)
		for k := 0; k < 200*16; k++ {
			settled.Frame()
		}

		hit, bestPh, bestCT := false, -1, -1
		for ph := 0; ph < 16 && !hit; ph++ {
			for ct := 0; ct < 48 && !hit; ct++ {
				w := cloneWorld(settled)
				w.Map = mA
				w.TotalFunds = meta[i-1].Funds
				w.CityTime = (w.CityTime/48)*48 + ct
				w.Fcycle = ph
				w.Rand.SetState(advanceRand(s, 4))

				got := 0
				for n := 0; n < 4000 && got <= want; n++ {
					if got == want && mapDiffM(&w.Map, &mB) == 0 {
						hit, bestPh, bestCT = true, ph, ct
						break
					}
					b := w.Rand.State()
					w.Frame()
					got += drawsBetween(b, w.Rand.State())
				}
			}
		}
		if hit {
			matched++
			t.Logf("段 %2d（%4d 次抽樣）✓ 相位 %d、CityTime%%48=%d", i, want, bestPh, bestCT)
		} else {
			t.Logf("段 %2d（%4d 次抽樣）✗", i, want)
		}
	}
	t.Logf("逐次元對上 %d/%d 段", matched, len(meta)-1)
	if matched < segParityBudget {
		t.Errorf("只對上 %d 段，低於現況 %d —— 有東西退步了", matched, segParityBudget)
	}
	if matched > segParityBudget {
		t.Errorf("對上 %d 段，比現況 %d 好 —— 請把 segParityBudget 調到 %d",
			matched, segParityBudget, matched)
	}
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
