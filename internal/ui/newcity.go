package ui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 「建造新城市」的對話框：市名欄 ＋ 技術等級 ＋ 確定。
//
// 原版的字串在執行檔裡（`SIMCITY.EXE` 映像 0x255b1 起連成一串：
// `Easy` `Medium` `Hard` `HERESVILLE` `Game Play Level` `%-18s` ` `
// `SIMCITY city name:` `OK` `%-18s`），不在 `.PTF` 訊息檔裡——
// 所以譯名走說明書而不是訊息檔。軟體世界說明書 p.11 已經譯過：
// **市名欄**、**技術等級**（簡易／適中／艱難），預設名稱 `HERESVILLE`，
// 市名可鍵入 17 個字。
//
// 版面全部量自原版（`workplace/dosbox/nc-01-after-new.png`，640×350 內容座標）。
const (
	ncX, ncY = 240, 70  // 對話框左上角
	ncW, ncH = 160, 210 // 大小
	ncBorder = 4        // 外框粗細

	ncNameLabelY = 78 // 「市名欄」標題的字格上緣
	ncFieldX     = 245
	ncFieldY     = 95
	ncFieldW     = 150
	ncFieldH     = 14
	ncFieldTextX = 248
	// 欄內文字的字格上緣。原版的欄位字型比介面字型寬（筆畫兩像素），
	// remake 只有一種字型，把 14 列的字格對齊欄位框就好。
	ncFieldTextY = 95

	ncLevelLabelY = 126 // 「技術等級」
	ncRadioX      = 277
	ncRadioY      = 151
	ncRadioPitch  = 28
	ncRadioW      = 14
	ncRadioH      = 20
	ncOptTextX    = 296
	ncOptTextY    = 154

	// ⚠ 確定鈕比原版寬：原版寫 `OK` 兩格就夠，中文「確定」要四格。
	// 中心對齊原版的 312，寬度改成放得下四格再加左右各三像素。
	ncOKX, ncOKY = 293, 235
	ncOKW, ncOKH = 38, 20
	ncOKTextX    = 296
	ncOKTextY    = 238

	// maxCityNameRunes 是市名欄的長度上限。說明書 p.11：「市名可以鍵入
	// 17 個字，但存到磁碟時只取前 8 個字」——後半是 DOS 的 8.3 檔名限制，
	// remake 不套（存檔走另一條路，見 saveas.go）。
	maxCityNameRunes = 17
)

// 原版這個對話框只用三個顏色：外框與字是 EGA 9、按鈕與欄位底是 EGA 11、
// 客戶區是白的。
// 地形編輯器自己設 EGA 的調色盤暫存器，用的不是遊戲那三個色。
// 量自原版（`workplace/dosbox/ter-20-random-terrain.png`）：
// 框與字 (0,0,255)、按鈕底 (0,170,255)、客戶區白。
// ⚠ (0,170,255) **不在 EGA 預設十六色裡**，是 64 色盤裡的一個——
// 每個通道兩位元（0／85／170／255）都合法，所以它是重新載入過的調色盤。
var (
	colTELine = color.RGBA{0x00, 0x00, 0xff, 0xff}
	colTEFill = color.RGBA{0x00, 0xaa, 0xff, 0xff}
)

var (
	colDlgLine = color.RGBA{0x55, 0x55, 0xff, 0xff}
	colDlgFill = color.RGBA{0x55, 0xff, 0xff, 0xff}
	colDlgBG   = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// levelName 是三個技術等級。繁中譯名出自說明書 p.11，其餘語言在 ui.tsv。
var levelKeys = [3]string{"level_easy", "level_medium", "level_hard"}

func (g *Game) levelName(i int) string {
	if i < 0 || i >= len(levelKeys) {
		i = 0
	}
	return g.txt.UI(levelKeys[i])
}

type newCityBox struct {
	name  []rune
	level int
	// terrain 是地形編輯器交下來的三個百分比（terrain_editor.go）。
	// nil 代表走原版遊戲的預設值（三個旋鈕都是 -1，隨機）。
	terrain *sim.TerrainParams
}

// openNewCity 打開對話框。原版是從標題畫面的「建造新城市」進來，
// remake 沒有標題畫面，所以掛在系統選單同一項底下。
func (g *Game) openNewCity() {
	g.newCityTitleBackdrop = false
	g.newCityDlg = &newCityBox{name: []rune("HERESVILLE")}
}

func (g *Game) openNewCityFromTitle() {
	g.openNewCity()
	g.newCityTitleBackdrop = true
}

// handleNewCityKeys 處理對話框的按鍵；回傳 true 代表這一格的鍵盤歸它。
func (g *Game) handleNewCityKeys() bool {
	b := g.newCityDlg
	if b == nil {
		return false
	}
	b.name = ebiten.AppendInputChars(b.name)
	if len(b.name) > maxCityNameRunes {
		b.name = b.name[:maxCityNameRunes]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(b.name) > 0 {
		b.name = b.name[:len(b.name)-1]
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyUp) && b.level > 0:
		b.level--
	case inpututil.IsKeyJustPressed(ebiten.KeyDown) && b.level < 2:
		b.level++
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter),
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter):
		g.startNewCity()
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		g.newCityDlg = nil
		g.newCityTitleBackdrop = false
	}
	return true
}

