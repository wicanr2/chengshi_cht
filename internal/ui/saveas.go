package ui

import (
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/wicanr2/chengshi_cht/internal/game"
)

// 「以……檔名儲存」（訊息檔第 17 段第 9 筆）。
//
// 這是系統選單裡唯一需要**文字輸入**的項目，而遊戲其他地方一個輸入框都沒有
// ——所以它一直沒接。這裡做一個最小的輸入列：可打字、退格、Enter 存檔、
// Esc 取消。
//
// ⚠ 中文檔名要能打。`ebiten.AppendInputChars` 給的是已經組好的字元
// （輸入法送出的結果），所以只要不假設一個字元一個位元組就行——
// 用 []rune 存，不要用 string 的 byte 索引。

// maxNameRunes 是檔名長度上限。原版的城市名欄位是 60 個位元組
// （docs/formats/01-city-file.md），但這裡存的是**檔案路徑**不是城市名，
// 上限只是防呆。
const maxNameRunes = 64

type saveAsBox struct {
	name []rune
}

func (g *Game) openSaveAs() {
	base := "city"
	if g.savePath != "" {
		base = strings.TrimSuffix(filepath.Base(g.savePath),
			filepath.Ext(g.savePath))
	} else if n := strings.TrimSpace(g.world.CityName); n != "" {
		base = n
	}
	g.saveAs = &saveAsBox{name: []rune(base)}
	g.win = winSaveAs
}

// handleSaveAsKeys 處理輸入列的按鍵。
func (g *Game) handleSaveAsKeys() {
	if g.saveAs == nil {
		g.win = winNone
		return
	}
	g.saveAs.name = ebiten.AppendInputChars(g.saveAs.name)
	if len(g.saveAs.name) > maxNameRunes {
		g.saveAs.name = g.saveAs.name[:maxNameRunes]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.saveAs.name) > 0 {
		g.saveAs.name = g.saveAs.name[:len(g.saveAs.name)-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		g.doSaveAs()
	}
}

// doSaveAs 存到輸入的檔名。
//
// ⚠ 只取檔名不取路徑：`filepath.Base` 擋掉 `../` 與絕對路徑。
// 玩家在遊戲裡打的字不該決定寫到磁碟哪裡。
func (g *Game) doSaveAs() {
	name := strings.TrimSpace(string(g.saveAs.name))
	if name == "" {
		g.setMessage("檔名不能是空的")
		return
	}
	name = filepath.Base(name)
	if filepath.Ext(name) == "" {
		name += ".cty"
	}
	dir := "."
	if g.savePath != "" {
		dir = filepath.Dir(g.savePath)
	}
	p := filepath.Join(dir, name)
	if err := game.SaveCity(p, g.world); err != nil {
		g.setMessage("存檔失敗：" + err.Error())
		return
	}
	g.savePath = p
	g.setMessage("已存檔：" + p)
	g.saveAs = nil
	g.win = winNone
}

func (g *Game) drawSaveAsWindow(dst *ebiten.Image, x, y, w, h int) {
	if g.saveAs == nil {
		return
	}
	line := g.font.Line()
	g.font.Draw(dst, "打字輸入檔名，Enter 儲存，Esc 取消", x, y, colDim)
	dir := "."
	if g.savePath != "" {
		dir = filepath.Dir(g.savePath)
	}
	g.font.Draw(dst, "存到："+dir, x, y+line, colDim)
	// 游標用一個底線，跟著字尾走。
	g.font.Draw(dst, string(g.saveAs.name)+"_", x+8, y+line*2, colOn)
	g.font.Draw(dst, "沒有副檔名的話自動補 .cty", x, y+line*3, colDim)
}
