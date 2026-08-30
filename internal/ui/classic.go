package ui

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 原版 DOS 版面。規格與量測：docs/spec/ui-layout.md
//
// 座標系是**原版畫面座標**（640×350，EGA 高解析），畫的時候乘 UIScale。
// 為什麼不用自己的座標系：所有數字都是從原版截圖量出來的，換算過一次
// 就再也對不回去了——要改版面得回頭看截圖，而截圖是 640×350 的。

const (
	// OrigW／OrigH 是原版的畫面尺寸。
	OrigW, OrigH = 640, 350

	// UIScale 是放大倍率。
	//
	// 為什麼是 3：原版一個字元格 8×8，×3 剛好 24×24 螢幕像素，
	// 而本專案的中文點陣字就是 24×24——**一個中文字剛好佔一個原版字元格**。
	// 英數是半形 12×24，佔半格。所以中文比它取代的英文**窄**，
	// 版面有餘裕，不是 CLAUDE.md §3.3 擔心的爆版。
	// 全部是整數倍，不會出現寬窄不一的像素（rulebook/81）。
	UIScale = 3
)

// 版面常數，單位是原版像素。全部量自 workplace/dosbox/ui-00-clean.png。
const (
	menuBarH = 18 // 選單列高

	// 編輯視窗外框。
	//
	// ⚠ 寬度是 **575 不是 236**。第一次量的時候 City Form 視窗蓋在它上面，
	// 量到的是「露出來的那一截」——兩個視窗是**重疊**的，不是並排。
	// 在原版裡對編輯視窗按一下右鍵會把它拉到前面，那時才看得到全寬。
	// 這種錯不會有任何症狀：小視窗照樣畫得出來、玩得動，只是視野少了三分之二。
	editX, editY = 5, 21
	// editFrameW 是編輯視窗外框的寬度，四邊都是 3 像素。
	editFrameW = 3
	// ditherTop 是網點區相對工具盤上緣的位移（原版量到網點從 y 235 起，
	// 工具盤在 y 55，所以是 180）。
	ditherTop = 180
	// 預設大小。玩家可以用「調整編輯窗大小」（Ctrl-R）改，改過的存在
	// Game 的 ew／eh，畫圖與命中判斷一律走 g.editSize()。
	editW, editH = 575, 304
	// 可調整的範圍。下限留得下工具盤（57 寬）加上幾格地圖。
	editMinW, editMinH = 200, 150

	// menuTextY 是選單列文字的**字格上緣**。原版量到的筆畫在 y 4–12
	// ——8×14 的字格從 2 起算，而 remake 的字格也是 14 原版像素高，
	// 所以直接用原版的字格上緣。
	menuTextY = 2

	editTitleY = 24 // 標題列
	editTitleH = 14
	editInfoY  = 38 // 資金／訊息帶
	editInfoH  = 17
	// 地圖區。⚠ 圖塊從 **55** 開始，不是 54——54 那一列是地圖區的白色外框。
	// 量法：把畫面上每個 16×16 位移拿去比對第 0 庫的 960 張圖塊，
	// 命中最多的位移就是格網原點（螢幕 y 除以 16 餘 15，換算成遊戲座標是 55）。
	// 先前寫 54，等於整張地圖上移一像素，而且沒有那圈框。
	editViewY  = 55 // 地圖區（圖塊的第一列）
	editViewH  = 256
	editToolY  = 311 // 目前工具帶

	// 帶內文字的字格位置，全部量自原版（見 drawEditWindow 的說明）。
	fundsTextX = 8   // `Funds:`
	msgTextX   = 136 // 訊息
	infoTextY  = 39  // 資金帶的字格上緣（筆畫 41–49）
	toolTextX  = 64  // 目前工具的名稱與造價
	editToolH  = 14

	// 工具盤圖（庫 2，57×182）。⚠ 是 (8,55) 不是 (6,53)：拿原版與 remake
	// 的同一張美術做二維位移搜尋（顏色先正規化到 EGA 四階，否則調色盤差異
	// 會蓋過對齊訊號），最佳位移是 dx=dy=−2，2436 點中對上 2402 點。
	editPalX, editPalY = 8, 55
	editViewX          = 64    // 地圖區左緣（圖塊的第一欄；63 是白框）
	editViewW          = 512   // 64–575，剛好 32 格
	// 需求指標（庫 3，46×39）。位置量自原版：C·R·I 的洋紅條在 x 15–53、
	// y 253–259，據此回推圖的左上角。先前是 (8,236)，偏左 4、偏上 1。
	editDemandX        = 12
	editDemandY        = 237

	// City Form 視窗。座標全部量自原版（`workplace/dosbox/w3-01-maps.png`，
	// 逐列逐行讀色）：外框 x 240–639、y 21–347；標題列 y 25–41；
	// 圖層圖示欄從 (244,44) 起，一格 25 像素高，九格；地圖本體 x 274–633、
	// y 44–343，剛好 120×100 格 × 3 像素。
	mapX, mapY = 240, 21 // City Form 視窗外框
	mapW, mapH = 400, 327
	// 圖示欄最左邊那一行黑線是視窗自己畫的，圖示美術（庫 5，26×226）
	// 從 245 起——照 244 blit 會整條左移一像素。
	mapIconEdgeX = 244
	mapIconX     = 245 // 圖層圖示（庫 5，26×226）
	mapIconY   = 44
	mapIconH   = 25 // 一格的高，選取的黃框照這個間距走
	// 原版的圖示欄只有**九格**，而圖層有十一個——最後兩個圖示各管兩個
	// 圖層（警力／消防那一格印著 `F` 與 `P`）。對應關係還沒解，
	// 記在 CONTEXT.md §5.5；黃框畫得出來的只有前九個。
	mapIconCount = 9
	mapTitleX    = 244 // 標題列
	mapTitleY    = 25
	mapTitleW    = 392
	mapContentY  = 42 // 內容區（白底 ＋ 綠邊的圖示欄 ＋ 地圖）
	mapContentH  = 304
	mapViewX   = 274 // 地圖本體
	mapViewY   = 44
)

