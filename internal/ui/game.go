package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 內部畫布 ＝ 原版畫面 × UIScale。
//
// 版面照原版 DOS 1.10 的 640×350 重現（docs/spec/ui-layout.md），
// 放大三倍成 1920×1050。三倍不是隨便挑的：原版一個字元格 8×8，×3 剛好
// 24×24，而本專案的中文點陣字就是 24×24——**一個中文字剛好佔一個原版
// 字元格**，英數半形佔半格。所以中文比它取代的英文窄，版面有餘裕。
// 全部整數倍放大，不會出現寬窄不一的像素（rulebook/81）。
const (
	CanvasW = OrigW * UIScale // 1920
	CanvasH = OrigH * UIScale // 1050

	tileScale = UIScale // 圖塊放大倍率跟著版面

	// 編輯視窗裡地圖區的大小（螢幕像素）。原版的編輯視窗很小——
	// 右半邊被 City Form 視窗佔著，那是原版的預設配置。
	// 預設的地圖區大小（螢幕像素）。玩家調整過視窗之後以 g.editViewSize() 為準。
	viewW = editViewW * UIScale
	viewH = editViewH * UIScale
)

// 配色。視窗內容是**深字白底**，照原版——原版的資料視窗（預算、評估、
// 統計圖）都是白底或亮青底配深藍字，不是現代的深色主題。
//
// ⚠ 換過一次：舊版是深底亮字，換成原版版面之後客戶區變白，
// 亮字會整片看不見，而且畫面上「有東西」（框、標題都在），
// 看起來像資料沒算出來，不像顏色錯了。
var (
	colBG     = color.RGBA{0xaa, 0xaa, 0xaa, 0xff} // 桌面灰
	colPanel  = color.RGBA{0xff, 0xff, 0xff, 0xff}
	colLine   = color.RGBA{0x00, 0x00, 0xaa, 0xff}
	colText   = color.RGBA{0x00, 0x00, 0x00, 0xff}
	colDim    = color.RGBA{0x55, 0x55, 0x55, 0xff}
	colOn     = color.RGBA{0x00, 0x00, 0xaa, 0xff}
	colMoneyN = color.RGBA{0xaa, 0x00, 0x00, 0xff}
	colDemR   = color.RGBA{0x00, 0xaa, 0x00, 0xff}
	// 需求長條的三個顏色，量自原版：商業亮藍、住宅亮綠、工業亮黃。
	colDemBarC = color.RGBA{0x55, 0x55, 0xff, 0xff}
	colDemBarR = color.RGBA{0x55, 0xff, 0x55, 0xff}
	colDemBarI = color.RGBA{0xff, 0xff, 0x55, 0xff}
	colDemC    = color.RGBA{0x00, 0x00, 0xaa, 0xff}
	colDemI    = color.RGBA{0xaa, 0x55, 0x00, 0xff}

	// 圖片訊息目前以深色遮罩顯示文字，不能沿用一般資料視窗的黑字配色。
	// 採 EGA 亮色，讓劇本簡介與災難通知在任何底圖上都清楚可讀。
	pictureBG     = color.RGBA{0x14, 0x18, 0x22, 0xf8}
	pictureText   = color.RGBA{0xff, 0xff, 0xff, 0xff}
	pictureHint   = color.RGBA{0xaa, 0xaa, 0xaa, 0xff}
	pictureBorder = color.RGBA{0x55, 0xff, 0xff, 0xff}
)

// toolButton 是工具列上的一個按鈕。
//
// 名稱與造價**不寫在這裡**：它們來自原版訊息檔第 1 段（「推土機：$1」
// 這種形式），經由 internal/i18n 取得。這樣切換城市風格時，工具名會跟著
// 換成該風格的說法——古代亞洲的發電廠叫「水井」，那是原版的設計。
//
// msgIdx 是那一筆在第 1 段裡的索引；cost 只用來判斷買不買得起（灰掉），
// 顯示的字一律用譯文。
type toolButton struct {
	tool   sim.Tool
	msgIdx int
	cost   int
	key    ebiten.Key
	// alt 是第二個鍵。舊版用的字母保留成別名，免得既有的說明與腳本失效；
	// 沒有別名的就填成與 key 相同。
	alt ebiten.Key
}

// ⚠ 造價以**原版資料檔**為準（訊息第 1 段自己寫著），不是 Micropolis 的
// CostOf[]：體育館 $3000、海港 $5000，兩者在 Micropolis 裡剛好對調。
// 見 docs/manual-cht/p23-58-operations.md 的「與 Micropolis 不一致的數字」。
//
// ⚠ **數字鍵不是工具鍵，是速度鍵**（`0`–`4`）。原版的《模擬城市參考附表》
// 與**訊息檔第 19 段自己印著「暫停 0／慢速 1／普通 2／快速 3／最快 4」**——
// 兩份一手資料一致。先前把 1–0 拿去選工具，遊戲裡的速度副選單於是在
// 叫玩家按一組會放住宅區的鍵。工具鍵改成字母：`B`／`R`／`P`／`T`／`Q`
// 是原版本來就有的，其餘是本專案新增（見 docs/spec/controls.md）。
var toolButtons = []toolButton{
	{sim.ToolBulldozer, 0, 1, ebiten.KeyB, ebiten.KeyD},
	{sim.ToolRoad, 1, 10, ebiten.KeyR, ebiten.KeyR},
	{sim.ToolWire, 2, 5, ebiten.KeyP, ebiten.KeyW},
	{sim.ToolRail, 3, 20, ebiten.KeyT, ebiten.KeyT},
	{sim.ToolPark, 4, 10, ebiten.KeyK, ebiten.KeyK},
	{sim.ToolResidential, 5, 100, ebiten.KeyZ, ebiten.KeyZ},
	{sim.ToolCommercial, 6, 100, ebiten.KeyX, ebiten.KeyX},
	{sim.ToolIndustrial, 7, 100, ebiten.KeyV, ebiten.KeyV},
	{sim.ToolPolice, 8, 500, ebiten.KeyO, ebiten.KeyO},
	{sim.ToolFireStation, 9, 500, ebiten.KeyF, ebiten.KeyF},
	{sim.ToolStadium, 10, 3000, ebiten.KeyS, ebiten.KeyS},
	{sim.ToolNuclear, 11, 5000, ebiten.KeyJ, ebiten.KeyJ},
	{sim.ToolSeaport, 12, 5000, ebiten.KeyH, ebiten.KeyH},
	{sim.ToolAirport, 13, 10000, ebiten.KeyA, ebiten.KeyA},
	{sim.ToolCoalPower, 14, 3000, ebiten.KeyG, ebiten.KeyG},
	{sim.ToolQuery, -1, 0, ebiten.KeyQ, ebiten.KeyQ},
}

