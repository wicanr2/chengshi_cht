package sim

import "testing"

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

		hit, bestPh, bestCT := false, -1, -1
		for ph := 0; ph < 16 && !hit; ph++ {
			for ct := 0; ct < 48 && !hit; ct++ {
				w := newTickParityWorld(mA, 0, meta[i-1].Funds, 0, false)
				// 地圖已經靜止，先跑 200 刻讓衍生陣列收斂，再歸位。
				for k := 0; k < 200*16; k++ {
					w.Frame()
				}
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
