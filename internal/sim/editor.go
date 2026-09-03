package sim

// 地形編輯器的規則層。
//
// 一手證據是軟體世界 1990 年那片磁片裡的 `TERRAIN.EXE`（解 LZEXE 之後的
// 325 728 位元組版本），拆解過程與位址在 docs/re/20-terrain-editor.md，
// 規格在 docs/spec/terrain-editor.md。
//
// 這一層完全 headless、完全決定性：畫筆、油漆桶、復原、四個地形動作
// 都只動 World.Map，不認識畫面。與 internal/sim 其餘部分一樣，
// 這是逐格對拍得起來的前提。
//
// ⚠ 這裡的規則**不是** Micropolis 的。Micropolis 沒有地形編輯器，
// 所以本檔每一條的出處都是 DOS 執行檔的位址，不是 `s_*.c` 的行號。

// EditorTool 是工具盤上那四個會寫地圖的畫筆。
//
// 編號就是原版的編號：`byte_595E0` 存的值，也是工具描述表
// （`dseg:0x2B42`，一列 18 位元組）的列號。5（FILL）與 6（UNDO）
// 不在這裡——它們在原版是**動作**不是畫筆，見 `sub_22636`。
type EditorTool int

const (
	EdDirt    EditorTool = 1 // DIRT
	EdTrees   EditorTool = 2 // TREES
	EdRiver   EditorTool = 3 // RIVER
	EdChannel EditorTool = 4 // CHANNEL
)

// editorCell 是每個畫筆寫下去的那個 16 位元字。
//
// 出處是工具描述表的第 +0x0C 個位元組（圖塊）與第 +0x02 個（旗標），
// 由 `sub_1EF36`＋0x1F0A6 相加寫進地圖：`Map[x][y] = tile + flags`。
// 表裡的原始值：
//
//	1  tile=0   flags=0x0000
//	2  tile=37  flags=0x3000  (BURNBIT|BULLBIT)
//	3  tile=3   flags=0x0000
//	4  tile=4   flags=0x0000
//
// **RIVER 寫的是 3（REDGE）不是 2（RIVER）**，跟油漆桶那一支一致；
// 真正的河面圖塊由之後的「平滑河流」算出來。
var editorCell = map[EditorTool]uint16{
	EdDirt:    DIRT,
	EdTrees:   WOODS | BURNBIT | BULLBIT,
	EdRiver:   REDGE,
	EdChannel: CHANNEL,
}

// Cell 回傳某個畫筆會寫下去的完整 16 位元字。
func (t EditorTool) Cell() uint16 { return editorCell[t] }

// 復原環的大小。原版是 `sub_1C9C2(0, idx±1, 0x1387)`，也就是 0–4999 折回，
// 所以環有 5000 格（`sub_106BA`＋0x107F3、`sub_10862`＋0x10886）。
const editorUndoRing = 5000

// 全圖快照最多四份（`sub_106BA`＋0x10704 的 `cmp word_4BFC2, 4`）。
// 第五份進來時最舊的那一份被擠掉，環的尾巴也跟著跳過它的標記。
const editorSnapshots = 4

// undoRec 是環裡的一格。x ＝ 0xFF 代表「這一步是全圖快照」，
// 值那一欄不看（原版在快照那條路徑用 x=−1 去索引地圖，讀到的是界外垃圾，
// 而復原時根本不會用到它）。
type undoRec struct {
	x, y uint8
	cell uint16
}

// Editor 是一個地形編輯階段：一張地圖加上它的復原歷史。
type Editor struct {
	w *World

	ring       [editorUndoRing]undoRec
	head, tail int // head ＝ 下一格要寫的位置，head == tail 代表沒得復原

	snaps [][WorldX][WorldY]uint16

	// Island 對應原版 `byte_52E72`：TERRAIN 選單的「造島」是一個**開關**，
	// 勾起來之後下一次「產生隨機地形」才會造島（`sub_10A0A`＋0x10C6B）。
	// 不是按下去就造島的動作。
	Island bool

	// Fill 對應原版 `byte_59194`：油漆桶是一個開關，按一次亮起來，
	// 倒完一次自己熄掉（`sub_229F0`＋0x22D98 的 `sub_22636(5)`）。
	Fill bool

	// Tool 是目前選的畫筆。原版開機預設 DIRT（截圖上 DIRT 亮著黃框）。
	Tool EditorTool
}

