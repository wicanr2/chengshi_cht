// Package i18n 提供遊戲內的文字，目前四種語言。
//
// 文本在 `messages/*.tsv`，一列一筆，以「段落.索引」為鍵——那是原版
// `.PTF` 的結構（見 docs/formats/04-ptf-messages.md）。
//
// ⚠ **原文不進版控。** 所以 TSV 只有 `zh_hant`／`zh_hans`／`ja` 三欄，
// 英文那一欄不存在——英文是**執行時從玩家自己那份 `.PTF` 讀出來的**
// （`SetEnglish`）。這不是省事，是 CLAUDE.md §8 的界線：原版的文字
// 屬於原權利人。
//
// TSV 是**唯一真相**，沒有產生器會覆寫它。先前那條「python 表 → toml」
// 的路把直接編在產出檔裡的六筆譯文洗掉過（asia 的「水車」變回
// 「核能發電廠」），所以工具改成只補空格子（`tools/i18n/fill_lang.py`）。
//
// 這個套件不相依 Ebiten，所以測試在無頭環境跑得起來。
package i18n

import (
	"embed"
	"fmt"
	"path"
	"strings"
)

//go:embed messages/*.tsv
var files embed.FS

// Lang 是語言代號，同 TSV 的欄名。
type Lang string

const (
	ZhHant Lang = "zh_hant" // 繁體中文（本專案的主語言）
	ZhHans Lang = "zh_hans" // 簡體中文（由繁體字面轉換，見 tools/i18n/hant2hans.py）
	Ja     Lang = "ja"      // 日文（本專案新譯，不是原版日文版的用語）
	En     Lang = "en"      // 英文 ＝ 玩家自己那份 `.PTF` 的原文
)

// Langs 是可以切換的語言，順序就是選單裡的順序。
var Langs = []Lang{ZhHant, ZhHans, Ja, En}

// LangName 是語言在選單裡顯示的名字，各自用自己的語言寫。
var LangName = map[Lang]string{
	ZhHant: "繁體中文", ZhHans: "简体中文", Ja: "日本語", En: "English",
}

// ParseLang 認得寬鬆一點的寫法，給命令列用。
func ParseLang(s string) (Lang, bool) {
	switch strings.ToLower(strings.NewReplacer("-", "_", ".", "_").Replace(s)) {
	case "zh_hant", "zh_tw", "zh", "hant", "cht":
		return ZhHant, true
	case "zh_hans", "zh_cn", "hans", "chs":
		return ZhHans, true
	case "ja", "jp", "ja_jp":
		return Ja, true
	case "en", "en_us", "english":
		return En, true
	}
	return ZhHant, false
}

// 原版 .PTF 的段落編號。用具名常數而不是裸數字：
// 「第 14 段的第 19 筆」看不出是什麼，`SecTileName, 19` 看得出。
const (
	SecStatus   = 0  // 狀態訊息（訊息欄）
	SecToolCost = 1  // 工具名稱與造價
	SecPicture  = 2  // 圖片訊息與劇本簡介
	SecBudget   = 3  // 預算欄位
	SecMonth    = 4  // 月份
	SecPowerSub = 5  // 發電廠副選單
	SecGraph    = 7  // 統計圖曲線
	SecMapTitle = 10 // 地圖視窗的十一種全貌圖
	// SecMapSub 是兩個「一格管兩個圖層」的圖示按住時跳出來的小選單：
	// 人口分佈／警力範圍／消防範圍／人口成長。九個圖示對十一個圖層，
	// 差的兩個就在這裡（實測見 docs/spec/controls.md §九）。
	SecMapSub   = 11
	SecClass    = 12 // 城市等級
	SecProblem  = 13 // 評估視窗的嚴重問題
	SecTileName = 14 // 查詢工具的地物名稱
	SecQuery    = 15 // 查詢工具的欄位
	SecMenu     = 16 // 主選單
	SecSysMenu  = 17 // 系統選單
	SecOptMenu  = 18 // 功能選單
	SecSpeed    = 19 // 速度副選單
	SecDisaster = 20 // 災難選單
	SecWinMenu  = 21 // 視窗選單
)

// 六個城市風格對應的翻譯檔前綴。
var styleFile = map[string]string{
	"asia": "asia_msg",
	"medi": "medi_msg",
	"west": "west_msg",
	"fusa": "fusa_msg",
	"feur": "feur_msg",
	"moon": "moon_msg",
}

// Catalog 是一個風格的文字表，四種語言放在同一份裡。
type Catalog struct {
	style string
	lang  Lang
	// text[語言][段落.索引]。英文那一層由 SetEnglish 從玩家的 `.PTF` 填。
	text map[Lang]map[string]string
	// base 是基本檔（`message.tsv`）。資料片只覆寫**用詞和基本檔不同**的鍵，
	// 其餘要從基本檔取；沒有這一層的話，載入資料片會讓非繁體的語言
	// 整片退回繁中。
	base map[Lang]map[string]string
}

