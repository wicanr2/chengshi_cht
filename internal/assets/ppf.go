package assets

import (
	"fmt"
	"image"
	"image/color"
)

// `.PPF` 是一整幅畫面（標題與劇本選單），格式寫在
// docs/formats/06-ppf-screen.md。外層 LZSS，解壓後沒有檔頭，
// **版面由長度認得出來**：
//
//	112000  CEGA   640×350，四個位元平面
//	 32000  sega   320×200，四個位元平面
//	 63680  mcga   320×199，每像素一個位元組（256 色）
//
// 位元平面是**逐列交錯**的——每一列 (寬/8) 位元組 × 平面數——而且
// **高位在前**：第一個平面是 EGA 的 I（亮度），最後一個才是 B。
//
// ⚠ 順序組反會得到版面分毫不差、字讀得出來、只有顏色整組錯位的畫面
// （招牌從綠色變紅色），而長度檢查照樣過。長度只能認版面，不能認順序。

// PPFWidth／PPFHeight 是 EGA 高解析（CEGA）的畫面尺寸。
const (
	PPFWidth  = 640
	PPFHeight = 350
)

// ppfLayout 是一種顯示模式的畫面版面。planes 為 0 代表每像素一個位元組。
type ppfLayout struct {
	name   string
	w, h   int
	planes int
}

var ppfLayouts = []ppfLayout{
	{"CEGA", 640, 350, 4},
	{"sega", 320, 200, 4},
	{"mcga", 320, 199, 0},
}

func (l ppfLayout) size() int {
	if l.planes == 0 {
		return l.w * l.h
	}
	return l.w * l.h / 8 * l.planes
}

// egaScreen 是標準 EGA 十六色。位元平面版的 `.PPF` 自己不帶調色盤，
// 用的就是這一組（與 DOS 1.10 實跑逐像素對拍確認）。
var egaScreen = [16]color.RGBA{
	{0x00, 0x00, 0x00, 0xff}, {0x00, 0x00, 0xaa, 0xff},
	{0x00, 0xaa, 0x00, 0xff}, {0x00, 0xaa, 0xaa, 0xff},
	{0xaa, 0x00, 0x00, 0xff}, {0xaa, 0x00, 0xaa, 0xff},
	{0xaa, 0x55, 0x00, 0xff}, {0xaa, 0xaa, 0xaa, 0xff},
	{0x55, 0x55, 0x55, 0xff}, {0x55, 0x55, 0xff, 0xff},
	{0x55, 0xff, 0x55, 0xff}, {0x55, 0xff, 0xff, 0xff},
	{0xff, 0x55, 0x55, 0xff}, {0xff, 0x55, 0xff, 0xff},
	{0xff, 0xff, 0x55, 0xff}, {0xff, 0xff, 0xff, 0xff},
}

// ParsePPF 把解壓後的畫面資料展成 RGBA。
//
// 256 色的 mcga 版沒有自己的調色盤，用**同一個圖形集的 `.PGF`** 那一份
// （實測：37 個用到的色號與 `westmcga.pgf` 全部相符，差距最大 3，
// 那是六位元 VGA 值展成八位元的兩種算法差異）。pal 傳 nil 的話
// 只有位元平面的兩種模式解得開。
func ParsePPF(d []byte, pal []PGFColor) (*image.RGBA, error) {
	for _, l := range ppfLayouts {
		if l.size() != len(d) {
			continue
		}
		if l.planes > 0 {
			return ppfPlanar(d, l), nil
		}
		if pal == nil {
			return nil, fmt.Errorf(".PPF：%s 是 256 色的，要傳同一個圖形集的調色盤", l.name)
		}
		return ppfLinear(d, l, pal), nil
	}
	return nil, fmt.Errorf(".PPF：解出 %d 位元組，對不上任何已知的版面", len(d))
}

func ppfPlanar(d []byte, l ppfLayout) *image.RGBA {
	bpr := l.w / 8
	im := image.NewRGBA(image.Rect(0, 0, l.w, l.h))
	for y := 0; y < l.h; y++ {
		row := y * l.planes * bpr
		for b := 0; b < bpr; b++ {
			for bit := 0; bit < 8; bit++ {
				sh := uint(7 - bit)
				idx := 0
				for p := 0; p < l.planes; p++ {
					idx |= int((d[row+p*bpr+b]>>sh)&1) << uint(l.planes-1-p)
				}
				im.SetRGBA(b*8+bit, y, egaScreen[idx])
			}
		}
	}
	return im
}

func ppfLinear(d []byte, l ppfLayout, pal []PGFColor) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, l.w, l.h))
	for y := 0; y < l.h; y++ {
		for x := 0; x < l.w; x++ {
			v := int(d[y*l.w+x])
			c := color.RGBA{A: 0xff}
			if v < len(pal) {
				c.R, c.G, c.B = pal[v].R, pal[v].G, pal[v].B
			}
			im.SetRGBA(x, y, c)
		}
	}
	return im
}

// LoadPPF 從原始檔案位元組解出畫面。
func LoadPPF(raw []byte, pal []PGFColor) (*image.RGBA, error) {
	d, err := DecompressLZSS(raw)
	if err != nil {
		return nil, err
	}
	return ParsePPF(d, pal)
}
