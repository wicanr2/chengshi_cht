package sim

// 世界的四種解析度。證據：docs/re/03-map-and-tiles.md §1
//
// headers/sim.h:160-167。SmY 特別注意：原版是 ((SimHeight+7)>>3)，
// 有進位，等於 13；寫成 SimHeight>>3 會少最外圈那一列，而且要等城市長到
// 地圖底部才看得出來。
const (
	WorldX = SimWidth  // 120
	WorldY = SimHeight // 100

	HWldX = SimWidth >> 1  // 60
	HWldY = SimHeight >> 1 // 50

	QWX = SimWidth >> 2  // 30
	QWY = SimHeight >> 2 // 25

	SmX = SimWidth >> 3        // 15
	SmY = (SimHeight + 7) >> 3 // 13 —— 不是 12

	// headers/sim.h:183。位元圖每列的 word 數。
	PowerMapRow = (WorldX + 15) / 16 // 8

	// 歷史統計。headers/sim.h:180-181 的 HISTLEN／MISCHISTLEN 是 **byte 數**，
	// 指標型別是 short*，所以實際容量是一半（s_alloc.c:206 起）。
	HistLen     = HISTLEN / 2     // 240
	MiscHistLen = MISCHISTLEN / 2 // 120
)

// 旗標的組合常數。headers/sim.h:254-256
//
// 這三個在標頭裡寫成 (BULLBIT+BURNBIT) 這種算式而不是數字字面量，
// 所以 tools/gen_tiles.py 抽不到（它只收數字）。在這裡以同樣的算式定義，
// 值由編譯器算，不手抄數字。
const (
	BLBNBIT   = BULLBIT + BURNBIT           // headers/sim.h:254
	BLBNCNBIT = BULLBIT + BURNBIT + CONDBIT // headers/sim.h:255
	BNCNBIT   = BURNBIT + CONDBIT           // headers/sim.h:256
)

