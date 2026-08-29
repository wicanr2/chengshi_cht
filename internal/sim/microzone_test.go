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

// 單一空商業區與單一空工業區的逐次元對拍。
//
// 住宅區那條路徑（doResidential）已經逐次元一致，但商業與工業各自有
// 自己的成長判斷（doCommercial 的 TrfGood 短路、doIndustrial 的
// EMarket），抽樣次數不一樣。三種分區各驗一次，才知道差異在哪一條。
//
// 實驗腳本 tools/oracle/tcl/micro-com.tcl 與 micro-ind.tcl，
// 版面與 micro-zone2.tcl 相同，只換了那九格的圖塊編號。
func TestMicroCommercialExactParity(t *testing.T) {
	runMicroZone(t, "com",
		[]int{59848, 15591, 6617, 20936},
		[]int{18584, 60667, 35844, 61981},
		1917)
}

func TestMicroIndustrialExactParity(t *testing.T) {
	runMicroZone(t, "ind",
		[]int{10521, 53672, 21440, 18600},
		[]int{22860, 14282, 24365, 33735},
		1921)
}

// runMicroZone 是三個微實驗共用的骨架。
//
// s0 是原版停在起始點時抽的四個數（狀態就在那四個數之後），
// s1 是停在結束點時抽的四個數（要往回捲四次才是停下來那一刻的狀態）。
//
// **相位與 Scycle 兩個都要搜。** 相位（Fcycle）決定這一刻跑十六段裡的
// 哪一段；Scycle 決定「這一刻要不要跑 PTLScan／CrimeScan／PopDenScan／
// FireAnalysis／DoPowerScan」（各自是 Scycle % 17／18／19／20／5）。
// 兩個都從外面觀察不到，Tcl 也沒有存取子。
//
// 只搜相位而把 Scycle 當 0 的話，工業區這個實驗會在跑了三十幾年之後
// 突然對不上——而且症狀很像實作錯誤：抽樣總數對、地圖也對，只有平手
// 擲骰的結果（CrimeMaxX／Y）偶爾不一樣。實際上是那幾個掃描落在不同的
// 刻，吃到的亂數值不同。見 docs/re/12-tick-parity.md §6。
func runMicroZone(t *testing.T, name string, first, last []int, year int) {
	t.Helper()
	s0, ok := RecoverState(first)
	if !ok {
		t.Fatal("反推失敗")
	}
	s1 := rewindRand(mustRec(last), 4)

	za := loadGoldenMap(t, "testdata/micro-"+name+"-za.csv")
	zb := loadGoldenMap(t, "testdata/micro-"+name+"-zb.csv")

	for startPhase := 0; startPhase < 16; startPhase++ {
		for scycle := 0; scycle < 1024; scycle++ {
			w := NewWorld(0)
			w.Map = za
			w.CityTime = (year - 1900) * 48
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
			w.Scycle = scycle
			w.Rand.SetState(s0)
			for n := 0; n <= 20000; n++ {
				if w.Rand.State() == s1 {
					d := mapDiffM(&w.Map, &zb)
					t.Logf("★ 相位 %d、Scycle %d：第 %d 個 frame 對上亂數狀態（%.1f 刻），地圖差 %d 格",
						startPhase, scycle, n, float64(n)/16, d)
					if d != 0 {
						t.Errorf("亂數對上了但地圖差 %d 格", d)
					}
					return
				}
				w.Frame()
			}
		}
	}
	t.Fatalf("十六個相位 × 1024 個 Scycle 都對不上亂數狀態（%s）", name)
}