// Game 是 Ebiten 的遊戲物件。
type Game struct {
	world *sim.World
	tiles *TileSet
	font  *Font
	txt   *i18n.Catalog

	win         window
	layer       mapLayer
	budgetRow   int
	disasterRow int
	sysRow      int

	// dataDir／style 讓系統選單換得了劇本與圖形集（見 sysmenu.go）。
	dataDir string
	style   string
	// quit 由「跳出遊戲」設起來，Update 下一次回報給 Ebiten。
	quit bool

	tool     sim.Tool
	camX     int // 視野左上角的格子座標
	camY     int
	dragging bool

	message  string
	msgTimer int

	// savePath 是 Ctrl-S 存檔的目標。空的話存到工作目錄下的城市名。
	savePath string

	// snd 是音效系統。nil 代表沒開（無頭環境或裝置開不起來）。
	snd *soundSystem

	// version 顯示在「關於本遊戲」，由進入點填。
	version string
	// saveAs 是「以……檔名儲存」的輸入列，nil 代表沒開。
	saveAs *saveAsBox

	// openMenu 是拉開的下拉選單（0 ＝ 沒有，1–4 對應選單列四個標題）。
	// menuRow 是游標停在哪一列（−1 ＝ 沒有）。
	openMenu int
	menuRow  int
	// waitEnterRelease 防止用 Enter 從下拉選單開啟子視窗時，同一次按鍵又被
	// 子視窗當成「選定第一列」；必須等實體鍵放開才接受下一次 Enter。
	waitEnterRelease bool
	// openLangNext 把「SYSTEM→設定」延後到本次畫面所有輸入處理完成後提交，
	// 避免開窗用的滑鼠／Enter 又被新視窗消費。
	openLangNext bool
	// openSaveFmtNext 同理，給「SYSTEM→存檔格式」用。
	openSaveFmtNext bool

	// 功能選單的三個 remake 端開關。前三個在 sim.World 裡（會存進城市檔），
	// 這三個只影響呈現層，所以放這裡。
	soundOff    bool
	animate     bool
	fastAnimate bool

	// winPos 是玩家搬過的視窗位置（原版座標）；沒搬過就用 winRect 的預設。
	winPos map[window][2]int
	// dragWin 是正在拖的視窗，dragDX／dragDY 是按下時游標與視窗左上角的差。
	dragWin        window
	dragDX, dragDY int

	// watch／frameNo 是卡死偵測用的，見 watchdog.go。
	watch    *watchState
	frameNo  uint64
	freezeAt time.Time
	// curBuf 是工具佔地框的暫存圖，見 drawToolCursor。
	curBuf *ebiten.Image
	// editFront 記錄編輯視窗有沒有被拉到 City Form 視窗前面。
	// 原版一開始是 City Form 在前面。
	editFront bool
	// mapClosed 是 City Form 視窗被關掉（Ctrl-C）的狀態。
	//
	// ⚠ 「關閉」與「隱藏」在原版是**兩件事**，實測過
	// （workplace/dosbox/uh-71..73）：`Ctrl-C` 真的把視窗關掉，
	// `Ctrl-H` 只是把最前面的視窗**移到最後面**，視窗還在。
	// 一開始兩個鍵都做成「收起來」，看起來一樣，但按第二次就分得出來——
	// 原版按第二次 `Ctrl-H` 會換成編輯視窗被壓到後面。
	mapClosed bool
	// speedLevel 是原版的五段速度（0 暫停 … 4 最快）。
	speedLevel int

	// graphOn 是統計圖六條曲線各自開著沒有，graphYears 是 10 或 120。
	// 原版用左邊那八個圖示按鈕切換，狀態不進存檔。
	graphOn    [6]bool
	graphYears int

	// querying 是「按住查詢」的狀態，queryTX／queryTY 是被查的那一格。
	querying         bool
	queryTX, queryTY int

	// ew／eh 是玩家調整過的編輯視窗大小（原版像素）；0 代表用預設值。
	ew, eh int
	// resizing 是「調整編輯窗大小」（Ctrl-R）的模式，
	// dragResize 是模式裡正在拖右下角。
	resizing   bool
	dragResize bool

	// gotoX／gotoY 是 Tab「前往災區」的目標，取自上一則帶座標的訊息。
	// 0,0 代表沒有目標——原版的 MesX／MesY 也是用 0,0 當「沒有」。
	gotoX, gotoY int
	// 招牌與劇本選單（原版的 `.PPF` 兩幅畫面）。screen 是「現在在哪一幕」。
	screen screenMode
	// loadFiles 是「讀取舊有檔案」列出來的城市檔。
	loadFiles []string
	// bundledCityDir 是發行包隨附的地圖目錄（cities/）。由啟動層注入，
	// 因為只有它知道 AppImage 的 APPDIR 與 macOS 的 .app 版面。
	bundledCityDir string
	// zoom 是**縮小**倍數：1 是原版的一格 16 像素，2 是減半，4 再減半。
	// 原版沒有這個功能，見 ZoomTile 的說明。
	zoom int
	// lang 是目前的語言，music 是背景音樂播放器。
	// **兩個原版都沒有**：原版只有英文，也沒有音樂（docs/re/19-no-music.md）。
	lang  i18n.Lang
	music *musicPlayer
	// saveFmt 是存檔要寫哪一種版面。兩種都是原版認得的格式，取捨見
	// internal/game/save.go 的 SaveFormat。
	saveFmt game.SaveFormat
	// savePrefs 由啟動層注入，把語言、存檔格式與顯示模式一起寫回設定檔。
	savePrefs func(i18n.Lang, string, string) error
	// mode 是目前的顯示模式（`internal/ui/tileset.go` 的 DisplayModes）。
	// 空字串代表沒指定，走 graphicsDirs 的挑選順序。
	mode string
	// saveLang 由啟動層注入，避免 UI 套件決定各平台設定檔位置。
	// nil 代表本次工作階段不持久化（例如測試或無法取得設定目錄）。
	saveLang func(i18n.Lang) error
	// mapPopup 是 City Form 裡按住共用圖示跳出來的小選單，−1 代表沒開。
	// mapSubArmed 分辨「按住拉開」與「點一下拉開」：拉開的那一下放開時
	// 不能把選單收掉，否則點一下永遠看不到它。
	mapPopup          int
	mapSubArmed       bool
	titlePic, scenPic *ebiten.Image
	// 前往災區之前的鏡頭。參考附表寫「再按一次返回原地」，所以 Tab 是
	// 來回切換，不是單程。backX 為 −1 代表現在人在原地。
	backX, backY int

	// mini 是全市地圖的畫布快取，見 minimap.go。
	mini *minimap

	// newCityBox 是「建造新城市」對話框，nil 代表沒開。見 newcity.go。
	newCityDlg *newCityBox
	terrainDlg *terrainBox // 地形編輯器的參數對話框
	// terrain 非 nil 代表整個畫面被地形編輯器接管（terrain_screen.go）。
	terrain              *terrainScreen
	newCityTitleBackdrop bool // 從招牌進入時，原版只畫選單列與灰色桌面

	// picture 是目前顯示的圖片訊息全文（多行）。空字串代表沒有。
	// 原版的圖片訊息會開一個視窗擋住畫面，玩家按一下才關掉——
	// 那是刻意的：那些訊息（爐心熔毀、彈劾、劇本簡介）必須被看到。
	picture         string
	pictureScenario bool
}

