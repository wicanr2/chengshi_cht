package assets

import (
	"fmt"
	"image"
	"image/color"
)

// `.PPF` 是一整幅畫面（標題與劇本選單），格式寫在
// docs/formats/06-ppf-screen.md：外層 LZSS，解壓後沒有檔頭，
// 位元平面**逐列交錯**——每一列 80 位元組 × 四個平面。
//
// ⚠ 不是「一整面接一整面」。那種擺法畫出來是橫條錯位的雜訊，
// 但長度一樣過得了檢查，所以長度不能當版面的證據。

// PPFWidth／PPFHeight 是 EGA 高解析（CEGA）的畫面尺寸。
const (
	PPFWidth  = 640
	PPFHeight = 350

	ppfBytesPerRow = PPFWidth / 8
	ppfPlanes      = 4
	ppfLen         = ppfBytesPerRow * ppfPlanes * PPFHeight // 112000
)

// egaScreen 是標準 EGA 十六色。`.PPF` 自己不帶調色盤。
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
func ParsePPF(d []byte) (*image.RGBA, error) {
	if len(d) != ppfLen {
		return nil, fmt.Errorf(".PPF：解出 %d 位元組，EGA 高解析應為 %d", len(d), ppfLen)
	}
	im := image.NewRGBA(image.Rect(0, 0, PPFWidth, PPFHeight))
	for y := 0; y < PPFHeight; y++ {
		row := y * ppfPlanes * ppfBytesPerRow
		for b := 0; b < ppfBytesPerRow; b++ {
			p0, p1 := d[row+b], d[row+ppfBytesPerRow+b]
			p2, p3 := d[row+2*ppfBytesPerRow+b], d[row+3*ppfBytesPerRow+b]
			for bit := 0; bit < 8; bit++ {
				sh := uint(7 - bit)
				// ⚠ 平面順序是**高位在前**：第一個平面是 EGA 的 I（亮度），
				// 最後一個才是 B。反過來組會得到版面完全正確、顏色整組
				// 錯位的畫面（招牌變成紅色的），而且長度檢查照樣過。
				idx := int((p0>>sh)&1)<<3 | int((p1>>sh)&1)<<2 |
					int((p2>>sh)&1)<<1 | int((p3>>sh)&1)
				im.SetRGBA(b*8+bit, y, egaScreen[idx])
			}
		}
	}
	return im, nil
}

// LoadPPF 從原始檔案位元組解出畫面。
func LoadPPF(raw []byte) (*image.RGBA, error) {
	d, err := DecompressLZSS(raw)
	if err != nil {
		return nil, err
	}
	return ParsePPF(d)
}
