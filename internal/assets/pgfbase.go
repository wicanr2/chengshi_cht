package assets

import (
	"encoding/binary"
	"fmt"
)

// 基本圖形檔（`CEGADAT.PGF`、`mcgadat.pgf`、`segadat.pgf`、`MONODAT.PGF`）
// 是「沒有資料片的原始城市外觀」。它和六個風格檔是**兩種版面**：
//
//   - 風格檔：CP437 橫幅 ＋ 名稱／模式／配套檔名 ＋ 壓縮流，流裡有一張
//     十二位元組一列的圖形庫表。
//   - 基本檔：沒有橫幅、沒有那張表。第一個位元組就是壓縮流，解開之後
//     直接是資料。
//
// 基本檔的圖形庫表是**行內**的：每一個庫前面三個位元組是
// `u8 平面數` ＋ `u16 資料長度`，庫裡每一張圖前面四個位元組是
// `u16 寬` ＋ `u16 高`。第 0 庫（960 張地圖圖塊）例外，它沒有任何表頭。
//
// 版面細節與判準見 docs/formats/03-pgf-graphics.md §8。

// egaPalette 是標準 EGA 十六色，取自風格檔自己帶的調色盤（實測值，
// 不是教科書上的 0x00/0x55/0xaa/0xff）。4 位元的基本檔不帶調色盤，
// 因為它用的就是這一組。
// ⚠ 值是**螢幕上顯示的**標準 EGA 四階 `0x00／0x55／0xaa／0xff`，
// 不是檔案裡存的 `0x00／0x50／0xa0／0xf0`——後者是六位元 VGA 值乘 4 的
// 近似值，照抄會讓每個像素偏暗一階。理由與量法見 `pgf.go` 的 `egaLevels`。
var egaPalette = [16][3]uint8{
	{0x00, 0x00, 0x00}, {0x00, 0x00, 0xaa}, {0x00, 0xaa, 0x00}, {0x00, 0xaa, 0xaa},
	{0xaa, 0x00, 0x00}, {0xaa, 0x00, 0xaa}, {0xaa, 0x55, 0x00}, {0xaa, 0xaa, 0xaa},
	{0x55, 0x55, 0x55}, {0x55, 0x55, 0xff}, {0x55, 0xff, 0x55}, {0x55, 0xff, 0xff},
	{0xff, 0x55, 0x55}, {0xff, 0x55, 0xff}, {0xff, 0xff, 0x55}, {0xff, 0xff, 0xff},
}

// monoPalette 是單色模式的兩色，同樣取自 `ASIAMONO.PGF`。
var monoPalette = [2][3]uint8{{0x00, 0x00, 0x00}, {0xff, 0xff, 0xff}}

