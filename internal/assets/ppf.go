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
// **版面由「模式 ＋ 長度」決定**，寬度與每列位元組數固定，高度由長度除出來：
//
//	模式    寬   每列位元組  像素排法        實測到的高度
//	CEGA    640     320      四個位元平面    350
//	sega    320     160      四個位元平面    200
//	tdy     320     160      四個位元平面    200（與 sega 同一個版面）
//	mcga    320     320      每像素一位元組  199、200
//	MONO    640      80      一個位元平面    336、347、348
//	CGA     640      80      一個位元平面    175、200
//
// ⚠ **高度不是每個模式一個常數。** `mcgantro.ppf` 是 320×199 而同一個目錄的
// `mcgascen.ppf` 是 320×200；CGA 的招牌是 175 列、劇本選單是 200 列；
// MONO 三份樣本是 336／347／348。把高度寫死成常數的話，同一個模式裡
// 有些畫面讀得出來、有些整幅讀不出來——而且在別的模式下玩完全看不出問題。
//
// ⚠ **MONO 與 CGA 的每列位元組數都是 80，所以光看長度分不出是哪一種。**
// 16000 可以是 CGA 640×200，也可以是 MONO 640×200。
//
// ⚠ CGA 是 **640×200 兩色**（`SIMCITY.CFG` 的解碼表寫 `C - CGA Monochrome`），
// 不是 320×200 四色。兩種讀法每列都是 80 個位元組，而錯的那一種
// **畫出來是一幅看得懂的招牌**——只是每個像素寬一倍、顏色變成 EGA 前四色。
// 判準是顏色數與原版實跑：`workplace/dosbox/cgatitle-00-title.png` 的招牌
// 佔滿 640 像素寬，整張畫面只有 #000000 與 #ffffff（`TestCGAPPFIsMonochrome`）。
//
// 要解 mono 與 cga 就得把模式傳進來（ParsePPFAs）——遊戲自己知道模式，
// 它是從 SIMCITY.CFG 決定要載哪一組檔案的。
// ParsePPF 只認長度沒有歧義的那三種。
//
// ⚠ **封裝式與位元平面吃掉的位元組數一樣。** 320×200 的封裝式 2bpp 與
// 640×200 的單平面都剛好用掉 16000 位元組，長度檢查兩種都會過，
// 而錯的那一種畫出來**是一幅看得懂的招牌**，只是每個像素寬一倍、
// 顏色變成 EGA 前四色。裁決方式是把兩種都畫出來，再與原版實跑對照。
//
// 六種模式的樣本來自 DOS 1.03（玩家自備）；Tandy 的版面另有軟體世界 1990 年
// 地形編輯器磁片的一份佐證，見 docs/formats/00-e220-terrain-editor.md。
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

// ppfKind 是像素怎麼排。
type ppfKind int

const (
	kindPlanar  ppfKind = iota // 逐列交錯的位元平面，高位在前
	kindLinear                 // 每像素一個位元組（256 色）
	kindPacked2                // 封裝式 2bpp，一個位元組四個像素，高位在左
	kindPacked4                // 封裝式 4bpp，一個位元組兩個像素，高位在左
)

// ppfLayout 是一種顯示模式的畫面版面。高度不記在這裡——它由檔案長度
// 除以 bytesPerRow 得到，因為同一個模式的不同畫面高度不一樣。
type ppfLayout struct {
	name        string
	w           int
	bytesPerRow int
	planes      int // 只有 kindPlanar 用得到
	kind        ppfKind
	minH, maxH  int // 合理的高度範圍，見 height 的說明
}

// PPFModes 是六種顯示模式的版面。鍵是 `SIMCITY.CFG` 與檔名共用的那組前綴。
var PPFModes = map[string]ppfLayout{
	"cega": {"cega", 640, 320, 4, kindPlanar, 300, 400},
	"sega": {"sega", 320, 160, 4, kindPlanar, 150, 250},
	"tdy":  {"tdy", 320, 160, 0, kindPacked4, 150, 250},
	"mcga": {"mcga", 320, 320, 0, kindLinear, 150, 250},
	"mono": {"mono", 640, 80, 1, kindPlanar, 300, 400},
	// ⚠ CGA 是 **640×200 兩色**（`SIMCITY.CFG` 的解碼表寫 `C - CGA
	// Monochrome`），不是 320×200 四色。兩種讀法**每列都是 80 個位元組**，
	// 長度分不出來，而 320 寬的四色讀法**畫出來是一幅看得懂的招牌**
	// ——只是每個像素寬了一倍、顏色是 EGA 前四色。
	// 判準是原版實跑：`workplace/dosbox/cgatitle-00-title.png` 的招牌
	// 佔滿 640 個像素寬，而且整張畫面只有 `#000000` 與 `#ffffff` 兩色。
	"cga": {"cga", 640, 80, 1, kindPlanar, 150, 250},
}

// unambiguous 是長度不會與別的模式相撞的那幾種，給沒有模式資訊的呼叫端用。
// mono 與 cga 不在內：兩者每列都是 80 個位元組，長度分不出來。
var unambiguous = []string{"cega", "sega", "mcga"}

