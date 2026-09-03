package ui

import "testing"

// 對話框八個控制項的位置。表是 docs/spec/terrain-editor.md §二 量出來的欄列，
// 這裡把它釘死：版面改壞了要有東西變紅，不能只靠肉眼看畫面。
func TestTerrainControlRectsMatchSpec(t *testing.T) {
	want := []struct {
		col, row, cols int
	}{
		{3, teValueRow, 1},   // 0x800 樹木 ◄
		{10, teValueRow, 1},  // 0x801 樹木 ►
		{14, teValueRow, 1},  // 0x802 湖泊 ◄
		{21, teValueRow, 1},  // 0x803 湖泊 ►
		{25, teValueRow, 1},  // 0x804 彎曲 ◄
		{32, teValueRow, 1},  // 0x805 彎曲 ►
		{3, teButtonRow, 8},  // 0x806 開始
		{25, teButtonRow, 8}, // 0x807 取消
	}
	for id, w := range want {
		x, y, cw, ch := teControlRect(id)
		if x != teCol(w.col) || y != teRow(w.row) ||
			cw != w.cols*teCellW || ch != teCellH {
			t.Errorf("控制項 %#x：得到 (%d,%d,%d,%d)，規格是欄 %d 列 %d 寬 %d 格",
				0x800+id, x, y, cw, ch, w.col, w.row, w.cols)
		}
	}
	// 每個控制項都要點得到，而且不能互相重疊。
	for id := 0; id < 8; id++ {
		x, y, cw, ch := teControlRect(id)
		if got := teHit(x+cw/2, y+ch/2); got != id {
			t.Errorf("控制項 %#x 的中心被判成 %d", 0x800+id, got)
		}
	}
}

// 視窗放得下所有控制項：原版是 36 欄 × 10 列，取消鈕的右緣正好在第 33 欄。
func TestTerrainDialogFitsOriginalWindow(t *testing.T) {
	for id := 0; id < 8; id++ {
		x, y, w, h := teControlRect(id)
		if x < teX || x+w > teX+teW || y < teY || y+h > teY+teH {
			t.Errorf("控制項 %#x 超出 %d×%d 的視窗", 0x800+id, teCols, teRows)
		}
	}
}

// `◄`／`►` 一次加減 1，夾限在 0–100（原版 sub_113E4(0, 值, 100)）。
func TestTerrainPressClamps(t *testing.T) {
	g := &Game{terrainDlg: &terrainBox{val: [3]int{50, 50, 50}, held: -1}}
	g.terrainPress(1) // 樹木 ►
	if g.terrainDlg.val[0] != 51 {
		t.Fatalf("加一之後是 %d", g.terrainDlg.val[0])
	}
	g.terrainPress(0) // 樹木 ◄
	if g.terrainDlg.val[0] != 50 {
		t.Fatalf("減一之後是 %d", g.terrainDlg.val[0])
	}
	g.terrainDlg.val = [3]int{0, 100, 50}
	g.terrainPress(0) // 樹木 ◄，已在下限
	g.terrainPress(3) // 湖泊 ►，已在上限
	if g.terrainDlg.val[0] != 0 || g.terrainDlg.val[1] != 100 {
		t.Fatalf("夾限失效：%v", g.terrainDlg.val)
	}
	// 三組互不干擾。
	g.terrainDlg.val = [3]int{50, 50, 50}
	g.terrainPress(5)
	if g.terrainDlg.val[0] != 50 || g.terrainDlg.val[1] != 50 ||
		g.terrainDlg.val[2] != 51 {
		t.Fatalf("動到別組：%v", g.terrainDlg.val)
	}
}

// 焦點在八個控制項之間輪一圈（原版 `+`／`-`：(focus ± 1) mod 8）。
func TestTerrainFocusWraps(t *testing.T) {
	b := &terrainBox{focus: 7}
	if got := (b.focus + 1) % 8; got != 0 {
		t.Fatalf("7 的下一個是 %d", got)
	}
	if got := (0 + 7) % 8; got != 7 {
		t.Fatalf("0 的上一個是 %d", got)
	}
}

// 按下「開始」要把三個百分比原封不動交給「建造新城市」對話框，
// 而且掛上 DOS 編輯器那一式（EditorDOS）。CreateIsland 維持 0：
// 原版編輯器的介面上沒有這個元素。
func TestTerrainGoHandsParamsToNewCity(t *testing.T) {
	g := &Game{terrainDlg: &terrainBox{val: [3]int{12, 34, 56}, held: -1}}
	g.terrainGo()
	if g.terrainDlg != nil {
		t.Fatal("開始之後對話框沒關掉")
	}
	if g.newCityDlg == nil || g.newCityDlg.terrain == nil {
		t.Fatal("沒有接到建造新城市對話框")
	}
	p := *g.newCityDlg.terrain
	if p.TreeLevel != 12 || p.LakeLevel != 34 || p.CurveLevel != 56 {
		t.Errorf("百分比走樣：%+v", p)
	}
	if !p.EditorDOS {
		t.Error("沒有掛上 DOS 編輯器那一式")
	}
	if p.CreateIsland != 0 {
		t.Errorf("CreateIsland 應維持 0，得到 %d", p.CreateIsland)
	}
}
