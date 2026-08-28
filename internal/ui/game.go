package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 內部畫布。
//
// 為什麼是 1280×960 而不是原版的 640×350：中文字要 24×24 才讀得清楚，
// 硬塞進原版的字位會糊成一團。正解是把畫布拉高、底圖用**整數倍**最近鄰
// 放大——這裡圖塊 16×16 放大兩倍成 32×32。非整數倍即使用最近鄰也會出現
// 寬窄不一的像素。詳見 rulebook/81。
const (
	CanvasW = 1280
	CanvasH = 960

	tileScale = 2  // 16×16 → 32×32
	panelW    = 256 // 右側工具列
	statusH   = 192 // 下方狀態列

	viewW = CanvasW - panelW // 1024
	viewH = CanvasH - statusH // 768
)

// 配色。刻意用低彩度的灰藍，讓原版的十六色底圖是畫面上最亮的東西。
var (
	colBG     = color.RGBA{0x1c, 0x20, 0x28, 0xff}
	colPanel  = color.RGBA{0x26, 0x2c, 0x36, 0xff}
	colLine   = color.RGBA{0x3a, 0x42, 0x50, 0xff}
	colText   = color.RGBA{0xdc, 0xe0, 0xe6, 0xff}
	colDim    = color.RGBA{0x8a, 0x93, 0xa0, 0xff}
	colOn     = color.RGBA{0xf0, 0xc0, 0x50, 0xff}
	colMoneyN = color.RGBA{0xe0, 0x70, 0x60, 0xff}
	colDemR   = color.RGBA{0x50, 0xc0, 0x60, 0xff}
	colDemC   = color.RGBA{0x60, 0x90, 0xe0, 0xff}
	colDemI   = color.RGBA{0xe0, 0xc0, 0x50, 0xff}
)

// toolButton 是工具列上的一個按鈕。譯名全部來自軟體世界說明書，
// 出處見 translations/glossary.md。
type toolButton struct {
	tool sim.Tool
	name string
	cost int
	key  ebiten.Key
}

var toolButtons = []toolButton{
	{sim.ToolBulldozer, "推土機", 1, ebiten.KeyD},
	{sim.ToolRoad, "道路", 10, ebiten.KeyR},
	{sim.ToolRail, "鐵軌", 20, ebiten.KeyT},
	{sim.ToolWire, "電力線", 5, ebiten.KeyW},
	{sim.ToolPark, "公園", 10, ebiten.KeyP},
	{sim.ToolResidential, "住宅區", 100, ebiten.Key1},
	{sim.ToolCommercial, "商業區", 100, ebiten.Key2},
	{sim.ToolIndustrial, "工業用地", 100, ebiten.Key3},
	{sim.ToolPolice, "警察局", 500, ebiten.Key4},
	{sim.ToolFireStation, "消防隊", 500, ebiten.Key5},
	{sim.ToolStadium, "體育館", 5000, ebiten.Key6},
	{sim.ToolCoalPower, "火力發電廠", 3000, ebiten.Key7},
	{sim.ToolNuclear, "核能發電廠", 5000, ebiten.Key8},
	{sim.ToolSeaport, "海港", 3000, ebiten.Key9},
	{sim.ToolAirport, "機場", 10000, ebiten.Key0},
	{sim.ToolQuery, "查詢", 0, ebiten.KeyQ},
}

// 速度的譯名同樣來自說明書 p.33。
var speedNames = [4]string{"暫停", "慢速", "普通", "最快"}

var monthNames = [12]string{
	"一月", "二月", "三月", "四月", "五月", "六月",
	"七月", "八月", "九月", "十月", "十一月", "十二月",
}

// Game 是 Ebiten 的遊戲物件。
type Game struct {
	world *sim.World
	tiles *TileSet
	font  *Font

	tool     sim.Tool
	camX     int // 視野左上角的格子座標
	camY     int
	dragging bool

	message  string
	msgTimer int
}

// NewGame 建一個新遊戲。
func NewGame(w *sim.World, ts *TileSet, f *Font) *Game {
	g := &Game{world: w, tiles: ts, font: f, tool: sim.ToolResidential}
	g.centerCamera()
	return g
}

func (g *Game) centerCamera() {
	g.camX = (sim.WorldX - viewW/(g.tiles.Size*tileScale)) / 2
	g.camY = (sim.WorldY - viewH/(g.tiles.Size*tileScale)) / 2
	g.clampCamera()
}

func (g *Game) tilesAcross() int { return viewW / (g.tiles.Size * tileScale) }
func (g *Game) tilesDown() int   { return viewH / (g.tiles.Size * tileScale) }

