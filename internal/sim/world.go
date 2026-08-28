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
	BLBNBIT   = BULLBIT + BURNBIT             // headers/sim.h:254
	BLBNCNBIT = BULLBIT + BURNBIT + CONDBIT   // headers/sim.h:255
	BNCNBIT   = BURNBIT + CONDBIT             // headers/sim.h:256
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

	Rand *Rand
}

// NewWorld 配置一個全空的世界。
// NewWorld 配置一個全空的世界，純量用 sim.c:167 sim_init() 的初值。
func NewWorld(seed uint32) *World {
	return &World{
		CityTime:     50,   // sim.c:183
		TotalFunds:   20000, // 實測 sim Funds；docs/re/01-oracle-harness.md §4
		CityTax:      7,    // sim.c:182
		SimSpeed:     3,    // sim.c:194
		AutoBulldoze: true, // sim.c:188
		AutoBudget:   true, // sim.c:189
		AutoGo:       true, // sim.c:181
		Rand:         NewRand(seed),
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