// World 是模擬的全部狀態。它不認識畫面，也不做 I/O。
//
// 索引順序照原版：s_alloc.c:160 的 Map[i] = mapPtr + i*WORLD_Y，
// 也就是 x 外層、y 內層。存檔的位元組順序依賴這個順序。
type World struct {
	// Map 每格是一個 16 位元字：低 10 位元圖塊、高 6 位元旗標。
	// docs/re/03-map-and-tiles.md §2
	Map [WorldX][WorldY]uint16

	// 半解析度（一格代表 2×2）。s_alloc.c:180 起。
	PopDensity   [HWldX][HWldY]uint8
	TrfDensity   [HWldX][HWldY]uint8
	PollutionMem [HWldX][HWldY]uint8
	LandValueMem [HWldX][HWldY]uint8
	CrimeMem     [HWldX][HWldY]uint8
	Tem          [HWldX][HWldY]uint8
	Tem2         [HWldX][HWldY]uint8

	// 四分之一解析度（一格代表 4×4）。s_alloc.c:194 起。
	TerrainMem [QWX][QWY]uint8
	Qtem       [QWX][QWY]uint8

	// 八分之一解析度（一格代表 8×8）。s_alloc.c:113 起。
	RateOGMem       [SmX][SmY]int16
	FireStMap       [SmX][SmY]int16
	PoliceMap       [SmX][SmY]int16
	PoliceMapEffect [SmX][SmY]int16
	FireRate        [SmX][SmY]int16
	ComRate         [SmX][SmY]int16
	STem            [SmX][SmY]int16

	// 歷史統計。s_alloc.c:206 起。
	ResHis       [HistLen]int16
	ComHis       [HistLen]int16
	IndHis       [HistLen]int16
	MoneyHis     [HistLen]int16
	PollutionHis [HistLen]int16
	CrimeHis     [HistLen]int16
	MiscHis      [MiscHistLen]int16

	// PowerMap 是位元圖。headers/sim.h:189 SETPOWERBIT。
	PowerMap [PowerMapRow * WorldY]uint16

	// 遊戲層的純量。sim.c:167 sim_init() 的初值；載入城市檔時由檔案覆蓋。
	CityTime     int  // 遊戲刻。一年 48 刻（s_fileio.c:396 起的劇本表）
	TotalFunds   int  // 市庫
	CityTax      int  // 稅率。sim.c:182 初值 7
	SimSpeed     int  // sim.c:194 初值 3
	AutoBulldoze bool // sim.c:188 初值 true
	AutoBudget   bool // sim.c:189 初值 true
	AutoGo       bool // sim.c:181 初值 true

	CityName string   // setAnyCityName()
	Scenario Scenario // ScenarioID。0 代表不是劇本

	// 撥款百分比。w_budget.c:64-68 初值 0.0，InitFundingLevel() 設成 1.0。
	PolicePercent float64
	FirePercent   float64
	RoadPercent   float64

	// 掃描游標。原版是全域（s_alloc.c:68-69），MapScan 每一格更新。
	SMapX, SMapY int
	CChr         uint16 // 目前這一格的完整字
	CChr9        int    // 目前這一格的圖塊編號（CChr & LOMASK）

	// 每一輪普查歸零、由 MapScan 累計的計數。s_sim.c:524 ClearCensus
	PwrdZCnt, UnPwrdZCnt      int
	FirePop                   int
	RoadTotal, RailTotal      int
	ResPop, ComPop, IndPop    int
	ResZPop, ComZPop, IndZPop int
	HospPop, ChurchPop        int
	PolicePop, FireStPop      int
	StadiumPop                int
	CoalPop, NuclearPop       int
	PortPop, APortPop         int

	// 三態旗標：1 需要、0 剛好、−1 太多。s_sim.c:598 TakeCensus
	NeedHosp, NeedChurch int

	// 需求閥與上限。s_sim.c:414 SetValves
	RValve, CValve, IValve int
	ResCap, ComCap, IndCap bool
	TotalPop, LastTotalPop int
	EMarket                float64
	CrimeRamp, PolluteRamp int
	GameLevel              int
	CityClass, CityScore   int

	// 預算與撥款效果。s_sim.c:398 SetCommonInits、:641 CollectTax
	RoadEffect, PoliceEffect, FireEffect int
	TaxFlag                              bool
	TaxFund, CashFlow                    int
	AvCityTax                            int
	RoadFund, PoliceFund, FireFund       int
	RoadSpend, PoliceSpend, FireSpend    int

	// 主迴圈的計數器。s_sim.c:207 DoSimInit、:96 SimFrame
	Fcycle, Scycle, Spdcycle int
	NewPower                 bool

	// 劇本災難排程。s_sim.c:333 SimLoadInit
	DisasterEvent, DisasterWait int
	ScoreType, ScoreWait        int
	NoDisasters                 bool

	// 訊息埠。s_msg.c:227
	//
	// 一次只放得下一則。正數（純文字）先到先得，負數（有圖）會覆蓋。
	// MesX／MesY 是「前往」按鈕的目標，0,0 代表沒有座標。
	// DoAnimation 決定爆炸要不要用動畫格。原版是使用者選項
	// （w_stubs.c），預設開。關掉時瓦礫用固定的 SOMETINYEXP，
	// **而且不擲骰**——這會改變亂數數列。
	DoAnimation bool

	MessagePort  int
	MesX, MesY   int
	LastPicNum   int
	LastCityPop  int // CheckGrowth 的上一次人口（算法與 TotalPop 不同）
	LastCategory int // 上一次發出的人口里程碑，用來去重
	GameOver     GameOverHook

	// 交通的走訪堆疊。s_traf.c:69
	posStack           [maxTrafDis + 1][2]int
	posStackN          int
	lDir               int
	zSource            int
	TrafMaxX, TrafMaxY int

	// 掃描的輸出。s_scan.c:69-72
	CCx, CCy         int // 城市重心（全解析度座標）
	CCx2, CCy2       int // 重心的半解析度座標
	PolMaxX, PolMaxY int
	// PolluteTot／PolluteNum 是 PolluteAverage 的分子與分母。原版沒有
	// 這兩個變數（`s_scan.c:250` 是區域變數），這裡留下來純粹是為了
	// 觀測：DOS 原版在同一張地圖上算出來的汙染均值比我們低 17%–48%，
	// 要拆開是分子不同還是分母不同就得看得到這兩個數。
	// 見 docs/re/18-dos-parity.md §六。
	PolluteTot, PolluteNum int
	CrimeMaxX, CrimeMaxY   int
	LVAverage              int // 地價平均
	PolluteAverage         int // 汙染平均
	CrimeAverage           int // 犯罪平均

	// MeltX/MeltY 是最近一次熔毀的位置。s_sim.c:1161
	MeltX, MeltY int
	// 水災與墜機的狀態。s_disast.c:68-70
	FloodCnt       int
	FloodX, FloodY int
	CrashX, CrashY int

	// InitSimLoad：2 ＝ 新城市、1 ＝ 剛載入、0 ＝ 已初始化。s_sim.c:213
	InitSimLoad   int
	DoInitialEval bool

	// Eval 是最近一次城市評分的結果。s_eval.c
	Eval Evaluation

	// HasAirCrash 對應原版的 NO_AIRCRASH 建置旗標。
	// 官方 makefile 帶了 -DNO_AIRCRASH，所以隨機空難在正式建置裡不會發生。
	HasAirCrash bool

	spriteSys *spriteSystem

	// soundQueue 是規則層發出的音效事件，呈現層每個畫格取走一次。
	// 不進存檔、不參與對拍，見 sound.go。
	soundQueue []int

	// Sprites 是精靈系統。nil 代表沒有（見 mapscan.go 的 SpriteHooks）。
	Sprites SpriteHooks

	Rand *Rand
}

