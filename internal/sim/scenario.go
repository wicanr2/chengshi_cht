package sim

// 八個劇本。證據：docs/formats/01-city-file.md／一手出處：s_fileio.c:396 LoadScenario
//
// 軟體世界中文說明書把 SCENARIOS 譯為「悲情城市」（珍藏版 29，第 13 頁）。
// 譯名以 translations/glossary.md 為準，這一層只放編號與規則數值。

// Scenario 是劇本編號。s_fileio.c:406 起的 switch，1 到 8；超出範圍會被夾成 1。
type Scenario int

const (
	ScenarioNone         Scenario = 0
	ScenarioDullsville   Scenario = 1
	ScenarioSanFrancisco Scenario = 2
	ScenarioHamburg      Scenario = 3
	ScenarioBern         Scenario = 4
	ScenarioTokyo        Scenario = 5
	ScenarioDetroit      Scenario = 6
	ScenarioBoston       Scenario = 7
	ScenarioRio          Scenario = 8
)

// ScenarioInfo 是劇本表的一列。
type ScenarioInfo struct {
	Name       string // 原版的英文城市名。s_fileio.c 的 name 變數
	File       string // 資源檔名。snro.111 … snro.888
	Year       int    // 起始年份
	CityTime   int    // (Year-1900)*48 + 2 —— 一年 48 刻
	StartFunds int    // SetFunds() 的參數
}

// TicksPerYear 是一年的遊戲刻數。
// 由劇本表的 (Year-1900)*48 + 2 反推（s_fileio.c:400 起，八列一致）。
const TicksPerYear = 48

// scenarioTable 逐列照抄 s_fileio.c:406-447。
//
// ⚠ 東京是 **1957**，不是 1967。啟動畫面的縮圖美術上印的是 1967，
// 但程式碼（CityTime = (1957-1900)*48+2）、訊息字串（res/micropolis.tcl:451
// 的 `TOKYO, JAPAN  1957`）與 IBM PC 版官方手冊三者都寫 1957。
// **一手資料贏美術字樣。**
var scenarioTable = map[Scenario]ScenarioInfo{
	ScenarioDullsville:   {"Dullsville", "snro.111", 1900, (1900-1900)*TicksPerYear + 2, 5000},
	ScenarioSanFrancisco: {"San Francisco", "snro.222", 1906, (1906-1900)*TicksPerYear + 2, 20000},
	ScenarioHamburg:      {"Hamburg", "snro.333", 1944, (1944-1900)*TicksPerYear + 2, 20000},
	ScenarioBern:         {"Bern", "snro.444", 1965, (1965-1900)*TicksPerYear + 2, 20000},
	ScenarioTokyo:        {"Tokyo", "snro.555", 1957, (1957-1900)*TicksPerYear + 2, 20000},
	ScenarioDetroit:      {"Detroit", "snro.666", 1972, (1972-1900)*TicksPerYear + 2, 20000},
	ScenarioBoston:       {"Boston", "snro.777", 2010, (2010-1900)*TicksPerYear + 2, 20000},
	ScenarioRio:          {"Rio de Janeiro", "snro.888", 2047, (2047-1900)*TicksPerYear + 2, 20000},
}

// Info 回傳劇本表那一列。編號超出 1..8 時夾成 1，照 s_fileio.c:404
// `if ((s < 1) || (s > 8)) s = 1;`。
func (s Scenario) Info() ScenarioInfo {
	if s < ScenarioDullsville || s > ScenarioRio {
		s = ScenarioDullsville
	}
	return scenarioTable[s]
}

// LoadScenarioFile 照 s_fileio.c:396 LoadScenario() 的語意載入一個劇本。
//
// ⚠ 與 LoadCityFile 的關鍵差別：**劇本不套用檔案裡的 MiscHis 純量**。
// 原版的 LoadScenario 呼叫的是 `_load_file()`（只讀七個陣列），
// 不是 `loadFile()`（會套用 MiscHis）。所以 snro.111 裡殘留的 CityTime 1716
// 從來沒有生效過——劇本表寫死的 2 才算數。
func (w *World) LoadScenarioFile(cf *CityFile, s Scenario) {
	info := s.Info()

	// _load_file()：只讀七個陣列。
	w.Map = cf.Map
	w.ResHis = cf.ResHis
	w.ComHis = cf.ComHis
	w.IndHis = cf.IndHis
	w.CrimeHis = cf.CrimeHis
	w.PollutionHis = cf.PollutionHis
	w.MoneyHis = cf.MoneyHis
	w.MiscHis = cf.MiscHis

	// LoadScenario 自己設的純量。s_fileio.c:404-455
	w.CityTime = info.CityTime
	w.TotalFunds = info.StartFunds
	w.CityTax = 7  // s_fileio.c:455
	w.SimSpeed = 3 // s_fileio.c:454 setSpeed(3)
	w.Scenario = s
	w.CityName = info.Name

	// InitFundingLevel()。w_budget.c:83
	w.PolicePercent, w.FirePercent, w.RoadPercent = 1.0, 1.0, 1.0
}
