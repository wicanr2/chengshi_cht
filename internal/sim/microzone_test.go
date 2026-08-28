package sim

import "testing"

// 單一空住宅區的逐次元對拍。這是最小的、能走完
// doResidential → Rand(35) → evalRes → growShrink → doResOut 的配置：
// 空地 ＋ 一座沒有電、沒有路的 3×3 住宅區。
//
// 實驗腳本 tools/oracle/tcl/micro-zone2.tcl；證據見
// docs/re/12-tick-parity.md §5。

func TestMicroZoneExactParity(t *testing.T) {
	s0, ok := RecoverState([]int{33364, 62653, 20487, 60478})
	if !ok {
		t.Fatal("反推失敗")
	}
	s1 := rewindRand(mustRec([]int{8652, 12643, 29191, 27260}), 4)

	za := loadGoldenMap(t, "testdata/micro-zone-za.csv")
	zb := loadGoldenMap(t, "testdata/micro-zone-zb.csv")
	if za != zb {
		t.Log("注意：起始與結束地圖不同")
	}

	for startPhase := 0; startPhase < 16; startPhase++ {
		w := NewWorld(0)
		w.Map = za
		w.CityTime = (1922 - 1900) * 48
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
		w.Rand.SetState(s0)
		for n := 0; n <= 20000; n++ {
			if w.Rand.State() == s1 {
				t.Logf("★ 起始相位 %d：第 %d 個 frame 對上亂數狀態（%.1f 刻），地圖差 %d 格",
					startPhase, n, float64(n)/16, mapDiffM(&w.Map, &zb))
				return
			}
			w.Frame()
		}
	}
	t.Fatal("16 個相位都對不上亂數狀態")
}

