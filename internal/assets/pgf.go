package assets

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// .PGF 圖形檔。證據：docs/formats/03-pgf-graphics.md
//
// 版面：
//
//	[CP437 橫幅 + 0x1a + 風格名\0 + 模式位元組 + 三個檔名\0]  ← 只有風格檔有
//	LZSS 壓縮流：
//	  u16 圖形庫數量
//	  u8  位元深度（MCGA 8、CEGA/SEGA 4 平面、MONO 1 平面）
//	  u16 風格編號
//	  調色盤：8 位元 RGB，8bpp 是 256 色、其餘 16 色
//	  圖形庫 × N：
//	    u16 寬、u16 高、u16 張數、u16 旗標、u32 資料長度
//	    資料（旗標 bit0 = 每張圖前面多 4 個位元組）
//
// 基本檔（`mcgadat.pgf` 等）沒有橫幅也沒有那五個位元組的檔頭，
// 解壓後直接就是調色盤——那是另一種版本，本套件目前只解風格檔。

// PGFColor 是一個調色盤項目（8 位元 RGB）。
type PGFColor struct{ R, G, B uint8 }

// PGFImage 是圖形庫裡的一張圖。Head 是旗標 bit0 時每張圖前面那四個
// 位元組，語意未解（見文件 §4），先原樣保留。
type PGFImage struct {
	Head   [4]byte
	Pixels []uint8 // 每格一個調色盤索引，長度 = 寬 × 高
}

// PGFBank 是一組同尺寸的圖。
type PGFBank struct {
	Width, Height int
	Flags         uint16
	Images        []PGFImage
}

// PGF 是解開的一個圖形檔。
type PGF struct {
	Name         string // 風格顯示名，例如 "Ancient Asia"
	Mode         byte   // 顯示模式碼：'2' MCGA、'E' CEGA、'V' MONO、'e' SEGA
	MsgFile      string
	SoundFile    string
	MonoFile     string
	BitsPerPixel int
	StyleID      int
	Palette      []PGFColor
	Banks        []PGFBank
}

// 旗標位元。
const (
	pgfPerImageHead = 0x0001 // 每張圖前面多四個位元組
	pgfSinglePlane  = 0x0100 // 這一庫只有一個平面
	//
	// 對 4 平面的檔案，單平面的庫是**遮罩**，與前一庫配對；
	// 對 1 平面的 MONO 檔，每一庫都是單平面，這個位元恆為 1。
)

// pgfHeader 拆掉 CP437 橫幅與後面的中繼資料，回傳壓縮流的起點。
//
// ⚠ 橫幅後面那個 0x1a 是 DOS 的檔尾字元——`TYPE ASIAMCGA.PGF` 只會印出
// 橫幅就停住。那是刻意的：讓使用者看到「這是什麼檔」而不是滿螢幕亂碼。
func pgfHeader(raw []byte) (int, string, byte, []string) {
	eof := bytes.IndexByte(raw, 0x1a)
	if eof < 0 || eof > 300 {
		return 0, "", 0, nil // 基本檔：沒有橫幅
	}
	p := eof + 1
	z := bytes.IndexByte(raw[p:], 0)
	if z < 0 || z > 64 {
		return 0, "", 0, nil
	}
	name := string(raw[p : p+z])
	p += z + 1
	if p >= len(raw) {
		return 0, "", 0, nil
	}
	mode := raw[p]
	p++
	var files []string
	for i := 0; i < 3; i++ {
		z := bytes.IndexByte(raw[p:], 0)
		if z < 0 || z > 64 {
			break
		}
		files = append(files, string(raw[p:p+z]))
		p += z + 1
	}
	return p, name, mode, files
}