// SetSavePath 設定存檔位置。
func (g *Game) SetSavePath(p string) { g.savePath = p }

// NewGame 建一個新遊戲。
func NewGame(w *sim.World, ts *TileSet, f *Font, txt *i18n.Catalog) *Game {
	g := &Game{world: w, tiles: ts, font: f, txt: txt, tool: sim.ToolResidential,
		animate: true, fastAnimate: true, menuRow: -1, graphYears: 10,
		backX: -1, zoom: 1, mapPopup: -1, lang: i18n.ZhHant}
	// 一開始六條曲線都畫，跟原版一樣。
	for i := range g.graphOn {
		g.graphOn[i] = true
	}
	g.resetCamera()
	return g
}

// inEditView 判斷畫面座標在不在編輯視窗的地圖區裡。
func (g *Game) inEditView(mx, my int) bool {
	vw, vh := g.editViewSize()
	return mx >= editViewX*UIScale && mx < (editViewX+vw)*UIScale &&
		my >= editViewY*UIScale && my < (editViewY+vh)*UIScale
}

// resetCamera 把鏡頭擺到**地圖原點 (0,0)**——原版就是這樣。
//
// ⚠ 先前是置中（`(WorldX - 可見格數)/2`），那是憑直覺訂的，不是量出來的。
// 量法：讓原版載入劇本後立刻暫停並存檔，再從截圖把每一格解回圖塊編號、
// 在存出來的地圖上滑動找最吻合的位置（`tools/shot_locate.py`）。
// 兩個劇本都指到 (0,0) 而且尖峰很銳利：
//
//	波士頓    128 個有辨識力的格中 118 格對上，次佳 15
//	達斯維利  149 格全中，次佳 19
//
// 拿存出來的城市檔當基準是關鍵：直接拿劇本檔比會被 `DoSimInit` 觸發的
// 劇本災難（波士頓是核災）弄髒，那時候怎麼比都對不上。
func (g *Game) resetCamera() {
	g.camX, g.camY = 0, 0
	g.clampCamera()
}

func (g *Game) tilesAcross() int {
	vw, _ := g.editViewSize()
	return vw / g.tileSize()
}

func (g *Game) tilesDown() int {
	_, vh := g.editViewSize()
	return vh / g.tileSize()
}

// tileSize 是目前一格佔幾個原版像素。縮小之後變小，其餘的版面不動——
// 地圖區的邊界是原版量出來的，不跟著縮。
func (g *Game) tileSize() int {
	z := g.zoom
	if z < 1 {
		z = 1
	}
	sz := g.tiles.Size / z
	if sz < 1 {
		sz = 1
	}
	return sz
}

// zoomLevels 是可選的縮小倍數。上限 4 的理由：一格 4 像素時整張 120×100
// 的地圖是 480×400 原版像素，已經超過畫面本身，再縮沒有意義。
var zoomLevels = []int{1, 2, 4}

// setZoom 換縮小倍數，並把鏡頭中心留在原地——否則縮小之後畫面會跳到
// 左上角，玩家得重新找自己剛才在看哪裡。
func (g *Game) setZoom(z int) {
	old := g.zoom
	if old < 1 {
		old = 1
	}
	if z < 1 {
		z = 1
	}
	if z == old {
		return
	}
	cx := g.camX + g.tilesAcross()/2
	cy := g.camY + g.tilesDown()/2
	g.zoom = z
	g.LookAt(cx, cy)
	g.setMessage(fmt.Sprintf(g.txt.UI("zoom"), z))
}

// stepZoom 往清單的下一／上一級走。
func (g *Game) stepZoom(d int) {
	cur := 0
	for i, z := range zoomLevels {
		if z == g.zoom {
			cur = i
		}
	}
	i := cur + d
	if i < 0 || i >= len(zoomLevels) {
		return
	}
	g.setZoom(zoomLevels[i])
}

// OpenWindow 依名稱開一個視窗。給 -window 旗標與截圖驗收用。
func (g *Game) OpenWindow(name string) bool {
	switch name {
	case "maps":
		g.showCityForm()
	case "graphs":
		g.win = winGraphs
	case "budget":
		g.win = winBudget
	case "eval":
		g.win = winEval
	case "about":
		g.win = winAbout
	case "saveas":
		g.openSaveAs()
	case "newcity":
		g.openNewCity()
	case "terrain":
		g.openTerrainScreen()
	case "terrainparams":
		g.openTerrainScreen()
		g.openTerrainEditor()
	case "language", "settings":
		g.openLangSettings()
	case "load":
		g.load()
	default:
		return false
	}
	return true
}

