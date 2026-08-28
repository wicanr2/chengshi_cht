package assets

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// DOS 版劇本檔（`.PSN`）。證據：docs/formats/02-dos-lzss.md §4
//
// 解壓之後是 27264 位元組：**144 位元組的檔頭 ＋ 27120 位元組的城市檔**，
// 而那 27120 與 Micropolis 的 `.cty` 是同一個版面（大端 16 位元、同樣的陣列順序）。
// 這件事直接證實 **DOS 版的地圖也是 120×100**。

const (
	// PSNTotalLen 是解壓後的總長度。
	PSNTotalLen = 27264
	// PSNHeaderLen 是城市資料前面的檔頭長度。
	PSNHeaderLen = 144
	// PSNBodyLen 是城市資料的長度，與 Micropolis 的 `.cty` 相同。
	PSNBodyLen = PSNTotalLen - PSNHeaderLen // 27120
)

// PSNMagic 出現在檔頭裡，是「這是一個城市檔」的標記。
var PSNMagic = []byte("CITYMCRP")

// Scenario 是一個解開的 DOS 劇本。
type Scenario struct {
	Name   string // 檔頭裡長度前綴的城市名
	Header []byte // 完整的 144 位元組檔頭，語意大多未解，原樣保留
	Body   []byte // 27120 位元組的城市資料，可直接餵給 sim.ParseCityFile
}

// LoadPSN 解壓並解析一個 `.PSN` 檔。
func LoadPSN(raw []byte) (*Scenario, error) {
	d, err := DecompressLZSS(raw)
	if err != nil {
		return nil, err
	}
	if len(d) != PSNTotalLen {
		return nil, fmt.Errorf("劇本解出 %d 位元組，應為 %d", len(d), PSNTotalLen)
	}
	if !bytes.Contains(d[:PSNHeaderLen], PSNMagic) {
		return nil, fmt.Errorf("檔頭裡找不到 %q 標記", PSNMagic)
	}
	// 城市名是大端 16 位元長度前綴的字串。
	n := int(binary.BigEndian.Uint16(d[0:2]))
	if n < 0 || 2+n > PSNHeaderLen {
		return nil, fmt.Errorf("城市名長度 %d 不合理", n)
	}
	return &Scenario{
		Name:   string(d[2 : 2+n]),
		Header: append([]byte(nil), d[:PSNHeaderLen]...),
		Body:   append([]byte(nil), d[PSNHeaderLen:]...),
	}, nil
}
