package assets

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// 訊息檔（`.PTF`）。證據：docs/formats/04-ptf-messages.md
//
// 解壓之後是一串**段落**，一段一個 UI 用途（狀態訊息、工具造價、
// 月份名、選單項目…）：
//
//	u16 段落長度（不含這四個位元組）
//	u16 筆數
//	段落內容：連續的 NUL 結尾字串
//
// ⚠ 「筆數」不等於字串數。第 0 段（狀態訊息）每一筆是**兩個字串**——
// 文字加上兩個位元組的屬性（`0xFE` 開頭，第二個位元組疑似音效或嚴重度）。
// 其餘段落一筆一個字串。
//
// 這個差別是最容易踩的坑：把整份檔案當成「文字、屬性」交替去讀，
// 月份會變成「一月、三月、五月……」——**二月被當成一月的屬性吃掉了**，
// 而且輸出看起來完全合理，只是少了一半。

// Section 是訊息檔裡的一個段落。
type Section struct {
	Index  int      // 段落序號
	Count  int      // 檔案宣告的筆數
	Strings []string // 段落內的字串，順序即索引
}

// Message 是一筆訊息。中文化以 段落.索引 為鍵。
type Message struct {
	Section int
	Index   int
	Text    string
}

// ParsePTF 解一個**已經解壓**的訊息檔。
func ParsePTF(data []byte) ([]Section, error) {
	var out []Section
	p := 0
	for p+4 <= len(data) {
		size := int(binary.LittleEndian.Uint16(data[p:]))
		count := int(binary.LittleEndian.Uint16(data[p+2:]))
		if size == 0 {
			break
		}
		end := p + 4 + size
		if end > len(data) {
			end = len(data)
		}
		body := data[p+4 : end]
		s := Section{Index: len(out), Count: count}
		q := 0
		for q < len(body) {
			z := bytes.IndexByte(body[q:], 0)
			if z < 0 {
				z = len(body) - q
			}
			s.Strings = append(s.Strings, string(body[q:q+z]))
			q += z + 1
		}
		out = append(out, s)
		p = end
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("解不出任何段落，這可能不是 .PTF")
	}
	return out, nil
}

// LoadPTF 解壓並解析一個 `.PTF` 檔。
func LoadPTF(raw []byte) ([]Section, error) {
	d, err := DecompressLZSS(raw)
	if err != nil {
		return nil, fmt.Errorf("解壓失敗：%w", err)
	}
	return ParsePTF(d)
}

// TextMessages 抽出所有可翻譯的字串，過濾掉屬性位元組與純控制字元。
//
// 判準是**內容**不是位置：一筆字串只要含有可列印的 ASCII 就是文字。
// 用位置判斷（「偶數是文字、奇數是屬性」）在段落 0 以外會全錯。
func TextMessages(secs []Section) []Message {
	var out []Message
	for _, s := range secs {
		for i, str := range s.Strings {
			if !hasPrintable(str) {
				continue
			}
			out = append(out, Message{Section: s.Index, Index: i, Text: trimControl(str)})
		}
	}
	return out
}

func hasPrintable(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x20 && s[i] < 0x7f && s[i] != ' ' {
			return true
		}
	}
	return false
}

// trimControl 去掉字串前面的控制位元組。
//
// ⚠ 圖片訊息（段落 2）的前面有兩到三個位元組的前綴（例如 `\xfe\xf4\xff`），
// 那是圖片編號之類的中繼資料，不是文字的一部分。留著會讓譯文長度算錯，
// 也會在畫面上出現亂碼。
func trimControl(s string) string {
	i := 0
	for i < len(s) && (s[i] < 0x20 && s[i] != '\n') {
		i++
	}
	// 有些前綴後面還跟著一個高位位元組（0xff），一併去掉。
	for i < len(s) && s[i] >= 0x7f {
		i++
	}
	return s[i:]
}