// showCityForm 對應原版 WINDOWS → Maps／Ctrl-M：City Form 本來就是常駐的
// 地圖視窗，這兩個入口只把它叫回來並移到最前面，不另開第二個地圖覆蓋層。
func (g *Game) showCityForm() {
	g.win = winNone
	g.mapClosed = false
	g.editFront = false
}

// SetLayer 設定地圖視窗的圖層。
func (g *Game) SetLayer(n int) {
	if n >= 0 && n < int(layerCount) {
		g.layer = mapLayer(n)
	}
}

// scrollDir 讀出捲動方向，−1／0／1 各一個軸。
//
// 原版的捲動鍵是 **`Ctrl` ＋ 方向鍵或數字鍵盤**（英文手冊「WITH A KEYBOARD」段，
// 2026-08-30 用 DOS 1.10 實測：`Ctrl-Right` ×20 讓編輯視窗的地圖確實往右移，
// 而單按方向鍵、`Ins`、`Del`、數字鍵盤都沒有反應——手冊把那些寫成
// **沒有滑鼠驅動時**才走的路徑，DOSBox 裡裝了滑鼠所以量不到）。
//
// remake 兩種都吃：單按方向鍵直接捲（既有行為，滑鼠一定在），
// 加不加 `Ctrl` 都一樣。八個方向照手冊的對應：
// `Home`／`7` 左上、`PgUp`／`9` 右上、`End`／`1` 左下、`PgDn`／`3` 右下。
func scrollDir() (dx, dy int) {
	down := func(ks ...ebiten.Key) bool {
		for _, k := range ks {
			if ebiten.IsKeyPressed(k) {
				return true
			}
		}
		return false
	}
	left := down(ebiten.KeyLeft, ebiten.KeyKP4, ebiten.KeyHome, ebiten.KeyKP7,
		ebiten.KeyEnd, ebiten.KeyKP1)
	right := down(ebiten.KeyRight, ebiten.KeyKP6, ebiten.KeyPageUp, ebiten.KeyKP9,
		ebiten.KeyPageDown, ebiten.KeyKP3)
	up := down(ebiten.KeyUp, ebiten.KeyKP8, ebiten.KeyHome, ebiten.KeyKP7,
		ebiten.KeyPageUp, ebiten.KeyKP9)
	bottom := down(ebiten.KeyDown, ebiten.KeyKP2, ebiten.KeyEnd, ebiten.KeyKP1,
		ebiten.KeyPageDown, ebiten.KeyKP3)
	if left && !right {
		dx = -1
	} else if right && !left {
		dx = 1
	}
	if up && !bottom {
		dy = -1
	} else if bottom && !up {
		dy = 1
	}
	return
}

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
	g.beat()
	g.maybeFreeze()
	if g.quit {
		return ebiten.Termination
	}
	if g.updateTitle() {
		return nil
	}
	// 地形編輯器接管整個畫面：自己的選單列、工具盤與視窗，
	// 模擬也停著（原版的編輯器是獨立程式，不跑模擬）。
	//
	// ⚠ **只換掉輸入那一段，不是整個 Update 早退**：訊息計時、音效播放器的
	// 回收（`pumpSounds` 的 `reap`）與音樂都在下面，早退會讓存檔訊息永遠不消失，
	// 而且音效播放器不回收——那正是 issue #1 找到的那條沒有上界的資源。
	if !g.updateTerrain() {
		g.handleKeys()
		g.handleMouse()
	}
	if g.openSaveFmtNext {
		g.openSaveFmtNext = false
		g.openSubMenu(winSaveFmt)
		g.waitEnterRelease = true
	}
	if g.openLangNext {
		g.openLangNext = false
		g.openLangSettings()
	}
	// 「最快」是同一個速率下一個畫格多跑幾次模擬（Micropolis 的 sim_skips），
	// 不是第五個速率——見 speedMsgIdx 的說明。
	g.updateMusic()
	if g.terrain == nil {
		for i := 0; i < simFramesPerTick[clamp(g.speedLevel, 0, 4)]; i++ {
			g.world.Frame()
		}
	}
	// 圖塊動畫。原版的條件是 `DoAnimation && SimSpeed && !TilesAnimated`
	// （`w_editor.c:874`）——暫停時不動，而且一個畫格只做一次。
	// ⚠ 它會改地圖，所以只有呈現層能呼叫，`internal/sim` 自己不碰。
	if g.world.DoAnimation && g.world.SimSpeed != 0 {
		if g.animate {
			g.world.AnimateTiles()
		}
	}
	if g.msgTimer > 0 {
		g.msgTimer--
	}
	g.pumpSimMessage()
	g.pumpSounds()
	return nil
}

// pumpSimMessage 把模擬層的訊息埠取出來顯示。
//
// ⚠ 負數代表「有圖」。原版會開一個視窗放整段文字，而且**同一張圖不會
// 重送**（靠 LastPicNum 去重）。正數是訊息欄的一行字，先到先得。
// 兩者不能混：把有圖的當成一行字，玩家就永遠看不到劇本簡介與彈劾通知。
func (g *Game) pumpSimMessage() {
	n := g.world.MessagePort
	if n == 0 {
		return
	}
	g.world.MessagePort = 0
	// 情境配樂是 remake 新增功能：沿用同一個已證實的訊息入口，避免音樂
	// 另猜一套災難狀態。固定映射見 docs/spec/adaptive-music.md。
	g.cueDisasterMusic(n)
	if g.world.MesX != 0 || g.world.MesY != 0 {
		g.gotoX, g.gotoY = g.world.MesX, g.world.MesY
		// 功能選單的「自動前往災難現場」。原版的開關存在城市檔裡。
		if g.world.AutoGo {
			g.LookAt(g.gotoX, g.gotoY)
		}
	}
	if s := n; s != 0 {
		// 警笛在訊息**第一次顯示**的那一刻播，判準是訊息類別。
		// 原版同一支常式（doMessage 的 firstTime 分支）。
		if s < 0 {
			s = -s
		}
		if sim.WantsSiren(s) && !g.soundOff {
			g.snd.play(sim.SoundSiren)
		}
	}
	if n < 0 {
		if p := g.txt.Picture(-n); p != "" {
			g.picture = p
			g.pictureScenario = false
			return
		}
		n = -n
	}
	// 訊息編號是 1 起算，第 0 段的文字在偶數索引。
	g.setMessage(g.txt.S(i18n.SecStatus, (n-1)*2))
}

