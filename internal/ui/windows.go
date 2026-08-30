package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 四個視窗。快速鍵沿用原版（說明書 p.35）：Ctrl-M 地圖、Ctrl-G 統計圖、
// Ctrl-B 預算、Ctrl-U 評估。
type window int

const (
	winNone window = iota
	winMaps
	winGraphs
	winBudget
	winEval
	// winDisaster 是災難選單（原版 Alt-D，說明書 p.34）。原版是下拉選單，
	// 這裡做成視窗，選項與順序照訊息檔第 20 段。
	winDisaster
	// winSystem 是系統選單（原版 Alt-S，說明書 p.29–31），
	// winScenario／winStyle 是它的兩個副選單。見 sysmenu.go。
	winSystem
	winScenario
	winStyle
	// winAbout 是「關於本遊戲」（訊息檔第 17 段第 0 筆）。授權條款要求
	// 副本要附條款與 Required Notice，而 APK 沒地方放文字檔——見 about.go。
	winAbout
	// winSaveAs 是「以……檔名儲存」（同段第 9 筆），遊戲裡唯一的文字輸入。
	winSaveAs
	// winSpeed 是功能選單的速度副選單（訊息檔第 19 段，五段速度）。
	winSpeed
	// winPower 是工具盤發電廠那一格的副選單（訊息檔第 5 段）。
	winPower
	// winLoad 是「讀取舊有檔案」的檔案清單（同段第 8 筆）。原版開的是
	// 檔名對話框；這裡列出存檔目錄裡的 `.cty`，上下鍵選、Enter 讀。
	winLoad
	// winLangSel／winMusic 是 remake 自己加的兩個副選單：換語言、開關音樂。
	// 原版沒有——它只有一種語言，也沒有音樂（docs/re/19-no-music.md）。
	winLangSel
	winMusic
)

// disasterItems 是災難選單的六個項目，順序照訊息檔第 20 段。
// 原版的選單直接呼叫這六支（`res/whead.tcl:271` 起）。
var disasterItems = []func(*sim.World){
	(*sim.World).MakeFire,
	(*sim.World).MakeFlood,
	(*sim.World).MakeAirCrash,
	(*sim.World).MakeTornado,
	(*sim.World).MakeEarthquake,
	(*sim.World).MakeMonster,
}

// 地圖視窗的十種全貌圖。順序與訊息檔第 10 段一致，名稱從那裡取。
type mapLayer int

const (
	layerCityForm mapLayer = iota
	layerPower
	layerTransport
	layerPopDensity
	layerTraffic
	layerPollution
	layerCrime
	layerLandValue
	layerPolice
	layerFire
	layerGrowth
	layerCount
)

// 分佈密度表的配色：由低到高。原版用的是十六色調色盤裡的一組漸層，
// 這裡自己配一組在深色底上讀得出來的。
// 數值圖層的色階。**三個 EGA 色 ＋ 兩段一像素網點，共五級**，
// 這是量原版得到的（`workplace/dosbox/mi-04-icon3.png` 的人口分佈圖，
// 3000 個 6×6 密度格逐格分類）：
//
//	青 523 格／青＋洋紅網點 925／洋紅 97／洋紅＋紅網點 159／紅 7，
//	另外 1285 格完全沒有色階色——那是**值太低，直接畫地形**。
//
// 三個色就是色階圖例（庫 6，`Max`／`Min`）自己用的顏色。
var rampLow = color.RGBA{0x55, 0xff, 0xff, 0xff}  // 青
var rampMid = color.RGBA{0xaa, 0x00, 0xaa, 0xff}  // 洋紅
var rampHigh = color.RGBA{0xff, 0x55, 0x55, 0xff} // 紅

// rampSteps 是五級的門檻。
//
// ⚠ **前四個是已確認的**（Micropolis `g_map.c:105 GetCI`：50／100／150／200
// 分成 none／low／medium／high／veryhigh），**第五個是暫代值**——
// X11 版只有四級，DOS 版實測是五級，多出來的那一級門檻沒有量到，
// 先照等距外推 250。記在 CONTEXT.md §5.5，不要當成已確認。
var rampSteps = [5]int{50, 100, 150, 200, 250}

// rampColorAt 回傳一格的色階色；回 false 代表值太低，該畫地形。
//
// 網點在原版是**逐像素**的，remake 的縮圖一格 3×3 像素，所以改成
// **逐格**交錯——同一塊區域裡兩色相間，遠看的明度一樣，近看格子大一點。
func rampColorAt(v, max, x, y int) (color.RGBA, bool) {
	// 兩種來源的量程不同（密度類 0–255、警消覆蓋 0–1000），先正規化到
	// Micropolis 的 0–255 再比門檻。
	if max <= 0 {
		return rampLow, false
	}
	n := v * 255 / max
	switch {
	case n < rampSteps[0]:
		return rampLow, false
	case n < rampSteps[1]:
		return rampLow, true
	case n < rampSteps[2]:
		return dither2(rampLow, rampMid, x, y), true
	case n < rampSteps[3]:
		return rampMid, true
	case n < rampSteps[4]:
		return dither2(rampMid, rampHigh, x, y), true
	}
	return rampHigh, true
}

func dither2(a, b color.RGBA, x, y int) color.RGBA {
	if (x+y)%2 == 0 {
		return a
	}
	return b
}

// rampColor 保留給還沒改成「值太低就畫地形」的呼叫端。
func rampColor(v, max int) color.RGBA {
	c, _ := rampColorAt(v, max, 0, 0)
	return c
}

