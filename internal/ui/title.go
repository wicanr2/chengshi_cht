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

// 招牌上的按鈕框（原版 640×350 座標，量白色框線得到）。
//
// ⚠ 前三個是原版的；**第四個是 remake 加的**（與系統選單那條「地形編輯器」
// 同類）。位置不是隨便挑的：`209–423 × 255–288` 那一塊量出來是**整片招牌綠、
// 沒有任何原版美術**，所以放一個與其他三個同尺寸（214×30）的按鈕蓋不掉東西。
// 高度比前三個矮四像素（26 而不是 30），為的是讓它上下各留兩列綠——
// 原版的「SELECT SCENARIO」上面就是留兩列綠再接深灰外圈，照同一個節奏。
// 這一塊會讓招牌的畫面對拍多一段已知差異，記在 docs/spec/ui-layout.md。
var titleButtons = [4]image.Rectangle{
	image.Rect(95, 183, 309, 213),  // START NEW CITY
	image.Rect(319, 183, 533, 213), // LOAD A CITY
	image.Rect(209, 223, 423, 253), // SELECT SCENARIO
	image.Rect(209, 259, 423, 285), // 地形編輯器 —— remake 加的
}

// titleButtonFill 是招牌按鈕內側的原版色盤綠（目前 NTRO.PPF 實際解碼為
// RGB 0,170,0）。框線與陰影仍保留原圖，只覆蓋固定在圖裡的英文操作文字。
var titleButtonFill = color.RGBA{0x00, 0xaa, 0x00, 0xff}

// titleFill／titleFrame 是**目前顯示模式**下招牌按鈕該用的填色與框色。
// 單色模式的招牌是黑底白字，用招牌綠會在一片黑白裡冒出一塊彩色。
func titleFill() color.RGBA {
	if chromeSelDither { // 單色（見 chrome.go）
		return colInk
	}
	return titleButtonFill
}

func (g *Game) titleButtonLabels() [4]string {
	return [4]string{
		g.txt.UI("title_new_city"),
		g.txt.UI("title_load_city"),
		g.txt.UI("title_scenario"),
		g.txt.UI("title_terrain"),
	}
}

// 招牌按鈕的外框配色，逐像素量自原版的「SELECT SCENARIO」那一顆
// （x 209–423、y 223–253）：外圈兩像素深灰，內側左與上是白、右與下是淺灰，
// 中間招牌綠。第四顆是 remake 加的，得自己照這個配方畫一顆一樣的。
var (
	titleBtnDark  = color.RGBA{0x55, 0x55, 0x55, 0xff}
	titleBtnLight = color.RGBA{0xff, 0xff, 0xff, 0xff}
	titleBtnShade = color.RGBA{0xaa, 0xaa, 0xaa, 0xff}
)

// drawTitleAddedButton 畫第四顆按鈕的框。前三顆的框是原版美術自己畫的，
// 這裡只補 remake 加的那一顆。
func drawTitleAddedButton(dst *ebiten.Image, r image.Rectangle) {
	x, y, w, h := r.Min.X, r.Min.Y, r.Dx(), r.Dy()
	// 外圈深灰兩像素。
	dark, light, shade := titleBtnDark, titleBtnLight, titleBtnShade
	if chromeSelDither {
		dark, light, shade = colInkLight, colInkLight, colInkLight
	}
	fill(dst, x-2, y-2, w+6, 2, dark)
	fill(dst, x-2, y+h, w+6, 2, dark)
	fill(dst, x-2, y-2, 2, h+4, dark)
	fill(dst, x+w+2, y-2, 2, h+4, dark)
	// 內側：底先鋪綠，再壓左上白、右下淺灰。
	fill(dst, x, y, w+2, h, titleFill())
	fill(dst, x, y, w, 2, light)
	fill(dst, x, y, 2, h, light)
	fill(dst, x+w, y, 2, h, shade)
	fill(dst, x+2, y+h-2, w, 2, shade)
}