// height 由長度除出來。除不盡或高度落在該模式的合理範圍外就回 0。
//
// ⚠ **高度範圍不是裝飾，是消歧義的必要條件。** 少了它，32000 個位元組
// 除以 CEGA 的每列 320 剛好是 100，於是 sega 的畫面會被當成 CEGA 的
// 640×100 解出來——畫面出得來、長度檢查也過，只是整幅是錯的。
// 範圍取自實際的顯示模式：200 列那一批（sega／mcga／CGA）與
// 350 列那一批（CEGA／MONO），各留寬裕的邊。
func (l ppfLayout) height(n int) int {
	if l.bytesPerRow == 0 || n%l.bytesPerRow != 0 {
		return 0
	}
	h := n / l.bytesPerRow
	if h < l.minH || h > l.maxH {
		return 0
	}
	return h
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
	for _, name := range unambiguous {
		if h := PPFModes[name].height(len(d)); h > 0 {
			return parseWith(d, pal, PPFModes[name], h)
		}
	}
	return nil, fmt.Errorf(".PPF：解出 %d 位元組，對不上長度無歧義的版面"+
		"（mono 與 cga 每列都是 80 個位元組，要用 ParsePPFAs 指定模式）", len(d))
}

// ParsePPFAs 由呼叫端指定顯示模式。遊戲自己知道模式——`SIMCITY.CFG` 決定
// 要載哪一組檔案，檔名前綴就是模式。mono 與 cga 只能走這一條。
func ParsePPFAs(d []byte, pal []PGFColor, mode string) (*image.RGBA, error) {
	l, ok := PPFModes[mode]
	if !ok {
		return nil, fmt.Errorf(".PPF：不認得的顯示模式 %q", mode)
	}
	h := l.height(len(d))
	if h == 0 {
		return nil, fmt.Errorf(".PPF：%s 模式下 %d 個位元組不是每列 %d 的整數倍，"+
			"或高度離譜", mode, len(d), l.bytesPerRow)
	}
	return parseWith(d, pal, l, h)
}

func parseWith(d []byte, pal []PGFColor, l ppfLayout, h int) (*image.RGBA, error) {
	switch l.kind {
	case kindPlanar:
		return ppfPlanar(d, l, h), nil
	case kindPacked2:
		return ppfPacked2(d, l, h), nil
	case kindPacked4:
		return ppfPacked4(d, l, h), nil
	}
	if pal == nil {
		return nil, fmt.Errorf(".PPF：%s 是 256 色的，要傳同一個圖形集的調色盤", l.name)
	}
	return ppfLinear(d, l, h, pal), nil
}

// monoScreen 是單平面模式的兩色。Hercules 與 EGA Mono 都是單色顯示器，
// 亮的那一色是白不是 `egaScreen[1]` 的藍——照十六色表解會得到一幅
// 藍底黑字、看起來「有畫面」但顏色錯的圖。
var monoScreen = [2]color.RGBA{
	{0x00, 0x00, 0x00, 0xff}, {0xff, 0xff, 0xff, 0xff},
}

func ppfPlanar(d []byte, l ppfLayout, h int) *image.RGBA {
	bpr := l.w / 8
	im := image.NewRGBA(image.Rect(0, 0, l.w, h))
	for y := 0; y < h; y++ {
		row := y * l.planes * bpr
		for b := 0; b < bpr; b++ {
			for bit := 0; bit < 8; bit++ {
				sh := uint(7 - bit)
				idx := 0
				for p := 0; p < l.planes; p++ {
					idx |= int((d[row+p*bpr+b]>>sh)&1) << uint(l.planes-1-p)
				}
				if l.planes == 1 {
					im.SetRGBA(b*8+bit, y, monoScreen[idx])
				} else {
					im.SetRGBA(b*8+bit, y, egaScreen[idx])
				}
			}
		}
	}
	return im
}

// ppfPacked2 解 CGA 的封裝式 2bpp：一個位元組四個像素，最高的兩位在最左邊。
// 色號是 CGA 四色盤的索引，這裡沿用 egaScreen 的前四色當佔位——
// CGA 真正的四色盤（黑／青／洋紅／白等組合）還沒從資料裡讀出來。
func ppfPacked2(d []byte, l ppfLayout, h int) *image.RGBA {
	bpr := l.w / 4
	im := image.NewRGBA(image.Rect(0, 0, l.w, h))
	for y := 0; y < h; y++ {
		for b := 0; b < bpr; b++ {
			v := d[y*bpr+b]
			for i := 0; i < 4; i++ {
				im.SetRGBA(b*4+i, y, egaScreen[(v>>uint(6-2*i))&3])
			}
		}
	}
	return im
}

// ppfPacked4 解 Tandy／PCjr 的封裝式 4bpp：一個位元組兩個像素，高位在左，
// 色號是 EGA 十六色的索引。
//
// ⚠ **這個模式與 4 平面 planar 吃掉的位元組數一模一樣**：320 像素在
// 4bpp 封裝下是 160 個位元組，在 4 平面下是 40×4＝160，長度檢查兩邊都過。
// 版面初版照 sega 抄成 planar，解出來的尺寸正確、`.PPF` 測試全綠，
// 而畫面是一整片直條雜訊——**尺寸對不代表解對**，要看圖。
// 同一個坑 CGA 也有（2bpp 封裝 vs 平面），兩次都是靠把圖畫出來裁掉的。
func ppfPacked4(d []byte, l ppfLayout, h int) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, l.w, h))
	for y := 0; y < h; y++ {
		for b := 0; b < l.bytesPerRow; b++ {
			v := d[y*l.bytesPerRow+b]
			im.SetRGBA(b*2, y, egaScreen[v>>4])
			im.SetRGBA(b*2+1, y, egaScreen[v&0x0f])
		}
	}
	return im
}

func ppfLinear(d []byte, l ppfLayout, h int, pal []PGFColor) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, l.w, h))
	for y := 0; y < h; y++ {
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
