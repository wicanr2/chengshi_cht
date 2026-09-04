package ui

import (
	"image/color"
	"testing"
)

// 彩色四個模式的配色**一個值都不能變**。這一支把 2026-09 之前寫死在
// classic.go／game.go／newcity.go 的那批值抄過來當基準——配色改成
// 跟著顯示模式換之後，EGA 那條路徑必須逐項相同，否則就是重構把畫面改掉了。
func TestChromeColorUnchanged(t *testing.T) {
	defer setChrome('E')
	setChrome('E')
	for _, c := range []struct {
		name string
		got  color.RGBA
		want color.RGBA
	}{
		{"桌面", colDesktop, color.RGBA{0xaa, 0xaa, 0xaa, 0xff}},
		{"選單列", colMenuBar, color.RGBA{0x55, 0xff, 0xff, 0xff}},
		{"選單字", colMenuInk, color.RGBA{0x00, 0x00, 0xaa, 0xff}},
		{"編輯視窗框", colEditFrm, color.RGBA{0x00, 0x00, 0xaa, 0xff}},
		{"工具帶的字", colBandInk, color.RGBA{0x00, 0x00, 0xaa, 0xff}},
		{"地圖視窗框", colMapFrm, color.RGBA{0xff, 0x55, 0x55, 0xff}},
		{"地圖視窗框外圈", colMapFrmD, color.RGBA{0xaa, 0x00, 0x00, 0xff}},
		{"資金帶", colInfoBand, color.RGBA{0x55, 0x55, 0x55, 0xff}},
		{"資金帶的字", colInfoInk, color.RGBA{0xff, 0xff, 0xff, 0xff}},
		{"標題列", colTitleBar, color.RGBA{0xaa, 0xaa, 0xaa, 0xff}},
		{"圖示欄", colMapGreen, color.RGBA{0x00, 0xaa, 0x00, 0xff}},
		{"對話框外框", colDlgLine, color.RGBA{0x55, 0x55, 0xff, 0xff}},
		{"對話框按鈕", colDlgFill, color.RGBA{0x55, 0xff, 0xff, 0xff}},
		{"需求長條商業", colDemBarC, color.RGBA{0x55, 0x55, 0xff, 0xff}},
		{"需求長條住宅", colDemBarR, color.RGBA{0x55, 0xff, 0x55, 0xff}},
		{"需求長條工業", colDemBarI, color.RGBA{0xff, 0xff, 0x55, 0xff}},
		{"選取框", colSel, color.RGBA{0xff, 0xff, 0x55, 0xff}},
		{"負數金額", colMoneyN, color.RGBA{0xaa, 0x00, 0x00, 0xff}},
	} {
		if c.got != c.want {
			t.Errorf("%s = %v，原本是 %v", c.name, c.got, c.want)
		}
	}
	if chromeSelDither {
		t.Error("彩色模式的選取框應該是實線，不是網點")
	}
}

// 單色 VGA（`V`）與 CGA Mono（`C`）的介面**只能有黑與白**。
//
// 判準來自原版實跑：`workplace/dosbox/chrome-mono-00-ingame.png` 與
// `chrome-cga-00-ingame.png` 整張畫面各只有 **2 種顏色**
// （#000000 與 #ffffff，`tools/chromesample` 數的）。
//
// ⚠ 少了這一支的後果不是崩潰，是**畫面上有東西看不見**：
// 拿 EGA 的深藍去畫那兩個模式的視窗框，框還在、只是顏色跟美術對不起來；
// 而工具帶如果沿用視窗框的顏色，在單色模式會變成白底白字——
// 那條帶還在，字整個消失，看起來像「這個模式不顯示工具名」。
func TestChromeMonoIsBlackAndWhite(t *testing.T) {
	defer setChrome('E')
	bw := func(c color.RGBA) bool {
		return c == color.RGBA{0, 0, 0, 0xff} || c == color.RGBA{0xff, 0xff, 0xff, 0xff}
	}
	for _, mode := range []byte{'V', 'C'} {
		setChrome(mode)
		for _, c := range []struct {
			name string
			v    color.RGBA
		}{
			{"桌面", colDesktop}, {"選單列", colMenuBar}, {"選單字", colMenuInk},
			{"編輯視窗框", colEditFrm}, {"工具帶的字", colBandInk},
			{"地圖視窗框", colMapFrm}, {"地圖視窗框外圈", colMapFrmD},
			{"資金帶", colInfoBand}, {"資金帶的字", colInfoInk},
			{"標題列", colTitleBar}, {"圖示欄", colMapGreen},
			{"網點深", colInk}, {"網點淺", colInkLight},
			{"對話框外框", colDlgLine}, {"對話框按鈕", colDlgFill},
			{"對話框底", colDlgBG}, {"視窗框線", colLine}, {"視窗客戶區", colPanel},
			{"字", colText}, {"次要字", colDim}, {"選中字", colOn},
			{"負數金額", colMoneyN},
			{"需求長條商業", colDemBarC}, {"需求長條住宅", colDemBarR},
			{"需求長條工業", colDemBarI},
			{"統計圖商業", colDemC}, {"統計圖住宅", colDemR}, {"統計圖工業", colDemI},
		} {
			if !bw(c.v) {
				t.Errorf("模式 %q 的%s = %v，單色模式只能有黑白兩色", mode, c.name, c.v)
			}
		}
		// 底與字不能同色，否則畫面上那塊東西整個消失。
		for _, p := range []struct {
			what   string
			bg, fg color.RGBA
		}{
			{"選單列", colMenuBar, colMenuInk},
			{"資金帶", colInfoBand, colInfoInk},
			{"工具帶", colInkLight, colBandInk},
			{"資料視窗", colPanel, colText},
			{"對話框", colDlgBG, colDlgLine},
		} {
			if p.bg == p.fg {
				t.Errorf("模式 %q 的%s底與字同色（%v），字會整個看不見", mode, p.what, p.bg)
			}
		}
		if !chromeSelDither {
			t.Errorf("模式 %q 的選取框要畫網點：只有兩色時實線框會與格子本身的外框混在一起", mode)
		}
	}
}
