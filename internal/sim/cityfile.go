package sim

import (
	"encoding/binary"
	"fmt"
)

// 城市檔（`.cty`／`snro.*`）的讀寫。
// 證據：docs/formats/01-city-file.md／規格：docs/spec/city-file.md
// 一手出處：s_fileio.c
//
// 這一層只做序列化，不套用到 World。原因是 CLAUDE.md §4 的「改寫不是重建」：
// 未解的位元組要能原樣 round-trip，所以 MiscHis 整塊原樣保留，
// 語意欄位只是它上面的視圖。

// 檔案大小。s_fileio.c:207-215 的 switch 只認這三個。
const (
	CityFileSize1x1 = 27120 // 120×100
	CityFileSize2x2 = 99120
	CityFileSize3x3 = 219120
)

// MiscHis 裡的欄位位置（以 int16 為單位）。s_fileio.c:245-285、:325-357
const (
	miscCityTime      = 8  // 與 9，32 位元
	miscTotalFunds    = 50 // 與 51，32 位元
	miscAutoBulldoze  = 52
	miscAutoBudget    = 53
	miscAutoGo        = 54
	miscUserSoundOn   = 55
	miscCityTax       = 56
	miscSimSpeed      = 57
	miscPolicePercent = 58 // 與 59，32 位元定點（×65536）
	miscFirePercent   = 60 // 與 61
	miscRoadPercent   = 62 // 與 63
)

// CityFile 是一個城市檔的完整內容。欄位順序就是檔案順序。
// s_fileio.c:217-227（載入）與 :358-366（存檔）
type CityFile struct {
	ResHis       [HistLen]int16
	ComHis       [HistLen]int16
	IndHis       [HistLen]int16
	CrimeHis     [HistLen]int16
	PollutionHis [HistLen]int16
	MoneyHis     [HistLen]int16
	MiscHis      [MiscHistLen]int16
	Map          [WorldX][WorldY]uint16
}

// ParseCityFile 解一個 27120 位元組的城市檔。
//
// 所有 16 位元值都是 big-endian：s_fileio.c 的 NOOP_ON_BE 巨集在小端機器上
// 才會交換位元組（`int test = 1; if (!*(unsigned char*)&test) return;`
// —— 小端時第一個位元組是 1，不 return，於是交換）。所以檔案裡是大端。
func ParseCityFile(b []byte) (*CityFile, error) {
	if len(b) != CityFileSize1x1 {
		// 2×2 與 3×3 的大小原版認得，但那是 MEGA 版的地圖尺寸，本專案不支援。
		return nil, fmt.Errorf("城市檔大小 %d，只支援 %d（120×100）", len(b), CityFileSize1x1)
	}
	cf := &CityFile{}
	off := 0
	readShorts := func(dst []int16) {
		for i := range dst {
			dst[i] = int16(binary.BigEndian.Uint16(b[off:]))
			off += 2
		}
	}
	readShorts(cf.ResHis[:])
	readShorts(cf.ComHis[:])
	readShorts(cf.IndHis[:])
	readShorts(cf.CrimeHis[:])
	readShorts(cf.PollutionHis[:])
	readShorts(cf.MoneyHis[:])
	readShorts(cf.MiscHis[:])
	// 地圖是 x 外層、y 內層（s_alloc.c:160）。
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			cf.Map[x][y] = binary.BigEndian.Uint16(b[off:])
			off += 2
		}
	}
	if off != CityFileSize1x1 {
		return nil, fmt.Errorf("解析後位移 %d，應為 %d", off, CityFileSize1x1)
	}
	return cf, nil
}

// Bytes 把城市檔序列化回位元組。與 ParseCityFile 逐位元組互逆。
func (cf *CityFile) Bytes() []byte {
	b := make([]byte, CityFileSize1x1)
	off := 0
	writeShorts := func(src []int16) {
		for _, v := range src {
			binary.BigEndian.PutUint16(b[off:], uint16(v))
			off += 2
		}
	}
	writeShorts(cf.ResHis[:])
	writeShorts(cf.ComHis[:])
	writeShorts(cf.IndHis[:])
	writeShorts(cf.CrimeHis[:])
	writeShorts(cf.PollutionHis[:])
	writeShorts(cf.MoneyHis[:])
	writeShorts(cf.MiscHis[:])
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			binary.BigEndian.PutUint16(b[off:], cf.Map[x][y])
			off += 2
		}
	}
	return b
}

// misc32 讀 MiscHis 裡跨兩個 int16 的 32 位元值。
//
// 原版在記憶體裡用 `*(QUAD*)(MiscHis + n)` 加上 HALF_SWAP_LONGS 來處理，
// 那是為了對付「小端機器上 long 的兩個半字順序」；反推到檔案上，
// 結果就是一個單純的大端 32 位元整數：MiscHis[n] 是高半字，MiscHis[n+1] 是低半字。
func (cf *CityFile) misc32(n int) int32 {
	return int32(uint32(uint16(cf.MiscHis[n]))<<16 | uint32(uint16(cf.MiscHis[n+1])))
}

func (cf *CityFile) setMisc32(n int, v int32) {
	cf.MiscHis[n] = int16(uint32(v) >> 16)
	cf.MiscHis[n+1] = int16(uint32(v) & 0xffff)
}

// CityTime 是遊戲內的時間刻。s_fileio.c:250
func (cf *CityFile) CityTime() int32 { return cf.misc32(miscCityTime) }

// TotalFunds 是市庫。s_fileio.c:246
func (cf *CityFile) TotalFunds() int32 { return cf.misc32(miscTotalFunds) }