// ShowScenarioBrief 顯示劇本簡介。載入劇本時呼叫一次。
func (g *Game) ShowScenarioBrief() {
	if g.world.Scenario == 0 {
		return
	}
	g.picture = g.txt.ScenarioBrief(int(g.world.Scenario))
	g.pictureScenario = true
}

func (g *Game) setMessage(s string) {
	if s == "" {
		return
	}
	g.message = s
	g.msgTimer = 60 * 8
}

func (g *Game) handleKeys() {
	if g.waitEnterRelease {
		if ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeyKPEnter) {
			return
		}
		g.waitEnterRelease = false
	}
	// 地形編輯器與新城市對話框都是**強制回應**的。
	if g.handleTerrainKeys() {
		return
	}
	// 新城市對話框是**強制回應**的：原版要選完等級與市名才進得了遊戲。
	if g.handleNewCityKeys() {
		return
	}
	// 下拉選單拉開時，鍵盤全部歸它——否則方向鍵會同時捲地圖。
	if g.handleMenuKeys() {
		return
	}
	// 調整編輯視窗大小（Ctrl-R）：方向鍵改大小，不捲地圖。
	if g.resizing {
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			g.resizeEdit(1, 0)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			g.resizeEdit(-1, 0)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			g.resizeEdit(0, 1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.resizeEdit(0, -1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.resizing = false
		}
		return
	}
	// ⚠ 工具鍵要排除三種情況，每一種都造成過「按了甲卻順便做了乙」：
	//
	//   - **Ctrl 按著**：Ctrl-B（預算）會順便選到推土機。
	//   - **Alt 按著**：Alt-S（系統選單）會順便選到體育館、
	//     Alt-D（災難選單）順便選到道路。開個選單就換掉玩家的工具，
	//     而且狀態列只顯示新工具，玩家下一次點擊才發現。
	//   - **視窗開著**：「以……檔名儲存」要打字，每一個字母都會換工具。
	//
	// 前兩種是截圖驗收時發現的：開「關於本遊戲」的那張圖上，
	// 工具列的高亮跑到體育館去了。
	if !ebiten.IsKeyPressed(ebiten.KeyControl) &&
		!ebiten.IsKeyPressed(ebiten.KeyAlt) && g.win == winNone {
		for _, b := range toolButtons {
			if inpututil.IsKeyJustPressed(b.key) ||
				(b.alt != b.key && inpututil.IsKeyJustPressed(b.alt)) {
				g.tool = b.tool
				g.setMessage(g.toolLabel(b))
			}
		}
	}
	step := 1
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		step = 5
	}
	if g.win != winNone {
		step = 0 // 視窗開著時方向鍵歸視窗用
	}
	// 縮小／放大。**這是 remake 加的功能，原版沒有**（原版的編輯視窗永遠
	// 一格 16 像素）。鍵位選 `-`／`=` 而不是原版參考附表上的 `+`／`-`：
	// 那兩個鍵在原版是「在視窗的可點處之間循環」，留給它。
	if g.win == winNone && g.newCityDlg == nil && g.terrainDlg == nil &&
		g.picture == "" &&
		!ebiten.IsKeyPressed(ebiten.KeyControl) &&
		!ebiten.IsKeyPressed(ebiten.KeyAlt) {
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) ||
			inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
			g.stepZoom(1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) ||
			inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
			g.stepZoom(-1)
		}
		// 背景音樂，同樣是 remake 加的（原版沒有音樂，見 music.go）。
		if inpututil.IsKeyJustPressed(ebiten.KeyM) {
			g.toggleMusic()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeftBracket) {
			g.stepTrack(-1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRightBracket) {
			g.stepTrack(1)
		}
	}

	dx, dy := scrollDir()
	g.camX += dx * step
	g.camY += dy * step
	g.clampCamera()

	// 速度：原版是 `0`–`4`（暫停／慢速／普通／快速／最快）。
	// ⚠ 原版有**五段**，Micropolis 的規則層只有四段（0–3），所以「快速」與
	// 「最快」在規則層是同一段。這是已知的未解處，記在 docs/spec/controls.md。
	// F1–F4 是本專案舊版用的鍵，保留成別名。
	if g.win == winNone && !ebiten.IsKeyPressed(ebiten.KeyControl) {
		for i, k := range []ebiten.Key{
			ebiten.Key0, ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4,
		} {
			if inpututil.IsKeyJustPressed(k) {
				g.setSpeed(i)
			}
		}
	}
	// 選單：原版有兩組鍵。`Alt` ＋ 首字母（說明書 p.29–35），
	// 以及 **`F2`–`F5`**（手冊寫「沿用 Tandy Deskmate」）。
	//
	// ⚠ `F2`–`F5` **先前被拿去當速度的別名**，那和原版衝突。
	// 兩份一手資料對「哪個 F 鍵開哪個選單」說法不同，2026-08-30 用 DOS 原版
	// 逐鍵實測裁決（`tools/dosbox/act-m-F*.txt`，畫面存在
	// `workplace/dosbox/mF*-menu.png`）：
	//
	//	F1 不開選單／F2 SYSTEM／F3 OPTIONS／F4 DISASTERS／F5 WINDOWS
	//
	// 與官方英文手冊一致；軟體世界《參考附表》的 `F1F2` 系統／`F3F4` 災難／
	// `F5F6` 功能對這個版本**不成立**（註記在 `docs/manual-cht/ref-card.md`）。
	// 速度鍵是 `0`–`4`，那才是原版的（訊息檔第 19 段自己印著）。
	menuKeys := []ebiten.Key{ebiten.KeyS, ebiten.KeyO, ebiten.KeyD, ebiten.KeyW}
	fKeys := []ebiten.Key{ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5}
	alt := ebiten.IsKeyPressed(ebiten.KeyAlt)
	for i := range menuKeys {
		hit := inpututil.IsKeyJustPressed(fKeys[i]) ||
			(alt && inpututil.IsKeyJustPressed(menuKeys[i]))
		if !hit {
			continue
		}
		if g.openMenu == i+1 {
			g.openMenu = 0
		} else {
			g.openMenu, g.menuRow = i+1, g.firstMenuRow(i)
		}
	}

	// 視窗快速鍵沿用原版（說明書 p.35）。
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	if ctrl {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyM):
			// 原版：Ctrl-M 把既有的 City Form 叫到最前面。
			g.showCityForm()
		case inpututil.IsKeyJustPressed(ebiten.KeyG):
			g.toggleWindow(winGraphs)
		case inpututil.IsKeyJustPressed(ebiten.KeyB):
			g.toggleWindow(winBudget)
		case inpututil.IsKeyJustPressed(ebiten.KeyU):
			g.toggleWindow(winEval)
		case inpututil.IsKeyJustPressed(ebiten.KeyC):
			// 原版：Ctrl-C 關閉前視窗。有資料視窗開著就關它，
			// 否則關掉 City Form（編輯視窗關不掉，關了就沒得玩）。
			if g.win != winNone {
				g.win = winNone
			} else {
				g.mapClosed = true
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyH):
			// 原版：Ctrl-H 把**最前面的視窗移到最後面**（實測，見 mapClosed
			// 的說明）。這裡只有兩個常駐視窗，所以就是換 editFront。
			if g.win != winNone {
				g.win = winNone
			} else {
				g.editFront = !g.editFront
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyR):
			// 原版：Ctrl-R 調整編輯窗大小。進模式之後方向鍵改大小。
			g.resizing = true
		case inpututil.IsKeyJustPressed(ebiten.KeyL):
			g.load()
		case inpututil.IsKeyJustPressed(ebiten.KeyX):
			g.quit = true
		case inpututil.IsKeyJustPressed(ebiten.KeyE):
			// 原版：Ctrl-E 打開編輯視窗 —— 把它叫到最前面。
			g.win = winNone
			g.editFront = true
		case inpututil.IsKeyJustPressed(ebiten.KeyA):
			g.world.AutoBulldoze = !g.world.AutoBulldoze
			if g.world.AutoBulldoze {
				g.setMessage("自動整地：開")
			} else {
				g.setMessage("自動整地：關")
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.picture != "" {
			g.picture = ""
			g.pictureScenario = false
		} else {
			g.win = winNone
		}
	}
	if g.picture != "" && (inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter)) {
		g.picture = ""
		g.pictureScenario = false
	}
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.save()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.gotoEvent()
	}
	g.handleWindowKeys()
}

