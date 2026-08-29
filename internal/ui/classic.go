package ui

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
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

	editX, editY = 5, 21    // 編輯視窗外框左上角
	editW, editH = 236, 304 // 外框大小（含框線）

	editTitleY = 24 // 標題列
	editTitleH = 14
	editInfoY  = 38 // 資金／訊息帶
	editInfoH  = 16
	editViewY  = 54 // 地圖區
	editViewH  = 257
	editToolY  = 311 // 目前工具帶
	editToolH  = 10

	editPalX, editPalY = 6, 53 // 工具盤圖（庫 2，57×182）
	editViewX          = 64    // 地圖區左緣
	editViewW          = 176   // 240 − 64
	editDemandX        = 8     // 需求指標（庫 3，46×39）
	editDemandY        = 236

	mapX, mapY = 241, 21 // City Form 視窗外框
	mapW, mapH = 398, 327
	mapIconX   = 244 // 圖層圖示（庫 5，26×226）
	mapIconY   = 33
	mapViewX   = 272 // 地圖本體
	mapViewY   = 33
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
	g.drawEditWindow(dst)
	g.drawCityFormWindow(dst)
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
		g.font.Draw(dst, label, x, 2*UIScale, c)
	}
}

// drawEditWindow 畫編輯視窗：標題列、資金／訊息帶、工具盤、地圖、工具帶。
func (g *Game) drawEditWindow(dst *ebiten.Image) {
	fill(dst, editX, editY, editW, editH, colEditFrm)
	fill(dst, editX+1, editTitleY, editW-2, editTitleH, colTitleBar)
	fill(dst, editX+1, editInfoY, editW-2, editInfoH, colInfoBand)
	fill(dst, editX+1, editViewY, editW-2, editViewH, colDesktop)

	// 資金與訊息：原版是同一條帶，資金在左、訊息接在後面。
	g.font.Draw(dst, g.fundsText(), (editX+3)*UIScale, (editInfoY+2)*UIScale, colInkLight)
	if g.message != "" {
		g.font.Draw(dst, g.message, (editX+70)*UIScale, (editInfoY+2)*UIScale, colInkLight)
	}

	g.drawEditMap(dst)
	blit(dst, g.tiles.UIImage(BankToolPalette, 0), editPalX, editPalY)
	g.drawToolHighlight(dst)
	blit(dst, g.tiles.UIImage(BankDemand, 0), editDemandX, editDemandY)

	// 目前工具帶。
	fill(dst, editX+1, editToolY, editW-2, editToolH, colEditFrm)
	g.font.Draw(dst, g.currentToolText(), (editX+40)*UIScale, editToolY*UIScale, colInkLight)
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
// ⚠ 十五個建造工具只有十四格：**發電廠兩種共用一格**，點下去原版會開
// 一個副選單（訊息檔第 5 段，三筆）。這裡先固定成火力發電廠，
// 副選單還沒做——記在 docs/spec/ui-layout.md。
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

// drawCityFormWindow 畫 City Form 視窗：標題列、圖層圖示、全市地圖。
func (g *Game) drawCityFormWindow(dst *ebiten.Image) {
	fill(dst, mapX, mapY, mapW, mapH, colMapFrm)
	fill(dst, mapX+2, mapY+13, mapW-4, mapH-15, colInk)
	fill(dst, mapX+1, mapY+1, mapW-2, 11, colDesktop)
	title := trimMenu(g.txt.S(i18n.SecMapTitle, 0))
	g.font.Draw(dst, title,
		(mapX+mapW/2)*UIScale-g.font.Measure(title)/2, (mapY+1)*UIScale, colInk)
	blit(dst, g.tiles.UIImage(BankMapIcons, 0), mapIconX, mapIconY)

	// 全市地圖：120×100 格，一格畫 3×3 原版像素剛好 360×300——放不下，
	// 所以取能塞進視窗的最大整數倍。
	sc := (mapX + mapW - 2 - mapViewX) / sim.WorldX
	if s2 := (mapY + mapH - 2 - mapViewY) / sim.WorldY; s2 < sc {
		sc = s2
	}
	if sc < 1 {
		sc = 1
	}
	for ty := 0; ty < sim.WorldY; ty++ {
		for tx := 0; tx < sim.WorldX; tx++ {
			fill(dst, mapViewX+tx*sc, mapViewY+ty*sc, sc, sc, g.layerColor(tx, ty))
		}
	}
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
