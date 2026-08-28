package sim

import "testing"

// 掃描對拍。
//
// 這四個掃描互相回饋（PTLScan 的地價吃上一輪的汙染與犯罪，CrimeScan 吃這一輪的
// 地價），所以逐格的中間值沒辦法單獨驗。可驗的是**收斂後的三個平均值**，
// 它們各自由 3000 個半解析度格子算出來，鑑別力夠。
//
// 實驗地圖刻意**完全不含 ZONEBIT 分區**：這樣 PopDenScan 的重心會退化成
// 固定值（HWLDX, HWLDY），警察局與消防局涵蓋圖全零，其餘輸入就完全由地圖決定。
// 用的圖塊（650 工業美術、700 港區、電線）都不會被 MapScan 改動——
// 實驗跑完倒回來的地圖與寫進去的完全一樣，證明這一點。
//
// oracle 實測（tools/oracle/tcl/scan-experiment.tcl，跑兩次結果相同）：
// LandValue 25、Pollution 144、Crime 102。
func TestScansMatchOracleAverages(t *testing.T) {
	w := NewWorld(1)
	w.Map = loadGoldenMap(t, "testdata/scan-experiment.csv")

	// 跑到固定點。原版在跑的時候這四個掃描以不同的週期反覆執行，
	// 地圖不變時會收斂；這裡照 Simulate 的相位順序重複到穩定。
	var lastLV, lastPol, lastCrime int
	stable := 0
	for i := 0; i < 200; i++ {
		w.PTLScan()
		w.CrimeScan()
		w.PopDenScan()
		w.FireAnalysis()
		if w.LVAverage == lastLV && w.PolluteAverage == lastPol && w.CrimeAverage == lastCrime {
			stable++
			if stable >= 5 {
				break
			}
		} else {
			stable = 0
		}
		lastLV, lastPol, lastCrime = w.LVAverage, w.PolluteAverage, w.CrimeAverage
	}
	if stable < 5 {
		t.Fatalf("200 輪還沒收斂：LV=%d 汙染=%d 犯罪=%d",
			w.LVAverage, w.PolluteAverage, w.CrimeAverage)
	}

	if w.LVAverage != 25 {
		t.Errorf("地價平均 = %d，原版 25", w.LVAverage)
	}
	if w.PolluteAverage != 144 {
		t.Errorf("汙染平均 = %d，原版 144", w.PolluteAverage)
	}
	if w.CrimeAverage != 102 {
		t.Errorf("犯罪平均 = %d，原版 102", w.CrimeAverage)
	}
}

// 沒有分區時重心退化成 (HWLDX, HWLDY)。s_scan.c:130
//
// ⚠ 原版在這裡把**半解析度**的 HWLDX/HWLDY 指派給**全解析度**的 CCx/CCy。
// 看起來像單位錯了，但那是原版行為，照抄。
func TestCityCenterFallsBackToHalfResConstants(t *testing.T) {
	w := NewWorld(1)
	w.PopDenScan()
	if w.CCx != HWldX || w.CCy != HWldY {
		t.Errorf("重心 = (%d,%d)，原版無分區時是 (HWLDX,HWLDY) = (%d,%d)",
			w.CCx, w.CCy, HWldX, HWldY)
	}
	if w.CCx2 != HWldX>>1 || w.CCy2 != HWldY>>1 {
		t.Errorf("半解析度重心 = (%d,%d)", w.CCx2, w.CCy2)
	}
}

// 汙染貢獻表。s_scan.c:257 GetPValue
func TestGetPValueTable(t *testing.T) {
	cases := []struct {
		loc  int
		want int
		why  string
	}{
		{DIRT, 0, "空地"},
		{RIVER, 0, "水"},
		{WOODS, 0, "樹"},
		{RADTILE, 255, "輻射（原始碼註解問為什麼不是負的，照抄 255）"},
		{FIREBASE + 1, 90, "火"},
		{ROADS, 0, "普通道路"},
		{LTRFBASE, 50, "稀疏車流"},
		{HTRFBASE, 75, "壅塞車流"},
		{POWERBASE, 0, "電線"},
		{LASTIND, 0, "工業區上界之內"},
		{LASTIND + 1, 50, "工業"},
		{PORTBASE, 100, "海港"},
		{POWERPLANT, 100, "電廠"},
		{LASTPOWERPLANT + 1, 0, "電廠之後"},
	}
	for _, c := range cases {
		if got := getPValue(c.loc); got != c.want {
			t.Errorf("getPValue(%d) = %d，應為 %d（%s）", c.loc, got, c.want, c.why)
		}
	}
}

// 分區人口公式。s_zone.c:428-455
func TestZonePopulationFormulas(t *testing.T) {
	if got := rzPop(RZB); got != 16 {
		t.Errorf("rzPop(RZB) = %d，應為 16", got)
	}
	if got := czPop(COMCLR); got != 0 {
		t.Errorf("czPop(COMCLR) = %d，空商業區應為 0", got)
	}
	if got := izPop(INDCLR); got != 0 {
		t.Errorf("izPop(INDCLR) = %d，空工業區應為 0", got)
	}
	if got := czPop(CZB); got != 1 {
		t.Errorf("czPop(CZB) = %d，應為 1", got)
	}
	if got := izPop(IZB); got != 1 {
		t.Errorf("izPop(IZB) = %d，應為 1", got)
	}
}

// 兩個平滑核不一樣，不可互換。
//
//	doSmooth：  (四鄰居和 + 自己) >> 2
//	smoothFSMap：((四鄰居和 >> 2) + 自己) >> 1
func TestSmoothKernelsDiffer(t *testing.T) {
	w := NewWorld(1)
	// 半解析度：中央一點 100，四周 0。
	w.Tem[10][10] = 100
	w.doSmooth()
	// (0+0+0+0+100)>>2 = 25
	if w.Tem2[10][10] != 25 {
		t.Errorf("doSmooth 中央 = %d，(0+100)>>2 應為 25", w.Tem2[10][10])
	}
	// 鄰居：(100 + 0)>>2 = 25
	if w.Tem2[9][10] != 25 {
		t.Errorf("doSmooth 鄰居 = %d，應為 25", w.Tem2[9][10])
	}

	w2 := NewWorld(1)
	w2.FireStMap[5][5] = 100
	w2.smoothFSMap()
	// ((0>>2) + 100)>>1 = 50
	if w2.FireStMap[5][5] != 50 {
		t.Errorf("smoothFSMap 中央 = %d，((0>>2)+100)>>1 應為 50", w2.FireStMap[5][5])
	}
	// 鄰居：((100>>2) + 0)>>1 = 12
	if w2.FireStMap[4][5] != 12 {
		t.Errorf("smoothFSMap 鄰居 = %d，((100>>2)+0)>>1 應為 12", w2.FireStMap[4][5])
	}
}