// winRect 是每個視窗的預設位置與大小，單位是**原版像素**。
//
// 統計圖、預算、評估三個是量原版截圖來的（docs/spec/ui-layout.md）：
// 統計圖 240,103 304×125；預算 171,27 285×309；評估 39,70 513×196。
// 其餘視窗是 remake 自己加的（系統選單的副選單、關於、存檔輸入），
// 原版沒有對應物，位置自己定。
var winRect = map[window]struct{ x, y, w, h int }{
	winGraphs:   {240, 103, 304, 125},
	winBudget:   {171, 27, 285, 309},
	winEval:     {39, 70, 513, 210},
	// 寬度是照「圖層清單 ＋ 縮圖」量出來的：縮圖的倍率被高度夾在 7，
	// 120 格 × 7 ＝ 840 螢幕像素，加上清單那一欄剛好 380 原版像素。
	// 先前寫 520，右邊會空掉三分之一。
	winMaps:     {60, 40, 380, 270},
	winDisaster: {200, 60, 240, 160},
	winSystem:   {90, 20, 240, 210},
	winScenario: {150, 40, 240, 180},
	winStyle:    {150, 40, 240, 180},
	winSpeed:    {180, 30, 240, 140},
	winPower:    {70, 180, 220, 110},
	// 「關於」是 remake 自己的頁（原版沒有），所以尺寸不是量原版來的：
	// 它要放得下十八行 15 原版像素的字 ＋ 條款要求的 Required Notice。
	winAbout:  {8, 20, 624, 326},
	winSaveAs: {120, 90, 400, 150},
	winLoad:   {120, 40, 400, 240},
	// remake 自己加的兩個
	winLangSel: {170, 60, 300, 140},
	winMusic:   {120, 40, 400, 240},
}

// winFrame 回傳目前視窗的外框（螢幕像素）。玩家搬過的話用搬過的位置。
func (g *Game) winFrame() (x, y, w, h int) {
	r, ok := winRect[g.win]
	if !ok {
		r = struct{ x, y, w, h int }{60, 40, 520, 270}
	}
	px, py := r.x, r.y
	if p, ok := g.winPos[g.win]; ok {
		px, py = p[0], p[1]
	}
	return px * UIScale, py * UIScale, r.w * UIScale, r.h * UIScale
}

// titleBarHit 判斷畫面座標在不在目前視窗的標題列上。
func (g *Game) titleBarHit(mx, my int) bool {
	if g.win == winNone {
		return false
	}
	x, y, w, _ := g.winFrame()
	return mx >= x && mx < x+w && my >= y && my < y+13*UIScale
}

// closeBoxHit 判斷畫面座標在不在關閉鈕上。原版的關閉鈕在標題列左端。
func (g *Game) closeBoxHit(mx, my int) bool {
	if g.win == winNone {
		return false
	}
	x, y, _, _ := g.winFrame()
	return mx >= x+2*UIScale && mx < x+10*UIScale &&
		my >= y+2*UIScale && my < y+10*UIScale
}

