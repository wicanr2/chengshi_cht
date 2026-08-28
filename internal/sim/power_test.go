package sim

import "testing"

// 電力傳導對拍：拿 oracle 上收斂後的地圖，把 PWRBIT 全部清掉，
// 用 Go 版重算一次，結果必須逐格相同。
//
// 實驗地圖是在 oracle 上現搭的：一座燃煤電廠、一條 57 格橫向電線、
// 一條 34 格縱向分支、一段刻意不相連的 15 格電線。
// 取法見 tools/oracle/tcl/power-experiment.tcl。
func TestPowerScanMatchesOracle(t *testing.T) {
	golden := loadGoldenMap(t, "testdata/power-experiment.csv")

	w := NewWorld(1)
	w.Map = golden
	// 清掉全部 PWRBIT，讓 Go 版自己重算。
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			w.Map[x][y] &^= uint16(PWRBIT)
		}
	}

	res := w.DoPowerScan()

	diff, fx, fy := 0, -1, -1
	for y := 0; y < WorldY; y++ {
		for x := 0; x < WorldX; x++ {
			if w.Map[x][y] != golden[x][y] {
				if diff == 0 {
					fx, fy = x, y
				}
				diff++
			}
		}
	}
	if diff != 0 {
		t.Fatalf("有 %d 格不同。第一處 (%d,%d)：得到 %d（PWR=%v），原版 %d（PWR=%v）",
			diff, fx, fy,
			w.Map[fx][fy], w.Map[fx][fy]&PWRBIT != 0,
			golden[fx][fy], golden[fx][fy]&PWRBIT != 0)
	}
	t.Logf("燃煤 %d 座、核能 %d 座、通電 %d 格、供電上限未超過 = %v",
		res.CoalPop, res.NuclearPop, res.Powered, !res.OutOfPower)
}

// 孤立的電線不該通電——這是「泛洪真的有沿著連線走」的反面證據。
func TestIsolatedWireHasNoPower(t *testing.T) {
	golden := loadGoldenMap(t, "testdata/power-experiment.csv")
	w := NewWorld(1)
	w.Map = golden
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			w.Map[x][y] &^= uint16(PWRBIT)
		}
	}
	w.DoPowerScan()

	// 實驗裡孤立的那一段在 y=60、x=80..94。
	for x := 80; x < 95; x++ {
		if w.Map[x][60]&CONDBIT == 0 {
			t.Fatalf("(%d,60) 不是電線，測試資料變了", x)
		}
		if w.Map[x][60]&PWRBIT != 0 {
			t.Errorf("(%d,60) 是孤立電線，不該通電", x)
		}
	}
}

// 供電上限：一座燃煤電廠只能供 700 格。超過就中止整個掃描，
// **剩下的電線一格都不通**（s_power.c:200 的 return，不是 break）。
func TestPowerCapacityAbortsWholeScan(t *testing.T) {
	w := NewWorld(1)
	// 一座電廠 + 一條長到超過 700 格的蛇形電線。
	const wire = uint16(HPOWER) | CONDBIT | BURNBIT | BULLBIT
	w.Map[5][5] = uint16(POWERPLANT) | BNCNBIT | ZONEBIT
	n := 0
	for y := 5; y < 100 && n < 1500; y++ {
		for x := 6; x < 120 && n < 1500; x++ {
			w.Map[x][y] = wire
			n++
		}
	}
	res := w.DoPowerScan()
	if !res.OutOfPower {
		t.Fatalf("鋪了 %d 格電線，一座燃煤只有 %d 格容量，應該要超過", n, CoalPowerCapacity)
	}
	if res.Powered > CoalPowerCapacity+16 {
		t.Errorf("通電 %d 格，超過容量 %d 太多", res.Powered, CoalPowerCapacity)
	}
}

// 沒有電廠就沒有電。
func TestNoPlantNoPower(t *testing.T) {
	w := NewWorld(1)
	const wire = uint16(HPOWER) | CONDBIT | BURNBIT | BULLBIT
	for x := 10; x < 50; x++ {
		w.Map[x][20] = wire
	}
	res := w.DoPowerScan()
	if res.Powered != 0 {
		t.Errorf("沒有電廠卻有 %d 格通電", res.Powered)
	}
}

// 電廠本身永遠算有電，即使 PowerMap 那一位是 0。s_zone.c:639
func TestPlantIsAlwaysPowered(t *testing.T) {
	w := NewWorld(1)
	w.Map[60][60] = uint16(NUCLEAR) | BNCNBIT | ZONEBIT
	w.DoPowerScan()
	if w.Map[60][60]&PWRBIT == 0 {
		t.Error("核電廠本身應該永遠帶 PWRBIT")
	}
}

// 端到端：劇本 1 載入後，用 Go 版的電力傳導重算 PWRBIT，
// 應該要對得上 oracle 載入後的狀態（扣掉會逐刻換幀的動畫格）。
//
// 這一條把 docs/formats/01-city-file.md §3.3 留下的 266 格 PWRBIT 差異收掉。
func TestScenario1PowerMatchesOracle(t *testing.T) {
	res := micropolisRes(t)
	raw, err := readFileOrSkip(t, res, "snro.111")
	if err != nil {
		return
	}
	cf, err := ParseCityFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	golden := loadGoldenMap(t, "testdata/scenario1-dullsville.csv")

	w := NewWorld(1)
	w.LoadScenarioFile(cf, ScenarioDullsville)
	scan := w.DoPowerScan()

	var unexplained, animated int
	fx, fy := -1, -1
	for y := 0; y < WorldY; y++ {
		for x := 0; x < WorldX; x++ {
			got, exp := w.Map[x][y], golden[x][y]
			if got == exp {
				continue
			}
			if got&ANIMBIT != 0 || exp&ANIMBIT != 0 {
				animated++
				continue
			}
			if unexplained == 0 {
				fx, fy = x, y
			}
			unexplained++
		}
	}
	if unexplained != 0 {
		t.Fatalf("有 %d 格差異無法解釋。第一處 (%d,%d)：得到 %d（PWR=%v），原版 %d（PWR=%v）",
			unexplained, fx, fy,
			w.Map[fx][fy], w.Map[fx][fy]&PWRBIT != 0,
			golden[fx][fy], golden[fx][fy]&PWRBIT != 0)
	}
	t.Logf("電力重算後只剩動畫格 %d 格差異；燃煤 %d 座、核能 %d 座、通電 %d 格",
		animated, scan.CoalPop, scan.NuclearPop, scan.Powered)
}
