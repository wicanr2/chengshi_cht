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
//	tdy     960 ÷ 960 =  1（封裝式 4bpp，色號在高四位）→ 1×1
//
// ⚠ Tandy 走**封裝式**，不是平面式（同 pgf.go 的 pgfPixels）。
// 少了那一支，`per % planes` 過不了（1 % 4 ≠ 0），`parseMiniTiles` 直接回 nil，
// City Form 的縮圖整片消失而**畫面照樣畫得出來**——只是退回純色方塊，
// 看起來像「那個模式的地圖就長這樣」。
//
// 這一塊在**風格檔**六個模式都解得出來；**基本檔**只有 CEGA 與 sega 的
// 位置對得上（緊接在第 0 庫之後），MONO／mcga／tdy／CGA 的基本檔中間還隔著
// 一段（MONO 2 816＝128 個字×(8+14) 位元組，tdy 4 096＝128×32），
// 目前一律解不出來，退回純色方塊。這是既有缺口，不是換顯示模式帶進來的。
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
func parseMiniTiles(d []byte, bpp int, mode byte) (*MiniTiles, int) {
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
	if mode == modeTandy && bpp == 4 {
		// Tandy 走封裝式：一個位元組兩個像素、高四位在前。
		// 實測六個風格檔**都是** 960 ÷ 960 = 1 個位元組一格，而且 960 張的
		// 低四位全部是 0——所以一格只有一個像素，色號在高四位，低四位是
		// 封裝式的填充。驗證：取高四位得到的色號與同風格的 sega 相同
		// （ASIA 等五個是空地 6、水 1、樹林 2、道路 7；MOON 自己一組）。
		//
		// per 不是 1 的情形手上沒有樣本，寬度就無從量起（3 個像素與 4 個
		// 像素同樣佔 2 個位元組，兩種都自洽）。與其猜一個，不如回 nil ——
		// 猜錯是整張地圖歪掉，回 nil 只是退回純色方塊。
		if per != 1 {
			return nil, 0
		}
		m.Width, m.Height = 1, 1
		m.Pixels = make([]uint8, tileCount)
		for t := 0; t < tileCount; t++ {
			m.Pixels[t] = body[t] >> 4
		}
		return m, 3 + n
	}
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
