package ui

// 地形編輯器的整個畫面，照原版 `TERRAIN.EXE`（MAXIS SimCity Terrain
// Editor 1.0，軟體世界 1990 年那片磁片）重製。
//
// 原版是**獨立程式**：整個 640×350 都是它的——最上面一條選單列
// （SYSTEM／TERRAIN／PARAMETERS）、左邊編輯視窗加直立的六格工具盤、
// 右邊 City Map 全市地圖視窗、編輯視窗底下一條狀態列。
// 它用的是**與遊戲本體同一套視窗系統**：編輯視窗的外框、標題列、
// 地圖區、工具帶的座標與遊戲一模一樣（docs/spec/ui-layout.md §二），
// 差別只在工具盤換成六個文字按鈕、資金帶是空的、地圖視窗沒有圖層圖示。
//
// 拆解過程與每一條的位址在 docs/re/20-terrain-editor.md，
// 版面與行為規格在 docs/spec/terrain-editor.md（READY）。
//
// remake 的兩處對應（都標明不是原版行為）：
//   - 原版的「Exit」是離開程式；這裡是離開編輯器回到遊戲。
//   - 原版的「Print」印紙本地圖；這裡沿用遊戲那一條，存成 PNG。
//
// 編輯器改的是**自己那一張地圖**，不是玩家正在玩的城市。出口照原版：
// 存成城市檔，回遊戲之後自己讀那個檔（使用者定案 2026-09-03）。

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 工具盤的版面，單位是原版像素，量自 `workplace/dosbox/tep-00-ui.png`。
//
//	外框     x 8–63、y 55–210（白 2 像素、內一圈黑）
//	按鈕內容 x 11–62，寬 52
//	第 i 個  y ＝ 59 ＋ 25i，高 24；按鈕之間一條深灰 (85,85,85) 分隔
//	選取     內縮的黃色 2 像素框（實測 DIRT 選取時 y 59–60 與 80–81 是黃）
const (
	tePalX, tePalY       = 8, 55
	tePalW, tePalH       = 56, 156
	tePalBtnX, tePalBtnW = 11, 52
	teBtnY0              = 59
	teBtnPitch           = 25
	teBtnH               = 24
)

// teTools 是工具盤由上而下的六格。編號就是原版 `byte_595E0` 的值，
// 也是 `sub_22636` 那支分派函式看的值：1–4 是畫筆，5 是油漆桶開關，
// 6 是復原動作。
var teTools = [6]struct {
	id   int
	key  string     // ui.tsv 的鍵
	tile int        // 按鈕底鋪哪一張地圖圖塊；-1 代表自己畫
	ink  color.RGBA // 字的顏色，量自原版
}{
	{1, "te_tool_dirt", sim.DIRT, color.RGBA{0xff, 0xff, 0x00, 0xff}},
	{2, "te_tool_trees", sim.WOODS, color.RGBA{0xff, 0xff, 0xff, 0xff}},
	{3, "te_tool_river", sim.RIVER, color.RGBA{0x00, 0xaa, 0xff, 0xff}},
	{4, "te_tool_channel", sim.CHANNEL, color.RGBA{0x00, 0xaa, 0xff, 0xff}},
	{5, "te_tool_fill", -1, color.RGBA{0x00, 0x00, 0x00, 0xff}},
	{6, "te_tool_undo", -1, color.RGBA{0x00, 0x00, 0x00, 0xff}},
}

// 工具盤的配色，量自原版的彩色模式。
var (
	colTeSep    = color.RGBA{0x55, 0x55, 0x55, 0xff} // 按鈕之間的深灰分隔
	colTeSel    = color.RGBA{0xff, 0xff, 0x00, 0xff} // 選取的黃框
	colTeFill   = color.RGBA{0xaa, 0xaa, 0xaa, 0xff} // 油漆桶按鈕的灰底
	colTeFillIn = color.RGBA{0x00, 0x00, 0xff, 0xff} // 上面那些藍斜線
	colTeUndo   = color.RGBA{0xff, 0x00, 0x00, 0xff} // 復原按鈕的紅底
)