// 選單列的四個標題，中心 x 量自原版。
var menuTitles = []struct {
	centerX int
	sec     int // 訊息檔第 16 段（主選單）的索引
}{
	{112, 0}, // SYSTEM
	{250, 1}, // OPTIONS
	{402, 2}, // DISASTERS
	{554, 3}, // WINDOWS
}

// EGA 十六色裡本版面用得到的幾個。值取自原版截圖的實際位元組
// （不是教科書上的 0x55／0xaa，見 docs/formats/03-pgf-graphics.md §3）。
var (
	colDesktop  = color.RGBA{0xaa, 0xaa, 0xaa, 0xff} // 桌面灰
	colMenuBar  = color.RGBA{0x55, 0xff, 0xff, 0xff} // 選單列亮青
	colMenuInk  = color.RGBA{0x00, 0x00, 0xaa, 0xff} // 選單字深藍
	colEditFrm  = color.RGBA{0x00, 0x00, 0xaa, 0xff} // 編輯視窗框深藍
	colMapFrm   = color.RGBA{0xff, 0x55, 0x55, 0xff} // 地圖視窗框亮紅
	colInfoBand = color.RGBA{0x55, 0x55, 0x55, 0xff} // 資金帶深灰
	colTitleBar = color.RGBA{0xaa, 0xaa, 0xaa, 0xff} // 標題列灰
	colMapFrmD  = color.RGBA{0xaa, 0x00, 0x00, 0xff} // 地圖視窗框外圈暗紅
	colMapGreen = color.RGBA{0x00, 0xaa, 0x00, 0xff} // 圖層圖示欄的綠邊
	colInk      = color.RGBA{0x00, 0x00, 0x00, 0xff}
	colInkLight = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// s 把原版座標換成螢幕像素。
func s(v int) float32 { return float32(v * UIScale) }

// fill 在原版座標畫一個實心矩形。
func fill(dst *ebiten.Image, x, y, w, h int, c color.RGBA) {
	vector.DrawFilledRect(dst, s(x), s(y), s(w), s(h), c, false)
}

// blit 把一張介面美術畫在原版座標 (x,y)，放大 UIScale 倍。
func blit(dst *ebiten.Image, img *ebiten.Image, x, y int) {
	if img == nil {
		return
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(UIScale, UIScale)
	op.GeoM.Translate(float64(x*UIScale), float64(y*UIScale))
	dst.DrawImage(img, &op)
}

// drawClassic 畫原版版面。
func (g *Game) drawClassic(dst *ebiten.Image) {
	fill(dst, 0, 0, OrigW, OrigH, colDesktop)
	g.drawMenuBar(dst)
	// 兩個視窗是重疊的，畫的順序就是疊的順序。原版一開始 City Form 在前面，
	// 對編輯視窗按一下就換它到前面。
	switch {
	case g.mapClosed:
		g.drawEditWindow(dst)
	case g.editFront:
		g.drawCityFormWindow(dst)
		g.drawEditWindow(dst)
	default:
		g.drawEditWindow(dst)
		g.drawCityFormWindow(dst)
	}
	if g.resizing {
		g.drawResizeHint(dst)
	}
	g.drawQueryPanel(dst)
}

// inCityForm 判斷畫面座標在不在 City Form 視窗上。
func inCityForm(mx, my int) bool {
	x, y := mx/UIScale, my/UIScale
	return x >= mapX && x < mapX+mapW && y >= mapY && y < mapY+mapH
}

// editSize 回傳編輯視窗目前的大小（原版像素）。玩家沒調過就是預設值。
func (g *Game) editSize() (int, int) {
	w, h := g.ew, g.eh
	if w == 0 {
		w = editW
	}
	if h == 0 {
		h = editH
	}
	return w, h
}

// inEditWindow 判斷畫面座標在不在編輯視窗上。
func (g *Game) inEditWindow(mx, my int) bool {
	w, h := g.editSize()
	x, y := mx/UIScale, my/UIScale
	return x >= editX && x < editX+w && y >= editY && y < editY+h
}

// editViewSize 回傳地圖區的大小（原版像素），跟著視窗大小走。
// 一律取整到整格，否則邊緣會出現半格，而半格看起來很像「圖塊畫錯了」。
func (g *Game) editViewSize() (int, int) {
	w, h := g.editSize()
	sz := g.tiles.Size
	vw := (w - (editViewX - editX) - 4) / sz * sz
	// ⚠ 扣掉的是**工具帶的高度**（14），不是 18。原版的地圖區是 y 55–310
	// ＝ 256 像素 ＝ 16 列；扣 18 只剩 240，等於**少畫一列地圖**，
	// 而且會在地圖與工具帶之間留下 12 像素的灰色空隙。
	// 實測：原版 y 55–310 全是地圖，311 才是工具帶的第一列。
	vh := (h - (editViewY - editY) - editToolH) / sz * sz
	if vw < sz {
		vw = sz
	}
	if vh < sz {
		vh = sz
	}
	return vw, vh
}

// raiseWindowAt 依點擊位置決定哪個視窗到前面。
// 回傳 true 代表這一下**只**用來換疊放順序，不該再當成別的操作。
//
// ⚠ 只有點在**兩個視窗重疊的那一塊**時才吞掉這一下。點在編輯視窗只屬於
// 自己的地方（City Form 蓋不到的左半邊）要照樣蓋東西——吞掉的話，
// 玩家每次切回編輯視窗都要多點一下，而且第一下「沒反應」。
// 試玩腳本就是這樣少蓋了一座發電廠，而畫面像素檢查照樣過（疊放順序換了，
// 畫面確實變了），只有存檔內容檢查抓得到。
//
// 原版是按右鍵拉到前面；這裡用左鍵，因為 remake 的右鍵沒有別的用途。
func (g *Game) raiseWindowAt(mx, my int) bool {
	if g.mapClosed {
		if g.inEditWindow(mx, my) {
			g.editFront = true
		}
		return false
	}
	if inCityForm(mx, my) {
		if g.editFront {
			g.editFront = false
			return true // 重疊區：這一下只用來把 City Form 叫回前面
		}
		return false
	}
	if g.inEditWindow(mx, my) {
		g.editFront = true // 非重疊區：順手拉到前面，但這一下照樣算數
	}
	return false
}

func (g *Game) drawMenuBar(dst *ebiten.Image) {
	fill(dst, 0, 0, OrigW, menuBarH, colMenuBar)
	for i, m := range menuTitles {
		label := trimMenu(g.txt.S(i18n.SecMenu, m.sec))
		w := g.font.Measure(label)
		x := m.centerX*UIScale - w/2
		c := colMenuInk
		if g.openMenu == i+1 {
			// 拉開的那一項反白：原版是把標題塗成深藍、字變青。
			fill(dst, m.centerX-w/(2*UIScale)-2, 1, w/UIScale+4, menuBarH-2, colMenuInk)
			c = colMenuBar
		}
		g.font.Draw(dst, label, x, menuTextY*UIScale, c)
	}
}

// drawEditWindow 畫編輯視窗：標題列、資金／訊息帶、工具盤、地圖、工具帶。
func (g *Game) drawEditWindow(dst *ebiten.Image) {
	ew, eh := g.editSize()
	vw, vh := g.editViewSize()
	// 工具帶：原版是視窗最底下 14 個像素（y 311–324），與地圖區緊鄰，
	// 中間沒有空隙。
	toolY := editY + eh - editToolH
	fill(dst, editX, editY, ew, eh, colEditFrm)
	// ⚠ 外框是**三個像素**，四邊一樣（原版量到左 x 5–7、右 577–579、
	// 上 y 21–23）。先前左右只畫一像素，上緣卻是三像素——自己就不一致。
	fill(dst, editX+editFrameW, editTitleY, ew-2*editFrameW, editTitleH, colTitleBar)
	fill(dst, editX+editFrameW, editInfoY, ew-2*editFrameW, editInfoH, colInfoBand)
	fill(dst, editX+editFrameW, editViewY-1, ew-2*editFrameW, toolY-editViewY+1, colDesktop)
	// 工具盤與需求指標之間、以及需求指標下方，原版填的是**黑白一像素網點**
	// （x 8–63、y 235–310，(x+y) 偶數為黑）。先前那裡是空的桌面灰。
	drawLeftColumnDither(dst, toolY)

	g.drawEditTitle(dst, ew)

	// 資金與訊息：原版是同一條帶，資金在左、訊息接在後面。
	//
	// ⚠ 欄位是**量出來的字格**，不是估的：`Funds:` 在 x 8、金額在 64、
	// 訊息在 136（原版 8 像素一格，量自 workplace/dosbox/ui-04-windows.png）。
	g.font.Draw(dst, g.fundsText(), fundsTextX*UIScale, infoTextY*UIScale, colInkLight)
	if g.message != "" {
		g.font.Draw(dst, g.message, msgTextX*UIScale, infoTextY*UIScale, colInkLight)
	}

	g.drawEditMap(dst)
	g.drawEditViewFrame(dst, vw, vh)
	blit(dst, g.tiles.UIImage(BankToolPalette, 0), editPalX, editPalY)
	g.drawToolHighlight(dst)
	blit(dst, g.tiles.UIImage(BankDemand, 0), editDemandX, editDemandY)
	g.drawDemandBars(dst)

	// 目前工具帶。底色是一像素的白藍網點（實測原版：y 311 整列白、
	// 312–321 網點、322–324 實心藍，x 5–579 滿版寬）。
	//
	// ⚠ **字是藍色畫在白底上，不是白字疊在網點上。** 原版在文字寬度上鋪一塊
	// 白底再寫藍字（實測 `y=320` 有一段連續白 x 64–200，正好是 "Residential:
	// $100" 十七個字 × 8 像素），右端的 `+` 把手也有自己的白底。
	// 先前照「白字疊網點」畫，中文的細筆畫整個融進網點裡，**完全讀不出來**。
	//
	// 唯一的取捨：原版的白底是 8 列（8×8 字型），remake 的字格是 14 列，
	// 所以白底跟著長到整條帶高。這是 CJK 字高的必然結果。
	drawToolBandBG(dst, editX, toolY, ew)
	toolText := g.currentToolText()
	if tw := g.font.Measure(toolText) / UIScale; tw > 0 {
		fill(dst, toolTextX, toolY, tw, editToolH, colInkLight)
	}
	g.font.Draw(dst, toolText, toolTextX*UIScale, toolY*UIScale, colEditFrm)
	// 右下角的 `+` 是原版的改變大小把手（CP437 0x2B，8×8 字型），
	// 同樣是白底藍字（原版量到白底 x 567–576）。
	fill(dst, editX+ew-14, toolY+2, 10, 8, colInkLight)
	drawGlyph8(dst, glyphPlus, editX+ew-13, toolY+2, colEditFrm)
}

// 需求指標的三根長條。**順序是 C·R·I**（洋紅條上的字就是這個順序），
// 不是 R·C·I——照 R·C·I 畫會讓三根長條全部對錯欄。
//
// 幾何量自原版（`workplace/dosbox/bos-00-view.png` 的字元圖）：
//
//	欄 x      20–24（商）／33–37（住）／46–50（工），各寬 5
//	往上      y 240–251，最多 12 列，底邊固定在 251
//	往下      y 261–271，最多 11 列，頂邊固定在 261
//
// 上下不對稱是原版就這樣（面板下半被那塊深紅記號與網點吃掉空間）。
//
// 高度公式：valve 夾在 ±1500 之後線性換算。兩個極值都是實測的——
// 波士頓存檔 `CValve=-1500` 對到向下 11 列、`RValve=2000`（夾成 1500）
// 對到向上 12 列（`MiscHis[5..7]` 就是三個 valve，`s_sim.c:428`）。
// **中間值是假設線性**：Micropolis 自己也是線性的（`w_update.c:230
// SetDemand` 送出 `valve/100`），但 DOS 版沒有中段的實測點。
const (
	demandBarW   = 5
	demandUpMax  = 12  // 往上最多幾列
	demandDnMax  = 11  // 往下最多幾列
	demandUpBot  = 251 // 往上長的底邊
	demandDnTop  = 261 // 往下長的頂邊
	demandClamp  = 1500
)

// demandBarX 是三根長條的左緣（原版座標），順序 C、R、I。
var demandBarX = [3]int{20, 33, 46}

func (g *Game) drawDemandBars(dst *ebiten.Image) {
	w := g.world
	bars := [3]struct {
		v int
		c color.RGBA
	}{
		{w.CValve, colDemBarC},
		{w.RValve, colDemBarR},
		{w.IValve, colDemBarI},
	}
	for i, b := range bars {
		v := b.v
		if v > demandClamp {
			v = demandClamp
		}
		if v < -demandClamp {
			v = -demandClamp
		}
		switch {
		case v > 0:
			h := v * demandUpMax / demandClamp
			if h > 0 {
				fill(dst, demandBarX[i], demandUpBot-h+1, demandBarW, h, b.c)
			}
		case v < 0:
			h := -v * demandDnMax / demandClamp
			if h > 0 {
				fill(dst, demandBarX[i], demandDnTop, demandBarW, h, b.c)
			}
		}
	}
}

// drawLeftColumnDither 填工具盤下方那一塊黑白網點。
//
// 量自原版（`workplace/screen-parity/dos.png`）：x 8–63、y 235–310，
// `(x+y)` 為偶數的像素是黑、奇數是白。需求指標的白色面板疊在它上面，
// 所以只有面板四周看得到網點——先前那裡留成桌面灰，整條左欄的明度都不對。
//
// 上緣接在工具盤圖（庫 2，57×182，畫在 (8,55)）的下面，下緣接到工具帶。
func drawLeftColumnDither(dst *ebiten.Image, toolY int) {
	x0, x1 := editX+editFrameW, editViewX-1
	y0, y1 := editPalY+ditherTop, toolY-1
	for y := y0; y <= y1; y++ {
		for x := x0; x < x1; x++ {
			c := colInkLight
			if (x+y)%2 == 0 {
				c = colInk
			}
			fill(dst, x, y, 1, 1, c)
		}
	}
}

// drawToolBandBG 畫工具帶的底：一列白線、十列一像素白藍網點、三列實心藍。
//
// 量自原版（`workplace/screen-parity/dos.png`，x=536–543 逐列讀色）：
//
//	y 311      整列白
//	y 312–321  白藍交錯，(x+y) 偶數為白
//	y 322–324  實心藍
//
// 橫向是滿版：x 5–579，蓋過視窗外框那兩欄。
func drawToolBandBG(dst *ebiten.Image, x0, y0, w int) {
	fill(dst, x0, y0, w, 1, colInkLight)
	for y := 1; y <= 10; y++ {
		for x := 0; x < w; x++ {
			c := colEditFrm
			if (x0+x+y0+y)%2 == 0 {
				c = colInkLight
			}
			fill(dst, x0+x, y0+y, 1, 1, c)
		}
	}
	fill(dst, x0, y0+11, w, 3, colEditFrm)
}

// drawEditViewFrame 畫地圖區四周那一圈**一像素白框**。
//
// 原版量出來的位置（遊戲座標，workplace/dosbox/q9ctlA-09-back.png）：
// 上緣 y=54 從 x=63 到 576、左緣 x=63、右緣 x=576、下緣 y=311。
// 圖塊本身是 x 64–575、y 55–310，剛好 512×256 ＝ 32×16 格。
//
// ⚠ remake 的工具帶比原版高三個像素（只有一種字高），所以下緣的
// 絕對位置跟原版差三格——那是既有的已知偏差，不是這裡要修的。
func (g *Game) drawEditViewFrame(dst *ebiten.Image, vw, vh int) {
	l, t := editViewX-1, editViewY-1
	r, b := editViewX+vw, editViewY+vh
	fill(dst, l, t, r-l+1, 1, colInkLight)
	fill(dst, l, b, r-l+1, 1, colInkLight)
	fill(dst, l, t, 1, b-t+1, colInkLight)
	fill(dst, r, t, 1, b-t+1, colInkLight)
}

// drawEditTitle 畫編輯視窗標題列：關閉記號、城市名、年月。
//
// 位置是拖著原版的視窗改大小量出來的（workplace/dosbox/t2-*.png，
// 右緣從 579 移到 435）：
//
//   - 城市名**置中**於 [editX, 右緣−72]：兩張圖的中心都比視窗中心左 36.5。
//   - 年月**靠右**：左緣 ＝ 右緣−99，位移量與右緣完全同步。
//   - 關閉記號固定在左端 editX+3。
func (g *Game) drawEditTitle(dst *ebiten.Image, ew int) {
	right := editX + ew - 1
	drawGlyph8(dst, glyphSun, editX+3, editTitleY+2, colInk)

	name := g.world.CityName
	if name != "" {
		g.font.DrawCentered(dst, name, editX*UIScale, editTitleY*UIScale,
			(right-72-editX)*UIScale, colInk)
	}
	g.font.Draw(dst, g.dateText(), (right-99)*UIScale, editTitleY*UIScale, colInk)
}

// drawEditMap 畫編輯視窗裡的地圖。原版一格 16×16，可見 11×16 格。
func (g *Game) drawEditMap(dst *ebiten.Image) {
	across, down := g.tilesAcross(), g.tilesDown()
	sz := g.tiles.Size
	for ty := 0; ty < down; ty++ {
		for tx := 0; tx < across; tx++ {
			mx, my := g.camX+tx, g.camY+ty
			if mx < 0 || my < 0 || mx >= sim.WorldX || my >= sim.WorldY {
				continue
			}
			img := g.tiles.TileImage(g.world.TileNum(mx, my))
			var op ebiten.DrawImageOptions
			op.GeoM.Scale(UIScale, UIScale)
			op.GeoM.Translate(
				float64((editViewX+tx*sz)*UIScale),
				float64((editViewY+ty*sz)*UIScale))
			dst.DrawImage(img, &op)
		}
	}
}

// drawToolHighlight 把目前選的工具那一格框起來。
// 原版是把該格的外框畫成黃色（見 workplace/dosbox/ui-00-clean.png）。
func (g *Game) drawToolHighlight(dst *ebiten.Image) {
	i := paletteIndexOf(g.tool)
	if i < 0 {
		return
	}
	x, y := paletteCell(i)
	vector.StrokeRect(dst, s(x), s(y), s(palCellW), s(palCellH),
		float32(UIScale), color.RGBA{0xff, 0xff, 0x55, 0xff}, false)
}

// 工具盤的格線。量自 workplace/gfx/bank02-00.png：
// 兩欄的暗框在 x=2／31，七列在 y=5、30、55、80、105、130、155，
// 所以間距是 29×25、格子 26×23。
const (
	palCols   = 2
	palRows   = 7
	palCellW  = 26
	palCellH  = 23
	palPitchX = 29
	palPitchY = 25
)

func paletteCell(i int) (int, int) {
	c, r := i%palCols, i/palCols
	return editPalX + 2 + c*palPitchX, editPalY + 5 + r*palPitchY
}

// paletteOrder 是工具盤十四格的工具，順序照原版由左到右、由上到下。
//
// ⚠ 十五個建造工具只有十四格：**發電廠兩種共用一格**，點下去會開一個
// 副選單（訊息檔第 5 段，兩筆）選火力或核能——`powerCell`／`winPower`／
// `powerTools`。
var paletteOrder = [palCols * palRows]sim.Tool{
	sim.ToolBulldozer, sim.ToolRoad,
	sim.ToolWire, sim.ToolRail,
	sim.ToolPark, sim.ToolResidential,
	sim.ToolCommercial, sim.ToolIndustrial,
	sim.ToolPolice, sim.ToolFireStation,
	sim.ToolStadium, sim.ToolCoalPower,
	sim.ToolSeaport, sim.ToolAirport,
}

func paletteIndexOf(t sim.Tool) int {
	// 核能發電廠與火力共用一格，highlight 要一起亮。
	if t == sim.ToolNuclear {
		return powerCell
	}
	for i, v := range paletteOrder {
		if v == t {
			return i
		}
	}
	return -1
}

// paletteHit 把畫面座標換成工具盤格號；不在盤上回 −1。
func paletteHit(px, py int) int {
	x, y := px/UIScale, py/UIScale
	for i := range paletteOrder {
		cx, cy := paletteCell(i)
		if x >= cx && x < cx+palCellW && y >= cy && y < cy+palCellH {
			return i
		}
	}
	return -1
}

// ditherRect 畫一塊一像素棋盤網點。原版的標題列與空白欄位都用它，
// `(x+y)` 為偶數畫 a、奇數畫 b。
func ditherRect(dst *ebiten.Image, x, y, w, h int, a, b color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			c := b
			if (i+j)%2 == 0 {
				c = a
			}
			fill(dst, i, j, 1, 1, c)
		}
	}
}

