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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/assets"

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
	g.loadEnglish()
}

func (g *Game) sysMenuLen() int {
	switch g.win {
	case winSystem:
		return len(sysItems)
	case winLangSel:
		return len(i18n.Langs)
	case winMusic:
		return len(g.MusicTracks()) + 1
	case winScenario:
		return 8
	case winStyle:
		return len(styleOrder)
	case winSpeed:
		return 5
	case winPower:
		return len(powerTools)
	case winLoad:
		return len(g.loadFiles)
	case winSaveFmt:
		return 2
	}
	return 0
}

// sysMenuLabel 回傳目前選單第 i 列要顯示的字。
func (g *Game) sysMenuLabel(i int) string {
	switch g.win {
	case winSystem:
		// 負的訊息編號是 remake 自己加的項目，訊息檔裡沒有對應的字。
		switch sysItems[i].msg {
		case -1:
			return g.txt.UI("lang_title")
		case -2:
			return g.txt.UI("music_title")
		}
		return trimMenu(g.txt.S(i18n.SecSysMenu, sysItems[i].msg))
	case winLangSel:
		return i18n.LangName[i18n.Langs[i]]
	case winMusic:
		if i == 0 {
			return fmt.Sprintf(g.txt.UI("music_toggle"), g.onOff(g.musicOn()))
		}
		return g.MusicTracks()[i-1]
	case winSaveFmt:
		if i == 0 {
			return g.txt.UI("savefmt_dos")
		}
		return g.txt.UI("savefmt_bare")
	case winScenario:
		return game.ScenarioNameZH(i + 1)
	case winStyle:
		return styleOrder[i].name
	case winSpeed:
		return trimMenu(g.txt.S(i18n.SecSpeed, i))
	case winPower:
		return trimMenu(g.txt.S(i18n.SecPowerSub, powerTools[i].msg))
	case winLoad:
		return filepath.Base(g.loadFiles[i])
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
	case winLoad:
		g.loadFile(g.loadFiles[i])
	case winLangSel:
		g.setLang(i18n.Langs[i])
	case winSaveFmt:
		f := game.SaveWithHeader
		if i == 1 {
			f = game.SaveBareBody
		}
		g.setSaveFormat(f)
	case winMusic:
		if i == 0 {
			g.toggleMusic()
		} else {
			// 選單點歌與 `[`／`]` 同樣進手動模式；一般情境不搶歌，災難仍可插播。
			if g.music != nil {
				g.music.cur = i - 1
				g.stepTrack(0)
			}
		}
		g.win = winNone
	case winScenario:
		g.loadScenario(i + 1)
	case winStyle:
		g.loadStyle(styleOrder[i].key)
	case winPower:
		g.tool = powerTools[i].tool
		g.win = winNone
	case winSpeed:
		// 第 19 段由快到慢：最快 4、快速 3、普通 2、慢速 1、暫停 0，
		// 所以選單的第 i 列對到速度 4−i。五段都接得起來，見 speedMsgIdx。
		g.setSpeed(4 - i)
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
		case "lang":
			g.openSubMenu(winLangSel)
		case "music":
			g.openSubMenu(winMusic)
		case "quit":
			g.quit = true
		}
	}
}

func (g *Game) openSubMenu(w window) {
	g.win = w
	g.sysRow = 0
}

func (g *Game) openLangSettings() {
	g.win = winLangSel
	g.waitEnterRelease = true
	g.sysRow = 0
	for i, lang := range i18n.Langs {
		if lang == g.lang {
			g.sysRow = i
			break
		}
	}
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
	txt, err := i18n.LoadLang(key, g.lang)
	if err != nil {
		g.setMessage("文字載入失敗：" + err.Error())
		return
	}
	g.tiles, g.txt, g.style = ts, txt, key
	// 換圖形集要重載英文原文（每個資料片一份 `*_MSG.PTF`）與語言設定，
	// 不然換完之後英文那一欄留著上一個資料片的字。
	g.txt.SetLang(g.lang)
	g.loadEnglish()
	g.win = winNone
	g.setMessage(trimMenu(g.txt.S(i18n.SecSysMenu, 3)))
}

