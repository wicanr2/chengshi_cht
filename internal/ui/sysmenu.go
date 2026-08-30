package ui

// 系統選單（原版 Alt-S，說明書 p.29–31）與它的兩個副選單。
//
// 為什麼要有這個：在它之前，換劇本、換圖形集、開新城市、讀取存檔
// **全部只能靠命令列參數**——玩家要玩第二個悲情城市得先關掉遊戲。
// 原版的系統選單就有這些項目（訊息檔第 17 段），只是 remake 沒接。
//
// 選項名稱一律取自訊息檔；六個圖形集的中文名說明書沒收，
// 用的是 translations/glossary.md §十 登記的新譯。

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// sysItems 是系統選單的項目，順序與值都取自訊息檔第 17 段
// （分隔線 1／4／7／11 不列）。
//
// ⚠ 「印表」是 remake 自訂的對應物：原版印的是紙上的全市地圖，
// 這裡存成 PNG。標在 docs/spec/controls.md，不要寫成原版行為。
var sysItems = []struct {
	msg  int // 訊息檔第 17 段的索引
	kind string
}{
	{0, "about"},    // 關於本遊戲
	{2, "print"},    // 印表 → 存成 PNG
	{3, "style"},    // 讀取圖形集
	{5, "scenario"}, // 讀取悲情城市
	{6, "new"},      // 重新建造一新城市
	{8, "load"},     // 讀取舊有檔案  Ctrl-L
	{9, "saveas"},   // 以……檔名儲存
	{10, "save"},    // 儲存現有城市  Ctrl-S
	{12, "quit"},    // 跳出遊戲      Ctrl-X
}

// styleOrder 是圖形集的順序。base 是沒有資料片的原始外觀，排第一。
var styleOrder = []struct{ key, name string }{
	{"base", "基本"},
	{"asia", "古代亞洲"},
	{"medi", "中世紀"},
	{"west", "西部拓荒"},
	{"fusa", "未來美國"},
	{"feur", "未來歐洲"},
	{"moon", "月球殖民地"},
}

// SetDataDir 告訴呈現層原版資料在哪、目前用的是哪個圖形集。
// 沒設的話系統選單裡需要重讀資料的項目會停用。
func (g *Game) SetDataDir(dir, style string) {
	g.dataDir, g.style = dir, style
}

func (g *Game) sysMenuLen() int {
	switch g.win {
	case winSystem:
		return len(sysItems)
	case winScenario:
		return 8
	case winStyle:
		return len(styleOrder)
	case winSpeed:
		return 5
	case winPower:
		return len(powerTools)
	}
	return 0
}

// sysMenuLabel 回傳目前選單第 i 列要顯示的字。
func (g *Game) sysMenuLabel(i int) string {
	switch g.win {
	case winSystem:
		return trimMenu(g.txt.S(i18n.SecSysMenu, sysItems[i].msg))
	case winScenario:
		return game.ScenarioNameZH(i + 1)
	case winStyle:
		return styleOrder[i].name
	case winSpeed:
		return trimMenu(g.txt.S(i18n.SecSpeed, i))
	case winPower:
		return trimMenu(g.txt.S(i18n.SecPowerSub, powerTools[i].msg))
	}
	return ""
}

// handleSysMenuKeys 處理三個選單共用的上下選與 Enter。
func (g *Game) handleSysMenuKeys() {
	n := g.sysMenuLen()
	if n == 0 {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.sysRow = (g.sysRow + 1) % n
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.sysRow = (g.sysRow + n - 1) % n
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		g.sysMenuPick(g.sysRow)
	}
}

// sysMenuPick 執行目前選單的第 i 列。
func (g *Game) sysMenuPick(i int) {
	switch g.win {
	case winScenario:
		g.loadScenario(i + 1)
	case winStyle:
		g.loadStyle(styleOrder[i].key)
	case winPower:
		g.tool = powerTools[i].tool
		g.win = winNone
	case winSpeed:
		// 第 19 段由快到慢：最快 4、快速 3、普通 2、慢速 1、暫停 0。
		// 規則層只有四段（0–3），「最快」與「快速」是同一段——
		// 已知的未解，記在 docs/spec/controls.md。
		g.setSpeed(min(4-i, 3))
		g.win = winNone
	case winSystem:
		switch sysItems[i].kind {
		case "about":
			g.win = winAbout
		case "print":
			g.printMap()
			g.win = winNone
		case "saveas":
			g.openSaveAs()
		case "style":
			g.openSubMenu(winStyle)
		case "scenario":
			g.openSubMenu(winScenario)
		case "new":
			g.newCity()
		case "load":
			g.load()
		case "save":
			g.save()
			g.win = winNone
		case "quit":
			g.quit = true
		}
	}
}

func (g *Game) openSubMenu(w window) {
	g.win = w
	g.sysRow = 0
}

// loadScenario 換一個悲情城市。鏡頭重新置中，並顯示劇本簡介。
func (g *Game) loadScenario(n int) {
	if g.dataDir == "" {
		g.setMessage("沒有原版資料目錄，換不了劇本")
		return
	}
	w, err := game.LoadScenario(g.dataDir, n)
	if err != nil {
		g.setMessage("載入失敗：" + err.Error())
		return
	}
	g.swapWorld(w)
	g.ShowScenarioBrief()
}

// loadStyle 換圖形集。**圖塊與文字一起換**——古代亞洲的發電廠叫「水井」，
// 那是原版的設計，只換圖不換字會是半套。
func (g *Game) loadStyle(key string) {
	if g.dataDir == "" {
		g.setMessage("沒有原版資料目錄，換不了圖形集")
		return
	}
	ts, err := LoadTileSet(g.dataDir, key)
	if err != nil {
		g.setMessage("圖形集載入失敗：" + err.Error())
		return
	}
	txt, err := i18n.Load(key)
	if err != nil {
		g.setMessage("文字載入失敗：" + err.Error())
		return
	}
	g.tiles, g.txt, g.style = ts, txt, key
	g.win = winNone
	g.setMessage(trimMenu(g.txt.S(i18n.SecSysMenu, 3)))
}

// newCity 產生一張新地圖。原版的「重新建造一新城市」還會問資金等級與
// 難度，remake 目前用預設值（$20,000、簡單）——差異記在 controls.md。
func (g *Game) newCity() {
	s := sim.RandomSeed()
	w := sim.NewWorld(s)
	w.GenerateMap(s, sim.DefaultTerrainParams())
	w.DoSimInit()
	g.swapWorld(w)
}

// load 讀存檔。原版的 Ctrl-L 會開檔名對話框，remake 讀的是 -save 指的
// 那一個檔——對「存了檔想接著玩」這件事夠用，而且不必做檔案瀏覽器。
func (g *Game) load() {
	p := g.savePath
	if p == "" {
		p = "city.cty"
	}
	w, err := game.LoadCity(p)
	if err != nil {
		g.setMessage("讀檔失敗：" + err.Error())
		return
	}
	g.swapWorld(w)
	g.setMessage("已讀取：" + p)
}

// swapWorld 換掉整個世界。鏡頭、視窗、選單游標都要跟著重設——
// 不重設的話會停在上一座城市的座標上，看起來像「讀檔讀到空地」。
func (g *Game) swapWorld(w *sim.World) {
	g.world = w
	g.win = winNone
	g.sysRow = 0
	g.picture = ""
	g.centerCamera()
}