// LookAt 把鏡頭移到某一格附近。示範模式與「前往災區」都用它。
func (g *Game) LookAt(x, y int) {
	g.camX = x - g.tilesAcross()/2
	g.camY = y - g.tilesDown()/2
	g.clampCamera()
}

func (g *Game) clampCamera() {
	maxX, maxY := sim.WorldX-g.tilesAcross(), sim.WorldY-g.tilesDown()
	g.camX = clamp(g.camX, 0, maxX)
	g.camY = clamp(g.camY, 0, maxY)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Layout 固定回傳內部畫布尺寸；Ebiten 會自己縮放到視窗。
func (g *Game) Layout(int, int) (int, int) { return CanvasW, CanvasH }

// Update 每個 frame 跑一次：先收輸入，再推進模擬。
//
// ⚠ 順序不能反。玩家這一個 frame 蓋的東西要在同一個 frame 進模擬，
// 否則「按下去到看到反應」會多一格延遲，手感會鬆。
func (g *Game) Update() error {
	g.handleKeys()
	g.handleMouse()
	g.world.Frame()
	if g.msgTimer > 0 {
		g.msgTimer--
	}
	g.pumpSimMessage()
	return nil
}

// pumpSimMessage 把模擬層的訊息埠取出來顯示。
func (g *Game) pumpSimMessage() {
	n := g.world.MessagePort
	if n == 0 {
		return
	}
	g.world.MessagePort = 0
	if n < 0 {
		n = -n
	}
	g.setMessage(messageText(n))
}

func (g *Game) setMessage(s string) {
	if s == "" {
		return
	}
	g.message = s
	g.msgTimer = 60 * 8
}

func (g *Game) handleKeys() {
	for _, b := range toolButtons {
		if inpututil.IsKeyJustPressed(b.key) {
			g.tool = b.tool
			g.setMessage(fmt.Sprintf("選擇工具：%s", b.name))
		}
	}
	step := 1
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		step = 5
	}
	switch {
	case ebiten.IsKeyPressed(ebiten.KeyLeft):
		g.camX -= step
	case ebiten.IsKeyPressed(ebiten.KeyRight):
		g.camX += step
	}
	switch {
	case ebiten.IsKeyPressed(ebiten.KeyUp):
		g.camY -= step
	case ebiten.IsKeyPressed(ebiten.KeyDown):
		g.camY += step
	}
	g.clampCamera()

	for i, k := range []ebiten.Key{
		ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4,
	} {
		if inpututil.IsKeyJustPressed(k) {
			g.world.SimSpeed = i
			g.setMessage("模擬速度：" + speedNames[i])
		}
	}
}

func (g *Game) handleMouse() {
	mx, my := ebiten.CursorPosition()
	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	just := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	// 工具列
	if just && mx >= viewW && my < viewH {
		if i := (my - 8) / 40; i >= 0 && i < len(toolButtons) {
			g.tool = toolButtons[i].tool
			g.setMessage("選擇工具：" + toolButtons[i].name)
		}
		return
	}
	// 地圖
	if mx >= viewW || my >= viewH {
		g.dragging = false
		return
	}
	if !pressed {
		g.dragging = false
		return
	}
	// 道路、鐵軌、電力線可以拖曳；其餘只在按下的那一刻動作，
	// 免得手一抖就蓋出一排體育館。
	drag := g.tool == sim.ToolRoad || g.tool == sim.ToolRail ||
		g.tool == sim.ToolWire || g.tool == sim.ToolBulldozer
	if !just && !(drag && g.dragging) {
		return
	}
	g.dragging = true
	px := g.tiles.Size * tileScale
	tx, ty := g.camX+mx/px, g.camY+my/px
	g.applyTool(tx, ty)
}

func (g *Game) applyTool(tx, ty int) {
	if g.tool == sim.ToolQuery {
		g.setMessage(fmt.Sprintf("(%d,%d) 圖塊 %d", tx, ty, g.world.TileNum(tx, ty)))
		return
	}
	before := g.world.TotalFunds
	switch r := g.world.ApplyTool(g.tool, tx, ty); r {
	case sim.ToolOK:
		if spent := before - g.world.TotalFunds; spent > 0 {
			g.setMessage(fmt.Sprintf("花費 $%d", spent))
		}
	case sim.ToolNoMoney:
		g.setMessage("資金不足")
	case sim.ToolNeedsClear:
		g.setMessage("這裡不能蓋——地要先整平")
	case sim.ToolBlocked:
		g.setMessage("這裡蓋不了")
	}
}