// newCity 開「建造新城市」對話框（市名欄 ＋ 技術等級），實作在 newcity.go。
// 原版是從標題畫面進這個對話框；remake 沒有標題畫面，掛在系統選單同一項。
func (g *Game) newCity() {
	g.openNewCity()
}

// load 是「讀取舊有檔案」（`Ctrl-L`）。原版開的是檔名對話框，這裡列出
// 存檔目錄裡的城市檔讓玩家挑——DOS 版的存檔與 `.cty` 都吃得下
// （`internal/game/save.go` 的 normalizeCityBytes 認三種長度）。
//
// 目錄裡只有一個檔的時候不必挑，直接讀；一個都沒有就說一聲。
func (g *Game) load() {
	g.loadFiles = g.cityFilesInSaveDir()
	switch len(g.loadFiles) {
	case 0:
		g.setMessage(g.txt.UI("no_city_files"))
		g.win = winNone
	case 1:
		g.loadFile(g.loadFiles[0])
	default:
		g.openSubMenu(winLoad)
	}
}

// cityFilesInSaveDir 列出存檔目錄裡的城市檔，照檔名排序。
// SetBundledCityDir 注入發行包隨附的地圖目錄。空字串代表沒有。
func (g *Game) SetBundledCityDir(dir string) { g.bundledCityDir = dir }

// isBundledCity 判斷這個檔是不是隨附地圖。隨附的那些在 AppImage 裡是唯讀的，
// 讀了之後不能把存檔位置指過去，否則 Ctrl-S 會寫進暫存掛載點。
func (g *Game) isBundledCity(p string) bool {
	return g.bundledCityDir != "" && filepath.Dir(p) == filepath.Clean(g.bundledCityDir)
}

func ctyFilesIn(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".cty") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

// cityFilesInSaveDir 列出「讀取舊有檔案」要顯示的城市檔：玩家自己的存檔，
// 再接上發行包隨附的地圖。
//
// ⚠ **隨附的地圖一定要列進來。** AppImage 把整包掛在暫存目錄，玩家的工作
// 目錄是別的地方；只掃存檔目錄的話，包裡明明有五張（完整版十四張）地圖，
// 玩家在遊戲裡一張都看不到。
//
// 同名時存檔優先——玩家自己存的那份才是他要的。
func (g *Game) cityFilesInSaveDir() []string {
	dir := "."
	if g.savePath != "" {
		dir = filepath.Dir(g.savePath)
	}
	out := ctyFilesIn(dir)
	if g.bundledCityDir == "" || filepath.Clean(g.bundledCityDir) == filepath.Clean(dir) {
		return out
	}
	seen := map[string]bool{}
	for _, p := range out {
		seen[strings.ToLower(filepath.Base(p))] = true
	}
	for _, p := range ctyFilesIn(g.bundledCityDir) {
		if !seen[strings.ToLower(filepath.Base(p))] {
			out = append(out, p)
		}
	}
	return out
}

// loadFile 讀一個城市檔並換掉整個世界。
func (g *Game) loadFile(p string) {
	w, err := game.LoadCity(p)
	if err != nil {
		g.setMessage("讀檔失敗：" + err.Error())
		return
	}
	g.swapWorld(w)
	// 隨附的地圖是唯讀的（AppImage 掛在暫存目錄），存檔位置不能指過去。
	if !g.isBundledCity(p) {
		g.savePath = p
	}
	g.setMessage(fmt.Sprintf(g.txt.UI("loaded"), p))
}

// swapWorld 換掉整個世界。鏡頭、視窗、選單游標都要跟著重設。
// 鏡頭回到原版可重播證據確認的 (0,0)，不沿用上一座城市的位置。
func (g *Game) swapWorld(w *sim.World) {
	g.world = w
	g.win = winNone
	g.sysRow = 0
	g.picture = ""
	g.pictureScenario = false
	// 速度是**存進城市檔的**（`MiscHis[57]`，s_fileio.c:263），所以讀檔之後
	// 呈現層的五段要跟著回來。存檔只記 0–3，第五段「最快」是執行期的
	// `sim_skips`——原版讀檔時也是 `setSkips(0)`，所以「最快」不會被記住。
	g.speedLevel = clamp(w.SimSpeed, 0, 4)
	g.resetCamera()
}