// handleWindowMouse 處理視窗的關閉鈕與標題列拖曳。
//
// 原版的視窗可以搬（選單裡的「視窗位置 Ctrl-P」），這裡做成直接拖標題列——
// 原版是先選選單再用方向鍵搬，兩種都到得了同一個結果，而拖曳是玩家會先試的。
func (g *Game) handleWindowMouse(mx, my int) bool {
	if g.dragWin != winNone {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.dragWin = winNone
			return true
		}
		if g.winPos == nil {
			g.winPos = map[window][2]int{}
		}
		g.winPos[g.dragWin] = [2]int{
			clampInt((mx-g.dragDX)/UIScale, 0, OrigW-40),
			clampInt((my-g.dragDY)/UIScale, menuBarH, OrigH-20),
		}
		return true
	}
	if g.win == winNone || !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	if g.closeBoxHit(mx, my) {
		g.win = winNone
		return true
	}
	if g.titleBarHit(mx, my) {
		x, y, _, _ := g.winFrame()
		g.dragWin, g.dragDX, g.dragDY = g.win, mx-x, my-y
		return true
	}
	return false
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// drawWindow 畫目前開著的視窗。
//
// 原版是**可重疊的浮動視窗**，各有標題列與左上角的關閉鈕，不是全畫面
// 覆蓋層。位置與大小照原版量出來的預設值。
func (g *Game) drawWindow(dst *ebiten.Image) {
	if g.win == winNone {
		return
	}
	x, y, w, h := g.winFrame()
	// 外框 ＋ 標題列 ＋ 客戶區，配色照原版：外框亮青、內容白底。
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h),
		colMenuBar, false)
	vector.DrawFilledRect(dst, float32(x+2*UIScale), float32(y+13*UIScale),
		float32(w-4*UIScale), float32(h-15*UIScale),
		color.RGBA{0xff, 0xff, 0xff, 0xff}, false)

	title := ""
	switch g.win {
	case winMaps:
		title = g.txt.S(i18n.SecWinMenu, 0)
	case winGraphs:
		// 原版的標題就是目前的區間（`Last 10 years`），不是視窗名。
		if g.graphYears == 10 {
			title = g.txt.S(i18n.SecGraph, 6)
		} else {
			title = g.txt.S(i18n.SecGraph, 7)
		}
	case winBudget:
		// ⚠ 帶年份的 `%d Fiscal Budget`（執行檔 0x027d3a）是**視窗裡面**
		// 那一行標題，不是視窗的標題列——原版這兩個視窗根本沒有標題列。
		// 兩邊都印會變成同一行字出現兩次，見 drawBudgetWindow。
		title = g.txt.S(i18n.SecWinMenu, 2)
	case winEval:
		// 評估視窗沒有視窗內的標題，所以帶年份的
		// `%d City Evaluation`（執行檔 0x0265bb）畫在標題列。
		//
		// ⚠ 中文語序與英文相反，年份在前、名稱在後——`CLAUDE.md` §3.3
		// 說的「模板要能重排參數順序」就是這一類。
		title = fmt.Sprintf(g.txt.UI("eval_title"), 1900+g.world.CityTime/48)
	case winDisaster:
		title = g.txt.S(i18n.SecMenu, 2)
	case winSystem:
		title = g.txt.S(i18n.SecMenu, 0)
	case winScenario:
		title = g.txt.S(i18n.SecSysMenu, 5)
	case winStyle:
		title = g.txt.S(i18n.SecSysMenu, 3)
	case winAbout:
		title = g.txt.S(i18n.SecSysMenu, 0)
	case winSaveAs:
		title = g.txt.S(i18n.SecSysMenu, 9)
	case winLoad:
		title = g.txt.S(i18n.SecSysMenu, 8)
	case winLangSel:
		title = g.txt.UI("lang_title")
	case winMusic:
		title = g.txt.UI("music_title")
	case winSpeed:
		title = g.txt.S(i18n.SecOptMenu, 4)
	case winPower:
		// 原版沒有這個視窗（見 classic.go 的說明），所以標題是 remake 自己的。
		title = "發電廠"
	}
	t := trimMenu(title)
	g.font.Draw(dst, t, x+w/2-g.font.Measure(t)/2, y+UIScale, colMenuInk)
	// 左上角的關閉鈕。原版是一個小方塊，點下去關掉這個視窗。
	vector.StrokeRect(dst, float32(x+2*UIScale), float32(y+2*UIScale),
		float32(8*UIScale), float32(8*UIScale), float32(UIScale), colMenuInk, false)

	inner := image.Rect(x+6*UIScale, y+16*UIScale, x+w-6*UIScale, y+h-4*UIScale)
	switch g.win {
	case winMaps:
		g.drawMapWindow(dst, inner.Min.X, inner.Min.Y, inner.Dx(), inner.Dy())
	case winGraphs:
		g.drawGraphWindow(dst, inner.Min.X, inner.Min.Y, inner.Dx(), inner.Dy())
	case winBudget:
		g.drawBudgetWindow(dst, inner.Min.X, inner.Min.Y, inner.Dx(), inner.Dy())
	case winEval:
		g.drawEvalWindow(dst, inner.Min.X, inner.Min.Y, inner.Dx(), inner.Dy())
	case winDisaster:
		g.drawDisasterWindow(dst, inner.Min.X, inner.Min.Y, inner.Dx(), inner.Dy())
	case winSpeed, winPower:
		g.drawSysMenu(dst, inner.Min.X, inner.Min.Y)
	case winAbout:
		g.drawAboutWindow(dst, inner.Min.X, inner.Min.Y, inner.Dx(), inner.Dy())
	case winSaveAs:
		g.drawSaveAsWindow(dst, inner.Min.X, inner.Min.Y, inner.Dx(), inner.Dy())
	case winSystem, winScenario, winStyle, winLoad, winLangSel, winMusic:
		g.drawSysMenu(dst, inner.Min.X, inner.Min.Y)
	}
}

// drawSysMenu 畫系統選單與它的兩個副選單。三個共用同一套排版與游標。
func (g *Game) drawSysMenu(dst *ebiten.Image, x, y int) {
	// ⚠ 提示要短。中文一個字佔兩格（32 原版像素），這幾個副選單最窄的
	// 只有 220 原版像素——把「上下鍵選擇」也寫進去就會**衝出視窗右緣**。
	g.font.Draw(dst, g.txt.UI("menu_hint_ok"), x, y, colDim)
	for i := 0; i < g.sysMenuLen(); i++ {
		c, mark := colDim, "  "
		if i == g.sysRow {
			c, mark = colOn, "> "
		}
		g.font.Draw(dst, mark+g.sysMenuLabel(i), x+8, y+g.font.Line()*(i+1), c)
	}
}

// drawDisasterWindow 畫災難選單。六個項目 ＋ 一列取消，名稱取自訊息檔。
func (g *Game) drawDisasterWindow(dst *ebiten.Image, x, y, w, h int) {
	g.font.Draw(dst, g.txt.UI("menu_hint_fire"), x, y, colDim)
	for i := range disasterItems {
		c := colDim
		mark := "  "
		if i == g.disasterRow {
			c, mark = colOn, "> "
		}
		label := fmt.Sprintf("%s%d. %s", mark, i+1,
			trimMenu(g.txt.S(i18n.SecDisaster, i)))
		g.font.Draw(dst, label, x+8, y+g.font.Line()*(i+1), c)
	}
	// 訊息檔第 20 段第 7 筆就是「取消」。
	g.font.Draw(dst, trimMenu(g.txt.S(i18n.SecDisaster, 7)),
		x+8, y+g.font.Line()*(len(disasterItems)+2), colDim)
}

