package game

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// SaveCity 把城市存成原版格式的 `.cty`。
//
// 用原版格式而不是自創格式，是刻意的：存出來的檔案可以拿去餵原版
// SimCity 或 Micropolis，反過來也一樣。remake 的存檔如果自成一格，
// 玩家的城市就被鎖在這個實作裡了。
func SaveCity(path string, w *sim.World) error {
	if filepath.Ext(path) == "" {
		path += ".cty"
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	b := w.ToCityFile().Bytes()
	if len(b) != sim.CityFileSize1x1 {
		return fmt.Errorf("打包出來是 %d 位元組，應為 %d", len(b), sim.CityFileSize1x1)
	}
	return os.WriteFile(path, b, 0o644)
}

// dosSaveSize 是 DOS 原版自己存出來的城市檔大小。
const dosSaveSize = sim.CityFileSize1x1 + 128 // 27248

// psnSize 是解壓後的 DOS 劇本檔大小（144 位元組檔頭 ＋ 完整 27120）。
const psnSize = sim.CityFileSize1x1 + 144 // 27264

// normalizeCityBytes 把各種來源的城市檔攤成本專案的標準 27120 版面。
//
// 三種來源：
//
//	27120  Micropolis 與本專案寫的，沒有檔頭
//	27264  解壓後的 DOS 劇本 `.PSN`：144 位元組檔頭 ＋ 完整的 27120
//	27248  **DOS 原版自己存的城市檔**，版面不一樣：
//
//	    0      檔頭 128 位元組（`00 0d`、城市名、零填、魔數 `CITYMCRP`）
//	    128    六個歷史陣列，2880 位元組
//	    3008   MiscHis —— **256 位元組（128 個 short）**，不是 Micropolis 的 240
//	    3264   地圖 120×100，24000 位元組
//	    27248  檔案在這裡結束——**比 3264+24000 少 16**，
//	           也就是地圖最後 8 格（x=119、y=92–99）沒寫進去
//
// ⚠ **不要用檔案大小去推檔頭長度。** 27248 − 27120 = 128 會讓人以為
// 「檔頭 128、其餘照舊」，而那個答案**看起來完全正常**：純量全部讀得對
// （CityTime 3、資金 5000、稅率 7），地物的格數分布也對——因為整張地圖
// 只是**平移了 8 格**，counts 不變。只有逐格比才看得出來：
// 那樣讀 12 000 格裡只有 3191 格對得上，正確讀法有 11 684 格
// （其餘是存檔那一刻模擬已經跑掉的差異）。
//
// 量法：把 `.PSN` 解出來的地圖位元組拿去存檔裡搜，位移是 3264。
// 三種讀法的實測見 docs/formats/01-city-file.md。
//
// 少的那 16 位元組原因未解。讀的時候補零——那 8 格在地圖右下角，
// Dullsville 那裡是空地，補零與原值相同。
func normalizeCityBytes(raw []byte) ([]byte, error) {
	switch len(raw) {
	case sim.CityFileSize1x1:
		return raw, nil
	case psnSize:
		return raw[144:], nil
	case dosSaveSize:
		body := make([]byte, 0, sim.CityFileSize1x1)
		body = append(body, raw[128:128+3120]...) // 六個陣列 ＋ MiscHis 的前 240
		body = append(body, raw[3264:]...)        // 地圖（尾端短 16）
		return append(body, make([]byte, sim.CityFileSize1x1-len(body))...), nil
	}
	return nil, fmt.Errorf("城市檔是 %d 位元組，不是 %d（無檔頭）、%d（DOS 存檔）"+
		"或 %d（解壓後的 DOS 劇本）—— 這可能是還沒解壓的 `.PSN`",
		len(raw), sim.CityFileSize1x1, dosSaveSize, psnSize)
}

// LoadCity 讀一個原版格式的 `.cty`。
func LoadCity(path string) (*sim.World, error) {
	return LoadCitySeed(path, sim.RandomSeed())
}

// LoadCityFileRaw 只把城市檔攤成標準版面並解出結構，**不建 World、
// 不跑 DoSimInit**。
//
// 為什麼需要：`DoSimInit` 會跑一次 PTLScan／CrimeScan，把 `MiscHis` 裡
// 原版自己記下來的地價、犯罪、汙染平均全部覆蓋掉。要拿「原版自己算出來的
// 數字」跟 remake 比就不能經過那一步（docs/re/18-dos-parity.md §五之三）。
func LoadCityFileRaw(path string) (*sim.CityFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	body, err := normalizeCityBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("%s：%w", filepath.Base(path), err)
	}
	return sim.ParseCityFile(body)
}

// LoadCitySeed 同 LoadCity，但種子由呼叫端指定。理由與
// LoadScenarioSeed 相同：`DoSimInit` 的那次 `MapScan` 會擲亂數。
func LoadCitySeed(path string, seed uint32) (*sim.World, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	body, err := normalizeCityBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("%s：%w", filepath.Base(path), err)
	}
	cf, err := sim.ParseCityFile(body)
	if err != nil {
		return nil, err
	}
	w := sim.NewWorld(seed)
	w.LoadCityFile(cf)
	w.InitSimLoad = 1
	w.DoSimInit()
	return w, nil
}
