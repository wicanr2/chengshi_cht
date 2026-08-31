package ui

import (
	"errors"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

func TestShowCityFormRaisesExistingWindow(t *testing.T) {
	g := &Game{win: winGraphs, mapClosed: true, editFront: true}
	g.showCityForm()
	if g.win != winNone || g.mapClosed || g.editFront {
		t.Fatalf("City Form 沒有回到最前面：win=%v closed=%v editFront=%v",
			g.win, g.mapClosed, g.editFront)
	}
}

func TestOpenMapsUsesExistingCityForm(t *testing.T) {
	g := &Game{win: winBudget, mapClosed: true, editFront: true}
	if !g.OpenWindow("maps") {
		t.Fatal("OpenWindow(maps) 回報失敗")
	}
	if g.win != winNone || g.mapClosed || g.editFront {
		t.Fatalf("OpenWindow(maps) 另開了視窗或沒有叫回 City Form：win=%v closed=%v editFront=%v",
			g.win, g.mapClosed, g.editFront)
	}
}

func TestDataWindowGeometryMatchesDOS(t *testing.T) {
	want := map[window][4]int{
		winGraphs: {240, 103, 304, 125},
		winBudget: {171, 27, 285, 309},
		winEval:   {39, 70, 513, 196},
	}
	for w, box := range want {
		r := winRect[w]
		got := [4]int{r.x, r.y, r.w, r.h}
		if got != box {
			t.Errorf("視窗 %v 幾何 = %v，DOS 原版量測 = %v", w, got, box)
		}
	}
}

func TestNewCityTitleBackdropOnlyForTitlePath(t *testing.T) {
	g := &Game{}
	g.openNewCityFromTitle()
	if g.newCityDlg == nil || !g.newCityTitleBackdrop {
		t.Fatal("招牌的新城市入口沒有保留原版灰色桌面背景")
	}
	g.openNewCity()
	if g.newCityTitleBackdrop {
		t.Fatal("遊戲內重新建城不應清掉目前城市背景")
	}
}

func TestSystemMenuKeepsOriginalRowsAndAppendsSettings(t *testing.T) {
	txt, err := i18n.LoadLang("base", i18n.ZhHant)
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{txt: txt}
	items := g.menuEntries(0)
	if len(items) != 15 {
		t.Fatalf("SYSTEM 有 %d 列，預期原版 13 列加 2 列 remake 擴充", len(items))
	}
	if items[13] != "-" || items[14] != txt.UI("settings_title") {
		t.Fatalf("SYSTEM 擴充尾端 = %q, %q", items[13], items[14])
	}
	// 原版最後一列仍停在索引 12；不能因新增設定把 Exit 的動作錯位。
	if items[12] != trimMenu(txt.S(i18n.SecSysMenu, 12)) {
		t.Fatalf("原版 Exit 列被改動：%q", items[12])
	}
	g.pickSystem(14)
	if !g.openLangNext {
		t.Fatal("SYSTEM→設定沒有排入畫面末端狀態轉移")
	}
}

func TestLanguageSettingsHighlightsCurrentLanguage(t *testing.T) {
	g := &Game{lang: i18n.Ja}
	g.openLangSettings()
	if g.win != winLangSel || g.sysRow != 2 {
		t.Fatalf("日文設定視窗 = win %v row %d，預期 winLangSel row 2", g.win, g.sysRow)
	}
}

func TestSpeedMenuMouseHitUsesRenderedRows(t *testing.T) {
	txt, err := i18n.LoadLang("base", i18n.ZhHant)
	if err != nil {
		t.Fatal(err)
	}
	font, err := LoadFont()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{txt: txt, font: font, world: sim.NewWorld(1), win: winSpeed}
	x, y, _, _ := g.winFrame()
	innerX, innerY := x+6*UIScale, y+16*UIScale
	for want := 0; want < g.sysMenuLen(); want++ {
		mx := innerX + 20
		my := innerY + g.font.Line()*(want+1) + g.font.Line()/2
		if got := g.sysMenuHit(mx, my); got != want {
			t.Errorf("速度列 %d 的滑鼠命中 = %d", want, got)
		}
	}
	if got := g.sysMenuHit(innerX+20, innerY+g.font.Line()/2); got != -1 {
		t.Fatalf("操作提示被誤認為速度列 %d", got)
	}
}

func TestSpeedMenuMousePickChangesSpeedAndCloses(t *testing.T) {
	txt, err := i18n.LoadLang("base", i18n.ZhHant)
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{txt: txt, world: sim.NewWorld(1), win: winSpeed}
	g.sysMenuPick(2) // 第 19 段第 2 列是普通速度。
	if g.win != winNone || g.speedLevel != 2 || g.world.SimSpeed != 2 {
		t.Fatalf("滑鼠選普通速度後 win=%v level=%d SimSpeed=%d",
			g.win, g.speedLevel, g.world.SimSpeed)
	}
}

func TestLanguageSelectionPersistsAndUpdatesCatalog(t *testing.T) {
	txt, err := i18n.LoadLang("base", i18n.ZhHant)
	if err != nil {
		t.Fatal(err)
	}
	var saved i18n.Lang
	g := &Game{txt: txt, lang: i18n.ZhHant, saveLang: func(l i18n.Lang) error {
		saved = l
		return nil
	}}
	g.setLang(i18n.Ja)
	if g.lang != i18n.Ja || txt.Lang() != i18n.Ja || saved != i18n.Ja {
		t.Fatalf("語言切換未完成：game=%s catalog=%s saved=%s", g.lang, txt.Lang(), saved)
	}
}

func TestSystemSettingsPathSelectsEnglishAndPersists(t *testing.T) {
	txt, err := i18n.LoadLang("base", i18n.ZhHant)
	if err != nil {
		t.Fatal(err)
	}
	var saved i18n.Lang
	g := &Game{txt: txt, lang: i18n.ZhHant, saveLang: func(l i18n.Lang) error {
		saved = l
		return nil
	}}
	g.pickSystem(14)
	if !g.openLangNext {
		t.Fatal("SYSTEM→設定沒有排入開窗")
	}
	g.openLangNext = false
	g.openLangSettings()
	g.sysMenuPick(3)
	if g.win != winNone || g.lang != i18n.En || txt.Lang() != i18n.En || saved != i18n.En {
		t.Fatalf("完整設定路徑未完成：win=%v game=%s catalog=%s saved=%s",
			g.win, g.lang, txt.Lang(), saved)
	}
}

func TestLanguageSaveFailureIsVisible(t *testing.T) {
	txt, err := i18n.LoadLang("base", i18n.En)
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{txt: txt, lang: i18n.En, saveLang: func(i18n.Lang) error {
		return errors.New("disk full")
	}}
	g.setLang(i18n.ZhHans)
	if g.message == "" {
		t.Fatal("設定寫入失敗沒有顯示給玩家")
	}
}
