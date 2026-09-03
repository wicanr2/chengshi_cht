package ui

// 地形編輯器畫面的繪製與命中判斷。版面來源見 terrain_screen.go 的檔頭。

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// ------------------------------------------------------------ 命中判斷

// teMenuHit 把畫面 x 換成選單列的標題編號（0–2），不在標題上回 −1。
// 判定範圍照遊戲本體那一套：標題文字左右各留幾個像素。
func teMenuHit(x int) int {
	for i, cx := range teMenuCenterX {
		if x >= cx-48 && x <= cx+48 {
			return i
		}
	}
	return -1
}

// teFirstRow 回傳某個選單第一個「不是分隔線」的列號。
func teFirstRow(m int) int {
	for i, it := range teMenus[m].items {
		if it.key != "" {
			return i
		}
	}
	return 0
}

// teMenuGeom 算出拉開的選單的外框與每一列的 y（原版座標）。
func (g *Game) teMenuGeom(m int) (x, y, w, h int, rows []int) {
	w = menuMinW
	for _, it := range teMenus[m].items {
		if it.key == "" {
			continue
		}
		if tw := g.font.Measure(g.txt.UI(it.key))/UIScale + menuPadX*2; tw > w {
			w = tw
		}
	}
	y = menuBarH
	cur := y + menuPadY
	for _, it := range teMenus[m].items {
		rows = append(rows, cur)
		if it.key == "" {
			cur += menuSepH
		} else {
			cur += menuItemH
		}
	}
	h = cur + menuPadY - y
	x = teMenuCenterX[m] - w/2
	if x < 0 {
		x = 0
	}
	if x+w > OrigW {
		x = OrigW - w
	}
	return
}

// teMenuRowAt 把畫面座標換成拉開的選單的列號，不在選單上回 −1。
func (g *Game) teMenuRowAt(x, y int) int {
	ts := g.terrain
	if ts == nil || ts.openMenu == 0 {
		return -1
	}
	m := ts.openMenu - 1
	bx, by, bw, bh, rows := g.teMenuGeom(m)
	if x < bx || x >= bx+bw || y < by || y >= by+bh {
		return -1
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if y >= rows[i]-1 {
			return i
		}
	}
	return -1
}

// tePaletteHit 把畫面座標換成工具盤的格號（0–5），不在盤上回 −1。
func tePaletteHit(x, y int) int {
	if x < tePalBtnX || x >= tePalBtnX+tePalBtnW {
		return -1
	}
	for i := 0; i < 6; i++ {
		y0 := teBtnY0 + i*teBtnPitch
		if y >= y0 && y < y0+teBtnH {
			return i
		}
	}
	return -1
}

// 確認框的版面。原版的確認框（`sub_1C4B2`）版面未解，這裡沿用參數對話框
// 那一套字元格：置中、白底藍框，底下兩個 70×20 的按鈕。
const (
	teCfW, teCfH = 320, 110
	teCfBtnY     = 70 // 相對框頂
)

func teCfRect() (int, int) { return (OrigW - teCfW) / 2, (OrigH - teCfH) / 2 }

// teConfirmHit 回傳 0（確定）、1（取消），沒點到回 −1。
func (g *Game) teConfirmHit(x, y int) int {
	bx, by := teCfRect()
	cy := by + teCfBtnY
	if y < cy || y >= cy+teCtrlH {
		return -1
	}
	if x >= bx+40 && x < bx+40+teBtnW2 {
		return 0
	}
	if x >= bx+teCfW-40-teBtnW2 && x < bx+teCfW-40 {
		return 1
	}
	return -1
}

// teBtnW2 是確認框那兩個按鈕的寬，照參數對話框的 70×20。
const teBtnW2 = 70

// ---------------------------------------------------------------- 繪製