// gotoEvent 是原版參考附表上的 Tab「前往災區」。
//
// 目標是**上一則帶座標的訊息**的位置：災難、墜毀、爆炸、交通壅塞都帶座標
// （`SendMesAt`），純提示訊息不帶。原版用 `MesX`／`MesY`，`0,0` 代表沒有。
func (g *Game) gotoEvent() {
	if g.backX >= 0 {
		g.camX, g.camY = g.backX, g.backY
		g.backX = -1
		g.clampCamera()
		return
	}
	if g.gotoX == 0 && g.gotoY == 0 {
		g.setMessage(g.txt.UI("goto_none"))
		return
	}
	g.backX, g.backY = g.camX, g.camY
	g.LookAt(g.gotoX, g.gotoY)
}

// fireDisaster 發動災難選單的第 i 項，然後把選單收掉。
func (g *Game) fireDisaster(i int) {
	if i < 0 || i >= len(disasterItems) {
		return
	}
	disasterItems[i](g.world)
	g.setMessage(trimMenu(g.txt.S(i18n.SecDisaster, i)))
	g.win = winNone
}

func (g *Game) toggleWindow(w window) {
	if g.win == w {
		g.win = winNone
		return
	}
	g.win = w
}

// handleWindowKeys 處理開著的視窗自己的按鍵。
func (g *Game) handleWindowKeys() {
	switch g.win {
	case winMaps:
		// 1–9、0、- 切換十一個圖層
		keys := []ebiten.Key{
			ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5,
			ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9, ebiten.Key0,
			ebiten.KeyMinus,
		}
		for i, k := range keys {
			if inpututil.IsKeyJustPressed(k) && i < int(layerCount) {
				g.layer = mapLayer(i)
			}
		}
	case winSystem, winScenario, winStyle, winMode, winSpeed, winPower, winLoad,
		winLangSel, winMusic, winSaveFmt:
		g.handleSysMenuKeys()
	case winSaveAs:
		g.handleSaveAsKeys()
	case winDisaster:
		n := len(disasterItems)
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			g.disasterRow = (g.disasterRow + 1) % n
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.disasterRow = (g.disasterRow + n - 1) % n
		}
		for i := 0; i < n; i++ {
			if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
				g.disasterRow = i
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
			inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
			g.fireDisaster(g.disasterRow)
		}
	case winBudget:
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			g.budgetRow = (g.budgetRow + 1) % 3
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.budgetRow = (g.budgetRow + 2) % 3
		}
		step := 0.01
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			step = 0.10
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			g.adjustFunding(step)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			g.adjustFunding(-step)
		}
		// 稅率
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
			g.world.CityTax = clamp(g.world.CityTax+1, 0, 20)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
			g.world.CityTax = clamp(g.world.CityTax-1, 0, 20)
		}
	}
}

// adjustFunding 調整一項的編列百分比。上限 100%，下限 0。
func (g *Game) adjustFunding(d float64) {
	p := []*float64{&g.world.RoadPercent, &g.world.PolicePercent, &g.world.FirePercent}[g.budgetRow]
	v := *p + d
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	*p = v
}