// LoadPGFBase 讀一個基本圖形檔。
//
// tile 是地圖圖塊的邊長（CEGA／MONO 是 16，mcga／sega 是 8），
// bpp 是位元深度（CEGA／sega 4、mcga 8、MONO 1）。這兩個值由目錄決定，
// 檔案裡沒寫——基本檔連位元深度都沒有欄位可放。
func LoadPGFBase(raw []byte, tile, bpp int) (*PGF, error) {
	data, err := DecompressLZSS(raw)
	if err != nil {
		return nil, fmt.Errorf("解壓失敗：%w", err)
	}
	g := &PGF{Name: "基本", BitsPerPixel: bpp}

	off := 0
	switch bpp {
	case 8:
		// 256 色的 6 位元 VGA 值（0x00–0x3f），要放大到 8 位元。
		if len(data) < 768 {
			return nil, fmt.Errorf("解出來只有 %d 位元組，放不下調色盤", len(data))
		}
		for i := 0; i < 256; i++ {
			g.Palette = append(g.Palette, PGFColor{
				R: data[i*3] * 255 / 63, G: data[i*3+1] * 255 / 63, B: data[i*3+2] * 255 / 63,
			})
		}
		off = 768
	case 4:
		for _, c := range egaPalette {
			g.Palette = append(g.Palette, PGFColor{R: c[0], G: c[1], B: c[2]})
		}
	case 1:
		for _, c := range monoPalette {
			g.Palette = append(g.Palette, PGFColor{R: c[0], G: c[1], B: c[2]})
		}
	default:
		return nil, fmt.Errorf("位元深度 %d 不是 1/4/8", bpp)
	}

	// 第 0 庫：960 張地圖圖塊，沒有逐張表頭。
	planes := 4
	if bpp == 1 {
		planes = 1
	}
	one := baseImageBytes(tile, tile, planes, bpp)
	end := off + tileCount*one
	if end > len(data) {
		return nil, fmt.Errorf("第 0 庫要 %d 位元組，檔案只有 %d", end, len(data))
	}
	b0 := PGFBank{Width: tile, Height: tile}
	for i := 0; i < tileCount; i++ {
		px, _, err := pgfPixels(data, off+i*one, tile, tile, bpp, flagsFor(planes, bpp))
		if err != nil {
			return nil, fmt.Errorf("第 0 庫第 %d 張：%w", i, err)
		}
		b0.Images = append(b0.Images, PGFImage{Pixels: px})
	}
	g.Banks = append(g.Banks, b0)

	// 第 0 庫與行內圖形庫表之間夾著一塊共用的資料（六個風格檔裡它在
	// 檔尾，內容一模一樣），長度隨模式不同。它不一定是 3 位元組表頭的
	// 格式，所以往後掃到**第一個能一路走到檔尾的位移**——二十幾個庫
	// 全部要對齊才算數，撞對的機率可以忽略。
	start := -1
	for s := end; s < len(data) && s < end+65536; s++ {
		if _, ok := walkBaseBanks(data, s, bpp, false); ok {
			start = s
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("從 %d 起找不到行內的圖形庫表", end)
	}
	banks, _ := walkBaseBanks(data, start, bpp, true)
	g.Banks = append(g.Banks, banks...)
	// 第 0 庫與行內庫表之間那一塊就是地圖縮圖（＋單色／256 色的介面字型）。
	if mini, next := parseMiniTiles(data[end:start], bpp); mini != nil {
		g.Mini = mini
		g.Fonts = parsePGFFonts(data[end+next:start], bpp)
	}
	return g, nil
}

const tileCount = 960

func flagsFor(planes, bpp int) uint16 {
	if bpp != 8 && planes == 1 {
		return pgfSinglePlane
	}
	return 0
}

func baseImageBytes(w, h, planes, bpp int) int {
	if bpp == 8 {
		return w * h
	}
	return ((w + 7) / 8) * planes * h
}

// walkBaseBanks 從 off 走完行內的圖形庫表。
//
// collect 為假時只驗證走不走得到檔尾——掃描起點時用，省掉解像素的成本。
func walkBaseBanks(data []byte, off, bpp int, collect bool) ([]PGFBank, bool) {
	var out []PGFBank
	n := 0
	for off+3 <= len(data) {
		planes := int(data[off])
		ln := int(binary.LittleEndian.Uint16(data[off+1:]))
		if (planes != 1 && planes != 4 && planes != 8) || ln == 0 || off+3+ln > len(data) {
			return nil, false
		}
		q, end := off+3, off+3+ln
		var bank PGFBank
		bank.Flags = pgfPerImageHead | flagsFor(planes, bpp)
		for q < end {
			if q+4 > end {
				return nil, false
			}
			w := int(binary.LittleEndian.Uint16(data[q:]))
			h := int(binary.LittleEndian.Uint16(data[q+2:]))
			if w <= 0 || w > 640 || h <= 0 || h > 400 {
				return nil, false
			}
			if bank.Width == 0 {
				bank.Width, bank.Height = w, h
			}
			if collect {
				px, _, err := pgfPixels(data, q+4, w, h, bpp, bank.Flags&pgfSinglePlane)
				if err != nil {
					return nil, false
				}
				bank.Images = append(bank.Images, PGFImage{Pixels: px})
			} else {
				bank.Images = append(bank.Images, PGFImage{})
			}
			q += 4 + baseImageBytes(w, h, planes, bpp)
		}
		if q != end {
			return nil, false
		}
		out = append(out, bank)
		off = end
		n++
	}
	// 走到檔尾、而且庫數合理才算數
	return out, off == len(data) && n >= 10
}

// EGAPalette 回傳實測的 EGA 十六色。給工具用（cmd/pgfblk）。
func EGAPalette() []PGFColor {
	out := make([]PGFColor, 0, 16)
	for _, c := range egaPalette {
		out = append(out, PGFColor{R: c[0], G: c[1], B: c[2]})
	}
	return out
}