// drawMapWindow 畫全市小地圖與十個圖層。
//
// ⚠ 圖層的解析度不一樣：都市型態與運輸是逐格（120×100），密度類是半解析
// （60×50），警力與消防是八分之一（15×13）。畫的時候要按各自的解析度取值，
// 不能一律用格子座標——那會讓密度圖只顯示左上四分之一，而且**看起來像是
// 城市只發展了一角**。
func (g *Game) drawMapWindow(dst *ebiten.Image, x, y, w, h int) {
	// 左側：圖層清單
	const listW = 260
	line := g.font.Line()
	for i := 0; i < int(layerCount); i++ {
		c := colText
		if mapLayer(i) == g.layer {
			c = colOn
		}
		g.font.Draw(dst, trimMenu(g.txt.S(i18n.SecMapTitle, i)), x, y+i*line, c)
	}
	g.font.Draw(dst, g.txt.UI("map_hint"), x, y+int(layerCount)*line, colDim)

	// 右側：地圖本體，等比放大到剩下的空間
	mx, my := x+listW, y
	mw := w - listW
	scale := mw / sim.WorldX
	if s2 := h / sim.WorldY; s2 < scale {
		scale = s2
	}
	if scale < 1 {
		scale = 1
	}
	g.drawMinimap(dst, mx, my, scale)
	// 目前視野的框
	vector.StrokeRect(dst,
		float32(mx+g.camX*scale), float32(my+g.camY*scale),
		float32(g.tilesAcross()*scale), float32(g.tilesDown()*scale),
		2, colOn, false)
}

func (g *Game) layerColor(x, y int) color.RGBA {
	w := g.world
	t := w.TileNum(x, y)
	switch g.layer {
	case layerCityForm:
		return cityFormColor(t)
	case layerPower:
		if t >= sim.RESBASE && w.Map[x][y]&sim.PWRBIT != 0 {
			return color.RGBA{0xf0, 0xd0, 0x40, 0xff}
		}
		if t >= sim.RESBASE {
			return color.RGBA{0x60, 0x30, 0x30, 0xff}
		}
		if w.Map[x][y]&sim.CONDBIT != 0 {
			return color.RGBA{0x90, 0x80, 0x30, 0xff}
		}
		return color.RGBA{0x20, 0x24, 0x2c, 0xff}
	case layerTransport:
		if t >= sim.ROADBASE && t < sim.POWERBASE {
			return color.RGBA{0xd0, 0xd0, 0xd0, 0xff}
		}
		if t >= sim.RAILBASE && t < sim.RESBASE {
			return color.RGBA{0x90, 0x90, 0x60, 0xff}
		}
		return color.RGBA{0x20, 0x24, 0x2c, 0xff}
	case layerPopDensity:
		return g.overlay(int(w.PopDensity[x>>1][y>>1]), 255, x, y, t)
	case layerTraffic:
		return g.overlay(int(w.TrfDensity[x>>1][y>>1]), 255, x, y, t)
	case layerPollution:
		return g.overlay(int(w.PollutionMem[x>>1][y>>1]), 255, x, y, t)
	case layerCrime:
		return g.overlay(int(w.CrimeMem[x>>1][y>>1]), 255, x, y, t)
	case layerLandValue:
		return g.overlay(int(w.LandValueMem[x>>1][y>>1]), 255, x, y, t)
	case layerPolice:
		return g.overlay(int(w.PoliceMapEffect[x>>3][y>>3]), 1000, x, y, t)
	case layerFire:
		return g.overlay(int(w.FireStMap[x>>3][y>>3]), 1000, x, y, t)
	case layerGrowth:
		// 人口成長有正負，原版另給一張 `Pos`／`Neg` 的色階（庫 7）。
		// Micropolis 的門檻是 ±20／±100（`g_map.c:140`），已確認。
		v := int(w.RateOGMem[x>>3][y>>3])
		switch {
		case v > 100:
			return rampHigh
		case v > 20:
			return rampMid
		case v < -100:
			return color.RGBA{0x55, 0x55, 0xff, 0xff}
		case v < -20:
			return color.RGBA{0xff, 0xff, 0x55, 0xff}
		}
		return cityFormColor(t)
	}
	return cityFormColor(t)
}

// overlayColor 回傳一格要不要蓋色階。回 false 代表這一格畫底下的縮圖。
//
// 「都市型態」「電力網路」「運輸網路」三個圖層畫的是地物不是數值，
// 一律走 layerColor，不在這裡處理。
func (g *Game) overlayColor(x, y int) (color.RGBA, bool) {
	w := g.world
	switch g.layer {
	case layerCityForm:
		return color.RGBA{}, false
	case layerPower, layerTransport:
		return g.layerColor(x, y), true
	case layerPopDensity:
		return rampColorAt(int(w.PopDensity[x>>1][y>>1]), 255, x, y)
	case layerTraffic:
		return rampColorAt(int(w.TrfDensity[x>>1][y>>1]), 255, x, y)
	case layerPollution:
		return rampColorAt(int(w.PollutionMem[x>>1][y>>1]), 255, x, y)
	case layerCrime:
		return rampColorAt(int(w.CrimeMem[x>>1][y>>1]), 255, x, y)
	case layerLandValue:
		return rampColorAt(int(w.LandValueMem[x>>1][y>>1]), 255, x, y)
	case layerPolice:
		return rampColorAt(int(w.PoliceMapEffect[x>>3][y>>3]), 1000, x, y)
	case layerFire:
		return rampColorAt(int(w.FireStMap[x>>3][y>>3]), 1000, x, y)
	case layerGrowth:
		v := int(w.RateOGMem[x>>3][y>>3])
		switch {
		case v > 100:
			return rampHigh, true
		case v > 20:
			return rampMid, true
		case v < -100:
			return color.RGBA{0x55, 0x55, 0xff, 0xff}, true
		case v < -20:
			return color.RGBA{0xff, 0xff, 0x55, 0xff}, true
		}
		return color.RGBA{}, false
	}
	return color.RGBA{}, false
}