// NewEditor 開一個編輯階段。
func NewEditor(w *World) *Editor {
	return &Editor{w: w, Tool: EdDirt}
}

// World 讓呈現層拿到底下那張地圖。
func (e *Editor) World() *World { return e.w }

// wrapRing 是原版 `sub_1C9C2(min, val, max)`：小於下限折到上限、
// 大於上限折到下限。它是**折返**不是夾限——環形緩衝區靠這個繞回去。
func wrapRing(v int) int {
	switch {
	case v < 0:
		return editorUndoRing - 1
	case v > editorUndoRing-1:
		return 0
	}
	return v
}

// CanUndo 回報還有沒有東西可以復原。
// 原版用同一個判斷決定要不要發「工具失敗」的嗶聲（`sub_10862`＋0x1086F）。
func (e *Editor) CanUndo() bool { return e.head != e.tail }

// pushCell 把一格的舊值記進環裡。原版 `sub_106BA` 的後半段。
func (e *Editor) pushCell(x, y int, cell uint16) {
	e.ring[e.head] = undoRec{x: uint8(x), y: uint8(y), cell: cell}
	e.head = wrapRing(e.head + 1)
	// 環滿了就把最舊的那一格擠掉。
	if e.head == e.tail {
		e.tail = wrapRing(e.tail + 1)
	}
}

// PushSnapshot 記下一份全圖快照，給「會動很多格」的動作用
// （清除地圖、清除人造物、產生隨機地形、三個平滑、油漆桶）。
// 原版是 `sub_106BA(-1, -1)`。
func (e *Editor) PushSnapshot() {
	if len(e.snaps) == editorSnapshots {
		// 快照滿了：把環的尾巴推到最舊那個快照標記的**後面**，
		// 再把緩衝區整批往前搬一格（`sub_106BA`＋0x10713）。
		for e.ring[e.tail].x != 0xFF || e.ring[e.tail].y != 0xFF {
			e.tail = wrapRing(e.tail + 1)
		}
		e.tail = wrapRing(e.tail + 1)
		e.snaps = e.snaps[1:]
	}
	var snap [WorldX][WorldY]uint16
	for x := 0; x < WorldX; x++ {
		snap[x] = e.w.Map[x]
	}
	e.snaps = append(e.snaps, snap)
	// 快照也要在環裡佔一格，否則復原走不回來。x／y 都是 0xFF。
	e.ring[e.head] = undoRec{x: 0xFF, y: 0xFF}
	e.head = wrapRing(e.head + 1)
	if e.head == e.tail {
		e.tail = wrapRing(e.tail + 1)
	}
}

// Undo 復原一步。回傳 false 代表沒得復原（原版此時發第 7 號音效）。
// 原版 `sub_10862`。
func (e *Editor) Undo() bool {
	if !e.CanUndo() {
		return false
	}
	e.head = wrapRing(e.head - 1)
	r := e.ring[e.head]
	if r.x == 0xFF {
		if n := len(e.snaps); n > 0 {
			snap := e.snaps[n-1]
			for x := 0; x < WorldX; x++ {
				e.w.Map[x] = snap[x]
			}
			e.snaps = e.snaps[:n-1]
		}
		return true
	}
	e.w.Map[r.x][r.y] = r.cell
	return true
}

// Paint 用目前的畫筆蓋一格。回傳 true 代表地圖真的變了。
//
// 原版 `sub_1EF36`：界外直接回 0；值一樣就不寫也不記復原
// （`cmp [bx+di+4482h], ax; jz`）。**一次只有一格**——工具描述表裡
// 編輯器那四列的尺寸欄都是 1，3×3 那些是遊戲本體的分區工具。
func (e *Editor) Paint(t EditorTool, x, y int) bool {
	if !InBounds(x, y) {
		return false
	}
	cell, ok := editorCell[t]
	if !ok {
		return false
	}
	old := e.w.Map[x][y]
	if old == cell {
		return false
	}
	e.pushCell(x, y, old)
	e.w.Map[x][y] = cell
	return true
}

