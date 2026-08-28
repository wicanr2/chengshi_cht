// Package i18n 提供遊戲內的中文文字。
//
// 文本在 translations/messages/*.toml，以「段落.索引」為鍵——那是原版
// `.PTF` 的結構（見 docs/formats/04-ptf-messages.md）。**原文不進版控**，
// 翻譯檔只有鍵與譯文。
//
// 這個套件不相依 Ebiten，所以測試在無頭環境跑得起來。
package i18n

import (
	"embed"
	"fmt"
	"path"
	"strings"
)

//go:embed messages/*.toml
var files embed.FS

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
	SecMapTitle = 10 // 地圖視窗的十種全貌圖
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

// Catalog 是一個風格的文字表。
type Catalog struct {
	style string
	text  map[string]string
}

// Load 讀進某個風格的文字。風格代號不認得時退回基本檔。
func Load(style string) (*Catalog, error) {
	name, ok := styleFile[style]
	if !ok {
		name = "message"
	}
	m, err := parseFile(name)
	if err != nil {
		return nil, err
	}
	return &Catalog{style: style, text: m}, nil
}

// S 取一筆文字。查無或未翻譯時回空字串——**不要回鍵名或問號**：
// 空字串在畫面上是「少了一句話」，看得出來；回 "0.12" 之類的鍵
// 會被誤讀成遊戲資料，而且截圖看起來像正常內容。
func (c *Catalog) S(section, index int) string {
	return c.text[fmt.Sprintf("%d.%d", section, index)]
}

// Has 回報某一筆有沒有譯文。
func (c *Catalog) Has(section, index int) bool {
	_, ok := c.text[fmt.Sprintf("%d.%d", section, index)]
	return ok
}

// Count 回傳已翻譯的筆數，給測試與工具用。
func (c *Catalog) Count() int { return len(c.text) }

// parseFile 讀一個翻譯檔。
//
// 格式是我們自己產生的 TOML 子集，只有三種行：
//
//	["段落.索引"]
//	len = 數字
//	zh = "譯文"
//
// 所以不引第三方 TOML 套件。⚠ 但**跳脫序列要處理**：譯文裡有 `\n`
// （圖片訊息換行）與 `\"`。忽略它們的話長訊息會擠成一行。
func parseFile(name string) (map[string]string, error) {
	raw, err := files.ReadFile(path.Join("messages", name+".toml"))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	key := ""
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, `["`) && strings.HasSuffix(line, `"]`):
			key = line[2 : len(line)-2]
		case strings.HasPrefix(line, "zh = \""):
			if key == "" {
				continue
			}
			v := unquote(line[len("zh = "):])
			if v != "" {
				out[key] = v
			}
			key = ""
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s 裡一條譯文都沒有", name)
	}
	return out, nil
}

// unquote 解開 TOML 的雙引號字串。
func unquote(s string) string {
	if len(s) < 2 || s[0] != '"' {
		return ""
	}
	s = s[1:]
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			break
		}
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
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
		case '"':
			b.WriteByte('"')
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
	39:  7,  // 超級都會
	38:  8,  // 大都會
	37:  9,  // 首府
	36:  10, // 城市
	35:  11, // 城鎮
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
