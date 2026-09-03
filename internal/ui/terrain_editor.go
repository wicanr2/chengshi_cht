package ui

// 地形編輯器的參數對話框。
//
// 原版是**獨立程式** `TERRAIN.EXE`（MAXIS SimCity Terrain Editor 1.0），
// 不是遊戲的一部分：三個百分比、按 Go，程式產生地形再進遊戲。
// 版面規格在 docs/spec/terrain-editor.md（READY），拆解過程在
// docs/re/20-terrain-editor.md。
//
// 版面照原版的字元格擺：視窗 36 欄 × 10 列（`sub_1C010(&win, 0x24, 0x0A)`），
// 一格 8 × 14 原版像素。六個增減鍵與兩個按鈕的欄列都是量出來的。
//
// ⚠ 原版視窗在畫面上的**絕對位置未解**（藏在 `sub_1C010` 裡），這裡置中。

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

const (
	teCols, teRows   = 36, 10 // sub_1C010(&win, 0x24, 0x0A)
	teCellW, teCellH = 8, 14  // 一個字元格。8 出自版面換算，14 是 EGA 高解析的行高
	teW, teH         = teCols * teCellW, teRows * teCellH
	teX, teY         = (OrigW - teW) / 2, (OrigH - teH) / 2

	teLabelRow  = 3 // 原版兩行英文標籤佔 3 與 4；中文一行放得下，只用 3
	teValueRow  = 5 // 三個 %3d%% 與六個 ◄／► 同一列
	teButtonRow = 8
	teBtnCols   = 8 // `   Go   ` 與 ` Cancel ` 都是八格

	// teRepeat 是長按增減鍵的重複間隔。原版是「按著不放且經過 5 個計時單位」
	// 就再做一次（`sub_11402`＋0x118E9）。**5 個單位是多久未解**：計時來源
	// `sub_18CA9` 還沒讀。這裡假設是 BIOS 的 18.2 Hz 計時器，5 拍約 275 ms，
	// 在 60 TPS 下取 16 幀。等級：假說。
	teRepeat = 16
)

// teGroup 是一個參數的三個控制項：減、增、以及夾在中間的百分比。
// 欄位相對視窗左欄，取自 docs/spec/terrain-editor.md §二的表。
var teGroups = [3]struct {
	dec, inc int    // ◄ 與 ► 的欄
	key      string // 標籤的 ui.tsv 鍵
}{
	{3, 10, "terrain_trees"},
	{14, 21, "terrain_lakes"},
	{25, 32, "terrain_curve"},
}

// terrainBox 是對話框的狀態。三個值的初值都是 50（`dseg:0x0B6` 起 `32 00` ×3）。
type terrainBox struct {
	val   [3]int
	focus int // 0–7，對應原版的控制項編號 0x800–0x807
	held  int // 目前按著的控制項編號，-1 代表沒有
	frame int // 按著幾幀了，用來做自動重複
}

// openTerrainEditor 打開對話框。
func (g *Game) openTerrainEditor() {
	g.terrainDlg = &terrainBox{val: [3]int{50, 50, 50}, held: -1}
}

// teCol／teRow 把字元格換算成原版像素。
func teCol(c int) int { return teX + c*teCellW }
func teRow(r int) int { return teY + r*teCellH }

// teControlRect 回傳第 id 個控制項（0–7）的原版像素方框。
// 0–5 是三組 ◄／►，6 是開始，7 是取消。
func teControlRect(id int) (x, y, w, h int) {
	if id >= 6 {
		c := 3
		if id == 7 {
			c = 25
		}
		return teCol(c), teRow(teButtonRow), teBtnCols * teCellW, teCellH
	}
	gp := teGroups[id/2]
	c := gp.dec
	if id%2 == 1 {
		c = gp.inc
	}
	return teCol(c), teRow(teValueRow), teCellW, teCellH
}

// teHit 找出滑鼠落在哪個控制項上，沒有就回 -1。
func teHit(x, y int) int {
	for id := 0; id < 8; id++ {
		cx, cy, cw, ch := teControlRect(id)
		if x >= cx && x < cx+cw && y >= cy && y < cy+ch {
			return id
		}
	}
	return -1
}

// terrainPress 執行一個控制項的動作。
// 0x800–0x805 是加減 1 再夾限到 0–100（原版 `sub_113E4(0, 值, 100)`）；
// 0x806 是開始、0x807 是取消。
func (g *Game) terrainPress(id int) {
	b := g.terrainDlg
	switch {
	case id < 6:
		d := -1
		if id%2 == 1 {
			d = 1
		}
		v := b.val[id/2] + d
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		b.val[id/2] = v
	case id == 6:
		g.terrainGo()
	case id == 7:
		g.terrainDlg = nil
	}
}

// terrainGo 收下三個百分比，交給「建造新城市」對話框接手。
//
// 原版按 Go 之後是「Now terraforming」→「Smoothing...」→ 問年份 → 問難度
// （docs/spec/terrain-editor.md §三，等級：強證據）。remake 只接難度那一段——
// 進度訊息與年份輸入的版面**未解**，沒有版面就不自己編一個。
func (g *Game) terrainGo() {
	b := g.terrainDlg
	p := sim.TerrainParams{
		TreeLevel:    b.val[0],
		LakeLevel:    b.val[1],
		CurveLevel:   b.val[2],
		CreateIsland: 0, // 原版編輯器沒有這個介面元素，不要自己加
		EditorDOS:    true,
	}
	g.terrainDlg = nil
	g.openNewCity()
	g.newCityDlg.terrain = &p
}