// Load 讀進某個風格的文字。風格代號不認得時退回基本檔。
func Load(style string) (*Catalog, error) { return LoadLang(style, ZhHant) }

// LoadLang 同 Load，但指定語言。
func LoadLang(style string, lang Lang) (*Catalog, error) {
	name, ok := styleFile[style]
	if !ok {
		name = "message"
	}
	m, err := parseFile(name)
	if err != nil {
		return nil, err
	}
	c := &Catalog{style: style, lang: lang, text: m}
	if name != "message" {
		if b, err := parseFile("message"); err == nil {
			c.base = b
		}
	}
	return c, nil
}

// Lang 回傳目前的語言。
func (c *Catalog) Lang() Lang { return c.lang }

// SetLang 換語言。文字表已經載進來了，換語言不必重讀檔案。
func (c *Catalog) SetLang(l Lang) { c.lang = l }

// SetEnglish 把英文原文餵進來。來源是**玩家自己那份 `.PTF`**，
// 呼叫端負責讀檔（見 internal/game）。sections 的外層索引是段落編號。
func (c *Catalog) SetEnglish(sections [][]string) {
	m := map[string]string{}
	for si, list := range sections {
		for i, s := range list {
			if s != "" {
				m[fmt.Sprintf("%d.%d", si, i)] = s
			}
		}
	}
	c.text[En] = m
}

// HasLang 回報某個語言有沒有任何文字，給選單決定要不要停用那一項。
func (c *Catalog) HasLang(l Lang) bool { return len(c.text[l]) > 0 }

// lookup 依語言查一筆，查不到就走退路：
//
//	資料片的指定語言 → 基本檔的指定語言 →
//	資料片的繁體中文 → 基本檔的繁體中文 → 英文原文 → 空字串
//
// 資料片排在基本檔前面，「水車」才不會被基本檔的「核能發電廠」蓋掉。
//
// ⚠ 退路的終點是**空字串，不是鍵名**。空字串在畫面上是「少了一句話」，
// 看得出來；回 "0.12" 之類的鍵會被誤讀成遊戲資料，而且截圖看起來
// 像正常內容。
func (c *Catalog) lookup(key string) string {
	for _, l := range []Lang{c.lang, ZhHant} {
		if v := c.text[l][key]; v != "" {
			return v
		}
		if v := c.base[l][key]; v != "" {
			return v
		}
	}
	return c.text[En][key]
}

// S 取一筆文字。查無或未翻譯時回空字串——**不要回鍵名或問號**：
// 空字串在畫面上是「少了一句話」，看得出來；回 "0.12" 之類的鍵
// 會被誤讀成遊戲資料，而且截圖看起來像正常內容。
func (c *Catalog) S(section, index int) string {
	return c.lookup(fmt.Sprintf("%d.%d", section, index))
}

// Has 回報某一筆有沒有文字（走同一條退路）。
func (c *Catalog) Has(section, index int) bool {
	return c.lookup(fmt.Sprintf("%d.%d", section, index)) != ""
}

// Count 回傳目前語言有幾筆文字，給測試與工具用。
func (c *Catalog) Count() int { return len(c.text[c.lang]) }

// CountLang 回傳某個語言有幾筆。
func (c *Catalog) CountLang(l Lang) int { return len(c.text[l]) }

// parseFile 讀一個翻譯檔。
//
// 格式是 TSV，第一列是欄名：
//
//	key	len	zh_hant	zh_hans	ja
//
// 選 TSV 不選 TOML 的理由：**同一個鍵的各語言排在同一列**，
// 翻譯的人一眼看得到彼此，diff 也只動那一列；TOML 那種一鍵一段的排法
// 每加一個語言就要多一行分散在檔案各處。
//
// ⚠ 跳脫序列要處理：譯文裡有換行（圖片訊息）與 tab。TSV 本身用 tab
// 分欄，所以欄位裡的 tab 一定得跳脫，否則整列的欄位會位移，
// 而檔案看起來還是「合法的 TSV」。
func parseFile(name string) (map[Lang]map[string]string, error) {
	raw, err := files.ReadFile(path.Join("messages", name+".tsv"))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("%s 是空的", name)
	}
	head := strings.Split(lines[0], "\t")
	out := map[Lang]map[string]string{}
	col := map[int]Lang{}
	for i, h := range head {
		switch Lang(h) {
		case ZhHant, ZhHans, Ja:
			col[i] = Lang(h)
			out[Lang(h)] = map[string]string{}
		}
	}
	if len(col) == 0 {
		return nil, fmt.Errorf("%s 的欄名裡沒有任何語言欄", name)
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, "\t")
		key := f[0]
		if key == "" {
			continue
		}
		for i, l := range col {
			if i < len(f) && f[i] != "" {
				out[l][key] = unescapeTSV(f[i])
			}
		}
	}
	if len(out[ZhHant]) == 0 {
		return nil, fmt.Errorf("%s 裡一條繁體譯文都沒有", name)
	}
	return out, nil
}

