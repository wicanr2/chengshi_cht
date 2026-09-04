package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// 介面配色：**程式自己畫**的那些東西的顏色（選單列、視窗框、標題列、
// 資金帶、網點、對話框、需求長條），與從 `.PGF` 讀出來的美術無關。
//
// 為什麼要分兩套：原版六種顯示模式裡，**單色 VGA 與 CGA Mono 的整個畫面
// 只有黑白兩色**。拿 EGA 的深藍畫視窗框、亮青畫選單列，在那兩個模式底下
// 是「配色與美術對不起來」——工具盤是黑白點陣，外框卻是彩色的。
//
// 值是量出來的，不是挑的（`tools/chromesample`，原版實跑截圖在
// `workplace/dosbox/chrome-*-00-ingame.png`）：
//
//	chrome-mono-00-ingame.png  整張畫面 **2 種顏色**：#000000 / #ffffff
//	chrome-cga-00-ingame.png   整張畫面 **2 種顏色**：#000000 / #ffffff
//	chrome-cega / -tdy         標準 EGA 十六色
//	chrome-mcga                地圖是 256 色，但介面逐項與 EGA 相同——
//	                           選單列 #55ffff 底 #0000aa 字、桌面 #aaaaaa、
//	                           資金帶 #555555，三處都逐位元組相同
//
// 所以彩色那四個模式共用一套（就是原本那一套，一個值都沒改），
// 單色兩個模式一套。
//
// ⚠ **這是套件層的全域狀態**，`setChrome` 會直接改下面那些變數。
// 同一個行程只有一份遊戲畫面，所以夠用；但測試如果要驗顏色，
// 記得自己先呼叫一次 `setChrome`，不要假設是彩色那一套。
type chromeSet struct {
	desktop  color.RGBA // 桌面
	menuBar  color.RGBA // 選單列底
	menuInk  color.RGBA // 選單字
	editFrm  color.RGBA // 編輯視窗框
	mapFrm   color.RGBA // 地圖視窗框內圈
	mapFrmD  color.RGBA // 地圖視窗框外圈
	infoBand color.RGBA // 資金帶底
	infoInk  color.RGBA // 資金帶的字
	titleBar color.RGBA // 標題列底
	mapGreen color.RGBA // 圖層圖示欄的底
	ink      color.RGBA // 網點的深色
	inkLight color.RGBA // 網點的淺色
	dlgLine  color.RGBA // 對話框外框
	dlgFill  color.RGBA // 對話框按鈕與欄位底
	dlgBG    color.RGBA // 對話框客戶區
	line     color.RGBA // 資料視窗的框線
	panel    color.RGBA // 資料視窗的客戶區
	text     color.RGBA // 資料視窗的字
	dim      color.RGBA // 次要的字
	on       color.RGBA // 選中的字
	moneyN   color.RGBA // 負數金額
	demC     color.RGBA // 統計圖的商業
	demR     color.RGBA // 統計圖的住宅
	demI     color.RGBA // 統計圖的工業
	demBarC  color.RGBA // 需求長條：商業
	demBarR  color.RGBA // 需求長條：住宅
	demBarI  color.RGBA // 需求長條：工業
	// bandInk 是工具帶（編輯視窗最下面那條）的深色：網點的一半、
	// 工具名的字、右下角那個 `+`。
	//
	// ⚠ **不能沿用 editFrm。** 彩色模式兩者都是深藍，看起來像同一個顏色，
	// 但單色模式的視窗框是**白**的（畫在黑桌面上）、工具帶的字是**黑**的
	// （畫在白帶上）。共用一個變數的話工具名會變成白底白字——
	// 畫面上那條帶還在，只是**什麼都沒有**，看起來像「這個模式不顯示工具名」。
	bandInk color.RGBA
	// tableBG 是預算視窗那張表的底（原版是亮藍、白字）。
	tableBG color.RGBA
	// sel 是選取框（工具盤、圖層圖示、統計圖按鈕那一圈）。
	sel color.RGBA
	// selDither 為真時選取框改畫網點——單色模式只有黑白兩色，
	// 實線框會與格子本身的外框混在一起。原版就是這樣做的。
	selDither bool
}