// handleTerrainKeys 處理對話框的按鍵；回傳 true 代表這一格的鍵盤歸它。
//
// 原版的 `+`／`-` 是把**滑鼠游標**移到下／上一個控制項的中心
// （`sub_19139` → `sub_18F81`），八個輪一圈。Ebiten 沒有移動游標的介面，
// 所以改成移動一個選取框，另外補一個空白鍵按下它——**這一鍵是 remake 加的**，
// 原版靠移過去之後真的按滑鼠。
func (g *Game) handleTerrainKeys() bool {
	b := g.terrainDlg
	if b == nil {
		return false
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEqual),
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd):
		b.focus = (b.focus + 1) % 8
	case inpututil.IsKeyJustPressed(ebiten.KeyMinus),
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract):
		b.focus = (b.focus + 7) % 8
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.terrainPress(b.focus)
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter),
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter),
		inpututil.IsKeyJustPressed(ebiten.KeyG):
		g.terrainGo()
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		inpututil.IsKeyJustPressed(ebiten.KeyC):
		g.terrainDlg = nil
	}
	return true
}

// handleTerrainMouse 處理點擊與長按重複。
func (g *Game) handleTerrainMouse(mx, my int) bool {
	b := g.terrainDlg
	if b == nil {
		return false
	}
	x, y := mx/UIScale, my/UIScale
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if id := teHit(x, y); id >= 0 {
			b.focus, b.held, b.frame = id, id, 0
			g.terrainPress(id)
		}
		return true
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		b.held = -1
		return true
	}
	// 按著不放：原版只在游標還在那個控制項上時才重複。
	if b.held < 0 || b.held >= 6 || teHit(x, y) != b.held {
		return true
	}
	b.frame++
	if b.frame%teRepeat == 0 {
		g.terrainPress(b.held)
	}
	return true
}

// drawTerrainArrow 畫一個 ◄ 或 ►。原版用的是 CP437 的 0x11／0x10，
// 字型圖集裡沒有這兩個字，所以直接畫成三角形（原本也就是 8×14 的點陣字）。
func drawTerrainArrow(dst *ebiten.Image, x, y int, left bool, c color.RGBA) {
	for i := 0; i < 6; i++ {
		cx := x + 1 + i
		if !left {
			cx = x + 6 - i
		}
		fill(dst, cx, y+7-i, 1, 2*i+1, c)
	}
}

// drawTerrainEditor 畫對話框。
func (g *Game) drawTerrainEditor(dst *ebiten.Image) {
	b := g.terrainDlg
	if b == nil {
		return
	}
	fill(dst, teX-ncBorder, teY-ncBorder,
		teW+2*ncBorder, teH+2*ncBorder, colDlgLine)
	fill(dst, teX, teY, teW, teH, colDlgBG)

	g.font.DrawCentered(dst, g.txt.UI("terrain_title"),
		teX*UIScale, teRow(0)*UIScale, teW*UIScale, colDlgLine)

	for i, gp := range teGroups {
		// 標籤置中在 ◄ 與 ► 圍出來的範圍上。
		lx, lw := teCol(gp.dec), (gp.inc-gp.dec+1)*teCellW
		g.font.DrawCentered(dst, g.txt.UI(gp.key),
			lx*UIScale, teRow(teLabelRow)*UIScale, lw*UIScale, colDlgLine)

		ax, ay, _, _ := teControlRect(i * 2)
		bx, _, _, _ := teControlRect(i*2 + 1)
		drawTerrainArrow(dst, ax, ay, true, colDlgLine)
		drawTerrainArrow(dst, bx, ay, false, colDlgLine)

		// `%3d%%` 四格，左對齊在 ◄ 右邊兩格（原版 X ＝ 左欄×8+16+24）。
		// 字型的半形字正好一格寬，所以逐格對得上原版。
		g.font.Draw(dst, fmt.Sprintf("%3d%%", b.val[i]),
			teCol(gp.dec+2)*UIScale, teRow(teValueRow)*UIScale, colDlgLine)
	}

	for id, key := range [2]string{"terrain_go", "terrain_cancel"} {
		x, y, w, h := teControlRect(6 + id)
		fill(dst, x, y, w, h, colDlgLine)
		fill(dst, x+1, y+1, w-2, h-2, colDlgFill)
		g.font.DrawCentered(dst, g.txt.UI(key),
			x*UIScale, y*UIScale, w*UIScale, colDlgLine)
	}

	// 選取框。原版是把游標移過去，沒有框；這是 remake 的替代品。
	fx, fy, fw, fh := teControlRect(b.focus)
	outline(dst, fx-2, fy-2, fw+4, fh+4, colDlgLine)
}

// outline 畫一個一像素的方框。
func outline(dst *ebiten.Image, x, y, w, h int, c color.RGBA) {
	fill(dst, x, y, w, 1, c)
	fill(dst, x, y+h-1, w, 1, c)
	fill(dst, x, y, 1, h, c)
	fill(dst, x+w-1, y, 1, h, c)
}