// Draw 畫一個 frame。
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	g.drawMap(screen)
	g.drawPanel(screen)
	g.drawStatus(screen)
}

func (g *Game) drawMap(dst *ebiten.Image) {
	px := g.tiles.Size * tileScale
	view := dst.SubImage(rect(0, 0, viewW, viewH)).(*ebiten.Image)
	for y := 0; y < g.tilesDown(); y++ {
		for x := 0; x < g.tilesAcross(); x++ {
			n := g.world.TileNum(g.camX+x, g.camY+y)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(tileScale, tileScale)
			op.GeoM.Translate(float64(x*px), float64(y*px))
			// 最近鄰：像素畫放大要銳利，雙線性會糊掉。
			op.Filter = ebiten.FilterNearest
			view.DrawImage(g.tiles.TileImage(n), op)
		}
	}
}

func (g *Game) drawPanel(dst *ebiten.Image) {
	vector.DrawFilledRect(dst, viewW, 0, panelW, viewH, colPanel, false)
	vector.StrokeLine(dst, viewW, 0, viewW, viewH, 2, colLine, false)
	for i, b := range toolButtons {
		y := 8 + i*40
		c := colText
		if b.cost > g.world.TotalFunds {
			c = colDim
		}
		if b.tool == g.tool {
			vector.DrawFilledRect(dst, viewW+4, float32(y-4), panelW-8, 36,
				color.RGBA{0x3a, 0x46, 0x5c, 0xff}, false)
			c = colOn
		}
		g.font.Draw(dst, b.name, viewW+16, y, c)
		if b.cost > 0 {
			s := fmt.Sprintf("$%d", b.cost)
			g.font.Draw(dst, s, viewW+panelW-16-g.font.Measure(s), y, colDim)
		}
	}
}

func (g *Game) drawStatus(dst *ebiten.Image) {
	top := viewH
	vector.DrawFilledRect(dst, 0, float32(top), CanvasW, statusH, colPanel, false)
	vector.StrokeLine(dst, 0, float32(top), CanvasW, float32(top), 2, colLine, false)

	// 日期與資金
	year := 1900 + g.world.CityTime/48
	month := monthNames[(g.world.CityTime%48)/4]
	g.font.Draw(dst, fmt.Sprintf("%d 年 %s", year, month), 24, top+20, colText)

	fundColor := colText
	if g.world.TotalFunds < 0 {
		fundColor = colMoneyN
	}
	g.font.Draw(dst, fmt.Sprintf("資金 $%d", g.world.TotalFunds), 24, top+56, fundColor)
	g.font.Draw(dst, fmt.Sprintf("人口 %d", g.world.Eval.CityPop), 24, top+92, colText)
	g.font.Draw(dst, fmt.Sprintf("稅率 %d%%", g.world.CityTax), 24, top+128, colText)

	// 需求顯示表（說明書 p.47）：短柱指向上代表市民需要這一類。
	g.drawDemand(dst, 360, top+16)

	// 訊息欄
	if g.msgTimer > 0 {
		g.font.Draw(dst, g.message, 660, top+20, colOn)
	}
	g.font.Draw(dst, "速度："+speedNames[g.world.SimSpeed], 660, top+56, colDim)
	g.font.Draw(dst, "F1–F4 切速度　方向鍵移動　數字鍵選工具", 660, top+92, colDim)
	g.font.Draw(dst, "圖形："+g.tiles.Style, 660, top+128, colDim)
}

// drawDemand 畫 R／C／I 需求柱。原版用短柱的正負表示需要或過剩。
func (g *Game) drawDemand(dst *ebiten.Image, x, y int) {
	g.font.Draw(dst, "需求", x, y, colDim)
	bars := []struct {
		label string
		v     int
		c     color.RGBA
	}{
		{"住", g.world.RValve, colDemR},
		{"商", g.world.CValve, colDemC},
		{"工", g.world.IValve, colDemI},
	}
	const mid = 60 // 零線相對於 y 的位移
	for i, b := range bars {
		bx := float32(x + 40 + i*44)
		h := float32(b.v) / 2000 * 40
		if h > 40 {
			h = 40
		}
		if h < -40 {
			h = -40
		}
		if h >= 0 {
			vector.DrawFilledRect(dst, bx, float32(y+mid)-h, 24, h, b.c, false)
		} else {
			vector.DrawFilledRect(dst, bx, float32(y+mid), 24, -h, b.c, false)
		}
		vector.StrokeLine(dst, bx, float32(y+mid), bx+24, float32(y+mid), 1, colLine, false)
		g.font.Draw(dst, b.label, int(bx), y+mid+44, colDim)
	}
}
