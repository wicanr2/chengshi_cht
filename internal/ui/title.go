package ui

// 標題畫面與劇本選單畫面。
//
// 原版一啟動就是 `<模式>NTRO.PPF` 這幅招牌，上面三個按鈕
// （`START NEW CITY`／`LOAD A CITY`／`SELECT SCENARIO`）；選第三個會換成
// `<模式>SCEN.PPF`，八格縮圖排成 4×2，點一格就開始那個悲情城市。
// 兩幅畫面的解碼寫在 docs/formats/06-ppf-screen.md。
//
// remake 在這之前是**沒有**這條路徑的：一啟動就直接進城市，換劇本得走
// 系統選單。招牌與劇本縮圖是原版自己的美術，不畫等於少一塊。
//
// ⚠ 按鈕與八格的座標是**量原版那兩幅圖**得到的（找白色框線與灰色面板的
// 連續段），不是估的。量法記在 docs/spec/ui-layout.md。

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// screenMode 是「現在在哪一幕」。
type screenMode int

const (
	scrPlay  screenMode = iota // 城市
	scrTitle                   // 招牌
	scrScen                    // 劇本選單
)

// 招牌上三個按鈕的框（原版 640×350 座標，量白色框線得到）。
var titleButtons = [3]image.Rectangle{
	image.Rect(95, 183, 309, 213),  // START NEW CITY
	image.Rect(319, 183, 533, 213), // LOAD A CITY
	image.Rect(209, 223, 423, 253), // SELECT SCENARIO
}

// titleButtonFill 是招牌按鈕內側的原版色盤綠（目前 NTRO.PPF 實際解碼為
// RGB 0,170,0）。框線與陰影仍保留原圖，只覆蓋固定在圖裡的英文操作文字。
var titleButtonFill = color.RGBA{0x00, 0xaa, 0x00, 0xff}

func (g *Game) titleButtonLabels() [3]string {
	return [3]string{
		g.txt.UI("title_new_city"),
		g.txt.UI("title_load_city"),
		g.txt.UI("title_scenario"),
	}
}

func (g *Game) drawTitleButtonLabels(dst *ebiten.Image) {
	const inset = 3
	labels := g.titleButtonLabels()
	for i, r := range titleButtons {
		x := (r.Min.X + inset) * UIScale
		y := (r.Min.Y + inset) * UIScale
		w := (r.Dx() - inset*2) * UIScale
		h := (r.Dy() - inset*2) * UIScale
		vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), titleButtonFill, false)
		g.font.DrawCentered(dst, labels[i], x, y+(h-g.font.Height())/2, w, color.White)
	}
}

// 劇本選單八格的框：四欄兩列，欄距 160、列距 152（量灰色面板得到）。
func scenSlot(i int) image.Rectangle {
	col, row := i%4, i/4
	x, y := 3+col*160, 46+row*152
	return image.Rect(x, y, x+156, y+148)
}

// LoadTitleScreens 讀原版的招牌與劇本選單。讀不到就回錯，呼叫端自己決定
// 要不要直接進城市——沒有這兩個檔並不妨礙遊戲跑。
func (g *Game) LoadTitleScreens(dir string) error {
	ntro, err := loadPPF(dir, "NTRO")
	if err != nil {
		return err
	}
	scen, err := loadPPF(dir, "SCEN")
	if err != nil {
		return err
	}
	g.titlePic = ebiten.NewImageFromImage(ntro)
	g.scenPic = ebiten.NewImageFromImage(scen)
	g.screen = scrTitle
	return nil
}

// loadPPF 找 `CEGA/` 底下的 `*NTRO.PPF`／`*SCEN.PPF`。檔名的大小寫在不同
// 的重打包版本裡不一致，所以掃目錄比對後綴，不寫死檔名。
func loadPPF(dir, suffix string) (image.Image, error) {
	sub := filepath.Join(dir, "CEGA")
	ents, err := os.ReadDir(sub)
	if err != nil {
		return nil, err
	}
	for _, e := range ents {
		n := strings.ToUpper(e.Name())
		if strings.HasSuffix(n, suffix+".PPF") {
			raw, err := os.ReadFile(filepath.Join(sub, e.Name()))
			if err != nil {
				return nil, err
			}
			return assets.LoadPPF(raw, nil)
		}
	}
	return nil, os.ErrNotExist
}

// drawTitle 把招牌或劇本選單鋪滿畫布。兩幅都是 640×350，與原版畫面同尺寸，
// 所以只要整數倍放大。
func (g *Game) drawTitle(dst *ebiten.Image) {
	pic := g.titlePic
	if g.screen == scrScen {
		pic = g.scenPic
	}
	if pic == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(UIScale, UIScale)
	dst.DrawImage(pic, op)
	if g.screen == scrTitle {
		g.drawTitleButtonLabels(dst)
	}
}

// updateTitle 收招牌與劇本選單的輸入。回 true 代表這一幕吃掉了輸入，
// 城市那邊今天不用跑。
func (g *Game) updateTitle() bool {
	if g.screen == scrPlay {
		return false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.screen == scrScen {
			g.screen = scrTitle
		}
		return true
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	mx, my := ebiten.CursorPosition()
	p := image.Pt(mx/UIScale, my/UIScale)
	if g.screen == scrScen {
		for i := 0; i < 8; i++ {
			if p.In(scenSlot(i)) {
				g.screen = scrPlay
				g.loadScenario(i + 1)
				return true
			}
		}
		return true
	}
	switch {
	case p.In(titleButtons[0]): // 重新建造一新城市
		g.screen = scrPlay
		g.openNewCityFromTitle()
	case p.In(titleButtons[1]): // 讀取舊有檔案
		g.screen = scrPlay
		g.load()
	case p.In(titleButtons[2]): // 選悲情城市
		g.screen = scrScen
	}
	return true
}
