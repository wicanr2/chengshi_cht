package ui

import (
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

// pickOptions 是功能選單（第 18 段）。
//
// ⚠ 只有速度那一項接得起來：全自動整地、自動預算、自動前往、音效開關、
// 兩個動畫選項在 remake 裡分別對應不同的東西，還沒接。
// 沒接的項目點下去只顯示一行訊息，不要靜默無反應——玩家會以為當掉了。
func (g *Game) pickOptions(row int) {
	switch row {
	case 4:
		g.win = winSpeed
		g.sysRow = 0
	default:
		g.setMessage(trimMenu(g.txt.S(i18n.SecOptMenu, row)) + "（還沒接）")
	}
}

// pickDisaster 是災難選單（第 20 段）。第 6 筆是分隔線、第 7 筆是取消。
func (g *Game) pickDisaster(row int) {
	if row < len(disasterItems) {
		g.fireDisaster(row)
	}
}

// pickWindow 是視窗選單（第 21 段）。
func (g *Game) pickWindow(row int) {
	switch row {
	case 0:
		g.toggleWindow(winMaps)
	case 1:
		g.toggleWindow(winGraphs)
	case 2:
		g.toggleWindow(winBudget)
	case 3:
		g.win = winNone // 編輯視窗一直開著，選它就是把別的關掉
	case 4:
		g.toggleWindow(winEval)
	case 5, 6:
		g.win = winNone
	default:
		g.setMessage(trimMenu(g.txt.S(i18n.SecWinMenu, row)) + "（視窗大小固定，還沒接）")
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