// overlay 是數值圖層的一格：值夠高就畫色階，不夠就畫地形。
// **「值太低畫地形」是原版行為**（Micropolis `maybeDrawRect`：
// `VAL_NONE` 直接 return，底下那張 `drawAll` 就露出來），
// 先前 remake 一律畫色階，整張圖沒有地形。
func (g *Game) overlay(v, max, x, y, t int) color.RGBA {
	if c, ok := rampColorAt(v, max, x, y); ok {
		return c
	}
	return cityFormColor(t)
}

// cityFormColor 是都市型態圖：水、地、三種分區、道路各一色。
func cityFormColor(t int) color.RGBA {
	switch {
	case t == 0:
		return color.RGBA{0x3a, 0x2c, 0x20, 0xff}
	case t < sim.TREEBASE:
		return color.RGBA{0x20, 0x40, 0x90, 0xff}
	case t <= sim.WOODS5:
		return color.RGBA{0x28, 0x60, 0x30, 0xff}
	case t >= sim.ROADBASE && t < sim.POWERBASE:
		return color.RGBA{0x80, 0x80, 0x80, 0xff}
	case t >= sim.RESBASE && t < sim.COMBASE:
		return color.RGBA{0x50, 0xc0, 0x60, 0xff}
	case t >= sim.COMBASE && t < sim.INDBASE:
		return color.RGBA{0x60, 0x90, 0xe0, 0xff}
	case t >= sim.INDBASE && t <= sim.LASTZONE:
		return color.RGBA{0xe0, 0xc0, 0x50, 0xff}
	}
	return color.RGBA{0x40, 0x44, 0x50, 0xff}
}

// drawGraphWindow 畫六條歷史曲線。
//
// ⚠ 歷史陣列是**環狀**的嗎？不是——原版是把整個陣列往後推一格再寫入
// 索引 0（`s_sim.c` 的 `GraphDoer`），所以索引 0 是最新的、索引 119 最舊。
// 當成環狀去畫會得到一條時間軸亂跳的曲線，而且**看起來只是「資料有雜訊」**。
// 版面照原版排（workplace/dosbox/uw-10-graphs.png）：左邊一塊 2×4 的圖示
// 按鈕（`.PGF` 庫 4），右邊是曲線圖，底下標年份。
//
// 圖示的順序**就是訊息檔第 7 段的順序**：人口、犯罪、商業、現金流量、
// 工業、污染、近十年、近一百二十年——照著讀就對，不必另外對照。
func (g *Game) drawGraphWindow(dst *ebiten.Image, x, y, w, h int) {
	series := []struct {
		msg  int
		data *[240]int16
		c    color.RGBA
	}{
		{0, &g.world.ResHis, colDemR},
		{1, &g.world.CrimeHis, color.RGBA{0xd0, 0x50, 0x50, 0xff}},
		{2, &g.world.ComHis, colDemC},
		{3, &g.world.MoneyHis, color.RGBA{0xb0, 0x90, 0x30, 0xff}},
		{4, &g.world.IndHis, colDemI},
		{5, &g.world.PollutionHis, color.RGBA{0x50, 0xa0, 0x30, 0xff}},
	}

	// 左側圖示盤。庫 4 是一整張 51×102，格線量自 workplace/gfx/bank04-00.png：
	// 兩欄間距 25、四列間距 25、格子 24×23。
	blit(dst, g.tiles.UIImage(BankGraphBtns, 0), x/UIScale, y/UIScale)
	for i := 0; i < 8; i++ {
		on := false
		if i < 6 {
			on = g.graphOn[i]
		} else {
			on = (i == 6) == (g.graphYears == 10)
		}
		if !on {
			continue
		}
		cx, cy := graphCell(x/UIScale, y/UIScale, i)
		vector.StrokeRect(dst, s(cx), s(cy), s(graphCellW), s(graphCellH),
			float32(UIScale), color.RGBA{0xff, 0xff, 0x55, 0xff}, false)
	}

	// 右側曲線圖。
	gx := x + (graphPanelW+6)*UIScale
	gw, gh := x+w-gx, h-16*UIScale
	vector.DrawFilledRect(dst, float32(gx), float32(y), float32(gw), float32(gh),
		colMenuBar, false)
	vector.StrokeRect(dst, float32(gx), float32(y), float32(gw), float32(gh),
		float32(UIScale), colLine, false)

	// 格線與刻度。原版量出來的（`workplace/dosbox/uw-10-graphs.png`，
	// 繪圖區寬 245 原版像素）：
	//
	//	年格線：每 24 原版像素一條，滿高，藍色
	//	刻度  ：每  6 原版像素一格（＝每年四格），高 5，貼著底框上緣
	//
	// 少了它們的話曲線是浮在一片純色上的，讀不出「這個轉折是哪一年」——
	// 原版的十年圖每一年都有一條線，那是這個視窗最主要的可讀性來源。
	for gxp := gridPitch; gxp*UIScale < gw; gxp += gridPitch {
		lx := float32(gx + gxp*UIScale)
		vector.StrokeLine(dst, lx, float32(y), lx, float32(y+gh),
			float32(UIScale), colLine, false)
	}
	for txp := 0; txp*UIScale < gw; txp += tickPitch {
		tx := float32(gx + txp*UIScale)
		vector.StrokeLine(dst, tx, float32(y+gh-tickH*UIScale), tx, float32(y+gh),
			float32(UIScale), colLine, false)
	}

	// n 是要畫幾格。近十年是每月一格 120 格，近一百二十年是每年一格 120 格
	// ——**兩種都是 120 格**，差在資料取自哪一半（原版的歷史陣列前 120 筆
	// 是月、後 120 筆是年）。
	n, base := 120, 0
	if g.graphYears != 10 {
		base = 120
	}
	for _, sr := range series {
		if !g.graphOn[sr.msg] {
			continue
		}
		for i := 0; i < n-1; i++ {
			// 索引 0 是最新，畫的時候要反過來讓時間由左往右。
			x0 := float32(gx+gw) - float32(i)*float32(gw)/float32(n)
			x1 := float32(gx+gw) - float32(i+1)*float32(gw)/float32(n)
			y0 := float32(y+gh) - float32(clamp(int(sr.data[base+i]), 0, 255))*float32(gh)/255
			y1 := float32(y+gh) - float32(clamp(int(sr.data[base+i+1]), 0, 255))*float32(gh)/255
			vector.StrokeLine(dst, x0, y0, x1, y1, float32(UIScale), sr.c, false)
		}
	}
	// 底下的年份，原版是左中右三個。
	year := 1900 + g.world.CityTime/48
	span := g.graphYears
	labels := [3]string{
		fmt.Sprintf("%d", year-span),
		fmt.Sprintf("%d", year-span/2),
		fmt.Sprintf("%d", year),
	}
	ly := y + gh + 2*UIScale
	g.font.Draw(dst, labels[0], gx+2*UIScale, ly, colText)
	g.font.Draw(dst, labels[1], gx+gw/2-g.font.Measure(labels[1])/2, ly, colText)
	g.font.Draw(dst, labels[2], gx+gw-2*UIScale-g.font.Measure(labels[2]), ly, colText)
}