// 油漆桶認得的三個「自然地物帶」，出自 `sub_229F0`＋0x22A94 起的三段比較。
// 上界是開區間。
const (
	bandDirtHi  = 2           // [0, 2)   空地
	bandWaterHi = 0x15        // [2, 21)  水域與河岸
	bandTreeHi  = 0x28        // [21, 40) 樹林
	bandWaterLo = bandDirtHi  // 2
	bandTreeLo  = bandWaterHi // 21
)

// FillAt 把一整片同類地物換成目前畫筆的地物（油漆桶）。原版 `sub_229F0`。
//
// 三件事照原版：
//   - 帶是**依起點那一格的地物類別**決定的，不是依畫筆。點在水面上就換整片水，
//     不管畫筆是樹還是空地。
//   - 起點已經是畫筆要畫的那一類就什麼都不做（點水面選 RIVER／CHANNEL、
//     點空地選 DIRT、點樹林選 TREES）。
//   - 起點的地物編號 ≥ 40（不是空地、水、樹的任何一種，例如道路或建築）時
//     **退化成單格畫筆**，不做整片填色（`loc_22B64`）。
//
// 回傳 true 代表做了填色（呈現層據此把油漆桶熄掉——原版倒完一次就熄）。
func (e *Editor) FillAt(t EditorTool, x, y int) bool {
	if !InBounds(x, y) {
		return false
	}
	cell, ok := editorCell[t]
	if !ok {
		return false
	}
	cur := int(e.w.Map[x][y] & LOMASK)
	var lo, hi int
	switch {
	case cur < bandDirtHi:
		if t == EdDirt {
			return false
		}
		lo, hi = 0, bandDirtHi
	case cur < bandWaterHi:
		if t == EdRiver || t == EdChannel {
			return false
		}
		lo, hi = bandWaterLo, bandWaterHi
	case cur < bandTreeHi:
		if t == EdTrees {
			return false
		}
		lo, hi = bandTreeLo, bandTreeHi
	default:
		e.Paint(t, x, y)
		return true
	}

	e.PushSnapshot()

	// 掃描線填色。原版沿 x 軸走、上下兩列找新的種子；這裡形狀一樣。
	// 填色的**結果**與拜訪順序無關，所以不必逐步對拍堆疊。
	in := func(px, py int) bool {
		if px < 0 || px >= WorldX || py < 0 || py >= WorldY {
			return false
		}
		n := int(e.w.Map[px][py] & LOMASK)
		return n >= lo && n < hi
	}
	stack := [][2]int{{x, y}}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		px, py := p[0], p[1]
		if !in(px, py) {
			continue
		}
		x0 := px
		for x0 > 0 && in(x0-1, py) {
			x0--
		}
		up, down := false, false
		for x1 := x0; x1 < WorldX && in(x1, py); x1++ {
			e.w.Map[x1][py] = cell
			if n := in(x1, py-1); n && !up {
				stack = append(stack, [2]int{x1, py - 1})
				up = true
			} else if !n {
				up = false
			}
			if n := in(x1, py+1); n && !down {
				stack = append(stack, [2]int{x1, py + 1})
				down = true
			} else if !n {
				down = false
			}
		}
	}
	return true
}

// ClearMap 把整張圖填成空地。TERRAIN 選單第 0 列（Ctrl-C），原版 `sub_11EA4`。
func (e *Editor) ClearMap() {
	e.PushSnapshot()
	e.w.clearMap()
}

// ClearUnnatural 把所有不是空地／水／樹的格子清成空地。
// TERRAIN 選單第 1 列，原版 `sub_10A0A` 的 case 0x11（`loc_10B39`）：
// 逐格取低十位元，**大於 37（WOODS）就把整個字寫成 0**——旗標一起清掉。
func (e *Editor) ClearUnnatural() {
	e.PushSnapshot()
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			if int(e.w.Map[x][y]&LOMASK) > WOODS {
				e.w.Map[x][y] = DIRT
			}
		}
	}
}