// unescapeTSV 解開 TSV 欄位的跳脫。
func unescapeTSV(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case '\\':
			b.WriteByte('\\')
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// 圖片訊息的編號對照。
//
// ⚠ 第 2 段的索引**不是**訊息編號。編號寫在原版檔案裡每一筆的
// 三位元組前綴（見 docs/formats/04-ptf-messages.md §7），順序是
// 交通、犯罪、爐心熔毀、勝利、彈劾、怪獸、污染，然後五個人口里程碑，
// 最後八個劇本簡介——既不是遞增也不是遞減。
//
// 這張表是從原版檔案讀出來的，不是猜的；`simtool prefix` 可以重印。
var pictureIndex = map[int]int{
	12:  0,  // 交通壅塞
	11:  1,  // 犯罪
	43:  2,  // 爐心熔毀
	100: 3,  // 劇本過關
	200: 4,  // 劇本失敗
	21:  5,  // 怪獸
	10:  6,  // 污染
	39:  7,  // 大都會區
	38:  8,  // 大都會
	37:  9,  // 首都
	36:  10, // 城市
	35:  11, // 小鎮
}

// scenarioPicture 是八個劇本簡介在第 2 段的索引。
// 順序是原版檔案裡的順序，不是劇本編號。
var scenarioPicture = map[int]int{
	6: 12, 8: 13, 5: 14, 3: 15, 2: 16, 1: 17, 7: 18, 4: 19,
}

// Picture 回傳某個訊息編號對應的圖片訊息全文；沒有圖片就回空字串。
//
// 傳入的是**正數**訊息編號（模擬層送出的是負數，代表「有圖」）。
func (c *Catalog) Picture(msg int) string {
	if msg < 0 {
		msg = -msg
	}
	i, ok := pictureIndex[msg]
	if !ok {
		return ""
	}
	return c.S(SecPicture, i)
}

// ScenarioBrief 回傳第 n 個劇本（1–8）的簡介全文。
func (c *Catalog) ScenarioBrief(n int) string {
	i, ok := scenarioPicture[n]
	if !ok {
		return ""
	}
	return c.S(SecPicture, i)
}

// ── 介面字串（remake 自己的字，不在原版的 `.PTF` 裡）────────────────

// uiText 是 `messages/ui.tsv`，全遊戲共用一份（不隨圖形集換）。
//
// 為什麼要分開一個檔：`.PTF` 的字是**原版的**，鍵是「段落.索引」，
// 英文從玩家的檔案讀；這裡的字是**remake 自己加的**——「資金」那一列的
// 排版、縮小的提示、語言與音樂選單——原版執行檔裡有的（`Funds:`）我們
// 也沒有權利散布，所以英文那一欄是本專案自己寫的，不是原版字串。
var uiText map[Lang]map[string]string

// UI 取一筆介面字串。查無時回鍵名本身——**這裡與 `S` 相反**：
// 介面字串是排版的骨架（欄位標題、按鈕），少一句話會讓版面塌掉，
// 看到鍵名至少知道是哪一筆漏了。
func (c *Catalog) UI(key string) string {
	if uiText == nil {
		uiText, _ = parseUI()
	}
	for _, l := range []Lang{c.lang, ZhHant, En} {
		if v := uiText[l][key]; v != "" {
			return v
		}
	}
	return key
}

// UICount 回傳某個語言有幾筆介面字串，給測試用。
func UICount(l Lang) int {
	if uiText == nil {
		uiText, _ = parseUI()
	}
	return len(uiText[l])
}

func parseUI() (map[Lang]map[string]string, error) {
	raw, err := files.ReadFile("messages/ui.tsv")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	head := strings.Split(lines[0], "\t")
	out := map[Lang]map[string]string{}
	col := map[int]Lang{}
	for i, h := range head {
		switch Lang(h) {
		case ZhHant, ZhHans, Ja, En:
			col[i] = Lang(h)
			out[Lang(h)] = map[string]string{}
		}
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, "\t")
		for i, l := range col {
			if i < len(f) && f[i] != "" {
				out[l][f[0]] = unescapeTSV(f[i])
			}
		}
	}
	return out, nil
}