// 統計圖繪圖區的格線與刻度間距（原版像素，量自 uw-10-graphs.png）。
//
// ⚠ 這兩個數字**不隨十年／一百二十年改變**：原版兩種模式都畫 120 格資料，
// 差在資料取自歷史陣列的哪一半，格線是固定的版面不是資料的函式。
const (
	gridPitch = 24 // 年格線間距
	tickPitch = 6  // 刻度間距（每年四格）
	tickH     = 5  // 刻度高
)

// 統計圖圖示盤的格線（原版像素）。
const (
	graphCellW, graphCellH = 24, 23
	graphPitch             = 25
	graphPanelW            = 51
)

// graphCell 回傳第 i 個圖示格的左上角（原版座標）。
func graphCell(px, py, i int) (int, int) {
	return px + (i%2)*graphPitch, py + 2 + (i/2)*graphPitch
}

// graphHit 把畫面座標換成圖示格號；不在盤上回 −1。
func (g *Game) graphHit(mx, my int) int {
	x0, y0, _, _ := g.winFrame()
	px := x0/UIScale + 6
	py := y0/UIScale + 16
	x, y := mx/UIScale, my/UIScale
	for i := 0; i < 8; i++ {
		cx, cy := graphCell(px, py, i)
		if x >= cx && x < cx+graphCellW && y >= cy && y < cy+graphCellH {
			return i
		}
	}
	return -1
}