// 單色模式的那一套，量自原版跑 `V`（單色 VGA）的地形編輯器
// （`workplace/dosbox/te-mono-00-ui.png`，TERRAIN.CFG 第 0 個位元組
// 就是螢幕模式，與 SIMCITY.CFG 同一組代碼）：
//
//	DIRT／TREES／RIVER／CHANNEL  白字（黑外框），底是地圖圖塊本身
//	FILL                        淺網點底、白字
//	UNDO                        **白底黑字**
//	選取框                      白
//	按鈕之間的分隔               黑
var (
	monoTeSep    = color.RGBA{0x00, 0x00, 0x00, 0xff}
	monoTeSel    = color.RGBA{0xff, 0xff, 0xff, 0xff}
	monoTeFill   = color.RGBA{0xff, 0xff, 0xff, 0xff}
	monoTeFillIn = color.RGBA{0x00, 0x00, 0x00, 0xff}
	monoTeUndo   = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// teColors 回傳目前顯示模式該用的工具盤配色。
func teColors() (sep, sel, fillBG, fillIn, undo color.RGBA) {
	if chromeSelDither { // 單色（見 chrome.go）
		return monoTeSep, monoTeSel, monoTeFill, monoTeFillIn, monoTeUndo
	}
	return colTeSep, colTeSel, colTeFill, colTeFillIn, colTeUndo
}

// teInk 回傳第 i 個工具的字色。單色模式只有黑白：四個畫筆與油漆桶是
// 白字（畫在深色的圖塊底上），復原是黑字（它的底是白的）。
func teInk(i int) color.RGBA {
	if !chromeSelDither {
		return teTools[i].ink
	}
	if teTools[i].id == 6 { // UNDO
		return color.RGBA{0x00, 0x00, 0x00, 0xff}
	}
	return color.RGBA{0xff, 0xff, 0xff, 0xff}
}

// teItem 是選單裡的一列。key 是空字串代表分隔線。
type teItem struct {
	key string
	act string
}

// teMenus 是三個下拉選單。項目順序、分隔線的位置與快速鍵提示全部照原版
// （字串表在 `dseg:0x1950`／`0x1980`／`0x19CC`，見 docs/re/20 §十四）。
//
// ⚠ **命令碼就是列號**：原版把 `(選單編號 << 4) | 列號` 直接當成
// `sub_10A0A` 的參數，所以分隔線佔掉的列號（SYSTEM 的 1／3／5／9）
// 正好就是 IDA 標成 default case 的那幾個。順序不能重排。
var teMenus = [3]struct {
	title string
	items []teItem
}{
	{"te_sys", []teItem{
		{"te_about", "about"},
		{"", ""},
		{"te_print", "print"},
		{"", ""},
		{"te_new", "new"},
		{"", ""},
		{"te_load", "load"},
		{"te_saveas", "saveas"},
		{"te_save", "save"},
		{"", ""},
		{"te_exit", "exit"},
	}},
	{"te_terrain", []teItem{
		{"te_clear", "clear"},
		{"te_clear_art", "clearart"},
		{"", ""},
		{"te_random", "random"},
		{"", ""},
		{"te_smooth_trees", "smoothtrees"},
		{"te_smooth_rivers", "smoothrivers"},
		{"te_smooth_all", "smoothall"},
		{"", ""},
		{"te_island", "island"},
	}},
	{"te_params", []teItem{
		{"te_name_level", "namelevel"},
		{"te_year", "year"},
		{"", ""},
		{"te_sound", "sound"},
	}},
}

// teMenuCenterX 是三個標題的中心，量自原版（`tep-00-ui.png` 的選單列，
// 深藍筆畫的外接範圍：136–182、321–374、512–590）。
var teMenuCenterX = [3]int{159, 347, 551}

// terrainScreen 是一個開著的地形編輯器。
type terrainScreen struct {
	ed    *sim.Editor
	world *sim.World

	// saved 是進編輯器之前玩家正在玩的那張地圖。編輯器改的是自己那一張，
	// 離開時把這一份放回去——原版是獨立程式，編輯器動不到遊戲裡的城市。
	saved *sim.World
	// savedPath 是進編輯器之前的存檔路徑。
	//
	// ⚠ **不能與遊戲共用**：共用的話在編輯器裡按「儲存城市」會直接蓋掉
	// 玩家正在玩的那個存檔檔案，而畫面上只寫「已存檔」，看不出來出事了。
	// 原版的編輯器是獨立程式，本來就有自己的檔名緩衝區。
	savedPath string
	// fromTitle 記得玩家是從招牌那個按鈕進來的：離開時回招牌，
	// 不是掉進一座他沒選過的城市。
	fromTitle bool

	openMenu int // 0 ＝ 沒拉開，1–3
	menuRow  int
	dragging bool
	sound    bool

	// 進度訊息。原版在造地形與平滑時開一個小視窗寫
	// `Now terraforming`／`Smoothing...`；remake 的動作是瞬間完成的，
	// 所以照原版畫出來之後留幾格畫格再收掉。
	progress     string
	progressCols int
	progressLeft int
	yearInput    string
	yearOpen     bool
	confirmKey   string
	confirmAct   string
	aboutOpen    bool
}

// openTerrainScreen 進入地形編輯器。
//
// 原版一開機是一張全空地的地圖、DIRT 選著、年份 1900。
func (g *Game) openTerrainScreen() {
	w := sim.NewWorld(sim.RandomSeed())
	w.ClearMapTiles()
	// 原版的編輯器一開機是 1900 年 1 月（實測標題列 `Jan 1900`），
	// 不是遊戲那個 `CityTime = 50`（sim.c:183，會顯示 1901）。
	w.CityTime = 0
	// 市名不繼承玩家正在玩的那座城市：原版的編輯器是獨立程式，
	// 標題列也不印城市名。繼承的話「以……檔名儲存」會預設成同一個檔名，
	// 玩家一路按下去就把自己的存檔蓋掉了。留 NewWorld 的 HERESVILLE。
	ts := &terrainScreen{
		world: w,
		saved: g.world,
		sound: !g.soundOff,
	}
	ts.ed = sim.NewEditor(w)
	ts.savedPath = g.savePath
	g.savePath = "" // 第一次存檔一律走「以……檔名儲存」，不覆蓋遊戲的存檔
	g.world = w
	g.terrain = ts
	g.camX, g.camY = 0, 0
	g.win = winNone
	g.openMenu = 0
	// 原版的編輯器那條帶是空的。不清掉的話開機那句「繁體中文」會跟進來。
	g.message = ""
}

// closeTerrainScreen 離開編輯器，把玩家原本的城市放回去。
func (g *Game) closeTerrainScreen() {
	if g.terrain == nil {
		return
	}
	back := g.terrain.fromTitle
	g.world = g.terrain.saved
	g.savePath = g.terrain.savedPath
	g.terrain = nil
	g.win = winNone
	g.mini = nil
	g.LookAt(sim.WorldX/2, sim.WorldY/2)
	if back && g.titlePic != nil {
		g.screen = scrTitle
	}
}

// teCurTool 回傳目前畫筆。
func (ts *terrainScreen) tool() sim.EditorTool { return ts.ed.Tool }

// ---------------------------------------------------------------- 更新

// updateTerrain 是編輯器自己的一格。回傳 true 代表這一格歸編輯器，
// 遊戲的鍵盤與滑鼠處理不要再跑。
func (g *Game) updateTerrain() bool {
	ts := g.terrain
	if ts == nil {
		return false
	}
	// 讀取城市會換掉 g.world（sysmenu 那條路徑），編輯器要跟上。
	if g.world != ts.world {
		ts.world = g.world
		ts.ed = sim.NewEditor(g.world)
	}
	if ts.progressLeft > 0 {
		ts.progressLeft--
	}
	// 存讀檔那幾個視窗與「市名與難度」對話框開著時，鍵盤與滑鼠歸它們
	// （沿用遊戲本體的實作）。少了 newCityDlg 這一條，市名欄打不了字、
	// 「確定」也按不下去——而畫面看起來完全正常。
	if g.win != winNone || g.newCityDlg != nil {
		return false
	}
	g.terrainKeys()
	g.terrainMouse()
	return true
}

// terrainKeys 處理編輯器的鍵盤。
func (g *Game) terrainKeys() {
	ts := g.terrain
	switch {
	case ts.aboutOpen:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
			inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			ts.aboutOpen = false
		}
		return
	case ts.confirmKey != "":
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsKeyJustPressed(ebiten.KeyY) {
			act := ts.confirmAct
			ts.confirmKey, ts.confirmAct = "", ""
			g.terrainConfirmed(act)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
			inpututil.IsKeyJustPressed(ebiten.KeyN) {
			ts.confirmKey, ts.confirmAct = "", ""
		}
		return
	case ts.yearOpen:
		g.terrainYearKeys()
		return
	case g.terrainDlg != nil:
		g.handleTerrainKeys() // 參數對話框（terrain_editor.go）
		return
	}

	if ts.openMenu != 0 {
		g.terrainMenuKeys()
		return
	}

	// 快速鍵。原版把它們印在選單項目上：Ctrl-C／T／A／I／L／S／X。
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		for key, act := range map[ebiten.Key]string{
			ebiten.KeyC: "clear",
			ebiten.KeyT: "random",
			ebiten.KeyA: "smoothall",
			ebiten.KeyI: "island",
			ebiten.KeyL: "load",
			ebiten.KeyS: "save",
			ebiten.KeyX: "exit",
		} {
			if inpututil.IsKeyJustPressed(key) {
				g.terrainAct(act)
				return
			}
		}
		return
	}

	// 捲動地圖。原版用方向鍵與 Home／End／PgUp／PgDn 八個方向
	// （`sub_10F48`＋0x10FA0 起的八次掃描碼比對）。
	dx, dy := 0, 0
	for key, d := range map[ebiten.Key][2]int{
		ebiten.KeyLeft:     {-1, 0},
		ebiten.KeyRight:    {1, 0},
		ebiten.KeyUp:       {0, -1},
		ebiten.KeyDown:     {0, 1},
		ebiten.KeyHome:     {-1, -1},
		ebiten.KeyPageUp:   {1, -1},
		ebiten.KeyEnd:      {-1, 1},
		ebiten.KeyPageDown: {1, 1},
	} {
		if inpututil.IsKeyJustPressed(key) {
			dx += d[0]
			dy += d[1]
		}
	}
	if dx != 0 || dy != 0 {
		g.camX = clamp(g.camX+dx, 0, sim.WorldX-g.tilesAcross())
		g.camY = clamp(g.camY+dy, 0, sim.WorldY-g.tilesDown())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.terrainAct("exit")
	}
}