// CityTax 是稅率。s_fileio.c:258
func (cf *CityFile) CityTax() int16 { return cf.MiscHis[miscCityTax] }

// SimSpeed 是模擬速度。s_fileio.c:259
func (cf *CityFile) SimSpeed() int16 { return cf.MiscHis[miscSimSpeed] }

// AutoBulldoze／AutoBudget／AutoGo／UserSoundOn。s_fileio.c:254-257
func (cf *CityFile) AutoBulldoze() bool { return cf.MiscHis[miscAutoBulldoze] != 0 }
func (cf *CityFile) AutoBudget() bool   { return cf.MiscHis[miscAutoBudget] != 0 }
func (cf *CityFile) AutoGo() bool       { return cf.MiscHis[miscAutoGo] != 0 }
func (cf *CityFile) SoundOn() bool      { return cf.MiscHis[miscUserSoundOn] != 0 }

// LoadCityFile 把城市檔套用到世界，語意照 s_fileio.c:236 loadFile()。
//
// 三個夾限照抄（s_fileio.c:286-291）：CityTime 負數歸零、稅率超出 0..20 回 7、
// 速度超出 0..3 回 3。
//
// ⚠ **撥款百分比刻意不從檔案讀。** 原版讀了兩次（第二次漏掉 HALF_SWAP，
// 是個真的 bug），但緊接著呼叫 InitFundingLevel() 把三個百分比全部設回 1.0
// （w_budget.c:83），所以檔案裡那三個欄位**從來沒有生效過**。
// 這裡直接照最終行為做，並在筆記裡記下那個死 bug。
func (w *World) LoadCityFile(cf *CityFile) {
	// ⚠ 先清場，同 LoadScenarioFile：原版 loadFile() 之後會呼叫
	// InitWillStuff()，把衍生陣列歸零並清掉精靈。
	w.InitWillStuff()

	w.Map = cf.Map
	w.ResHis = cf.ResHis
	w.ComHis = cf.ComHis
	w.IndHis = cf.IndHis
	w.CrimeHis = cf.CrimeHis
	w.PollutionHis = cf.PollutionHis
	w.MoneyHis = cf.MoneyHis
	w.MiscHis = cf.MiscHis

	w.CityTime = int(cf.CityTime())
	w.TotalFunds = int(cf.TotalFunds())
	w.CityTax = int(cf.CityTax())
	w.SimSpeed = int(cf.SimSpeed())
	w.AutoBulldoze = cf.AutoBulldoze()
	w.AutoBudget = cf.AutoBudget()
	w.AutoGo = cf.AutoGo()

	if w.CityTime < 0 {
		w.CityTime = 0
	}
	if w.CityTax > 20 || w.CityTax < 0 {
		w.CityTax = 7
	}
	if w.SimSpeed < 0 || w.SimSpeed > 3 {
		w.SimSpeed = 3
	}

	// w_budget.c:83 InitFundingLevel()
	w.PolicePercent, w.FirePercent, w.RoadPercent = 1.0, 1.0, 1.0

	// ⚠ **載入城市會把劇本編號清成 0**（s_fileio.c:300 `ScenarioID = 0;`）。
	// 也就是說：把一個劇本存下來再讀回去，它就不再是劇本了——災難排程與
	// 勝敗判定都不會再觸發。這是原版的行為，不是漏掉。
	w.Scenario = 0
}

// ToCityFile 把目前的世界狀態打包成一個可存檔的城市檔。
// s_fileio.c:325 SaveFile()
//
// ⚠ **存檔寫回 MiscHis，載入時也從 MiscHis 讀**。那個陣列同時是
// 「歷史統計的一部分」與「純量的容器」——`miscCityTime` 這些索引
// 落在歷史陣列的後段。所以打包時要先複製整個 MiscHis 再覆蓋純量，
// 不能從零開始填：其餘欄位（劇本編號、城市等級、災難計時…）
// 會整批遺失，而**載入後看起來一切正常**，只有跑一陣子才會發現
// 劇本判定不觸發。
func (w *World) ToCityFile() *CityFile {
	cf := &CityFile{
		ResHis:       w.ResHis,
		ComHis:       w.ComHis,
		IndHis:       w.IndHis,
		CrimeHis:     w.CrimeHis,
		PollutionHis: w.PollutionHis,
		MoneyHis:     w.MoneyHis,
		MiscHis:      w.MiscHis,
		Map:          w.Map,
	}
	cf.setMisc32(miscCityTime, int32(w.CityTime))
	cf.setMisc32(miscTotalFunds, int32(w.TotalFunds))
	cf.MiscHis[miscCityTax] = int16(w.CityTax)
	cf.MiscHis[miscSimSpeed] = int16(w.SimSpeed)
	cf.MiscHis[miscAutoBulldoze] = boolToI16(w.AutoBulldoze)
	cf.MiscHis[miscAutoBudget] = boolToI16(w.AutoBudget)
	cf.MiscHis[miscAutoGo] = boolToI16(w.AutoGo)
	// 三個編列百分比是 32 位元定點（×65536）。s_fileio.c:352
	cf.setMisc32(miscPolicePercent, int32(w.PolicePercent*65536))
	cf.setMisc32(miscFirePercent, int32(w.FirePercent*65536))
	cf.setMisc32(miscRoadPercent, int32(w.RoadPercent*65536))
	return cf
}

func boolToI16(b bool) int16 {
	if b {
		return 1
	}
	return 0
}