// drawMapIconState 補圖層圖示欄的兩件事：選中那一格的黃框，
// 以及圖示下方那塊黑白網點。
//
// 量自原版（`workplace/dosbox/w3-01-maps.png`）：黃框是 x 247–267、
// y 44–66 的兩像素外框，一格 25 像素往下數；網點在 x 244–272、y 272–343，
// `(x+y)` 為偶數是黑、奇數是白，與工具盤下方那一塊同一種。
//
// ⚠ 網點先前留成黑色。畫面上看起來只是「下面暗一塊」，但那一塊佔了
// City Form 左欄三分之一的高度，整個視窗的明度都不對。
func (g *Game) drawMapIconState(dst *ebiten.Image) {
	if int(g.layer) < mapIconCount {
		y := mapIconY + mapIconH*int(g.layer)
		c := color.RGBA{0xff, 0xff, 0x55, 0xff}
		fill(dst, 247, y, 21, 2, c)
		fill(dst, 247, y+21, 21, 2, c)
		fill(dst, 247, y, 2, 23, c)
		fill(dst, 266, y, 2, 23, c)
	}
	y0 := mapIconY + mapIconH*mapIconCount + 3
	y1 := mapContentY + mapContentH - 3
	for y := y0; y <= y1; y++ {
		for x := mapIconEdgeX; x <= mapViewX-3; x++ {
			c := colInkLight
			if (x+y)%2 == 0 {
				c = colInk
			}
			fill(dst, x, y, 1, 1, c)
		}
	}
}

