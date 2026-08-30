package ui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
)

// 下拉選單。原版是**按住式**的：按住標題拉開，滑到項目上放開才選中。
//
// 這裡兩種都吃：按住拉開＋放開選中（原版），或者點一下拉開、再點一下選中
// （放開時還在標題上就讓選單留著）。後者不是原版行為，但按住式在觸控與
// 無障礙上很難用，而且不影響原版路徑——原版怎麼操作，這裡就怎麼有效。
//
// 項目文字一律取自訊息檔（第 17／18／20／21 段），連 `Ctrl-M` 那種快速鍵
// 提示都是原版自己印在選單裡的。文字是 `-` 的那一筆是分隔線。

// menuSections 是四個選單各自的訊息檔段落，順序同選單列。
var menuSections = [4]int{
	i18n.SecSysMenu,  // 系統
	i18n.SecOptMenu,  // 功能
	i18n.SecDisaster, // 災難
	i18n.SecWinMenu,  // 視窗
}

// 下拉選單的版面，單位是原版像素。
const (
	menuItemH = 14 // 一列的高
	menuPadX  = 6
	menuPadY  = 3
	menuSepH  = 6 // 分隔線那一列比較矮
	menuMinW  = 90
)

// menuEntries 回傳第 m 個選單（0–3）的項目文字。
func (g *Game) menuEntries(m int) []string {
	sec := menuSections[m]
	var out []string
	for i := 0; g.txt.Has(sec, i); i++ {
		out = append(out, trimMenu(g.txt.S(sec, i)))
	}
	return out
}

func isSeparator(s string) bool { return strings.TrimSpace(s) == "-" }

// menuGeom 算出第 m 個選單拉開之後的外框（原版座標）與每一列的 y。
func (g *Game) menuGeom(m int) (x, y, w, h int, rows []int) {
	items := g.menuEntries(m)
	w = menuMinW
	for _, s := range items {
		if tw := g.font.Measure(s)/UIScale + menuPadX*2; tw > w {
			w = tw
		}
	}
	y = menuBarH
	cur := y + menuPadY
	for _, s := range items {
		rows = append(rows, cur)
		if isSeparator(s) {
			cur += menuSepH
		} else {
			cur += menuItemH
		}
	}
	h = cur + menuPadY - y
	// 靠標題置中，但不要跑出畫面。
	x = menuTitles[m].centerX - w/2
	if x < 0 {
		x = 0
	}
	if x+w > OrigW {
		x = OrigW - w
	}
	return
}

// drawMenu 畫拉開的下拉選單。
func (g *Game) drawMenu(dst *ebiten.Image) {
	if g.openMenu == 0 {
		return
	}
	m := g.openMenu - 1
	x, y, w, h, rows := g.menuGeom(m)
	fill(dst, x, y, w, h, colMenuInk)
	fill(dst, x+1, y+1, w-2, h-2, colMenuBar)
	items := g.menuEntries(m)
	for i, s := range items {
		if isSeparator(s) {
			fill(dst, x+menuPadX, rows[i]+menuSepH/2, w-menuPadX*2, 1, colMenuInk)
			continue
		}
		c := colMenuInk
		if i == g.menuRow {
			fill(dst, x+2, rows[i]-1, w-4, menuItemH, colMenuInk)
			c = colMenuBar
		}
		// 功能選單的開關項在左邊畫一個小三角形表示開著，照原版。
		//
		// ⚠ **用畫的不要用字**：`▸`（U+25B8）在 Noto Sans CJK 裡沒有字形，
		// 烘出來是一個空心方框，而字集檢查照樣過——因為那個字**在**字集裡，
		// 只是字型畫不出來。看起來像版面錯了，其實是缺字形。
		if m == 1 && g.optionOn(i) {
			drawTriangle(dst, x+2, rows[i]+3, c)
		}
		g.font.Draw(dst, s, (x+menuPadX)*UIScale, rows[i]*UIScale, c)
	}
}