// NewWorld 配置一個全空的世界。
// NewWorld 配置一個全空的世界，純量用 sim.c:167 sim_init() 的初值。
func NewWorld(seed uint32) *World {
	return &World{
		CityTime:     50,    // sim.c:183
		TotalFunds:   20000, // 實測 sim Funds；docs/re/01-oracle-harness.md §4
		CityTax:      7,     // sim.c:182
		SimSpeed:     3,     // sim.c:194
		AutoBulldoze: true,  // sim.c:188
		AutoBudget:   true,  // sim.c:189
		DoAnimation:  true,  // sim.c:92
		AutoGo:       true,  // sim.c:181
		RoadEffect:   32,    // s_sim.c:401 SetCommonInits
		PoliceEffect: 1000,  // s_sim.c:402
		FireEffect:   1000,  // s_sim.c:403
		EMarket:      6.0,   // s_sim.c:318 InitSimMemory

		// 新城市的預設名稱。Micropolis 的 X11 版是問玩家，DOS 版不問：
		// 執行檔的新城市對話框把 `HERESVILLE` 當預設值
		// （`SIMCITY.EXE` 0x0255c2，緊接在 `SIMCITY city name:` 前面），
		// 實跑也是這個名字出現在標題列
		// （workplace/dosbox/t2-00-default.png）。
		CityName: "HERESVILLE",

		// 三項撥款比例的初值是 1.0（w_budget.c:82 InitFundingLevel）。
		// 檔案層宣告的 0.0 只是 C 的靜態初值，開新城市時會被覆蓋掉；
		// 漏掉這一步的話 roadValue 恆為 0 —— 錢只進不出，而且
		// RoadEffect 會被 UpdateFundEffects 算成 0，道路加速崩壞。
		PolicePercent: 1.0,
		FirePercent:   1.0,
		RoadPercent:   1.0,

		Rand: NewRand(seed),
	}
}

// InBounds 回報座標是否落在地圖內。
func InBounds(x, y int) bool {
	return x >= 0 && x < WorldX && y >= 0 && y < WorldY
}

// Tile 取一格的完整 16 位元字（含旗標）。
func (w *World) Tile(x, y int) uint16 { return w.Map[x][y] }

// SetTile 寫入一格的完整 16 位元字。
func (w *World) SetTile(x, y int, v uint16) { w.Map[x][y] = v }

// TileNum 取圖塊編號（低 10 位元）。headers/sim.h:252 LOMASK
func (w *World) TileNum(x, y int) int { return int(w.Map[x][y] & LOMASK) }

// TileFlags 取旗標（高 6 位元）。headers/sim.h:251 ALLBITS
func (w *World) TileFlags(x, y int) uint16 { return w.Map[x][y] & ALLBITS }

// SetTileNum 只換圖塊編號，旗標保持不變。
func (w *World) SetTileNum(x, y, tile int) {
	w.Map[x][y] = w.Map[x][y]&ALLBITS | uint16(tile)&LOMASK
}

// HasFlag 回報某個旗標是否設起來。用法：w.HasFlag(x, y, PWRBIT)
func (w *World) HasFlag(x, y int, flag uint16) bool {
	return w.Map[x][y]&flag != 0
}

// PowerWord 回傳 (x,y) 在電力位元圖裡的 word 索引與位元遮罩。
// headers/sim.h:188 POWERWORD / SETPOWERBIT
//
// 原版非 MEGA 版寫的是 (x>>4) + (y<<3)；(y<<3) 就是 y*PowerMapRow，
// 因為 PowerMapRow 剛好是 8。這裡寫成乘法，尺寸改了也不會錯。
func PowerWord(x, y int) (word int, mask uint16) {
	return (x >> 4) + y*PowerMapRow, 1 << uint(x&15)
}

// SetPowerBit 設起 (x,y) 的供電位元。
func (w *World) SetPowerBit(x, y int) {
	i, m := PowerWord(x, y)
	w.PowerMap[i] |= m
}

// TestPowerBit 讀 (x,y) 的供電位元。
func (w *World) TestPowerBit(x, y int) bool {
	i, m := PowerWord(x, y)
	return w.PowerMap[i]&m != 0
}