// drawCityFormWindow 畫 City Form 視窗：標題列、圖層圖示、全市地圖。
func (g *Game) drawCityFormWindow(dst *ebiten.Image) {
	// 外框兩圈：外圈暗紅、內圈亮紅。標題列 y 25–41 整塊是藍白一像素棋盤
	// （`(x+y)` 為偶數畫白）；標題本身是**白底藍字**（原版 y=27 與 y=39
	// 在 x 399–487 是整段白，中間才是字）。內容區 x 242–635、y 42–345，
	// 圖層圖示欄那一條 30 像素寬是綠邊，地圖本體 274,44 起 360×300。
	fill(dst, mapX, mapY, mapW, mapH, colMapFrmD)
	fill(dst, mapX+1, mapY+1, mapW-2, mapH-2, colMapFrm)
	ditherRect(dst, mapTitleX, mapTitleY, mapTitleW, 17, colInkLight, colMenuInk)
	fill(dst, mapX+2, mapContentY, mapW-6, mapContentH, colInkLight)
	fill(dst, mapX+2, mapContentY, 30, mapContentH, colMapGreen)
	fill(dst, mapViewX, mapViewY, sim.WorldX*3, sim.WorldY*3, colInk)
	title := trimMenu(g.txt.S(i18n.SecMapTitle, 0))
	tw := g.font.Measure(title) / UIScale
	fill(dst, mapX+mapW/2-tw/2-8, mapTitleY+2, tw+16, 13, colInkLight)
	g.font.Draw(dst, title,
		(mapX+mapW/2)*UIScale-g.font.Measure(title)/2, (mapTitleY+2)*UIScale, colMenuInk)
	// 圖示欄左右各有一條**一像素棋盤**的邊（x=244 與 270–271），
	// 庫 5 的美術只有中間那 25 行。原版的規則同其他網點：`(x+y)` 偶數是黑。
	ih := mapIconH * mapIconCount
	ditherRect(dst, mapIconEdgeX, mapIconY, 1, ih, colInk, colInkLight)
	ditherRect(dst, mapViewX-4, mapIconY, 2, ih, colInk, colInkLight)
	blit(dst, g.tiles.UIImage(BankMapIcons, 0), mapIconX, mapIconY)
	g.drawMapIconState(dst)

	// 全市地圖：120×100 格，一格 3×3 原版像素 ＝ 360×300，塞得進視窗。
	// 「都市型態」圖層畫的是圖形檔自帶的縮圖，不是純色方塊——見 minimap.go。
	sc := (mapX + mapW - 2 - mapViewX) / sim.WorldX
	if s2 := (mapY + mapH - 2 - mapViewY) / sim.WorldY; s2 < sc {
		sc = s2
	}
	if sc < 1 {
		sc = 1
	}
	g.drawMinimap(dst, mapViewX*UIScale, mapViewY*UIScale, sc*UIScale)
	// 目前視野的框。
	vector.StrokeRect(dst,
		s(mapViewX+g.camX*sc), s(mapViewY+g.camY*sc),
		s(g.tilesAcross()*sc), s(g.tilesDown()*sc),
		float32(UIScale), colMenuInk, false)
}