// drawBudgetWindow 畫預算視窗（說明書 p.43）。
// 版面照原版排（workplace/dosbox/uw-11-budget.png）：標題、稅率、稅收，
// 中間一個藍底的表格框（三欄表頭 ＋ 三列），底下現金流量與資金，最後一個按鈕。
// 欄位標題與原版一樣是兩行的（`Amount／Requested`）。
func (g *Game) drawBudgetWindow(dst *ebiten.Image, x, y, w, h int) {
	w0 := g.world
	rows := []struct {
		label string
		req   int
		pct   float64
	}{
		{trimMenu(g.txt.S(i18n.SecBudget, 0)), w0.RoadFund, w0.RoadPercent},
		{trimMenu(g.txt.S(i18n.SecBudget, 1)), w0.PoliceFund, w0.PolicePercent},
		{trimMenu(g.txt.S(i18n.SecBudget, 2)), w0.FireFund, w0.FirePercent},
	}
	// 一列的高 ＝ 原版量到的 15 原版像素（字格 14 加一列的行距）。
	line := g.font.Line()

	// 標題底下一條雙線，原版就有。
	title := fmt.Sprintf(g.txt.UI("budget_title"), 1900+w0.CityTime/48)
	g.font.Draw(dst, title, x+w/2-g.font.Measure(title)/2, y, colOn)
	vector.DrawFilledRect(dst, float32(x+w/8), float32(y+line),
		float32(w*3/4), float32(UIScale), colLine, false)

	// 稅率與稅收，兩行置中偏左。
	tax := fmt.Sprintf("%s　＋／－　%d%%", trimMenu(g.txt.S(i18n.SecBudget, 3)), w0.CityTax)
	g.font.Draw(dst, tax, x+w/2-g.font.Measure(tax)/2, y+line*2, colText)
	// ⚠ 「稅收」不在訊息檔裡（第 3 段只有交通／警局／消防／稅率），
	// 用譯名表的說法（說明書 p.43）。
	rev := fmt.Sprintf(g.txt.UI("tax_rev"), comma(w0.TaxFund))
	g.font.Draw(dst, rev, x+w/2-g.font.Measure(rev)/2, y+line*3, colText)

	// 表格框：原版是藍底、白框。
	tx, ty := x+2*UIScale, y+line*4
	tw, th := w-4*UIScale, line*6
	vector.DrawFilledRect(dst, float32(tx), float32(ty), float32(tw), float32(th),
		color.RGBA{0x55, 0x55, 0xff, 0xff}, false)
	vector.StrokeRect(dst, float32(tx), float32(ty), float32(tw), float32(th),
		float32(UIScale), colMenuBar, false)

	// 四欄的位置是量原版的（`workplace/dosbox/uw-11-budget.png`，內容座標）：
	// 項目 192、維護需求 240、實際撥給 304、撥款比例 368——視窗左緣在 171，
	// 客戶區從 177 起算，所以相對位移是 15／63／127／191。
	//
	// ⚠ 不要改回百分比切欄。中文一格 16 原版像素，四個字就是 64——
	// 用比例算出來的欄距在窄視窗裡會**互相疊在一起**，而且疊得很像
	// 「字型太大」，不像「欄位算錯」。
	col := [4]int{
		x + 15*UIScale, x + 63*UIScale, x + 127*UIScale, x + 191*UIScale,
	}
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	// 表頭兩行，原版也是兩行（`Amount／Requested`）。
	//
	// ⚠ 這四組字被 `translations/glossary.md` §八的「畫面上實際用的字」欄
	// 釘著（`TestGlossaryScreenTermsAppearInUI`）。說明書 p.43 給的是
	// 「維護需求額」「實際撥給金額」「編列（支付）百分比」——那是原文對照的
	// **說明**不是螢幕標籤，而這裡一欄只放得下四個字。要改先改譯名表。
	for i, h := range [4][2]string{
		{"", "項目"}, {"維護", "需求"}, {"實際", "撥給"}, {"編列", "比例"},
	} {
		g.font.Draw(dst, h[0], col[i], ty+2*UIScale, white)
		g.font.Draw(dst, h[1], col[i], ty+2*UIScale+line, white)
	}
	for i, r := range rows {
		yy := ty + 2*UIScale + (i+2)*line
		c := white
		if i == g.budgetRow {
			c = color.RGBA{0xff, 0xff, 0x55, 0xff}
		}
		g.font.Draw(dst, r.label, col[0], yy, c)
		g.font.Draw(dst, "$"+comma(r.req), col[1], yy, white)
		g.font.Draw(dst, "$"+comma(int(float64(r.req)*r.pct)), col[2], yy, white)
		g.font.Draw(dst, fmt.Sprintf("◀%d%%▶", int(r.pct*100+0.5)), col[3], yy, c)
	}

	// 底下三行摘要，原版是右對齊的數字欄。
	by := ty + th + line/2
	for i, s := range [][2]string{
		{"現金流量", "$" + comma(w0.CashFlow)},
		{"上年度結存", "$" + comma(w0.TotalFunds-w0.CashFlow)},
		{"目前資金", "$" + comma(w0.TotalFunds)},
	} {
		g.font.Draw(dst, s[0], x+w/4, by+i*line, colText)
		g.font.Draw(dst, s[1], x+w*3/4, by+i*line, colText)
	}

	// 原版底下有一個「Go with these figures」按鈕。這裡的預算是自動的，
	// 按鈕做成關閉視窗——按下去就是「照這些數字跑」。
	btn := "就照這些數字"
	bw := g.font.Measure(btn) + 12*UIScale
	bx := x + w/2 - bw/2
	byy := by + 3*line + line/2
	vector.StrokeRect(dst, float32(bx), float32(byy-2*UIScale),
		float32(bw), float32(line), float32(UIScale), colLine, false)
	g.font.Draw(dst, btn, bx+6*UIScale, byy, colOn)
	g.font.Draw(dst, g.txt.UI("budget_hint"), x+4*UIScale, byy+line*3/2, colDim)
}