// menuHit 把畫面座標換成選單列號；不在選單上回 −1。
func (g *Game) menuHit(mx, my int) int {
	if g.openMenu == 0 {
		return -1
	}
	m := g.openMenu - 1
	x, y, w, h, rows := g.menuGeom(m)
	px, py := mx/UIScale, my/UIScale
	if px < x || px >= x+w || py < y || py >= y+h {
		return -1
	}
	items := g.menuEntries(m)
	for i := range items {
		hgt := menuItemH
		if isSeparator(items[i]) {
			hgt = menuSepH
		}
		if py >= rows[i] && py < rows[i]+hgt {
			if isSeparator(items[i]) {
				return -1
			}
			return i
		}
	}
	return -1
}

// menuBarHit 回傳滑鼠在選單列上的哪一個標題（1–4），不在回 0。
func (g *Game) menuBarHit(mx, my int) int {
	if my >= menuBarH*UIScale {
		return 0
	}
	for i, t := range menuTitles {
		label := trimMenu(g.txt.S(i18n.SecMenu, t.sec))
		w := g.font.Measure(label)/UIScale + 8
		if px := mx / UIScale; px >= t.centerX-w/2 && px < t.centerX+w/2 {
			return i + 1
		}
	}
	return 0
}

// handleMenuMouse 處理選單列與下拉選單的滑鼠。回傳 true 代表這一格的
// 滑鼠事件歸選單，其他地方不要再處理。
func (g *Game) handleMenuMouse(mx, my int) bool {
	just := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	released := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	if just {
		if t := g.menuBarHit(mx, my); t != 0 {
			if g.openMenu == t {
				g.openMenu = 0
			} else {
				g.openMenu, g.menuRow = t, -1
			}
			return true
		}
	}
	if g.openMenu == 0 {
		return false
	}
	g.menuRow = g.menuHit(mx, my)
	// 放開時若停在項目上就選中；停在標題上則讓選單留著（點一下拉開）。
	if (released || just) && g.menuRow >= 0 {
		m, row := g.openMenu-1, g.menuRow
		g.openMenu, g.menuRow = 0, -1
		g.menuPick(m, row)
		return true
	}
	if just && g.menuBarHit(mx, my) == 0 {
		g.openMenu = 0
	}
	return true
}

// menuPick 執行第 m 個選單的第 row 列。
//
// 每一個選單的列號就是訊息檔段落裡的索引，**含分隔線**——所以不能用
// 「第幾個可點的項目」當索引，那會整個錯開一格。
func (g *Game) menuPick(m, row int) {
	switch m {
	case 0:
		g.pickSystem(row)
	case 1:
		g.pickOptions(row)
	case 2:
		g.pickDisaster(row)
	case 3:
		g.pickWindow(row)
	}
}

// pickSystem 是系統選單（第 17 段）。索引就是原版的列號。
func (g *Game) pickSystem(row int) {
	switch row {
	case 0:
		g.win = winAbout
	case 2:
		g.printMap()
	case 3:
		g.openSubMenu(winStyle)
	case 5:
		g.openSubMenu(winScenario)
	case 6:
		g.newCity()
	case 8:
		g.load()
	case 9:
		g.openSaveAs()
	case 10:
		g.save()
	case 12:
		g.quit = true
	}
}

// pickOptions 是功能選單（第 18 段）。前四項與最後兩項都是開關，
// 原版在項目左邊畫一個 `▸` 表示開著（見原版截圖）。
//
// 前三個開關的狀態**存在城市檔裡**（`MiscHis[52..54]`，見 cityfile.go），
// 所以讀檔會帶回玩家上次的設定——這不是 remake 自己加的狀態。
func (g *Game) pickOptions(row int) {
	w := g.world
	switch row {
	case 0:
		w.AutoBulldoze = !w.AutoBulldoze
	case 1:
		w.AutoBudget = !w.AutoBudget
	case 2:
		w.AutoGo = !w.AutoGo
	case 3:
		g.soundOff = !g.soundOff
	case 4:
		g.win = winSpeed
		g.sysRow = 0
		return
	case 5:
		g.animate = !g.animate
	case 6:
		g.fastAnimate = !g.fastAnimate
	}
	g.setMessage(trimMenu(g.txt.S(i18n.SecOptMenu, row)) + "：" + g.onOff(g.optionOn(row)))
}