// terrainMenuKeys 處理拉開的選單的鍵盤。
func (g *Game) terrainMenuKeys() {
	ts := g.terrain
	items := teMenus[ts.openMenu-1].items
	step := func(d int) {
		for i := 0; i < len(items); i++ {
			ts.menuRow = (ts.menuRow + d + len(items)) % len(items)
			if items[ts.menuRow].key != "" {
				return
			}
		}
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyDown):
		step(1)
	case inpututil.IsKeyJustPressed(ebiten.KeyUp):
		step(-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyLeft):
		ts.openMenu = (ts.openMenu+1)%3 + 1
		ts.menuRow = 0
	case inpututil.IsKeyJustPressed(ebiten.KeyRight):
		ts.openMenu = ts.openMenu%3 + 1
		ts.menuRow = 0
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		ts.openMenu = 0
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter),
		inpututil.IsKeyJustPressed(ebiten.KeySpace):
		act := items[ts.menuRow].act
		ts.openMenu = 0
		g.terrainAct(act)
	}
}

// terrainYearKeys 處理年份輸入。原版只收四位數字，按 Enter 送出。
func (g *Game) terrainYearKeys() {
	ts := g.terrain
	for _, k := range []ebiten.Key{
		ebiten.Key0, ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4,
		ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9,
	} {
		if inpututil.IsKeyJustPressed(k) && len(ts.yearInput) < 4 {
			ts.yearInput += string(rune('0' + int(k-ebiten.Key0)))
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && ts.yearInput != "" {
		ts.yearInput = ts.yearInput[:len(ts.yearInput)-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ts.yearOpen = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		g.terrainSetYear()
	}
}

// terrainSetYear 收下年份。原版 `sub_111E4`＋0x11365：
// 只有**恰好四個字**才算數，換算是 `CityTime = (年 − 1900) × 48`，
// 而且結果必須大於零（也就是年份要大於 1900），否則整個丟掉。
// 不合格時發第 7 號音效（工具失敗）。
func (g *Game) terrainSetYear() {
	ts := g.terrain
	defer func() { ts.yearOpen = false }()
	if len(ts.yearInput) != 4 {
		g.teFailSound()
		g.setMessage(g.txt.UI("te_year_bad"))
		return
	}
	y, err := strconv.Atoi(ts.yearInput)
	if err != nil {
		g.teFailSound()
		return
	}
	t := (y - 1900) * 48
	if t <= 0 {
		g.teFailSound()
		g.setMessage(g.txt.UI("te_year_bad"))
		return
	}
	ts.world.CityTime = t
}

// teFailSound 發原版的「工具失敗」音效（第 7 段）。
func (g *Game) teFailSound() {
	if g.terrain != nil && g.terrain.sound && !g.soundOff {
		g.snd.play(sim.SoundToolFail)
	}
}

// terrainMouse 處理編輯器的滑鼠。
func (g *Game) terrainMouse() {
	ts := g.terrain
	mx, my := ebiten.CursorPosition()
	x, y := mx/UIScale, my/UIScale
	just := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	held := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	if g.terrainDlg != nil {
		g.handleTerrainMouse(mx, my)
		return
	}
	if ts.aboutOpen || ts.confirmKey != "" || ts.yearOpen {
		if ts.confirmKey != "" && just {
			if i := g.teConfirmHit(x, y); i == 0 {
				act := ts.confirmAct
				ts.confirmKey, ts.confirmAct = "", ""
				g.terrainConfirmed(act)
			} else if i == 1 {
				ts.confirmKey, ts.confirmAct = "", ""
			}
		}
		if ts.aboutOpen && just {
			ts.aboutOpen = false
		}
		return
	}

	// 選單列：原版是按住式（按住標題拉開、滑到項目上放開才選中），
	// 這裡跟遊戲本體一樣兩種都吃。
	if just && y < menuBarH {
		if m := teMenuHit(x); m >= 0 {
			if ts.openMenu == m+1 {
				ts.openMenu = 0
			} else {
				ts.openMenu, ts.menuRow = m+1, teFirstRow(m)
			}
		}
		return
	}
	if ts.openMenu != 0 {
		if row := g.teMenuRowAt(x, y); row >= 0 {
			ts.menuRow = row
		}
		if just || inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			row := g.teMenuRowAt(x, y)
			if row >= 0 && teMenus[ts.openMenu-1].items[row].key != "" {
				act := teMenus[ts.openMenu-1].items[row].act
				ts.openMenu = 0
				g.terrainAct(act)
			} else if just {
				ts.openMenu = 0
			}
		}
		return
	}

	// 工具盤。
	if just {
		if i := tePaletteHit(x, y); i >= 0 {
			g.terrainPickTool(teTools[i].id)
			return
		}
	}

	// 地圖區：按著就一直畫（原版 `sub_1F0C0` 就是一個拖曳迴圈）。
	if !g.inEditView(mx, my) {
		ts.dragging = false
		return
	}
	if !held {
		ts.dragging = false
		return
	}
	if !just && !ts.dragging {
		return
	}
	ts.dragging = true
	px := g.tileSize() * tileScale
	tx := g.camX + (mx-editViewX*UIScale)/px
	ty := g.camY + (my-editViewY*UIScale)/px
	if ts.ed.Fill {
		if just && ts.ed.FillAt(ts.tool(), tx, ty) {
			ts.ed.Fill = false // 原版倒完一次就熄
			g.mini = nil
		}
		return
	}
	if ts.ed.Paint(ts.tool(), tx, ty) {
		g.mini = nil
	}
}

// terrainPickTool 是原版的 `sub_22636`：6 是復原（動作）、5 是油漆桶
// （開關）、1–4 才是換畫筆。
func (g *Game) terrainPickTool(id int) {
	ts := g.terrain
	switch {
	case id == 6:
		if !ts.ed.Undo() {
			g.teFailSound()
			return
		}
		g.mini = nil
	case id == 5:
		ts.ed.Fill = !ts.ed.Fill
	case id >= 1 && id <= 4:
		ts.ed.Tool = sim.EditorTool(id)
	}
}

// terrainAct 執行一條選單命令。
func (g *Game) terrainAct(act string) {
	ts := g.terrain
	switch act {
	case "about":
		ts.aboutOpen = true
	case "print":
		g.printMap()
	case "new":
		ts.confirmKey, ts.confirmAct = "te_confirm_new", "new!"
	case "load":
		g.load()
	case "saveas":
		g.openSaveAs()
	case "save":
		// 還沒指定過檔名就先問——不然會落到工作目錄的 city.cty，
		// 玩家不知道東西存到哪去了。原版的 `sub_1FE44(0)` 也是問。
		if g.savePath == "" {
			g.openSaveAs()
		} else {
			g.save()
		}
	case "exit":
		ts.confirmKey, ts.confirmAct = "te_confirm_exit", "exit!"
	case "clear":
		ts.ed.ClearMap()
		g.mini = nil
	case "clearart":
		ts.ed.ClearUnnatural()
		g.mini = nil
	case "random":
		g.openTerrainEditor() // 參數對話框，按「開始」之後回到這裡
	case "smoothtrees":
		ts.ed.SmoothTreesOnly()
		g.teProgress("te_smoothing", 16)
	case "smoothrivers":
		ts.ed.SmoothRiversOnly()
		g.teProgress("te_smoothing", 16)
	case "smoothall":
		ts.ed.SmoothEverything()
		g.teProgress("te_smoothing", 16)
	case "island":
		ts.ed.Island = !ts.ed.Island
	case "namelevel":
		g.openNewCityNameOnly()
	case "year":
		ts.yearOpen = true
		ts.yearInput = fmt.Sprintf("%d", 1900+ts.world.CityTime/48)
	case "sound":
		ts.sound = !ts.sound
	}
}

// terrainConfirmed 是確認框按下「確定」之後真正做的事。
func (g *Game) terrainConfirmed(act string) {
	switch act {
	case "new!":
		ts := g.terrain
		ts.ed.ClearMap()
		ts.world.CityTime = 0
		g.mini = nil
	case "exit!":
		g.closeTerrainScreen()
	}
}

// teProgress 讓進度訊息顯示幾格畫格。原版是在算的時候一直開著，
// remake 算完只要一瞬間，所以固定留幾格才看得到。
func (g *Game) teProgress(key string, frames int) {
	ts := g.terrain
	ts.progress = g.txt.UI(key)
	ts.progressCols = 16
	if key == "te_terraforming" {
		ts.progressCols = 20
	}
	ts.progressLeft = frames
	g.mini = nil
}