// drawTerrainScreen 畫整個編輯器。
func (g *Game) drawTerrainScreen(dst *ebiten.Image) {
	ts := g.terrain
	fill(dst, 0, 0, OrigW, OrigH, colDesktop)
	g.teDrawMenuBar(dst)
	g.teDrawEditWindow(dst)
	g.teDrawCityMap(dst)
	g.teDrawMenu(dst)
	g.drawTerrainEditor(dst) // 參數對話框（TERRAIN → 產生隨機地形）
	if ts.progressLeft > 0 {
		g.teDrawProgress(dst)
	}
	if ts.yearOpen {
		g.teDrawYear(dst)
	}
	if ts.confirmKey != "" {
		g.teDrawConfirm(dst)
	}
	if ts.aboutOpen {
		g.teDrawAbout(dst)
	}
	g.drawNewCity(dst) // 市名與難度
	g.drawWindow(dst)  // 讀取／儲存那幾個視窗沿用遊戲本體的
}

// teDrawMenuBar 畫三個標題的選單列。
func (g *Game) teDrawMenuBar(dst *ebiten.Image) {
	ts := g.terrain
	fill(dst, 0, 0, OrigW, menuBarH, colMenuBar)
	for i := range teMenus {
		label := g.txt.UI(teMenus[i].title)
		w := g.font.Measure(label)
		x := teMenuCenterX[i]*UIScale - w/2
		c := colMenuInk
		if ts.openMenu == i+1 {
			fill(dst, teMenuCenterX[i]-w/(2*UIScale)-2, 1,
				w/UIScale+4, menuBarH-2, colMenuInk)
			c = colMenuBar
		}
		g.font.Draw(dst, label, x, menuTextY*UIScale, c)
	}
}

// teDrawMenu 畫拉開的下拉選單。
func (g *Game) teDrawMenu(dst *ebiten.Image) {
	ts := g.terrain
	if ts.openMenu == 0 {
		return
	}
	m := ts.openMenu - 1
	x, y, w, h, rows := g.teMenuGeom(m)
	fill(dst, x, y, w, h, colMenuInk)
	fill(dst, x+1, y+1, w-2, h-2, colMenuBar)
	for i, it := range teMenus[m].items {
		if it.key == "" {
			fill(dst, x+menuPadX, rows[i]+menuSepH/2, w-menuPadX*2, 1, colMenuInk)
			continue
		}
		c := colMenuInk
		if i == ts.menuRow {
			fill(dst, x+2, rows[i]-1, w-4, menuItemH, colMenuInk)
			c = colMenuBar
		}
		// 「造島」是開關，勾起來時左邊畫一個小三角形，與遊戲本體的
		// 功能選單同一套畫法（原版 `sub_17E9A` 重畫選單時打勾）。
		if it.act == "island" && ts.ed.Island {
			drawTriangle(dst, x+2, rows[i]+3, c)
		}
		if it.act == "sound" && ts.sound {
			drawTriangle(dst, x+2, rows[i]+3, c)
		}
		g.font.Draw(dst, g.txt.UI(it.key), (x+menuPadX)*UIScale, rows[i]*UIScale, c)
	}
}