// optionOn 回報功能選單第 row 項現在是開還是關。
func (g *Game) optionOn(row int) bool {
	switch row {
	case 0:
		return g.world.AutoBulldoze
	case 1:
		return g.world.AutoBudget
	case 2:
		return g.world.AutoGo
	case 3:
		return !g.soundOff
	case 5:
		return g.animate
	case 6:
		return g.fastAnimate
	}
	return false
}

func (g *Game) onOff(b bool) string {
	if b {
		return g.txt.UI("on")
	}
	return g.txt.UI("off")
}

// pickDisaster 是災難選單（第 20 段）。前六筆是六種災難、第 6 筆是分隔線、
// **第 7 筆是「停用災難」**（原文 " Disable "）。
//
// ⚠ 第 7 筆先前沒有接：選了完全沒反應，而選單照樣畫得出來。
// 它對應 Micropolis 的 `NoDisasters`（`s_disast.c:87` 一開頭就 return），
// 模擬層早就有這個旗標，只差呈現層沒接上。
func (g *Game) pickDisaster(row int) {
	switch {
	case row < len(disasterItems):
		g.fireDisaster(row)
	case row == len(disasterItems)+1: // 跳過分隔線
		g.world.NoDisasters = !g.world.NoDisasters
		g.setMessage(trimMenu(g.txt.S(i18n.SecDisaster, row)) + "：" +
			g.onOff(g.world.NoDisasters))
		g.win = winNone
	}
}

// pickWindow 是視窗選單（第 21 段）。
func (g *Game) pickWindow(row int) {
	switch row {
	case 0: // 地圖視窗：City Form 收起來的話先叫回來
		if g.mapClosed {
			g.mapClosed = false
			g.editFront = false
		} else {
			g.toggleWindow(winMaps)
		}
	case 1:
		g.toggleWindow(winGraphs)
	case 2:
		g.toggleWindow(winBudget)
	case 3: // 編輯視窗：叫到最前面
		g.win = winNone
		g.editFront = true
	case 4:
		g.toggleWindow(winEval)
	case 5: // 關閉前視窗
		if g.win != winNone {
			g.win = winNone
		} else {
			g.mapClosed = true
		}
	case 6: // 隱藏前視窗 ＝ 把最前面的移到最後面（實測，不是收起來）
		if g.win != winNone {
			g.win = winNone
		} else {
			g.editFront = !g.editFront
		}
	case 7: // 視窗位置：直接拖標題列就好，這裡只提示
		g.setMessage("拖曳視窗的標題列就可以搬動")
	case 8: // 調整編輯窗大小
		g.resizing = true
	}
}

// firstMenuRow 回傳第 m 個選單第一個可選的列（跳過分隔線）。
func (g *Game) firstMenuRow(m int) int {
	for i, s := range g.menuEntries(m) {
		if !isSeparator(s) {
			return i
		}
	}
	return -1
}

// handleMenuKeys 讓下拉選單也能用鍵盤走。原版是滑鼠拉的，
// 但鍵盤走得動對截圖驗收與無障礙都必要。
func (g *Game) handleMenuKeys() bool {
	if g.openMenu == 0 {
		return false
	}
	items := g.menuEntries(g.openMenu - 1)
	step := 0
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		step = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		step = -1
	}
	if step != 0 && len(items) > 0 {
		r := g.menuRow
		for i := 0; i < len(items); i++ {
			r = (r + step + len(items)) % len(items)
			if !isSeparator(items[r]) {
				break
			}
		}
		g.menuRow = r
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.openMenu, g.menuRow = 0, -1
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		m, row := g.openMenu-1, g.menuRow
		g.openMenu, g.menuRow = 0, -1
		if row >= 0 {
			g.menuPick(m, row)
		}
	}
	return true
}

// drawTriangle 畫一個朝右的小三角形（原版座標，高 7 寬 4）。
func drawTriangle(dst *ebiten.Image, x, y int, c color.RGBA) {
	for i := 0; i < 4; i++ {
		h := 7 - i*2
		if h <= 0 {
			break
		}
		fill(dst, x+i, y+i, 1, h, c)
	}
}