func (g *Game) handleMouse() {
	mx, my := ebiten.CursorPosition()
	// 滾輪縮放：只在編輯視窗的地圖區上有效，其他地方滾輪留給視窗自己用。
	if _, wy := ebiten.Wheel(); wy != 0 && g.win == winNone && g.inEditView(mx, my) {
		if wy < 0 {
			g.stepZoom(1)
		} else {
			g.stepZoom(-1)
		}
	}
	if g.handleTerrainMouse(mx, my) {
		return
	}
	if g.handleNewCityMouse(mx, my) {
		return
	}
	if g.handleResizeMouse(mx, my) {
		return
	}
	if g.handleMenuMouse(mx, my) {
		return
	}
	if g.handleWindowMouse(mx, my) {
		return
	}
	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	just := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	// 疊放順序：**原版是按右鍵把視窗拉到前面**（`docs/spec/ui-layout.md` §二）。
	//
	// ⚠ 先前綁在左鍵上，於是兩個視窗重疊的那一塊，每一次左鍵都被拿去換
	// 疊放順序，蓋不了東西——玩家的回報是「就算把地圖視窗放到後面，
	// 點到原本的位置它還是會跳上來卡住編輯」（issue #1）。
	// 左鍵一律歸最前面那個視窗，這才是原版的行為。
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) &&
		g.raiseWindowAt(mx, my) {
		return
	}
	// City Form 的圖層圖示（含兩個共用圖示的小選單）。
	if g.handleMapIconMouse(mx, my, just, pressed) {
		return
	}
	// 工具盤：編輯視窗左緣，2 欄 × 7 列（classic.go）。
	if just {
		if i := g.paletteHit(mx, my); i >= 0 {
			if i == powerCell {
				// 發電廠那一格是副選單，不是直接選工具。
				g.openPowerSub()
			} else {
				g.tool = paletteOrder[i]
			}
			return
		}
	}
	// 圖片訊息擋住畫面時，點一下關掉，不要蓋東西。
	if g.picture != "" {
		if just {
			g.picture = ""
			g.pictureScenario = false
		}
		return
	}
	// 視窗開著時，點擊歸視窗，不要蓋東西。統計圖的圖示按鈕例外。
	if g.win != winNone {
		if just && g.win == winGraphs {
			if i := g.graphHit(mx, my); i >= 0 {
				switch {
				case i < 6:
					g.graphOn[i] = !g.graphOn[i]
				case i == 6:
					g.graphYears = 10
				default:
					g.graphYears = 120
				}
			}
		}
		return
	}
	// 地圖：編輯視窗裡的那一塊。座標要先減掉視窗原點——
	// 少減的話工具會蓋在離游標好幾格的地方，而且看起來像「格子算錯」。
	// City Form 在前面的時候，被它蓋住的那一塊不能蓋東西。
	if !g.inEditView(mx, my) || (!g.editFront && !g.mapClosed && inCityForm(mx, my)) {
		g.dragging = false
		return
	}
	if !pressed {
		g.dragging = false
		g.querying = false
		return
	}
	// 道路、鐵軌、電力線可以拖曳；其餘只在按下的那一刻動作，
	// 免得手一抖就蓋出一排體育館。
	// 查詢也算「拖曳」：按住不放時面板要跟著游標換格，那是原版的行為。
	drag := g.tool == sim.ToolRoad || g.tool == sim.ToolRail ||
		g.tool == sim.ToolWire || g.tool == sim.ToolBulldozer ||
		g.tool == sim.ToolQuery
	if !just && !(drag && g.dragging) {
		return
	}
	g.dragging = true
	px := g.tileSize() * tileScale
	tx := g.camX + (mx-editViewX*UIScale)/px
	ty := g.camY + (my-editViewY*UIScale)/px
	g.applyTool(tx, ty)
}

func (g *Game) applyTool(tx, ty int) {
	if g.tool == sim.ToolQuery {
		// 查詢不蓋東西：把面板打開，放開滑鼠才收（原版就是按住才顯示）。
		g.querying = true
		g.queryTX, g.queryTY = tx, ty
		return
	}
	before := g.world.TotalFunds
	r := g.world.ApplyTool(g.tool, tx, ty)
	g.toolSound(r, g.tool)
	switch r {
	case sim.ToolOK:
		if spent := before - g.world.TotalFunds; spent > 0 {
			g.setMessage(fmt.Sprintf("花費 $%d", spent))
		}
	case sim.ToolNoMoney:
		g.toolMessage(sim.MsgNoMoney)
	case sim.ToolNeedsClear:
		g.toolMessage(sim.MsgNeedsClear)
	case sim.ToolBlocked:
		// ⚠ **推論等級：假說。** Micropolis 對回傳 0 不出訊息，而 DOS 1.10 的
		// 訊息檔多了三則工具錯誤（44 不能蓋在水上、45 這裡不能蓋、
		// 46 這裡不能推平），Micropolis 完全沒有。DOS 版怎麼分派這三則還沒解，
		// 這裡按工具分兩種；44 目前沒有觸發點。見 docs/re/14-messages.md §5。
		if g.tool == sim.ToolBulldozer {
			g.toolMessage(msgCannotBulldozeHere)
		} else {
			g.toolMessage(msgCannotBuildHere)
		}
	}
}

// DOS 1.10 專屬的工具錯誤訊息（Micropolis 的訊息表沒有這三則）。
const (
	msgCannotBuildOnWater = 44
	msgCannotBuildHere    = 45
	msgCannotBulldozeHere = 46
)

// toolMessage 走訊息埠而不是直接寫字串——文字要從語言檔來，
// 而原版也是走同一條路（w_tool.c:1553 `ClearMes(); SendMes(34);`）。
func (g *Game) toolMessage(n int) {
	g.world.ClearMes()
	g.world.SendMes(n)
}

// Draw 畫一個 frame。
//
// 版面是原版 DOS 的重現，畫法在 classic.go。舊的固定版面（右側文字工具列
// ＋ 下方狀態列）已經換掉——原版是視窗系統，那才是「操作介面一樣」的意思。
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	if g.screen != scrPlay {
		g.drawTitle(screen)
		return
	}
	if g.newCityDlg != nil && g.newCityTitleBackdrop {
		fill(screen, 0, 0, OrigW, OrigH, colDesktop)
		g.drawMenuBar(screen)
		g.drawNewCity(screen)
		return
	}
	if g.terrain != nil {
		g.drawTerrainScreen(screen)
		return
	}
	g.drawClassic(screen)
	g.drawWindow(screen)
	g.drawPicture(screen)
	g.drawMenu(screen)
	g.drawNewCity(screen)
	g.drawTerrainEditor(screen)
}