// ParsePGF 解開一個 .PGF。
func ParsePGF(raw []byte) (*PGF, error) {
	start, name, mode, files := pgfHeader(raw)
	data, err := DecompressLZSS(raw[start:])
	if err != nil {
		return nil, fmt.Errorf("解壓失敗：%w", err)
	}
	if len(data) < 5 {
		return nil, fmt.Errorf("解出來只有 %d 位元組", len(data))
	}
	g := &PGF{Name: name, Mode: mode}
	if len(files) == 3 {
		g.MsgFile, g.SoundFile, g.MonoFile = files[0], files[1], files[2]
	}
	nBanks := int(binary.LittleEndian.Uint16(data[0:]))
	g.BitsPerPixel = int(data[2])
	g.StyleID = int(binary.LittleEndian.Uint16(data[3:]))
	switch g.BitsPerPixel {
	case 1, 4, 8:
	default:
		return nil, fmt.Errorf("位元深度 %d 不是 1/4/8 —— 這可能是沒有五位元組檔頭的基本檔", g.BitsPerPixel)
	}
	if nBanks == 0 || nBanks > 64 {
		return nil, fmt.Errorf("圖形庫數量 %d 不合理", nBanks)
	}

	// 調色盤長度是 2^位元深度：8bpp 256 色、4 平面 16 色、1 平面 2 色。
	//
	// ⚠ 用固定長度去讀會讓第一個圖形庫的表頭落在調色盤中間，
	// 解出來是一堆看似合理的尺寸——自洽但錯。
	nColors := 1 << uint(g.BitsPerPixel)
	p := 5
	if p+nColors*3 > len(data) {
		return nil, fmt.Errorf("調色盤讀不完")
	}
	g.Palette = make([]PGFColor, 256)
	for i := 0; i < nColors; i++ {
		g.Palette[i] = PGFColor{data[p+i*3], data[p+i*3+1], data[p+i*3+2]}
	}
	p += nColors * 3

	for i := 0; i < nBanks; i++ {
		if p+12 > len(data) {
			return nil, fmt.Errorf("第 %d 個圖形庫的表頭讀不完", i)
		}
		w := int(binary.LittleEndian.Uint16(data[p:]))
		h := int(binary.LittleEndian.Uint16(data[p+2:]))
		cnt := int(binary.LittleEndian.Uint16(data[p+4:]))
		fl := binary.LittleEndian.Uint16(data[p + 6:])
		size := int(binary.LittleEndian.Uint32(data[p+8:]))
		p += 12
		if w <= 0 || h <= 0 || cnt < 0 || p+size > len(data) {
			return nil, fmt.Errorf("第 %d 個圖形庫的表頭不合理：%d×%d ×%d 長度 %d", i, w, h, cnt, size)
		}
		b := PGFBank{Width: w, Height: h, Flags: fl}
		q := p
		for j := 0; j < cnt; j++ {
			var im PGFImage
			if fl&pgfPerImageHead != 0 {
				if q+4 > len(data) {
					return nil, fmt.Errorf("第 %d 庫第 %d 張的四位元組表頭讀不完", i, j)
				}
				copy(im.Head[:], data[q:q+4])
				q += 4
			}
			px, next, err := pgfPixels(data, q, w, h, g.BitsPerPixel, fl)
			if err != nil {
				return nil, fmt.Errorf("第 %d 庫第 %d 張：%w", i, j, err)
			}
			im.Pixels = px
			q = next
			b.Images = append(b.Images, im)
		}
		p += size
		g.Banks = append(g.Banks, b)
	}
	return g, nil
}

// pgfPixels 讀一張圖的像素。
//
// 8bpp 是每格一個位元組；4 平面與 1 平面是 EGA 式的**平面式**排列：
// 整張圖的第 0 平面在前，接著第 1 平面，以此類推。
func pgfPixels(data []byte, q, w, h, bpp int, fl uint16) ([]uint8, int, error) {
	out := make([]uint8, w*h)
	switch bpp {
	case 8:
		if q+w*h > len(data) {
			return nil, 0, fmt.Errorf("像素讀不完")
		}
		copy(out, data[q:q+w*h])
		return out, q + w*h, nil
	case 1, 4:
		planes := bpp
		if fl&pgfSinglePlane != 0 {
			planes = 1
		}
		rowBytes := (w + 7) / 8
		need := rowBytes * h * planes
		if q+need > len(data) {
			return nil, 0, fmt.Errorf("像素讀不完")
		}
		// ⚠ 平面是**逐列交錯**的：每一列先放第 0 平面的 rowBytes 個位元組，
		// 再放第 1 平面……而不是整張圖的第 0 平面放完再放第 1 平面。
		//
		// 兩種讀法用掉的位元組數一模一樣，所以長度檢查抓不到。錯的那一種
		// 畫出來會出現規律的橫條——看起來像掃描線或隔行掃描的瑕疵，
		// 很容易被誤判成「渲染器的縮放有問題」。
		//
		// ⚠ 而且平面順序是**倒著的**：一列裡第一組位元組是最高位平面。
		// 判準是水面（圖塊 2）：它整格只有一個平面全 1。順著讀得到色號 8
		// （深灰），倒著讀得到色號 1（藍）。水是藍的。
		for y := 0; y < h; y++ {
			rowBase := q + y*rowBytes*planes
			for pl := 0; pl < planes; pl++ {
				bit := planes - 1 - pl
				if planes == 1 {
					bit = 0
				}
				base := rowBase + pl*rowBytes
				for x := 0; x < w; x++ {
					if data[base+x/8]&(0x80>>uint(x%8)) != 0 {
						out[y*w+x] |= 1 << uint(bit)
					}
				}
			}
		}
		return out, q + need, nil
	}
	return nil, 0, fmt.Errorf("位元深度 %d 不支援", bpp)
}