// fundsText 是資金帶左半的字。原版寫 `Funds: $20,000`。
func (g *Game) fundsText() string {
	return fmt.Sprintf("資金 $%s", comma(g.world.TotalFunds))
}

// currentToolText 是工具帶的字。原版寫 `Residential: $100`——
// 名稱與造價都在訊息檔第 1 段裡，連冒號都是。
func (g *Game) currentToolText() string {
	for _, b := range toolButtons {
		if b.tool == g.tool && b.msgIdx >= 0 {
			return trimMenu(g.txt.S(i18n.SecToolCost, b.msgIdx))
		}
	}
	return ""
}

// comma 把金額加上千位逗號。原版就是這樣印的。
func comma(n int) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// 發電廠副選單。原版的工具盤只有十四格，但建造工具有十五個——
// 火力與核能共用第 11 格。
//
// ⚠ **副選單是 remake 加的，不是原版行為。** 原版點那一格會直接選中
// 三千元那一種，工具帶當場顯示它的名字；試過按住不放、連點三次、
// 以及把時間拉到 1957 年的東京（核能在 1950 年後才解鎖），
// 三種都沒有出現任何選單（`workplace/dosbox/pw-*.png`、`pw2-*`、`pw3-*`）。
// 原版怎麼蓋核能廠**還沒解**。
//
// 訊息檔第 5 段是**兩個發電廠的名字**，不是選單的三行字：
// ASIA 是 `Well`／`Water Wheel`、WEST 是 `Water Wheel`／`Steam Pump`，
// 第 0 筆對三千元那一種（火力）、第 1 筆對五千元那一種（核能）。
const powerCell = 11 // 工具盤第 11 格（0 起算）＝ 發電廠