// teDrawEditWindow 畫編輯視窗。座標與遊戲本體共用同一組常數——
// 原版的編輯器就是同一個視窗系統。
func (g *Game) teDrawEditWindow(dst *ebiten.Image) {
	ew, eh := editW, editH
	vw, vh := editViewW, editViewH
	toolY := editY + eh - editToolH

	fill(dst, editX, editY, ew, eh, colEditFrm)
	fill(dst, editX+editFrameW, editTitleY, ew-2*editFrameW, editTitleH, colTitleBar)
	fill(dst, editX+editFrameW, editInfoY, ew-2*editFrameW, editInfoH, colInfoBand)
	fill(dst, editX+editFrameW, editViewY-1, ew-2*editFrameW,
		toolY-editViewY+1, colDesktop)
	// 網點左欄：編輯器的工具盤只有 156 像素高（遊戲的工具盤加需求指標更長），
	// 所以網點從工具盤底下就開始，不是從遊戲那個 y=235。
	teLeftColumnDither(dst, tePalY+tePalH, toolY)

	// 標題列：原版的編輯器只印年月，沒有資金也沒有城市名
	// （實測 `workplace/dosbox/tef-15-filled.png` 的標題列只有 `Jan 1900`）。
	right := editX + ew - 1
	drawGlyph8(dst, glyphSun, editX+3, editTitleY+2, colInk)
	g.font.Draw(dst, g.dateText(), (right-99)*UIScale, editTitleY*UIScale, colInk)

	// 訊息帶：原版的編輯器這一條是空的（沒有資金也沒有訊息）。
	// remake 借它顯示存讀檔的結果——沒有這一行的話「已存檔：city.cty」
	// 會寫到一個畫不出來的地方，玩家按了存檔完全沒有回饋。
	if g.message != "" {
		g.font.Draw(dst, g.message, msgTextX*UIScale, infoTextY*UIScale, colInkLight)
	}

	g.teDrawPalette(dst)
	g.drawEditMap(dst)
	g.teDrawCursor(dst)
	g.drawEditViewFrame(dst, vw, vh)

	// 狀態列：原版在這裡印目前工具的名稱（實測 `Dirt`／`Trees`）。
	drawToolBandBG(dst, editX, toolY, ew)
	label := g.teToolLabel()
	if tw := g.font.Measure(label) / UIScale; tw > 0 {
		fill(dst, toolTextX, toolY, tw, editToolH, colInkLight)
	}
	g.font.Draw(dst, label, toolTextX*UIScale, toolY*UIScale, colEditFrm)
}

// teLeftColumnDither 畫工具盤下方那一塊黑白一像素網點，規則同遊戲本體
// （`(x+y)` 偶數為黑）。
func teLeftColumnDither(dst *ebiten.Image, y0, toolY int) {
	x0, x1 := editX+editFrameW, editViewX-1
	for y := y0; y < toolY; y++ {
		for x := x0; x < x1; x++ {
			c := colInkLight
			if (x+y)%2 == 0 {
				c = colInk
			}
			fill(dst, x, y, 1, 1, c)
		}
	}
}

// teToolLabel 是狀態列那一行字：目前畫筆的名稱，油漆桶亮著時附註。
func (g *Game) teToolLabel() string {
	ts := g.terrain
	for _, t := range teTools {
		if sim.EditorTool(t.id) == ts.ed.Tool {
			s := g.txt.UI(t.key)
			if ts.ed.Fill {
				s += "  " + g.txt.UI("te_tool_fill")
			}
			return s
		}
	}
	return ""
}

// teDrawPalette 畫六格工具盤。
//
// 每一格的底是**對應地物的圖塊平鋪**（空地、樹林、水面、水道），
// 油漆桶是灰底藍斜線、復原是紅底白網點；字帶一圈黑描邊。
// 選取的那一格外圍多一圈黃框。全部量自原版截圖。
func (g *Game) teDrawPalette(dst *ebiten.Image) {
	ts := g.terrain
	// 外框：白 2 像素，內一圈黑。
	fill(dst, tePalX, tePalY, tePalW, tePalH, colInkLight)
	fill(dst, tePalX+2, tePalY+2, tePalW-4, tePalH-4, colInk)
	fill(dst, tePalBtnX, tePalY+2, tePalBtnW, tePalH-4, colTeSep)

	for i, t := range teTools {
		y0 := teBtnY0 + i*teBtnPitch
		switch {
		case t.tile >= 0 && g.tiles != nil:
			// 平鋪那一格的地物圖塊。
			for by := 0; by < teBtnH; by += 16 {
				for bx := 0; bx < tePalBtnW; bx += 16 {
					w, h := 16, 16
					if bx+w > tePalBtnW {
						w = tePalBtnW - bx
					}
					if by+h > teBtnH {
						h = teBtnH - by
					}
					g.teBlitTilePart(dst, t.tile, tePalBtnX+bx, y0+by, w, h)
				}
			}
		case t.id == 5:
			fill(dst, tePalBtnX, y0, tePalBtnW, teBtnH, colTeFill)
			for y := 0; y < teBtnH; y++ {
				for x := 0; x < tePalBtnW; x++ {
					if (x+y)%7 == 0 {
						fill(dst, tePalBtnX+x, y0+y, 1, 1, colTeFillIn)
					}
				}
			}
		default:
			fill(dst, tePalBtnX, y0, tePalBtnW, teBtnH, colTeUndo)
			// 沒得復原時原版是紅白網點（按鈕看起來被停用）。
			if !ts.ed.CanUndo() {
				ditherRect(dst, tePalBtnX, y0, tePalBtnW, teBtnH, colInkLight, colTeUndo)
			}
		}
		g.teDrawLabel(dst, g.txt.UI(t.key), tePalBtnX, y0, tePalBtnW, teBtnH, t.ink)
		if sim.EditorTool(t.id) == ts.ed.Tool || (t.id == 5 && ts.ed.Fill) {
			fill(dst, tePalBtnX, y0, tePalBtnW, 2, colTeSel)
			fill(dst, tePalBtnX, y0+teBtnH-2, tePalBtnW, 2, colTeSel)
			fill(dst, tePalBtnX, y0, 2, teBtnH, colTeSel)
			fill(dst, tePalBtnX+tePalBtnW-2, y0, 2, teBtnH, colTeSel)
		}
	}
}

