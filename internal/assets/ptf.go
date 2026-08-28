package assets

import (
	"bytes"
	"fmt"
)

// 訊息檔（`.PTF`）。證據：docs/formats/02-dos-lzss.md §3
//
// 解壓之後的版面：4 個位元組的檔頭（三個資料位元組 ＋ 一個 NUL），
// 接著一串記錄。每筆記錄是「NUL 結尾的字串」＋「NUL 結尾的標記」，
// 標記第一個位元組是 0xFE，第二個位元組看起來是分類（訊息嚴重程度或音效編號），
// 尚未證實。
//
// 檔頭的三個位元組：前兩個是小端 16 位元（各檔不同，1438…1590，語意未解），
// 第三個固定是 0x32。**未解，原樣保留。**

// PTFHeaderLen 是解壓後檔頭的長度（含結尾的 NUL）。
const PTFHeaderLen = 4

// Message 是訊息檔裡的一筆。
type Message struct {
	Index int    // 在檔案裡的序號。**中文化以序號為鍵，不以原文為鍵**
	Text  string // 原文
	Mark  []byte // 記錄後面的標記位元組，語意未解，原樣保留
}

// ParsePTF 解一個已經解壓的訊息檔。
func ParsePTF(data []byte) ([]Message, error) {
	if len(data) < PTFHeaderLen {
		return nil, fmt.Errorf("訊息檔只有 %d 位元組，連檔頭都不夠", len(data))
	}
	var out []Message
	i := PTFHeaderLen
	for i < len(data) {
		j := bytes.IndexByte(data[i:], 0)
		if j < 0 {
			break
		}
		text := string(data[i : i+j])
		i += j + 1
		var mark []byte
		if i < len(data) {
			k := bytes.IndexByte(data[i:], 0)
			if k < 0 {
				k = len(data) - i
			}
			mark = append(mark, data[i:i+k]...)
			i += k + 1
		}
		out = append(out, Message{Index: len(out), Text: text, Mark: mark})
	}
	return out, nil
}

// LoadPTF 解壓並解析一個 `.PTF` 檔。
func LoadPTF(raw []byte) ([]Message, error) {
	d, err := DecompressLZSS(raw)
	if err != nil {
		return nil, err
	}
	return ParsePTF(d)
}
