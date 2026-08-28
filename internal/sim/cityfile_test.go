package sim

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// micropolisRes 回傳封存的 res/ 目錄；沒有封存時跳過測試（使用者自備）。
func micropolisRes(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(thisFile), "..", "..", "workplace", "ref",
		"micropolis", "micropolis-activity", "res")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Skip("沒有 Micropolis 封存，跳過（使用者自備）")
	}
	return p
}

// 八個劇本與封存附的城市檔都要解得開，而且**逐位元組 round-trip 相同**。
//
// 這一條守著 CLAUDE.md §4 的「改寫不是重建」：未解的位元組要原樣寫回去。
func TestCityFileRoundTrip(t *testing.T) {
	res := micropolisRes(t)
	dirs := []string{res, filepath.Join(res, "..", "cities")}
	var files []string
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			n := e.Name()
			if len(n) > 5 && (n[:5] == "snro." || filepath.Ext(n) == ".cty") {
				files = append(files, filepath.Join(d, n))
			}
		}
	}
	if len(files) == 0 {
		t.Skip("封存裡沒有 snro.* 或 .cty")
	}

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("%s：讀不到 %v", filepath.Base(f), err)
			continue
		}
		if len(raw) != CityFileSize1x1 {
			t.Errorf("%s：大小 %d，應為 %d", filepath.Base(f), len(raw), CityFileSize1x1)
			continue
		}
		cf, err := ParseCityFile(raw)
		if err != nil {
			t.Errorf("%s：解析失敗 %v", filepath.Base(f), err)
			continue
		}
		out := cf.Bytes()
		for i := range raw {
			if raw[i] != out[i] {
				t.Errorf("%s：第 %d 個位元組不同（%d vs %d）", filepath.Base(f), i, raw[i], out[i])
				break
			}
		}
	}
	t.Logf("round-trip 通過：%d 個城市檔", len(files))
}