// handleNewCityMouse 處理三個等級鈕與確定鈕。
func (g *Game) handleNewCityMouse(mx, my int) bool {
	b := g.newCityDlg
	if b == nil {
		return false
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	x, y := mx/UIScale, my/UIScale
	for i := 0; i < 3; i++ {
		ry := ncRadioY + i*ncRadioPitch
		// 判定範圍含標籤：原版的三個選項整列都點得到。
		if x >= ncRadioX && x < ncX+ncW-ncBorder &&
			y >= ry && y < ry+ncRadioH {
			b.level = i
			return true
		}
	}
	if x >= ncOKX && x < ncOKX+ncOKW && y >= ncOKY && y < ncOKY+ncOKH {
		g.startNewCity()
	}
	return true
}

// startNewCity 照對話框的設定產生一座新城市。
//
// 資金隨等級：簡易 $20,000、適中 $10,000、艱難 $5,000
// （Micropolis — src/sim/w_util.c:177 SetGameLevelFunds）。
func (g *Game) startNewCity() {
	b := g.newCityDlg
	s := sim.RandomSeed()
	w := sim.NewWorld(s)
	w.SetGameLevelFunds(b.level)
	if n := strings.TrimSpace(string(b.name)); n != "" {
		w.CityName = n
	}
	p := sim.DefaultTerrainParams()
	if b.terrain != nil {
		p = *b.terrain
	}
	w.GenerateMap(s, p)
	w.DoSimInit()
	g.newCityDlg = nil
	g.newCityTitleBackdrop = false
	g.swapWorld(w)
}

// drawNewCity 畫對話框。座標是原版像素，fill 會乘上 UIScale。
func (g *Game) drawNewCity(dst *ebiten.Image) {
	b := g.newCityDlg
	if b == nil {
		return
	}
	fill(dst, ncX, ncY, ncW, ncH, colDlgLine)
	fill(dst, ncX+ncBorder, ncY+ncBorder,
		ncW-2*ncBorder, ncH-2*ncBorder, colDlgBG)

	g.font.DrawCentered(dst, "市名欄", ncX*UIScale, ncNameLabelY*UIScale,
		ncW*UIScale, colDlgLine)

	// 市名欄：底色青、字白，游標是跟著字尾走的底線。
	fill(dst, ncFieldX, ncFieldY, ncFieldW, ncFieldH, colDlgFill)
	g.font.Draw(dst, string(b.name)+"_",
		ncFieldTextX*UIScale, ncFieldTextY*UIScale, colDlgLine)

	g.font.DrawCentered(dst, "技術等級", ncX*UIScale, ncLevelLabelY*UIScale,
		ncW*UIScale, colDlgLine)

	for i := range levelKeys {
		name := g.levelName(i)
		ry := ncRadioY + i*ncRadioPitch
		fill(dst, ncRadioX, ry, ncRadioW, ncRadioH, colDlgLine)
		fill(dst, ncRadioX+1, ry+1, ncRadioW-2, ncRadioH-2, colDlgBG)
		fill(dst, ncRadioX+3, ry+3, 8, 14, colDlgFill)
		if b.level == i {
			fill(dst, ncRadioX+4, ry+7, 5, 6, colDlgLine)
		}
		g.font.Draw(dst, name, ncOptTextX*UIScale,
			(ncOptTextY+i*ncRadioPitch)*UIScale, colDlgLine)
	}

	fill(dst, ncOKX, ncOKY, ncOKW, ncOKH, colDlgLine)
	fill(dst, ncOKX+1, ncOKY+1, ncOKW-2, ncOKH-2, colDlgFill)
	g.font.Draw(dst, "確定", ncOKTextX*UIScale, ncOKTextY*UIScale, colDlgLine)
}
