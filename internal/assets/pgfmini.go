package assets

import "encoding/binary"

// 地圖視窗（City Form）的縮圖與介面字型。
//
// 這一塊資料躺在**風格檔宣告的圖形庫之後**、**基本檔第 0 庫與行內庫表
// 之間**，兩處位元組相同（MOON 例外，它自己一份）。它的表頭是三個位元組：
//
//	u8  位元深度（4 平面寫 4、256 色寫 8）
//	u16 縮圖資料的長度
//
// 長度一律是 960 的倍數——每一張地圖圖塊一份縮圖。除以 960 得到一張佔幾個
// 位元組，再除以平面數得到高度：
//
//	CEGA 11 520 ÷ 960 = 12 ÷ 4 平面 = 3 列 → 3×3
//	MONO  2 880 ÷ 960 =  3 ÷ 1 平面 = 3 列 → 3×3
//	sega  3 840 ÷ 960 =  4 ÷ 4 平面 = 1 列 → 3×1
//	mcga    960 ÷ 960 =  1（256 色，一格一個色號）→ 1×1
//
// 寬度沒有寫在檔案裡，是從內容反推的：CEGA 與 MONO 的縮圖區**每一個位元組
// 的低五位都是 0**，只有最高三位在動——所以一列只有三個像素。3×3 也正好
// 對上原版地圖視窗量到的 360×300（120×100 格），見
// docs/formats/03-pgf-graphics.md §7。
//
// 縮圖之後，單色與 256 色模式還接著介面字型（各 128 字）。EGA 彩色模式
// 沒有，那兩個模式用 BIOS ROM 的字型，尾巴補的是 0x1a。
const (
	miniWidth  = 3   // 平面式縮圖的像素寬（反推值，見上）
	fontGlyphs = 128 // 自帶字型的字數
)

// MiniTiles 是 960 張地圖圖塊的縮圖，地圖視窗一格畫一張。
type MiniTiles struct {
	Width, Height int
	Pixels        []uint8 // 960 張，一張 Width*Height 個調色盤色號
}

// Tile 回傳第 n 張縮圖的色號，長度 Width*Height。
func (m *MiniTiles) Tile(n int) []uint8 {
	per := m.Width * m.Height
	if n < 0 || (n+1)*per > len(m.Pixels) {
		return nil
	}
	return m.Pixels[n*per : (n+1)*per]
}

// PGFFont 是檔案自帶的一份點陣字型。
type PGFFont struct {
	Width, Height int
	Pixels        []uint8 // Count 個字，一個字 Width*Height 個色號
}

// Count 是字數。
func (f *PGFFont) Count() int {
	per := f.Width * f.Height
	if per == 0 {
		return 0
	}
	return len(f.Pixels) / per
}

// parseMiniTiles 讀縮圖區，回傳縮圖與它之後的位移。
// 讀不出合理的版面就回 nil，讓呼叫端照舊當成未解位元組。
func parseMiniTiles(d []byte, bpp int) (*MiniTiles, int) {
	if len(d) < 3 {
		return nil, 0
	}
	n := int(binary.LittleEndian.Uint16(d[1:]))
	if n == 0 || 3+n > len(d) || n%tileCount != 0 {
		return nil, 0
	}
	per := n / tileCount
	body := d[3 : 3+n]
	m := &MiniTiles{}
	planes := 4
	switch bpp {
	case 8:
		// 256 色：一格就是一個色號，沒有平面。
		m.Width, m.Height = 1, per
		m.Pixels = append(m.Pixels, body...)
		return m, 3 + n
	case 1:
		planes = 1
	case 4:
	default:
		return nil, 0
	}
	if per%planes != 0 {
		return nil, 0
	}
	m.Width, m.Height = miniWidth, per/planes
	m.Pixels = make([]uint8, tileCount*m.Width*m.Height)
	for t := 0; t < tileCount; t++ {
		for y := 0; y < m.Height; y++ {
			for pl := 0; pl < planes; pl++ {
				// 平面順序**倒著**，與 pgfPixels 一致：一列的第一個位元組是最高位平面。
				bit := planes - 1 - pl
				if planes == 1 {
					bit = 0
				}
				b := body[t*per+y*planes+pl]
				for x := 0; x < m.Width; x++ {
					if b&(0x80>>uint(x)) != 0 {
						m.Pixels[(t*m.Height+y)*m.Width+x] |= 1 << uint(bit)
					}
				}
			}
		}
	}
	return m, 3 + n
}

// parsePGFFonts 讀縮圖之後的介面字型。
//
// ⚠ 長度要寫死，不能「把剩下的全吃掉」——基本檔的縮圖後面接的是行內
// 圖形庫表，吃掉就把表當成字型了。
func parsePGFFonts(d []byte, bpp int) []PGFFont {
	switch bpp {
	case 1:
		// 8×8 與 8×14 各 128 字，單色。
		if len(d) < fontGlyphs*(8+14) {
			return nil
		}
		return []PGFFont{
			mono1bpp(d[:fontGlyphs*8], 8),
			mono1bpp(d[fontGlyphs*8:fontGlyphs*(8+14)], 14),
		}
	case 8:
		// 8×8，一個像素一個位元組。
		const n = fontGlyphs * 8 * 8
		if len(d) < n {
			return nil
		}
		return []PGFFont{{Width: 8, Height: 8, Pixels: append([]uint8(nil), d[:n]...)}}
	}
	return nil
}

func mono1bpp(d []byte, h int) PGFFont {
	f := PGFFont{Width: 8, Height: h, Pixels: make([]uint8, len(d)*8)}
	for i, b := range d {
		for x := 0; x < 8; x++ {
			if b&(0x80>>uint(x)) != 0 {
				f.Pixels[i*8+x] = 1
			}
		}
	}
	return f
}