// 解出來的地圖要跟原版載入後的狀態相同——但要先扣掉兩類「不是檔案內容」的差異。
//
// 黃金資料是 oracle 執行 `sim LoadScenario 1` 之後倒出來的 12000 格
// （tools/oracle/tcl/scenario1.tcl）。**它不等於檔案內容**，因為 LoadScenario
// 之後原版還做了兩件事：
//
//  1. DoSimInit() 會跑一次電力掃描，把供電的格子設上 PWRBIT（266 格）。
//  2. 帶 ANIMBIT 的格子（車流、煙囪、體育場）每一刻換一幀，倒出來的是當下那一幀（67 格）。
//
// 扣掉這兩類之後差異是 0。等電力掃描（s_power.c）實作出來，第 1 類就能一起驗。
func TestScenario1MatchesOracle(t *testing.T) {
	res := micropolisRes(t)
	raw, err := os.ReadFile(filepath.Join(res, "snro.111"))
	if err != nil {
		t.Skip("封存裡沒有 snro.111")
	}
	cf, err := ParseCityFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := loadGoldenMap(t, "testdata/scenario1-dullsville.csv")

	var unexplained, powerOnly, animated int
	fx, fy := -1, -1
	for y := 0; y < WorldY; y++ {
		for x := 0; x < WorldX; x++ {
			got, exp := cf.Map[x][y], want[x][y]
			if got == exp {
				continue
			}
			if got&^uint16(PWRBIT) == exp&^uint16(PWRBIT) {
				powerOnly++
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
		t.Fatalf("有 %d 格差異無法解釋。第一處 (%d,%d)：解出 %d，原版 %d",
			unexplained, fx, fy, cf.Map[fx][fy], want[fx][fy])
	}
	t.Logf("差異全部有解釋：PWRBIT %d 格、動畫幀 %d 格，其餘 0", powerOnly, animated)
}

// ⚠ **LoadScenario 不套用檔案裡的 MiscHis 純量。**
//
// s_fileio.c 有兩層：`_load_file()` 只讀七個陣列；`loadFile()` 是它的包裝，
// 額外把 MiscHis 裡的市庫、時間、稅率等套進遊戲狀態。
// **LoadScenario 直接呼叫 `_load_file()`**，跳過那一層，自己先設好劇本表寫死的值。
//
// 所以 snro.111 裡的 CityTime 是 1716（存檔當下的殘留），
// 而 LoadScenario 設的是 (1900-1900)*48+2 = 2。**檔案裡那個值從來沒被用過。**
// oracle 實測 `sim Year` 回 1900、`sim Funds` 回 5000，與劇本表一致。
func TestScenarioFileScalarsAreIgnored(t *testing.T) {
	res := micropolisRes(t)
	raw, err := os.ReadFile(filepath.Join(res, "snro.111"))
	if err != nil {
		t.Skip("封存裡沒有 snro.111")
	}
	cf, err := ParseCityFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cf.CityTime() == 2 {
		t.Skip("檔案裡剛好是 2，這個測試就沒有鑑別力了")
	}

	w := NewWorld(1)
	w.LoadScenarioFile(cf, ScenarioDullsville)
	if w.CityTime != 2 {
		t.Errorf("CityTime = %d，劇本表設的是 (1900-1900)*48+2 = 2", w.CityTime)
	}
	if w.TotalFunds != 5000 {
		t.Errorf("市庫 = %d，Dullsville 是 5000（其餘七個是 20000）", w.TotalFunds)
	}
	if w.CityTax != 7 {
		t.Errorf("稅率 = %d，LoadScenario 設 7", w.CityTax)
	}
	if w.SimSpeed != 3 {
		t.Errorf("速度 = %d，LoadScenario 設 3", w.SimSpeed)
	}
}

// 八個劇本的表：名稱、年份、起始市庫。s_fileio.c:396-447
func TestScenarioTable(t *testing.T) {
	want := []struct {
		s     Scenario
		name  string
		year  int
		funds int
	}{
		{ScenarioDullsville, "Dullsville", 1900, 5000},
		{ScenarioSanFrancisco, "San Francisco", 1906, 20000},
		{ScenarioHamburg, "Hamburg", 1944, 20000},
		{ScenarioBern, "Bern", 1965, 20000},
		{ScenarioTokyo, "Tokyo", 1957, 20000},
		{ScenarioDetroit, "Detroit", 1972, 20000},
		{ScenarioBoston, "Boston", 2010, 20000},
		{ScenarioRio, "Rio de Janeiro", 2047, 20000},
	}
	for _, c := range want {
		info := c.s.Info()
		if info.Name != c.name {
			t.Errorf("%d 的名稱 = %q，應為 %q", c.s, info.Name, c.name)
		}
		if info.StartFunds != c.funds {
			t.Errorf("%s 的起始市庫 = %d，應為 %d", c.name, info.StartFunds, c.funds)
		}
		if got := (c.year-1900)*48 + 2; info.CityTime != got {
			t.Errorf("%s 的 CityTime = %d，(%d-1900)*48+2 = %d", c.name, info.CityTime, c.year, got)
		}
	}
}

// 載入之後撥款百分比一律是 1.0，不管檔案裡寫什麼。
// 原版讀了兩次（第二次是 bug），然後 InitFundingLevel() 全部蓋成 1.0。
func TestLoadCityFileResetsFundingPercents(t *testing.T) {
	cf := &CityFile{}
	cf.setMisc32(miscPolicePercent, 12345)
	cf.setMisc32(miscFirePercent, 999)
	cf.setMisc32(miscRoadPercent, 1)

	w := NewWorld(1)
	w.LoadCityFile(cf)
	if w.PolicePercent != 1.0 || w.FirePercent != 1.0 || w.RoadPercent != 1.0 {
		t.Errorf("撥款百分比 = %v/%v/%v，載入後應該全部是 1.0",
			w.PolicePercent, w.FirePercent, w.RoadPercent)
	}
}

// 三個夾限。s_fileio.c:286-291
func TestLoadCityFileClamps(t *testing.T) {
	cases := []struct {
		name              string
		time              int32
		tax, speed        int16
		wantTime          int
		wantTax, wantSpeed int
	}{
		{"負的時間歸零", -5, 7, 3, 0, 7, 3},
		{"稅率超過 20 回 7", 100, 21, 3, 100, 7, 3},
		{"稅率負數回 7", 100, -1, 3, 100, 7, 3},
		{"速度超過 3 回 3", 100, 7, 9, 100, 7, 3},
		{"合法值不動", 100, 12, 1, 100, 12, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cf := &CityFile{}
			cf.setMisc32(miscCityTime, c.time)
			cf.MiscHis[miscCityTax] = c.tax
			cf.MiscHis[miscSimSpeed] = c.speed
			w := NewWorld(1)
			w.LoadCityFile(cf)
			if w.CityTime != c.wantTime || w.CityTax != c.wantTax || w.SimSpeed != c.wantSpeed {
				t.Errorf("得到 time=%d tax=%d speed=%d，應為 %d/%d/%d",
					w.CityTime, w.CityTax, w.SimSpeed, c.wantTime, c.wantTax, c.wantSpeed)
			}
		})
	}
}

// 32 位元欄位的高低半字順序：MiscHis[n] 是高半字。
func TestMisc32IsBigEndianAcrossShorts(t *testing.T) {
	cf := &CityFile{}
	cf.setMisc32(miscTotalFunds, 0x12345678)
	if cf.MiscHis[miscTotalFunds] != 0x1234 {
		t.Errorf("高半字 = %#x，應為 0x1234", uint16(cf.MiscHis[miscTotalFunds]))
	}
	if uint16(cf.MiscHis[miscTotalFunds+1]) != 0x5678 {
		t.Errorf("低半字 = %#x，應為 0x5678", uint16(cf.MiscHis[miscTotalFunds+1]))
	}
	if got := cf.TotalFunds(); got != 0x12345678 {
		t.Errorf("讀回 %#x", got)
	}
}

// 大小不對就拒絕，不要猜。
func TestParseCityFileRejectsWrongSize(t *testing.T) {
	if _, err := ParseCityFile(make([]byte, 100)); err == nil {
		t.Error("大小不對卻沒有拒絕")
	}
	if _, err := ParseCityFile(make([]byte, CityFileSize2x2)); err == nil {
		t.Error("2×2 的地圖大小應該拒絕（本專案不支援 MEGA）")
	}
}
