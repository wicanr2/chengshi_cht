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
var densityRamp = []color.RGBA{
	{0x20, 0x28, 0x38, 0xff},
	{0x28, 0x50, 0x60, 0xff},
	{0x30, 0x80, 0x70, 0xff},
	{0x60, 0xa0, 0x50, 0xff},
	{0xc0, 0xb0, 0x40, 0xff},
	{0xd0, 0x70, 0x30, 0xff},
	{0xd0, 0x40, 0x40, 0xff},
}

func rampColor(v, max int) color.RGBA {
	if max <= 0 {
		return densityRamp[0]
	}
	i := v * (len(densityRamp) - 1) / max
	if i < 0 {
		i = 0
	}
	if i >= len(densityRamp) {
		i = len(densityRamp) - 1
	}
	return densityRamp[i]
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
	winEval:     {39, 70, 513, 196},
	winMaps:     {60, 40, 520, 270},
	winDisaster: {200, 60, 240, 160},
	winSystem:   {90, 20, 240, 210},
	winScenario: {150, 40, 240, 180},
	winStyle:    {150, 40, 240, 180},
	winSpeed:    {200, 30, 200, 140},
	winPower:    {70, 190, 170, 80},
	winAbout:    {40, 30, 560, 290},
	winSaveAs:   {120, 90, 400, 150},
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
		title = g.txt.S(i18n.SecWinMenu, 1)
	case winBudget:
		title = g.txt.S(i18n.SecWinMenu, 2)
	case winEval:
		title = g.txt.S(i18n.SecWinMenu, 4)
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
	case winSpeed:
		title = g.txt.S(i18n.SecOptMenu, 4)
	case winPower:
		title = g.txt.S(i18n.SecPowerSub, 0)
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
	case winSystem, winScenario, winStyle:
		g.drawSysMenu(dst, inner.Min.X, inner.Min.Y)
	}
}

// drawSysMenu 畫系統選單與它的兩個副選單。三個共用同一套排版與游標。
func (g *Game) drawSysMenu(dst *ebiten.Image, x, y int) {
	g.font.Draw(dst, "上下鍵選擇，Enter 確定，Esc 取消", x, y, colDim)
	for i := 0; i < g.sysMenuLen(); i++ {
		c, mark := colDim, "  "
		if i == g.sysRow {
			c, mark = colOn, "> "
		}
		g.font.Draw(dst, mark+g.sysMenuLabel(i), x+8, y+40+i*28, c)
	}
}

// drawDisasterWindow 畫災難選單。六個項目 ＋ 一列取消，名稱取自訊息檔。
func (g *Game) drawDisasterWindow(dst *ebiten.Image, x, y, w, h int) {
	g.font.Draw(dst, "上下鍵選擇，Enter 發動，Esc 取消", x, y, colDim)
	for i := range disasterItems {
		c := colDim
		mark := "  "
		if i == g.disasterRow {
			c, mark = colOn, "> "
		}
		label := fmt.Sprintf("%s%d. %s", mark, i+1,
			trimMenu(g.txt.S(i18n.SecDisaster, i)))
		g.font.Draw(dst, label, x+8, y+40+i*28, c)
	}
	// 訊息檔第 20 段第 7 筆就是「取消」。
	g.font.Draw(dst, trimMenu(g.txt.S(i18n.SecDisaster, 7)),
		x+8, y+40+len(disasterItems)*28+16, colDim)
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
	for i := 0; i < int(layerCount); i++ {
		c := colText
		if mapLayer(i) == g.layer {
			c = colOn
		}
		g.font.Draw(dst, trimMenu(g.txt.S(i18n.SecMapTitle, i)), x, y+i*32, c)
	}
	g.font.Draw(dst, "1–9 0 - 切圖層", x, y+int(layerCount)*32+12, colDim)

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
	for ty := 0; ty < sim.WorldY; ty++ {
		for tx := 0; tx < sim.WorldX; tx++ {
			c := g.layerColor(tx, ty)
			vector.DrawFilledRect(dst, float32(mx+tx*scale), float32(my+ty*scale),
				float32(scale), float32(scale), c, false)
		}
	}
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
		return rampColor(int(w.PopDensity[x>>1][y>>1]), 255)
	case layerTraffic:
		return rampColor(int(w.TrfDensity[x>>1][y>>1]), 255)
	case layerPollution:
		return rampColor(int(w.PollutionMem[x>>1][y>>1]), 255)
	case layerCrime:
		return rampColor(int(w.CrimeMem[x>>1][y>>1]), 255)
	case layerLandValue:
		return rampColor(int(w.LandValueMem[x>>1][y>>1]), 255)
	case layerPolice:
		return rampColor(int(w.PoliceMapEffect[x>>3][y>>3]), 1000)
	case layerFire:
		return rampColor(int(w.FireStMap[x>>3][y>>3]), 1000)
	case layerGrowth:
		v := int(w.RateOGMem[x>>3][y>>3])
		if v > 0 {
			return rampColor(v, 200)
		}
		return color.RGBA{uint8(clamp(-v, 0, 200) + 40), 0x20, 0x20, 0xff}
	}
	return densityRamp[0]
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
func (g *Game) drawGraphWindow(dst *ebiten.Image, x, y, w, h int) {
	series := []struct {
		label string
		data  *[240]int16
		c     color.RGBA
	}{
		{g.txt.S(i18n.SecGraph, 0), &g.world.ResHis, colDemR},
		{g.txt.S(i18n.SecGraph, 2), &g.world.ComHis, colDemC},
		{g.txt.S(i18n.SecGraph, 4), &g.world.IndHis, colDemI},
		{g.txt.S(i18n.SecGraph, 1), &g.world.CrimeHis, color.RGBA{0xd0, 0x50, 0x50, 0xff}},
		{g.txt.S(i18n.SecGraph, 5), &g.world.PollutionHis, color.RGBA{0x90, 0xd0, 0x50, 0xff}},
		{g.txt.S(i18n.SecGraph, 3), &g.world.MoneyHis, color.RGBA{0xe0, 0xd0, 0x90, 0xff}},
	}
	const n = 120 // 近十年（每格一個月）
	gw, gh := w-200, h-40
	vector.StrokeRect(dst, float32(x), float32(y), float32(gw), float32(gh), 1, colLine, false)
	for si, s := range series {
		g.font.Draw(dst, s.label, x+gw+20, y+si*32, s.c)
		for i := 0; i < n-1; i++ {
			// 索引 0 是最新，畫的時候要反過來讓時間由左往右。
			x0 := float32(x+gw) - float32(i)*float32(gw)/float32(n)
			x1 := float32(x+gw) - float32(i+1)*float32(gw)/float32(n)
			y0 := float32(y+gh) - float32(clamp(int(s.data[i]), 0, 255))*float32(gh)/255
			y1 := float32(y+gh) - float32(clamp(int(s.data[i+1]), 0, 255))*float32(gh)/255
			vector.StrokeLine(dst, x0, y0, x1, y1, 2, s.c, false)
		}
	}
	g.font.Draw(dst, g.txt.S(i18n.SecGraph, 6), x, y+gh+8, colDim)
}