// powerSubOpen 記錄副選單開著沒有；開著時畫在工具盤右邊。
func (g *Game) openPowerSub() {
	g.win = winPower
	g.sysRow = 0
}

// powerTools 是副選單的兩個選項，順序照訊息檔第 5 段。
var powerTools = []struct {
	msg  int
	tool sim.Tool
}{
	{0, sim.ToolCoalPower},
	{1, sim.ToolNuclear},
}

// SetCamera 把鏡頭左上角擺到指定的格子。給試玩腳本用（`-cam`）：
// 腳本要能算出「某一格在畫面上的哪裡」，而那需要鏡頭是已知的。
func (g *Game) SetCamera(x, y int) {
	g.camX, g.camY = x, y
	g.clampCamera()
}

// dateText 是標題列右邊的年月。原版寫 `Jan 1849`。
func (g *Game) dateText() string {
	year := 1900 + g.world.CityTime/48
	month := trimMenu(g.txt.S(i18n.SecMonth, (g.world.CityTime%48)/4))
	return fmt.Sprintf("%d %s", year, month)
}

// drawResizeHint 在調整大小時把編輯視窗的外框畫成醒目的顏色，
// 並在資金帶上提示操作。沒有提示的話玩家不知道自己進了另一個模式，
// 而方向鍵原本是捲地圖的——那會讓人以為畫面卡住了。
func (g *Game) drawResizeHint(dst *ebiten.Image) {
	ew, eh := g.editSize()
	vector.StrokeRect(dst, s(editX), s(editY), s(ew), s(eh),
		float32(2*UIScale), color.RGBA{0xff, 0xff, 0x55, 0xff}, false)
	msg := "方向鍵調整大小，Enter 或 Esc 結束"
	g.font.Draw(dst, msg, (editX+3)*UIScale, infoTextY*UIScale,
		color.RGBA{0xff, 0xff, 0x55, 0xff})
}

