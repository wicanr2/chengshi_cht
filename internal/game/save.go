package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// dosHeaderTail 是 DOS 存檔檔頭從 0x40 起的 64 個位元組：魔數 `CITYMCRP`
// 之後跟著一段常數。四份獨立的原版存檔——軟體世界 1990 年地形編輯器磁片上的
// `TAIWAN.CTY` 與 `KAOHSIUN.CTY`，以及 DOS 1.02 的 `DETROIT.CTY` 與
// `HAMBURG.CTY`（1995／1997 的時間戳）——這 64 個位元組**逐位元相同**，
// 所以照抄。語意未解，原樣保留。
var dosHeaderTail = [64]byte{
	0x00, 0x43, 0x49, 0x54, 0x59, 0x4d, 0x43, 0x52, 0x50, 0x01, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x88, 0x00, 0x00, 0x00, 0x69, 0xf0, 0x00,
	0x00, 0x00, 0x00, 0x9f, 0xe5, 0xe4, 0x00, 0x9f, 0xf2, 0x69,
}

// maxHeaderNameBytes 是名稱欄放得下的位元組數：檔頭前 64 個位元組扣掉
// 兩個位元組的長度前綴。第 65 個位元組起是 dosHeaderTail，不能蓋掉。
const maxHeaderNameBytes = 64 - 2

// buildCityHeader 組出 128 位元組的 DOS 檔頭：長度前綴 ＋ 城市名 ＋ 零填 ＋
// 魔數與常數尾段。
//
// ⚠ 長度前綴寫的是**名稱的實際位元組數**，不是原版那個固定的 13。
// 原版觀察到的四份存檔一律寫 13，而它們的名稱都是 DOS 8.3 檔名（最長 8 字）；
// remake 的城市名可以到 17 個字元而且可能是中文，寫死 13 會把長名字截斷。
// 讀取端兩種都吃得下（取前綴長度再切到第一個 NUL），見 cityHeaderName。
func buildCityHeader(name string) []byte {
	h := make([]byte, cityHeaderLen)
	b := []byte(name)
	if len(b) > maxHeaderNameBytes {
		// 不能從中間切斷 UTF-8 字元，往前退到字元邊界。
		b = b[:maxHeaderNameBytes]
		for len(b) > 0 && !utf8.Valid(b) {
			b = b[:len(b)-1]
		}
	}
	binary.BigEndian.PutUint16(h, uint16(len(b)))
	copy(h[2:], b)
	copy(h[64:], dosHeaderTail[:])
	return h
}

// SaveFormat 是存檔要寫哪一種版面。兩種都是原版認得的格式，差別在檔頭。
type SaveFormat int

const (
	// SaveWithHeader 寫 DOS 存檔那種 27248：128 位元組檔頭 ＋ 27120 檔身。
	// **城市名唯一的容身處就是那個檔頭**，檔身裡沒有這個欄位。
	SaveWithHeader SaveFormat = iota
	// SaveBareBody 寫 27120 的裸檔身，餵得進 Micropolis，代價是沒有城市名。
	SaveBareBody
)

// String 讓設定檔與命令列用同一組字。
func (f SaveFormat) String() string {
	if f == SaveBareBody {
		return "bare"
	}
	return "dos"
}

// ParseSaveFormat 把設定檔或命令列的字轉成格式；不認得的回 false。
func ParseSaveFormat(s string) (SaveFormat, bool) {
	switch s {
	case "dos", "":
		return SaveWithHeader, true
	case "bare", "micropolis":
		return SaveBareBody, true
	}
	return SaveWithHeader, false
}

// SaveCity 把城市存成原版格式的 `.cty`，用預設的 DOS 存檔版面。
func SaveCity(path string, w *sim.World) error {
	return SaveCityAs(path, w, SaveWithHeader)
}

// SaveCityAs same，但由呼叫端指定版面。
//
// 兩種格式的取捨是玩家自己的：
//
//	SaveWithHeader  城市名存得住，但**餵不進 Micropolis**（它只認裸檔身）
//	SaveBareBody    餵得進原版 SimCity 與 Micropolis，但**城市名會掉**
//
// 讀取端兩種都吃得下，所以換格式不會讓舊存檔變成孤兒。
func SaveCityAs(path string, w *sim.World, f SaveFormat) error {
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
	if f == SaveBareBody {
		return os.WriteFile(path, b, 0o644)
	}
	out := make([]byte, 0, dosSaveSize)
	out = append(out, buildCityHeader(w.CityName)...)
	out = append(out, b...)
	return os.WriteFile(path, out, 0o644)
}

// dosSaveSize 是 DOS 原版自己存出來的城市檔大小。
const dosSaveSize = sim.CityFileSize1x1 + 128 // 27248

// psnSize 是解壓後的 DOS 劇本檔大小。**檔頭也是 128**，多出來的 16 個位元組
// 在檔尾，用途未解（docs/formats/01-city-file.md §三）。
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
	body, _, err := splitCityBytes(raw)
	return body, err
}

// cityHeaderLen 是 DOS 存檔與解壓後 `.PSN` 的檔頭長度。
const cityHeaderLen = 128

// splitCityBytes 把原始位元組切成檔身與檔頭。沒有檔頭時第二個回傳值是 nil。
func splitCityBytes(raw []byte) (body, header []byte, err error) {
	switch len(raw) {
	case sim.CityFileSize1x1:
		return raw, nil, nil
	case dosSaveSize, psnSize:
		return raw[cityHeaderLen : cityHeaderLen+sim.CityFileSize1x1], raw[:cityHeaderLen], nil
	}
	return nil, nil, fmt.Errorf("城市檔是 %d 位元組，不是 %d（無檔頭）、%d（DOS 存檔）"+
		"或 %d（解壓後的 DOS 劇本）—— 這可能是還沒解壓的 `.PSN`",
		len(raw), sim.CityFileSize1x1, dosSaveSize, psnSize)
}

// cityHeaderName 取出檔頭裡的城市名。
//
// 版面是大端 16 位元長度前綴 ＋ 名稱 ＋ 零填 ＋ 魔數 `CITYMCRP`
// （docs/formats/01-city-file.md §三）。
//
// ⚠ **長度前綴不可盡信**：`TAIWAN`（名稱 6 字）、`KAOHSIUN`（8 字）與
// `Joffebrg.cty`（12 字）三個檔的前綴都寫 13，而同一批其他檔的前綴確實等於
// 字串長度。所以取前綴長度之後還要切到第一個 NUL，否則會把補零一起讀進來。
// 前綴為什麼會是 13 未解——寫檔的程式可能不只一支。
// 樣本來源與逐檔雜湊見 docs/formats/00-e220-terrain-editor.md。
func cityHeaderName(header []byte) string {
	if len(header) < 3 {
		return ""
	}
	n := int(binary.BigEndian.Uint16(header))
	if n <= 0 || n > len(header)-2 {
		n = len(header) - 2
	}
	name := header[2 : 2+n]
	if i := bytes.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	return strings.TrimSpace(string(name))
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
	body, header, err := splitCityBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("%s：%w", filepath.Base(path), err)
	}
	cf, err := sim.ParseCityFile(body)
	if err != nil {
		return nil, err
	}
	w := sim.NewWorld(seed)
	w.LoadCityFile(cf)
	// 城市名在檔頭裡，不在檔身。沒有檔頭（Micropolis 那種 27120 的裸檔）
	// 就保留 NewWorld 的預設名。
	//
	if n := cityHeaderName(header); n != "" {
		w.CityName = n
	}
	w.InitSimLoad = 1
	w.DoSimInit()
	return w, nil
}