// drawBudgetWindow 畫預算視窗（說明書 p.43）。
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
	g.font.Draw(dst, fmt.Sprintf("%s %d%%（＋／－ 調整）",
		trimMenu(g.txt.S(i18n.SecBudget, 3)), w0.CityTax), x, y, colText)
	// ⚠ 「稅收」不在訊息檔裡（第 3 段只有交通／警局／消防／稅率），
	// 用譯名表的說法（說明書 p.43）。
	g.font.Draw(dst, fmt.Sprintf("稅收 $%d", w0.TaxFund), x+400, y, colDim)

	g.font.Draw(dst, "項目", x, y+56, colDim)
	g.font.Draw(dst, "維護需求額", x+200, y+56, colDim)
	g.font.Draw(dst, "編列百分比", x+440, y+56, colDim)
	g.font.Draw(dst, "實際撥給", x+680, y+56, colDim)
	for i, r := range rows {
		yy := y + 96 + i*40
		c := colText
		if i == g.budgetRow {
			c = colOn
		}
		g.font.Draw(dst, r.label, x, yy, c)
		g.font.Draw(dst, fmt.Sprintf("$%d", r.req), x+200, yy, colText)
		g.font.Draw(dst, fmt.Sprintf("%d%%", int(r.pct*100+0.5)), x+440, yy, colText)
		g.font.Draw(dst, fmt.Sprintf("$%d", int(float64(r.req)*r.pct)), x+680, yy, colText)
	}
	g.font.Draw(dst, fmt.Sprintf("現金流量 $%d", w0.CashFlow), x, y+240, colText)
	g.font.Draw(dst, fmt.Sprintf("目前資金 $%d", w0.TotalFunds), x, y+280, colText)
	g.font.Draw(dst, "上下鍵選項目，左右鍵調整百分比", x, y+340, colDim)
}

// drawEvalWindow 畫評估視窗（說明書 p.54）。
func (g *Game) drawEvalWindow(dst *ebiten.Image, x, y, w, h int) {
	w0 := g.world
	g.font.Draw(dst, "公眾意見", x, y, colOn)
	// 兩個數字都直接讀，不要拿 100 減。空城的評估是 EvalInit 清成
	// 0／0（沒有人投票），用減法會變成「否 100%」——一座剛開的城市
	// 被說成全體反對，而原版是兩邊都 0（w_eval.c:101 的 goodyes／goodno）。
	g.font.Draw(dst, fmt.Sprintf("市長做得好嗎？　是 %d%%　否 %d%%",
		w0.Eval.CityYes, w0.Eval.CityNo), x, y+40, colText)

	g.font.Draw(dst, "嚴重問題", x, y+96, colOn)
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
			i+1, g.txt.S(i18n.SecProblem, p), w0.Eval.ProblemVotes[p]),
			x, y+136+i*36, colText)
	}

	g.font.Draw(dst, "統計數據", x+520, y+96, colOn)
	stats := [][2]string{
		{"人口總數", fmt.Sprintf("%d", w0.Eval.CityPop)},
		{"遷出入數", fmt.Sprintf("%d", w0.Eval.DeltaCityPop)},
		{"市有財產總數", fmt.Sprintf("$%d", w0.Eval.CityAssValue)},
		{"城市類別", g.txt.S(i18n.SecClass, clamp(w0.CityClass, 0, 5))},
		{"整體成績", fmt.Sprintf("%d", w0.CityScore)},
		{"年度成績", fmt.Sprintf("%+d", w0.Eval.DeltaCityScore)},
	}
	for i, s := range stats {
		g.font.Draw(dst, s[0], x+520, y+136+i*36, colDim)
		g.font.Draw(dst, s[1], x+760, y+136+i*36, colText)
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