// teBlitTilePart 把一張 16×16 圖塊的左上角 w×h 畫到指定位置。
func (g *Game) teBlitTilePart(dst *ebiten.Image, tile, x, y, w, h int) {
	img := g.tiles.ZoomTile(tile, 0)
	if img == nil {
		return
	}
	sub := img.SubImage(rect(0, 0, w, h)).(*ebiten.Image)
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(UIScale, UIScale)
	op.GeoM.Translate(float64(x*UIScale), float64(y*UIScale))
	dst.DrawImage(sub, &op)
}

// teDrawLabel 把字置中畫在按鈕上，先描一圈邊。
// 原版的六個標籤是帶描邊的粗體字，底下是有花紋的地物圖塊——
// 沒有描邊的話中文字的細筆畫會融進圖塊裡。
//
// ⚠ 描邊的顏色要跟字**相反**：原版四個亮字（黃／白／青）配黑邊，
// 而油漆桶與復原是**黑字**，再描黑邊就整團糊掉，那兩個要配白邊。
func (g *Game) teDrawLabel(dst *ebiten.Image, s string, x, y, w, h int, c color.RGBA) {
	tx := x*UIScale + (w*UIScale-g.font.Measure(s))/2
	ty := (y+(h-teCellH)/2)*UIScale + 2
	edge := colInk
	if int(c.R)+int(c.G)+int(c.B) < 3*0x80 {
		edge = colInkLight
	}
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		g.font.Draw(dst, s, tx+d[0]*UIScale, ty+d[1]*UIScale, edge)
	}
	g.font.Draw(dst, s, tx, ty, c)
}

// teDrawCursor 畫工具游標：一格見方的框。編輯器的四個畫筆都是 1×1
// （工具描述表那四列的尺寸欄都是 1）。
func (g *Game) teDrawCursor(dst *ebiten.Image) {
	ts := g.terrain
	if ts.openMenu != 0 || ts.aboutOpen || ts.confirmKey != "" ||
		ts.yearOpen || g.terrainDlg != nil || g.win != winNone {
		return
	}
	mx, my := ebiten.CursorPosition()
	if !g.inEditView(mx, my) {
		return
	}
	px := g.tileSize() * tileScale
	cx := (mx-editViewX*UIScale)/px*px + editViewX*UIScale
	cy := (my-editViewY*UIScale)/px*px + editViewY*UIScale
	vector.StrokeRect(dst, float32(cx), float32(cy), float32(px), float32(px),
		float32(2*UIScale), colTeSel, false)
}

