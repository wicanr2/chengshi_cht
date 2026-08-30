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
// 三種來源，**後兩種都只是「128 位元組檔頭 ＋ 標準檔身」**：
//
//	27120  Micropolis 與本專案寫的，沒有檔頭
//	27248  DOS 原版自己存的城市檔 —— 128 檔頭 ＋ 27120，剛好用完
//	27264  解壓後的 DOS 劇本 `.PSN` —— 128 檔頭 ＋ 27120，尾端多 16 位元組
//
// 檔頭是 `00 0d`、城市名、零填、魔數 `CITYMCRP`。
//
// ⚠ **這裡曾經錯過，而且錯得沒有症狀。** 先前的版本對 `.PSN` 跳 144、
// 對 DOS 存檔把地圖當成從 3264 開始（假設 MiscHis 是 128 個 short），
// 兩條路都比正確位置**晚了 16 位元組 ＝ 8 格**。地圖是欄優先存的，
// 所以 8 格就是**整張圖往下平移 8 列**。
//
// 為什麼一直沒被發現：
//
//   - 純量全部讀得對（CityTime、資金、稅率都不在平移範圍內）；
//   - 地物的格數分布也對——平移不改變 counts；
//   - **remake 與 DOS 存檔的逐格對拍也對**（`docs/re/18-dos-parity.md`），
//     因為兩邊都經過同一個錯誤的讀法，偏移互相抵銷。
//
// 抓到它的是**跟原版畫面對拍**：把 DOSBox 的截圖每一格解回圖塊編號，
// 再跟存檔的地圖比。把原版的鏡頭用方向鍵頂到左上角（一定被夾在 (0,0)），
// 畫面上 504 格**全部**吻合位移 3248 的讀法，位移 3264 只有 141 格。
// `.PSN` 那邊同樣指到 3248（98%，次佳 61%）。
// 工具：`tools/shot_locate.py`、`tools/shot_tilescan.py`。
func normalizeCityBytes(raw []byte) ([]byte, error) {
	switch len(raw) {
	case sim.CityFileSize1x1:
		return raw, nil
	case dosSaveSize, psnSize:
		return raw[128 : 128+sim.CityFileSize1x1], nil
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