// resizeEdit 依方向鍵改編輯視窗的大小。一次一格（16 原版像素），
// 這樣地圖區永遠是整數格。
func (g *Game) resizeEdit(dx, dy int) {
	ew, eh := g.editSize()
	sz := g.tiles.Size
	ew = clampInt(ew+dx*sz, editMinW, OrigW-editX-1)
	eh = clampInt(eh+dy*sz, editMinH, OrigH-editY-1)
	g.ew, g.eh = ew, eh
	g.clampCamera()
}

// handleResizeMouse 處理「改變編輯視窗大小」的拖曳。
//
// 原版的流程是 **Ctrl-R 讓外框變黃 → 拖右下角 → 放開就套用並離開模式**，
// 三件事都實測過（`workplace/dosbox/t2-*.png`、`t3-*.png`）：
//
//   - 只按 Ctrl-R 不拖，送方向鍵沒有反應——所以方向鍵不是原版的操作方式。
//   - 不按 Ctrl-R 直接拖右下角，視窗一動也不動。
//   - 拖完之後外框已經變回藍色，不必再按 Enter。
//
// ⚠ 原版**從哪裡開始拖**沒有量到唯一答案：實測那一次是從外框右下角
// (578,321) 起手，而右下角另有一個 `+` 記號。這裡兩個都接受。
// 方向鍵也留著（`resizeEdit`），那是 remake 多給的，不是原版行為。
func (g *Game) handleResizeMouse(mx, my int) bool {
	if !g.resizing {
		return false
	}
	if g.dragResize {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.setEditCorner(mx, my)
		} else {
			g.dragResize = false
			g.resizing = false
		}
		return true
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	ew, eh := g.editSize()
	// 右下角的把手區：`+` 記號那一格加上外框的轉角，一起算。
	hx, hy := editX+ew-14, editY+eh-14
	if mx < hx*UIScale || my < hy*UIScale {
		return false
	}
	g.dragResize = true
	g.setEditCorner(mx, my)
	return true
}

// setEditCorner 把編輯視窗的右下角拉到游標處，取整到整數格。
func (g *Game) setEditCorner(mx, my int) {
	g.ew = clampInt(mx/UIScale-editX+1, editMinW, OrigW-editX-1)
	g.eh = clampInt(my/UIScale-editY+1, editMinH, OrigH-editY-1)
	g.clampCamera()
}