// SmoothTreesOnly 是 TERRAIN 選單的「平滑樹林」（原版跑兩次）。
func (e *Editor) SmoothTreesOnly() {
	e.PushSnapshot()
	e.w.smoothTreesTwice()
}

// SmoothRiversOnly 是「平滑河流」：先把所有**臨接非水面的水格**打回 REDGE，
// 再跑一次 smoothRiver。原版 `sub_11A24` ＋ `sub_11FC4`。
func (e *Editor) SmoothRiversOnly() {
	e.PushSnapshot()
	e.w.ResetRiverEdges()
	e.w.smoothRiverOnly()
}

// SmoothEverything 是「全部平滑」（Ctrl-A）。
// 原版 case 0x17 把兩個位元都設起來，而且**先河後樹**（`loc_10DC8` 在
// `loc_10DC1` 之後）。
func (e *Editor) SmoothEverything() {
	e.PushSnapshot()
	e.w.ResetRiverEdges()
	e.w.smoothRiverOnly()
	e.w.smoothTreesTwice()
}

// GenerateRandom 是「產生隨機地形」（Ctrl-T）。三個百分比與造島開關
// 照參數對話框收到的值走，樹叢數量走 DOS 編輯器那一式。
func (e *Editor) GenerateRandom(seed uint32, tree, lake, curve int) {
	e.PushSnapshot()
	island := 0
	if e.Island {
		island = 1
	}
	e.w.GenerateMap(seed, TerrainParams{
		TreeLevel:    tree,
		LakeLevel:    lake,
		CurveLevel:   curve,
		CreateIsland: island,
		EditorDOS:    true,
	})
}

// ResetRiverEdges 把每一個**四鄰裡有非水面**的水格打回 REDGE。
//
// 原版 `sub_11A24`，只在「平滑河流」與「全部平滑」之前跑。
// 為什麼需要它：`smoothRiver` 只改寫本來就是 REDGE 的格子，
// 而畫筆畫出來的水塊內部是 CHANNEL 或整片 REDGE，沒有這一步的話
// 岸線算不出來。
//
// ⚠ 邊界的判斷照原版：地圖最外圈那一側的鄰居**不檢查**（x=0 不看左、
// x=119 不看右、y=0 不看上、y=99 不看下），所以貼著邊的水面不會被打回 REDGE。
func (w *World) ResetRiverEdges() {
	const lo, hi = RIVER, 0x14 // 2 與 20：水面與河岸的編號範圍
	isWater := func(x, y int) bool {
		n := int(w.Map[x][y] & LOMASK)
		return n >= lo && n <= hi
	}
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			if !isWater(x, y) {
				continue
			}
			edge := false
			if x > 0 && !isWater(x-1, y) {
				edge = true
			}
			if !edge && x < WorldX-1 && !isWater(x+1, y) {
				edge = true
			}
			if !edge && y > 0 && !isWater(x, y-1) {
				edge = true
			}
			if !edge && y < WorldY-1 && !isWater(x, y+1) {
				edge = true
			}
			if edge {
				w.Map[x][y] = REDGE
			}
		}
	}
}

// smoothRiverOnly／smoothTreesTwice 是 SmoothTerrain 的兩半，
// 讓編輯器能分開叫（原版的「平滑河流」與「平滑樹林」是兩條選單）。
func (w *World) smoothRiverOnly() {
	g := &terrainGen{w: w, p: DefaultTerrainParams()}
	g.smoothRiver()
}

func (w *World) smoothTreesTwice() {
	g := &terrainGen{w: w, p: DefaultTerrainParams()}
	g.smoothTrees()
	g.smoothTrees()
}

// ClearMapTiles 讓呈現層也能把整張圖填成空地（開新城市那一條）。
func (w *World) ClearMapTiles() { w.clearMap() }
