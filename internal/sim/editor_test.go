package sim

import "testing"

// 判準全部來自 docs/re/20-terrain-editor.md 讀出來的原版行為，
// 不是「remake 現在的樣子」。

func newEditorForTest() *Editor {
	w := NewWorld(1)
	w.clearMap()
	return NewEditor(w)
}

// 四個畫筆寫下去的 16 位元字，出自工具描述表（dseg:0x2B42）。
func TestEditorPaintCells(t *testing.T) {
	e := newEditorForTest()
	want := map[EditorTool]uint16{
		EdDirt:    0,
		EdTrees:   37 | 0x3000,
		EdRiver:   3,
		EdChannel: 4,
	}
	for tool, cell := range want {
		if got := tool.Cell(); got != cell {
			t.Errorf("工具 %d 寫的字 = %#04x，原版是 %#04x", tool, got, cell)
		}
	}
	if !e.Paint(EdTrees, 10, 10) {
		t.Fatal("在空地上畫樹應該要改到地圖")
	}
	if got := e.World().Map[10][10]; got != want[EdTrees] {
		t.Errorf("Map[10][10] = %#04x，要 %#04x", got, want[EdTrees])
	}
	// 原版 `sub_1EF36`：值一樣就不寫也不記復原。
	if e.Paint(EdTrees, 10, 10) {
		t.Error("同一格再畫同一個地物不應該回報有改動")
	}
	if e.Paint(EdTrees, -1, 0) || e.Paint(EdTrees, WorldX, 0) {
		t.Error("界外不該畫得下去")
	}
}

// 一格一步、復原走得回來；沒得復原時回 false（原版此時發第 7 號音效）。
func TestEditorUndoCells(t *testing.T) {
	e := newEditorForTest()
	if e.CanUndo() {
		t.Fatal("剛開的編輯階段不該有東西可以復原")
	}
	e.Paint(EdTrees, 1, 1)
	e.Paint(EdRiver, 2, 2)
	if !e.Undo() || e.World().Map[2][2] != 0 {
		t.Error("第一次復原應該把 (2,2) 還原成空地")
	}
	if !e.Undo() || e.World().Map[1][1] != 0 {
		t.Error("第二次復原應該把 (1,1) 還原成空地")
	}
	if e.Undo() {
		t.Error("沒東西了還回報復原成功")
	}
}

// 全圖快照：清除地圖之後復原要拿回原本整張圖。
func TestEditorUndoSnapshot(t *testing.T) {
	e := newEditorForTest()
	e.Paint(EdTrees, 5, 5)
	e.Paint(EdTrees, 6, 6)
	e.ClearMap()
	if e.World().Map[5][5] != 0 {
		t.Fatal("清除地圖沒清乾淨")
	}
	if !e.Undo() {
		t.Fatal("清除地圖之後應該可以復原")
	}
	if e.World().Map[5][5] != EdTrees.Cell() || e.World().Map[6][6] != EdTrees.Cell() {
		t.Error("復原沒把整張圖拿回來")
	}
}

// 快照最多四份（`sub_106BA`＋0x10704 的 `cmp word_4BFC2, 4`）；
// 第五份把最舊的擠掉，環的尾巴也跟著跳過它的標記。
func TestEditorSnapshotLimit(t *testing.T) {
	e := newEditorForTest()
	for i := 0; i < 5; i++ {
		e.Paint(EdTrees, i, 0)
		e.ClearMap()
	}
	if len(e.snaps) != editorSnapshots {
		t.Errorf("快照數 = %d，上限是 %d", len(e.snaps), editorSnapshots)
	}
	// 五次清除 ＋ 五次畫，復原一路退回去不該恐慌，也不該卡死。
	for i := 0; i < 32 && e.CanUndo(); i++ {
		e.Undo()
	}
}

// 清除人造物：低十位元 > 37 的整格寫成 0，其餘一個位元都不動。
func TestEditorClearUnnatural(t *testing.T) {
	e := newEditorForTest()
	w := e.World()
	w.Map[1][1] = 37 | 0x3000 // 樹林，留著
	w.Map[2][2] = 4           // 水道，留著
	w.Map[3][3] = 64 | 0x4000 // 道路，清掉
	w.Map[4][4] = 240         // 住宅區，清掉
	e.ClearUnnatural()
	if w.Map[1][1] != 37|0x3000 || w.Map[2][2] != 4 {
		t.Error("自然地物被誤清")
	}
	if w.Map[3][3] != 0 || w.Map[4][4] != 0 {
		t.Error("人造物沒清乾淨（連旗標一起寫 0）")
	}
}