var (
	// chromeColor 是 EGA／Tandy／MCGA 那四個模式的配色。
	// 值取自原版截圖的實際位元組（見 docs/formats/03-pgf-graphics.md §3）。
	chromeColor = chromeSet{
		desktop:  color.RGBA{0xaa, 0xaa, 0xaa, 0xff},
		menuBar:  color.RGBA{0x55, 0xff, 0xff, 0xff},
		menuInk:  color.RGBA{0x00, 0x00, 0xaa, 0xff},
		editFrm:  color.RGBA{0x00, 0x00, 0xaa, 0xff},
		mapFrm:   color.RGBA{0xff, 0x55, 0x55, 0xff},
		mapFrmD:  color.RGBA{0xaa, 0x00, 0x00, 0xff},
		infoBand: color.RGBA{0x55, 0x55, 0x55, 0xff},
		infoInk:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		titleBar: color.RGBA{0xaa, 0xaa, 0xaa, 0xff},
		mapGreen: color.RGBA{0x00, 0xaa, 0x00, 0xff},
		ink:      color.RGBA{0x00, 0x00, 0x00, 0xff},
		inkLight: color.RGBA{0xff, 0xff, 0xff, 0xff},
		dlgLine:  color.RGBA{0x55, 0x55, 0xff, 0xff},
		dlgFill:  color.RGBA{0x55, 0xff, 0xff, 0xff},
		dlgBG:    color.RGBA{0xff, 0xff, 0xff, 0xff},
		line:     color.RGBA{0x00, 0x00, 0xaa, 0xff},
		panel:    color.RGBA{0xff, 0xff, 0xff, 0xff},
		text:     color.RGBA{0x00, 0x00, 0x00, 0xff},
		dim:      color.RGBA{0x55, 0x55, 0x55, 0xff},
		on:       color.RGBA{0x00, 0x00, 0xaa, 0xff},
		moneyN:   color.RGBA{0xaa, 0x00, 0x00, 0xff},
		demC:     color.RGBA{0x00, 0x00, 0xaa, 0xff},
		demR:     color.RGBA{0x00, 0xaa, 0x00, 0xff},
		demI:     color.RGBA{0xaa, 0x55, 0x00, 0xff},
		demBarC:  color.RGBA{0x55, 0x55, 0xff, 0xff},
		demBarR:  color.RGBA{0x55, 0xff, 0x55, 0xff},
		demBarI:  color.RGBA{0xff, 0xff, 0x55, 0xff},
		bandInk:  color.RGBA{0x00, 0x00, 0xaa, 0xff},
		tableBG:  color.RGBA{0x55, 0x55, 0xff, 0xff},
		sel:      color.RGBA{0xff, 0xff, 0x55, 0xff},
	}

	// chromeMono 是單色 VGA 與 CGA Mono 的配色。**只有黑與白**——
	// 原版那兩個模式的整張畫面就只有這兩色（實測，見上）。
	//
	// 哪個元素配哪一色是照原版截圖對的：桌面黑、視窗客戶區白、
	// 選單列白底黑字、資金帶白底黑字（彩色版是深灰底白字，這裡反過來）、
	// 視窗框與圖示欄的底都是白。
	//
	// 三根需求長條在這裡**同色**（白）。原版是靠網點分的，remake 畫的是
	// 實心長條，分不出來；位置固定在 C·R·I 三個標籤下面，讀得出是哪一根。
	chromeMono = chromeSet{
		desktop:   black,
		menuBar:   white,
		menuInk:   black,
		editFrm:   white,
		mapFrm:    white,
		mapFrmD:   white,
		infoBand:  white,
		infoInk:   black,
		titleBar:  white,
		mapGreen:  white,
		ink:       black,
		inkLight:  white,
		dlgLine:   black,
		dlgFill:   white,
		dlgBG:     white,
		line:      black,
		panel:     white,
		text:      black,
		dim:       black,
		on:        black,
		moneyN:    black,
		demC:      black,
		demR:      black,
		demI:      black,
		demBarC:   white,
		demBarR:   white,
		demBarI:   white,
		bandInk:   black,
		tableBG:   black,
		sel:       black,
		selDither: true,
	}

	black = color.RGBA{0x00, 0x00, 0x00, 0xff}
	white = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// 目前生效的配色。畫圖的地方一律用這些變數，不要直接寫顏色常數。
var (
	colDesktop  color.RGBA
	colMenuBar  color.RGBA
	colMenuInk  color.RGBA
	colEditFrm  color.RGBA
	colMapFrm   color.RGBA
	colMapFrmD  color.RGBA
	colInfoBand color.RGBA
	colInfoInk  color.RGBA
	colTitleBar color.RGBA
	colMapGreen color.RGBA
	colInk      color.RGBA
	colInkLight color.RGBA
	colDlgLine  color.RGBA
	colDlgFill  color.RGBA
	colDlgBG    color.RGBA
	colLine     color.RGBA
	colPanel    color.RGBA
	colText     color.RGBA
	colDim      color.RGBA
	colOn       color.RGBA
	colMoneyN   color.RGBA
	colDemC     color.RGBA
	colDemR     color.RGBA
	colDemI     color.RGBA
	colDemBarC  color.RGBA
	colDemBarR  color.RGBA
	colDemBarI  color.RGBA
	colBandInk  color.RGBA
	colTableBG  color.RGBA
	colSel      color.RGBA
	// chromeSelDither 見 chromeSet.selDither。
	chromeSelDither bool
	// colBG 是資料視窗外面那一圈，與桌面同色。
	colBG color.RGBA
)

func init() { setChrome('E') }

// chromeFor 回傳一個顯示模式該用哪一套配色。
// 判準是**那個模式的畫面有幾種顏色**，不是模式名字：
// `V`（單色 VGA/MCGA）與 `C`（CGA Mono）實測只有兩色。
func chromeFor(mode byte) chromeSet {
	switch mode {
	case 'V', 'C', 'M', 'H': // 單色 VGA、CGA Mono、EGA 單色、Hercules
		return chromeMono
	}
	return chromeColor
}

// setChrome 把某個顯示模式的配色套上去。載入圖形集之後要呼叫一次。
func setChrome(mode byte) {
	c := chromeFor(mode)
	colDesktop, colBG = c.desktop, c.desktop
	colMenuBar, colMenuInk = c.menuBar, c.menuInk
	colEditFrm = c.editFrm
	colMapFrm, colMapFrmD = c.mapFrm, c.mapFrmD
	colInfoBand, colInfoInk = c.infoBand, c.infoInk
	colTitleBar, colMapGreen = c.titleBar, c.mapGreen
	colInk, colInkLight = c.ink, c.inkLight
	colDlgLine, colDlgFill, colDlgBG = c.dlgLine, c.dlgFill, c.dlgBG
	colLine, colPanel, colText = c.line, c.panel, c.text
	colDim, colOn, colMoneyN = c.dim, c.on, c.moneyN
	colDemC, colDemR, colDemI = c.demC, c.demR, c.demI
	colDemBarC, colDemBarR, colDemBarI = c.demBarC, c.demBarR, c.demBarI
	colBandInk, colTableBG, colSel = c.bandInk, c.tableBG, c.sel
	chromeSelDither = c.selDither
}

// selRect 畫一圈選取框，厚度 t（原版座標）。
//
// 彩色模式是實心亮黃；**單色模式畫網點**——原版就是這樣做的
// （只有黑白兩色，實線框在白格子上是黑的、在黑格子上是白的，
// 兩種都會與格子本身的外框混在一起看不出來）。
func selRect(dst *ebiten.Image, x, y, w, h, t int) {
	paint := func(rx, ry, rw, rh int) {
		if !chromeSelDither {
			fill(dst, rx, ry, rw, rh, colSel)
			return
		}
		ditherRect(dst, rx, ry, rw, rh, colInk, colInkLight)
	}
	paint(x, y, w, t)
	paint(x, y+h-t, w, t)
	paint(x, y, t, h)
	paint(x+w-t, y, t, h)
}

// strokeSel 是 selRect 的一像素版，給工具盤與統計圖按鈕用。
func strokeSel(dst *ebiten.Image, x, y, w, h int) { selRect(dst, x, y, w, h, 1) }