func (g *Game) drawTitleButtonLabels(dst *ebiten.Image) {
	const inset = 3
	drawTitleAddedButton(dst, titleButtons[3])
	labels := g.titleButtonLabels()
	for i, r := range titleButtons {
		x := (r.Min.X + inset) * UIScale
		y := (r.Min.Y + inset) * UIScale
		w := (r.Dx() - inset*2) * UIScale
		h := (r.Dy() - inset*2) * UIScale
		vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), titleFill(), false)
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
	ntro, err := loadPPF(dir, "NTRO", g.mode)
	if err != nil {
		return err
	}
	scen, err := loadPPF(dir, "SCEN", g.mode)
	if err != nil {
		return err
	}
	g.titlePic = ebiten.NewImageFromImage(ntro)
	g.scenPic = ebiten.NewImageFromImage(scen)
	g.screen = scrTitle
	return nil
}

// loadPPF 讀 `*NTRO.PPF`／`*SCEN.PPF`。檔名的大小寫在不同的重打包版本裡
// 不一致，所以掃目錄比對後綴，不寫死檔名。
//
// **先找選定顯示模式的目錄，尺寸不是 640×350 就退回 CEGA 的。**
//
// ⚠ 退回不是偷懶，是必要的：招牌上三個按鈕的**點擊區**
// （`titleButtons`）是照 CEGA 那幅 640×350 的美術量出來的，而
// sega／tdy／mcga／CGA 的招牌是 320×200 或 640×200——拉大填滿之後
// 按鈕會整個錯位，而畫面看起來只是「圖有點糊」。單色那一幅本來就是
// 640×350，所以換得過去。
func loadPPF(dir, suffix, mode string) (image.Image, error) {
	// as 是 `.PPF` 的顯示模式名。**單色與 CGA 的長度分不出版面**
	// （兩者每列都是 80 個位元組），一定要指名，見 assets.ParsePPFAs。
	try := func(sub, as string) (image.Image, error) {
		d := filepath.Join(dir, sub)
		ents, err := os.ReadDir(d)
		if err != nil {
			return nil, err
		}
		for _, e := range ents {
			n := strings.ToUpper(e.Name())
			if strings.HasSuffix(n, suffix+".PPF") {
				raw, err := os.ReadFile(filepath.Join(d, e.Name()))
				if err != nil {
					return nil, err
				}
				if as == "" {
					return assets.LoadPPF(raw, nil)
				}
				body, err := assets.DecompressLZSS(raw)
				if err != nil {
					return nil, err
				}
				return assets.ParsePPFAs(body, nil, as)
			}
		}
		return nil, os.ErrNotExist
	}
	if mode != "" && !strings.EqualFold(mode, ModeCEGA) {
		for _, g := range graphicsDirs {
			if !strings.EqualFold(g.dir, mode) {
				continue
			}
			if im, err := try(g.dir, strings.ToLower(mode)); err == nil {
				// 只有**滿版寬**的才換得過去，高度可以短一點
				// （單色的招牌是 640×336、劇本選單 640×348，
				// 兩幅都比 CEGA 的 640×350 矮，畫面下緣會露出桌面色）。
				if im.Bounds().Dx() == OrigW {
					return im, nil
				}
			}
		}
	}
	return try("CEGA", "")
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
	// ⚠ **短的招牌要貼齊下緣，不是上緣。** 單色那兩幅比 CEGA 矮
	// （招牌 640×336、劇本選單 640×348，CEGA 是 640×350），而少掉的
	// 是**上面**那幾列——貼齊上緣的話整幅圖往上跑十四列，
	// 三顆按鈕的點擊區就全部落在按鈕下方，而畫面看起來只是
	// 「中文標籤有點偏低」。
	dy := OrigH - pic.Bounds().Dy()
	if dy < 0 {
		dy = 0
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(UIScale, UIScale)
	op.GeoM.Translate(0, float64(dy*UIScale))
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
	case p.In(titleButtons[3]): // 地形編輯器（remake 加的）
		g.screen = scrPlay
		g.openTerrainScreen()
		g.terrain.fromTitle = true
	}
	return true
}