// teDrawCityMap 畫右邊的 City Map 視窗。
//
// 與遊戲的 City Form 是同一個視窗（外框、標題列、綠邊那一欄的座標全部相同），
// 差別只在**沒有圖層圖示也沒有色階圖例**——編輯器只有一個圖層。
func (g *Game) teDrawCityMap(dst *ebiten.Image) {
	fill(dst, mapX, mapY, mapW, mapH, colMapFrmD)
	fill(dst, mapX+1, mapY+1, mapW-2, mapH-2, colMapFrm)
	ditherRect(dst, mapTitleX, mapTitleY, mapTitleW, 17, colInkLight, colMenuInk)
	fill(dst, mapX+2, mapContentY, mapW-6, mapContentH, colInkLight)
	fill(dst, mapX+2, mapContentY, 30, mapContentH, colMapGreen)
	// 圖示欄在編輯器裡是空的，只留原版那兩條一像素棋盤邊。
	ih := mapIconH * mapIconCount
	ditherRect(dst, mapIconEdgeX, mapIconY, 1, ih, colInk, colInkLight)
	ditherRect(dst, mapIconX, mapIconY, mapViewX-4-mapIconX, mapContentH-2,
		colInk, colInkLight)
	ditherRect(dst, mapViewX-4, mapIconY, 2, ih, colInk, colInkLight)

	title := g.txt.UI("te_citymap")
	tw := g.font.Measure(title) / UIScale
	fill(dst, mapX+mapW/2-tw/2-8, mapTitleY+2, tw+16, 13, colInkLight)
	g.font.Draw(dst, title,
		(mapX+mapW/2)*UIScale-g.font.Measure(title)/2, (mapTitleY+2)*UIScale, colMenuInk)

	fill(dst, mapViewX, mapViewY, sim.WorldX*3, sim.WorldY*3, colInk)
	sc := (mapX + mapW - 2 - mapViewX) / sim.WorldX
	if s2 := (mapY + mapH - 2 - mapViewY) / sim.WorldY; s2 < sc {
		sc = s2
	}
	if sc < 1 {
		sc = 1
	}
	g.drawMinimap(dst, mapViewX*UIScale, mapViewY*UIScale, sc*UIScale)
	vector.StrokeRect(dst,
		s(mapViewX+g.camX*sc), s(mapViewY+g.camY*sc),
		s(g.tilesAcross()*sc), s(g.tilesDown()*sc),
		float32(UIScale), colMenuInk, false)
}

// teBox 畫一個原版樣式的對話框：藍框、白底，回傳客戶區的左上角。
//
// 版面規則沿用參數對話框量到的那一套（docs/spec/terrain-editor.md §六）：
// 視窗以字元格為單位建立，客戶區 ＝ (欄×8−8) × (列×14−8)，水平置中、
// 垂直比中央高七個像素。等級：假說——只有 36×10 那一個實際量過。
func teBoxRect(cols, rows int) (x, y, w, h int) {
	w = cols*8 - 8
	h = rows*14 - 8
	x = (OrigW - w) / 2
	y = (OrigH-h)/2 - 7
	return
}

func (g *Game) teDrawBox(dst *ebiten.Image, cols, rows int) (int, int, int, int) {
	x, y, w, h := teBoxRect(cols, rows)
	fill(dst, x-teBorder, y-teBorder, w+2*teBorder, h+2*teBorder, colTELine)
	fill(dst, x, y, w, h, colDlgBG)
	return x, y, w, h
}

// teDrawProgress 畫「正在造地形」／「平滑中……」。
// 原版是 20×5 與 16×5 個字元格的視窗（`sub_1C010(&win,0x14,5)`／`(…,0x10,5)`）。
func (g *Game) teDrawProgress(dst *ebiten.Image) {
	ts := g.terrain
	x, y, w, h := g.teDrawBox(dst, ts.progressCols, 5)
	g.font.DrawCentered(dst, ts.progress, x*UIScale,
		(y+(h-teCellH)/2)*UIScale, w*UIScale, colTELine)
}

