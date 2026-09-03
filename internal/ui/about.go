package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 系統選單裡原本沒接的三項：關於本遊戲、印表、以……檔名儲存。
//
// 「關於本遊戲」不只是禮貌。授權條款（RRSAL-1.0）要求
// 拿到副本的人同時拿到條款與 `Required Notice:` 那一行，而 **APK 沒有地方
// 放文字檔給玩家看**——CLAUDE.md §7 明寫那種情形的正解就是遊戲內的
// 「關於」頁。所以這一頁是發行條件之一，不是裝飾。
//
// 「印表」在 1989 年印的是全市地圖。remake 沒有印表機這個概念，
// 對應物做成**把全市地圖存成 PNG**——這是 remake 自訂的對應，不是原版行為，
// 文件要標明（docs/spec/controls.md）。

// aboutLines 是「關於本遊戲」的內容。
//
// ⚠ 第一行的 `Required Notice:` 是條款要求的，**不要改寫、不要省略**。
// 版本號由 SetVersion 填。
func (g *Game) aboutLines() []string {
	return []string{
		"城市 — 模擬城市繁體中文重製",
		"chengshi_cht " + g.version,
		"",
		"Required Notice: Copyright 2026 Wang Chun-Yu (wicanr2)",
		"https://github.com/wicanr2/chengshi_cht",
		"",
		"授權：RRSAL-1.0（非商業免費，商業洽談）",
		"非商業免費，含修改後散布；商業使用請洽 wicanr2@gmail.com。",
		"全文在發行包的 LICENSE 檔，或上面那個網址。",
		"原版《SimCity》(1989) 由 Will Wright 設計、Maxis 發行。",
		"SimCity 與 Maxis 是 Electronic Arts 的商標，",
		"本專案與 EA 無隸屬關係，商標僅作指示性使用。",
		"",
		"規則層參考 Micropolis（EA 2008 以 GPL-3.0 釋出的原始碼），",
		"讀完寫成規格後以 Go 重寫，未複製其程式碼。",
		"原版執行檔、資料檔、美術、音樂與說明書不隨本專案散布，",
		"玩家需自備一份合法的原版。",
		"中文點陣字取自 Noto Sans CJK TC（SIL Open Font License 1.1）。",
	}
}

// SetVersion 讓進入點把版本號交給「關於」頁。
func (g *Game) SetVersion(v string) { g.version = v }

func (g *Game) drawAboutWindow(dst *ebiten.Image, x, y, w, h int) {
	for i, s := range g.aboutLines() {
		c := colText
		switch {
		case i == 0:
			c = colOn
		case strings.HasPrefix(s, "Required Notice"):
			c = colOn
		case i == 1:
			c = colDim
		}
		g.font.Draw(dst, s, x, y+i*g.font.Line(), c)
	}
}

// printMap 把全市地圖存成 PNG，用的是地圖視窗目前選的圖層。
//
// 這是原版「印表」的對應物，**不是原版行為**：1989 年印的是紙。
// 尺寸取 4 倍（480×400），小到可以直接貼進聊天室、大到看得出分區。
func (g *Game) printMap() {
	const scale = 4
	img := image.NewRGBA(image.Rect(0, 0, sim.WorldX*scale, sim.WorldY*scale))
	for ty := 0; ty < sim.WorldY; ty++ {
		for tx := 0; tx < sim.WorldX; tx++ {
			c := g.layerColor(tx, ty)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.SetRGBA(tx*scale+dx, ty*scale+dy, color.RGBA(c))
				}
			}
		}
	}
	p := g.mapImagePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		g.setMessage("存圖失敗：" + err.Error())
		return
	}
	f, err := os.Create(p)
	if err != nil {
		g.setMessage("存圖失敗：" + err.Error())
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		g.setMessage("存圖失敗：" + err.Error())
		return
	}
	g.setMessage("地圖存成 " + p)
}

// mapImagePath 把地圖圖檔放在存檔旁邊——玩家找得到存檔就找得到它。
func (g *Game) mapImagePath() string {
	dir := "."
	if g.savePath != "" {
		dir = filepath.Dir(g.savePath)
	}
	name := strings.TrimSpace(g.world.CityName)
	if name == "" {
		name = "city"
	}
	return filepath.Join(dir, name+"-地圖.png")
}