// setSaveFormat 換存檔版面，並把選擇寫回設定檔。
//
// 兩種都是原版認得的格式，差別在檔頭：DOS 版面存得住城市名但餵不進
// Micropolis，裸檔身反之。取捨見 internal/game/save.go 的 SaveFormat。
func (g *Game) setSaveFormat(f game.SaveFormat) {
	g.saveFmt = f
	g.win = winNone
	if g.savePrefs != nil {
		if err := g.savePrefs(g.lang, f.String()); err != nil {
			g.setMessage(fmt.Sprintf(g.txt.UI("settings_save_failed"), err))
			return
		}
	}
	if f == game.SaveBareBody {
		g.setMessage(g.txt.UI("savefmt_bare"))
	} else {
		g.setMessage(g.txt.UI("savefmt_dos"))
	}
}

// SetSaveFormat 由啟動層注入玩家上次的選擇（或命令列覆蓋）。
func (g *Game) SetSaveFormat(f game.SaveFormat) { g.saveFmt = f }

// SetPrefsSaver 由啟動層注入，把語言與存檔格式一起寫回設定檔。
func (g *Game) SetPrefsSaver(save func(i18n.Lang, string) error) { g.savePrefs = save }

// setLang 換遊戲語言。文字表四種語言都在同一份裡，換語言不必重讀檔案，
// 也不必重載圖形集。
func (g *Game) setLang(l i18n.Lang) {
	g.lang = l
	g.txt.SetLang(l)
	g.win = winNone
	if g.savePrefs != nil {
		if err := g.savePrefs(l, g.saveFmt.String()); err != nil {
			g.setMessage(fmt.Sprintf(g.txt.UI("settings_save_failed"), err))
			return
		}
	} else if g.saveLang != nil {
		if err := g.saveLang(l); err != nil {
			g.setMessage(fmt.Sprintf(g.txt.UI("settings_save_failed"), err))
			return
		}
	}
	g.setMessage(i18n.LangName[l])
}

// loadEnglish 把玩家自己那份 `.PTF` 的**英文原文**餵進文字表。
//
// ⚠ 英文不在版控裡，也不會進發行包（CLAUDE.md §8）——它是原版的文字，
// 屬於原權利人。要看英文就得自備原版，這與圖形、音效是同一條界線。
//
// 檔名規則同音效：基本圖形集是 `DATA/MESSAGE.PTF`，六個資料片是
// `DATA/<風格>_MSG.PTF`（實測路徑 `C:\DATA\west_msg.ptf`，
// docs/re/16-dos-oracle.md §八）。
func (g *Game) loadEnglish() {
	if g.dataDir == "" || g.txt == nil {
		return
	}
	name := "MESSAGE.PTF"
	if g.style != "" && g.style != StyleBase {
		name = strings.ToUpper(g.style) + "_MSG.PTF"
	}
	p, err := findFile(filepath.Join(g.dataDir, "DATA"), name)
	if err != nil {
		return
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return
	}
	secs, err := assets.LoadPTF(raw)
	if err != nil {
		return
	}
	out := make([][]string, len(secs))
	for i, s := range secs {
		out[i] = s.Strings
	}
	g.txt.SetEnglish(out)
}

// SetLang 給命令列用：一開始就指定語言。
func (g *Game) SetLang(l i18n.Lang) { g.setLang(l); g.win = winNone }

// SetLangSaver 設定玩家從語言視窗選取後的持久化函式。
// 啟動時套用語言應先呼叫 SetLang，再注入 saver，避免 `-lang` 覆寫玩家偏好。
func (g *Game) SetLangSaver(save func(i18n.Lang) error) { g.saveLang = save }