// drawDemand 畫 R／C／I 需求柱。原版用短柱的正負表示需要或過剩。
func (g *Game) drawDemand(dst *ebiten.Image, x, y int) {
	g.font.Draw(dst, g.txt.UI("demand"), x, y, colDim)
	bars := []struct {
		label string
		v     int
		c     color.RGBA
	}{
		{g.txt.UI("demand_r"), g.world.RValve, colDemR},
		{g.txt.UI("demand_c"), g.world.CValve, colDemC},
		{g.txt.UI("demand_i"), g.world.IValve, colDemI},
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

// toolNameCost 從譯文取工具的名稱與造價。
//
// 原版把它們寫成一句「推土機：$1」，所以在冒號處拆開。譯文用的是全形
// 冒號，原文是半形——兩種都要認得，不然某些風格會整串擠在左邊。
func (g *Game) toolNameCost(b toolButton) (string, string) {
	if b.msgIdx < 0 {
		return "查詢", ""
	}
	s := g.txt.S(i18n.SecToolCost, b.msgIdx)
	if s == "" {
		return "?", ""
	}
	for _, sep := range []string{"：", ": "} {
		if i := strings.Index(s, sep); i >= 0 {
			return s[:i], strings.TrimSpace(s[i+len(sep):])
		}
	}
	return s, ""
}

func (g *Game) toolLabel(b toolButton) string {
	n, _ := g.toolNameCost(b)
	return "選擇工具：" + n
}

// setSpeed 設定模擬速度並回報。
// setSpeed 設定五段速度之一（0 暫停 … 4 最快）。
func (g *Game) setSpeed(n int) {
	if n < 0 {
		n = 0
	}
	if n >= len(simSpeedOf) {
		n = len(simSpeedOf) - 1
	}
	g.speedLevel = n
	g.world.SimSpeed = simSpeedOf[n]
	g.setMessage("模擬速度：" + g.speedName(n))
}

// speedName 從功能選單的速度副選單取名稱。
//
// ⚠ 原版那一段是「 最快  4」這種形式：前導空白給選單縮排、後面是數字。
// 直接顯示會多出一堆空白，所以只取中間的名稱。
//
// 五段速度。原版的副選單（訊息檔第 19 段）由快到慢是
// 最快 4／快速 3／普通 2／慢速 1／暫停 0，所以索引要倒過來查。
//
// ⚠ **Micropolis 的 `SimSpeed` 只有四段**（`w_util.c:145` 把參數夾在 0–3，
// 0 ＝ 暫停）。第五段「最快」不是第五個模擬速率，而是
// **同一個速率下一個畫格多跑幾次模擬**——Micropolis 自己就有這個機制
// （`setSkips`／`sim_skips`，`sim.c:71`）。所以五段是：
//
//	段 0 暫停    SimSpeed 0
//	段 1 慢速    SimSpeed 1
//	段 2 普通    SimSpeed 2
//	段 3 快速    SimSpeed 3，一個畫格一次
//	段 4 最快    SimSpeed 3，一個畫格三次
//
// ⚠ 用 `4-n` 直接算是錯的：那會把「最快」顯示成「快速」，
// 數字看起來很合理，但玩家在最高速時看到的是次高速的名稱。
var speedMsgIdx = [5]int{4, 3, 2, 1, 0} // 暫停、慢速、普通、快速、最快

// simFramesPerTick 是每一段速度一個畫格要跑幾次模擬。
var simFramesPerTick = [5]int{1, 1, 1, 1, 3}

// simSpeedOf 是每一段速度對到的 Micropolis `SimSpeed`。
var simSpeedOf = [5]int{0, 1, 2, 3, 3}

func (g *Game) speedName(n int) string {
	if n < 0 || n >= len(speedMsgIdx) {
		n = 0
	}
	idx := speedMsgIdx[n]
	s := strings.TrimSpace(g.txt.S(i18n.SecSpeed, idx))
	if i := strings.IndexAny(s, "0123456789"); i > 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// save 把城市存成原版格式的 .cty。
func (g *Game) save() {
	p := g.savePath
	if p == "" {
		p = "city.cty"
	}
	if err := game.SaveCityAs(p, g.world, g.saveFmt); err != nil {
		g.setMessage("存檔失敗：" + err.Error())
		return
	}
	g.setMessage("已存檔：" + p)
}

// handleMapIconMouse 處理 City Form 左緣那九個圖層圖示。
//
// 原版的兩個共用圖示是**按住式**的：按住不放跳出兩列的小選單，
// 拖到哪一列放開就選哪一列（2026-08-30 實測，`workplace/dosbox/mi6-*.png`）。
// 快速點一下等於選中游標所在的第一列，所以看起來像「直接選了警力範圍」——
// 這也是為什麼「再點一次會不會換」量到零像素：它根本不是切換式的。
//
// remake 兩種都吃：按住拖曳（原版），或點一下拉開、再點一下選中。
func (g *Game) handleMapIconMouse(mx, my int, just, pressed bool) bool {
	if g.mapClosed || g.win != winNone {
		return false
	}
	if g.mapPopup >= 0 {
		row := g.mapSubRowAt(g.mapPopup, mx, my)
		// 放開時停在某一列上 → 選它；停在別處 → 收起來。
		if !pressed {
			if row >= 0 {
				g.SetLayer(mapIconLayers[g.mapPopup][row])
				g.mapPopup = -1
			} else if g.mapSubArmed {
				g.mapPopup = -1
			}
			g.mapSubArmed = true
			return true
		}
		if just && row < 0 && mapIconAt(mx, my) != g.mapPopup {
			g.mapPopup = -1
		}
		return true
	}
	if !just {
		return false
	}
	i := mapIconAt(mx, my)
	if i < 0 {
		return false
	}
	if len(mapIconLayers[i]) > 1 {
		g.mapPopup = i
		g.mapSubArmed = false
		return true
	}
	g.SetLayer(mapIconLayers[i][0])
	return true
}