// teDrawYear 畫「輸入遊戲年份」。原版是 18×5 個字元格
// （`sub_111E4`＋0x1120C 的 `sub_1C010(&win, 0x12, 5)`），
// 標題是 `Enter Game Year:`，欄位預填 `%4d` 的目前年份。
func (g *Game) teDrawYear(dst *ebiten.Image) {
	ts := g.terrain
	x, y, w, h := g.teDrawBox(dst, 18, 5)
	g.font.DrawCentered(dst, g.txt.UI("te_year_prompt"), x*UIScale,
		(y+6)*UIScale, w*UIScale, colTELine)
	fx, fy, fw := x+w/2-32, y+h-teCtrlH-8, 64
	fill(dst, fx, fy, fw, teCtrlH, colTEFill)
	g.font.DrawCentered(dst, ts.yearInput+"_", fx*UIScale,
		(fy+(teCtrlH-teCellH)/2)*UIScale, fw*UIScale, colTELine)
}

// teDrawConfirm 畫確認框。原版的確認框版面未解，這裡沿用參數對話框的配色
// 與按鈕尺寸——**標成 remake 自訂**，不要寫成量自原版。
func (g *Game) teDrawConfirm(dst *ebiten.Image) {
	ts := g.terrain
	x, y := teCfRect()
	fill(dst, x-teBorder, y-teBorder, teCfW+2*teBorder, teCfH+2*teBorder, colTELine)
	fill(dst, x, y, teCfW, teCfH, colDlgBG)
	g.font.DrawCentered(dst, g.txt.UI(ts.confirmKey), x*UIScale, (y+16)*UIScale,
		teCfW*UIScale, colTELine)
	for i, key := range [2]string{"te_yes", "te_no"} {
		bx := x + 40
		if i == 1 {
			bx = x + teCfW - 40 - teBtnW2
		}
		by := y + teCfBtnY
		fill(dst, bx, by, teBtnW2, teCtrlH, colTELine)
		fill(dst, bx+1, by+1, teBtnW2-2, teCtrlH-2, colTEFill)
		g.font.DrawCentered(dst, g.txt.UI(key), bx*UIScale,
			(by+(teCtrlH-teCellH)/2)*UIScale, teBtnW2*UIScale, colTELine)
	}
}

// teAboutLines 是編輯器的「關於」。
//
// **不是把原版那一頁照抄**：原版印的是 Maxis 1989 年的職稱表加上當年的
// 地址、電話與傳真，把那些放進 remake 會變成拿別人的聯絡方式當自己的。
// 原版那一頁的**全文轉錄留在 docs/re/20-terrain-editor.md §十七**（保存），
// 這裡只留史實性的工作人員名單，加上本專案自己的授權聲明。
func (g *Game) teAboutLines() []string {
	return []string{
		"地形編輯器 — 照 1990 年磁片版重製",
		"chengshi_cht " + g.version,
		"",
		"原版：SimCity Terrain Editor（Maxis, 1989）",
		"　構想與設計：Will Wright",
		"　IBM 版程式：Paul Schmidt、Daniel Goldman",
		"　城市美術：Don Bayless",
		"　標題畫面與圖示：Richard Payne",
		"　說明文件：Michael Bremer",
		"",
		"本頁的工作人員名單轉錄自原版的關於畫面（史料）。",
		"SimCity 與 Maxis 是 Electronic Arts 的商標，",
		"本專案與 EA 無隸屬關係，商標僅作指示性使用。",
		"",
		"Required Notice: Copyright 2026 Wang Chun-Yu (wicanr2)",
		"授權：RRSAL-1.0（非商業免費，商業洽談）",
	}
}

func (g *Game) teDrawAbout(dst *ebiten.Image) {
	lines := g.teAboutLines()
	w, h := 460, len(lines)*16+24
	x, y := (OrigW-w)/2, (OrigH-h)/2
	fill(dst, x-teBorder, y-teBorder, w+2*teBorder, h+2*teBorder, colTeSel)
	fill(dst, x, y, w, h, colDlgBG)
	for i, s := range lines {
		g.font.Draw(dst, s, (x+10)*UIScale, (y+12+i*16)*UIScale, colTELine)
	}
}