// drawEvalWindow 畫評估視窗（說明書 p.54）。
// 版面照原版排（workplace/dosbox/ue-20-eval.png）：左半「公眾意見」兩個
// 白框，右半「統計數據」一個白框。原版的框是實線白底，這裡也一樣。
func (g *Game) drawEvalWindow(dst *ebiten.Image, x, y, w, h int) {
	w0 := g.world
	line := g.font.Line()
	half := w/2 - 4*UIScale
	// 左右兩欄的標題，原版是反白的小標。
	g.font.Draw(dst, g.txt.UI("eval_public"), x+half/2-g.font.Measure("公眾意見")/2, y, colOn)
	g.font.Draw(dst, g.txt.UI("eval_stats"), x+w/2+half/2-g.font.Measure("統計數據")/2, y, colOn)
	// 三個白框。
	// ⚠ 「嚴重問題」框要 **6 行**：一行標題 ＋ 四個名次，最後一名的字底
	// 落在 `y+line*10+4`。先前給 5 行，第四名的下緣被框切掉——資訊還在，
	// 但玩家讀不到，而且截圖看起來只是「排版有點擠」。
	boxes := [][4]int{
		{x, y + line, half, line * 3},       // 市長評價
		{x, y + line*5, half, line * 6},     // 嚴重問題
		{x + w/2, y + line, half, line * 9}, // 統計數據
	}
	for _, b := range boxes {
		vector.DrawFilledRect(dst, float32(b[0]), float32(b[1]),
			float32(b[2]), float32(b[3]), colPanel, false)
		vector.StrokeRect(dst, float32(b[0]), float32(b[1]),
			float32(b[2]), float32(b[3]), float32(UIScale), colLine, false)
	}
	_ = line
	// 兩個數字都直接讀，不要拿 100 減。空城的評估是 EvalInit 清成
	// 0／0（沒有人投票），用減法會變成「否 100%」——一座剛開的城市
	// 被說成全體反對，而原版是兩邊都 0（w_eval.c:101 的 goodyes／goodno）。
	g.font.Draw(dst, g.txt.UI("eval_mayor"), x+6*UIScale, y+line+4*UIScale, colText)
	g.font.Draw(dst, fmt.Sprintf(g.txt.UI("eval_yesno"), w0.Eval.CityYes, w0.Eval.CityNo),
		x+6*UIScale, y+line*2+4*UIScale, colText)

	g.font.Draw(dst, g.txt.UI("eval_problems"), x+6*UIScale, y+line*5+4*UIScale, colText)
	for i := 0; i < 4; i++ {
		p := w0.Eval.ProblemOrder[i]
		if p < 0 || p >= len(w0.Eval.ProblemVotes) {
			continue
		}
		// 沒有票的名次整列留白。ProblemOrder 在沒問題可排時填的是 7
		// （= ProbNone），而 7 是合法的索引，光看索引分辨不出來——
		// 原版也是拿票數當判準（w_eval.c:115 的三元運算）。
		if w0.Eval.ProblemVotes[p] == 0 {
			continue
		}
		g.font.Draw(dst, fmt.Sprintf("%d. %s　%d%%",
			i+1, trimMenu(g.txt.S(i18n.SecProblem, p)), w0.Eval.ProblemVotes[p]),
			x+6*UIScale, y+line*6+4*UIScale+i*line, colText)
	}

	// 七列，順序照原版。實拍：`workplace/dosbox/ue-20-eval.png`
	//
	//	Population        0
	//	Net Migration     0
	//	(last year)
	//	Assessed Value  $0
	//	Category:VILLAGE
	//	Game Level:Easy
	//	   Overall CityScore (0 - 1000)
	//	current score:annual change
	//
	// ⚠ **`(last year)` 是「Net Migration」標籤的第二行**，不是另一個數字
	// ——原版把它折成兩行是因為欄寬不夠。中文一行放得下，所以併回去。
	// ⚠ **`Game Level` 先前漏了**。說明書 p.54 的統計數據表列了它
	// （`docs/manual-cht/p23-58-operations.md`），實拍也有，而這裡沒畫。
	// 等級的譯名沿用開新城市對話框（說明書 p.11）。
	stats := [][2]string{
		{g.txt.UI("stat_pop"), fmt.Sprintf("%d", w0.Eval.CityPop)},
		{g.txt.UI("stat_migr"), fmt.Sprintf("%d", w0.Eval.DeltaCityPop)},
		{g.txt.UI("stat_assess"), fmt.Sprintf("$%d", w0.Eval.CityAssValue)},
		{g.txt.UI("stat_class"), g.txt.S(i18n.SecClass, clamp(w0.CityClass, 0, 5))},
		{g.txt.UI("stat_level"), g.levelName(w0.GameLevel)},
		{g.txt.UI("stat_score"), fmt.Sprintf("%d", w0.CityScore)},
		{g.txt.UI("stat_delta"), fmt.Sprintf("%+d", w0.Eval.DeltaCityScore)},
	}
	for i, s := range stats {
		yy := y + line + 4*UIScale + i*line
		g.font.Draw(dst, s[0], x+w/2+6*UIScale, yy, colDim)
		g.font.Draw(dst, s[1], x+w-10*UIScale-g.font.Measure(s[1]), yy, colText)
	}
}

// trimMenu 去掉原版選單項目的前導空白與快速鍵。
//
// 原版把「 地圖視窗      Ctrl-M」整串存成一筆，因為它要直接畫進固定寬度的
// 選單。當標題用的時候那些空白與快速鍵是雜訊。
func trimMenu(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			continue
		}
		s = s[i:]
		break
	}
	if i := indexOf(s, "Ctrl-"); i > 0 {
		s = s[:i]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// drawPicture 畫圖片訊息：整段文字擋在地圖上，按任意鍵或點一下關掉。
//
// 原版會配一張圖，本專案目前只放文字——圖形在 .PGF 的哪一個圖形庫還沒
// 確認（見 docs/formats/03-pgf-graphics.md §7 的未解項），與其猜一個
// 錯的貼上去，不如先只放文字。
func (g *Game) drawPicture(dst *ebiten.Image) {
	if g.picture == "" {
		return
	}
	lines := splitLines(g.picture)
	lh := g.font.Size() + 10
	w := 0
	for _, l := range lines {
		if m := g.font.Measure(l); m > w {
			w = m
		}
	}
	w += 80
	if w > viewW-80 {
		w = viewW - 80
	}
	h := len(lines)*lh + 96
	x := (viewW - w) / 2
	y := (viewH - h) / 2
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h),
		color.RGBA{0x14, 0x18, 0x22, 0xf8}, false)
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), 3, colOn, false)
	for i, l := range lines {
		g.font.Draw(dst, l, x+40, y+40+i*lh, colText)
	}
	hint := "按空白鍵或點一下繼續"
	g.font.Draw(dst, hint, x+(w-g.font.Measure(hint))/2, y+h-38, colDim)
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