// 油漆桶：帶由**起點那一格**決定；起點已經是要畫的那一類就不動；
// 起點不是自然地物（編號 ≥ 40）就退化成單格畫筆。
func TestEditorFillBands(t *testing.T) {
	e := newEditorForTest()
	w := e.World()

	// 整張空地填成樹：全部變 WOODS。
	if !e.FillAt(EdTrees, 0, 0) {
		t.Fatal("空地填樹應該成立")
	}
	if w.Map[50][50] != EdTrees.Cell() || w.Map[119][99] != EdTrees.Cell() {
		t.Error("填色沒有蓋到整張圖")
	}
	// 樹上再填樹：什麼都不做。
	if e.FillAt(EdTrees, 0, 0) {
		t.Error("同一類再填一次應該什麼都不做")
	}
	// 樹上填河：整張變 REDGE。
	if !e.FillAt(EdRiver, 0, 0) {
		t.Fatal("樹上填河應該成立")
	}
	if w.Map[50][50] != EdRiver.Cell() {
		t.Errorf("填河之後 Map[50][50] = %#04x，要 %#04x", w.Map[50][50], EdRiver.Cell())
	}
	// 水上填河／水道：不做。
	if e.FillAt(EdRiver, 0, 0) || e.FillAt(EdChannel, 0, 0) {
		t.Error("水上填水應該什麼都不做")
	}
	// 起點是道路（≥40）：退化成單格。
	e.ClearMap()
	w.Map[10][10] = 64 // 道路
	e.FillAt(EdTrees, 10, 10)
	if w.Map[10][10] != EdTrees.Cell() {
		t.Error("起點是人造物時應該只改那一格")
	}
	if w.Map[11][10] != 0 {
		t.Error("起點是人造物時不該做整片填色")
	}
}

// 填色只有一個圍起來的區塊會被換掉，牆外不動。
func TestEditorFillBounded(t *testing.T) {
	e := newEditorForTest()
	w := e.World()
	// 用水把 (1,1)–(3,3) 圍起來（水不在空地那個帶裡，所以擋得住）。
	for i := 0; i <= 4; i++ {
		w.Map[i][0], w.Map[i][4] = 4, 4
		w.Map[0][i], w.Map[4][i] = 4, 4
	}
	e.FillAt(EdTrees, 2, 2)
	for x := 1; x <= 3; x++ {
		for y := 1; y <= 3; y++ {
			if w.Map[x][y] != EdTrees.Cell() {
				t.Fatalf("圍牆內的 (%d,%d) 沒被填到", x, y)
			}
		}
	}
	if w.Map[6][6] != 0 {
		t.Error("填色漏出圍牆了")
	}
}

// ResetRiverEdges（原版 `sub_11A24`）：臨接非水面的水格打回 REDGE，
// 地圖最外圈那一側的鄰居不檢查。
func TestResetRiverEdges(t *testing.T) {
	w := NewWorld(1)
	w.clearMap()
	// 一塊 5×5 的水，內部應該留著、外圈變 REDGE。
	for x := 10; x < 15; x++ {
		for y := 10; y < 15; y++ {
			w.Map[x][y] = RIVER
		}
	}
	w.ResetRiverEdges()
	if got := w.Map[12][12]; got != RIVER {
		t.Errorf("水塊正中央變成 %d，應該留著 RIVER", got)
	}
	if got := w.Map[10][12]; got != REDGE {
		t.Errorf("水塊左緣 = %d，應該被打回 REDGE", got)
	}
	// 貼著地圖左緣的水：左鄰不檢查，其餘三鄰也是水 → 不動。
	w2 := NewWorld(1)
	w2.clearMap()
	for y := 0; y < WorldY; y++ {
		w2.Map[0][y] = RIVER
		w2.Map[1][y] = RIVER
	}
	w2.ResetRiverEdges()
	if got := w2.Map[0][50]; got != RIVER {
		t.Errorf("貼著左緣的水 = %d，原版不檢查界外，應該留著 RIVER", got)
	}
	if got := w2.Map[1][50]; got != REDGE {
		t.Errorf("第二欄的水右邊是空地，應該變 REDGE，實際是 %d", got)
	}
}

// 造島是**開關**不是動作：勾起來之後下一次產生地形才造島。
func TestEditorIslandIsToggle(t *testing.T) {
	e := newEditorForTest()
	e.Island = true
	e.GenerateRandom(7, 50, 50, 50)
	edge := 0
	for x := 0; x < WorldX; x++ {
		if int(e.World().Map[x][0]&LOMASK) >= RIVER &&
			int(e.World().Map[x][0]&LOMASK) <= 20 {
			edge++
		}
	}
	if edge < WorldX/2 {
		t.Errorf("造島之後上緣只有 %d 格是水，應該幾乎整排都是海", edge)
	}
}
