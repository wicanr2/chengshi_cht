// Package textfont 是中文點陣字型的**資料層**：圖集、字寬與量測。
//
// 為什麼要和 internal/ui 分開：Ebiten 的 package init 需要 DISPLAY，
// 所以任何 import 它的套件都不能在無頭環境跑測試。字集覆蓋率與字寬這兩件事
// 和繪圖無關，拆出來就能常態跑，不必為了測字型去架 Xvfb。
package textfont

import (
	"bytes"
	"embed"
	"encoding/json"
	"image"
	_ "image/png"
)

//go:embed assets/font24.png assets/font24.json
var FS embed.FS

// Glyph 記一個字在圖集裡的位置與顯示寬度。
type Glyph struct {
	Index int // 在圖集裡的序號
	Width int // 顯示寬度：全形 24、半形 12
}

// Atlas 是解析好的字型圖集。
type Atlas struct {
	Image  image.Image
	Size   int // 一個全形字的邊長
	Cols   int // 圖集每列幾個字
	Face   string
	Glyphs map[rune]Glyph
}

type meta struct {
	Size   int              `json:"size"`
	Cols   int              `json:"cols"`
	Face   string           `json:"face"`
	Glyphs map[string][]int `json:"glyphs"`
}

// Load 讀進內嵌的圖集。
func Load() (*Atlas, error) {
	raw, err := FS.ReadFile("assets/font24.png")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	mRaw, err := FS.ReadFile("assets/font24.json")
	if err != nil {
		return nil, err
	}
	var m meta
	if err := json.Unmarshal(mRaw, &m); err != nil {
		return nil, err
	}
	a := &Atlas{
		Image:  img,
		Size:   m.Size,
		Cols:   m.Cols,
		Face:   m.Face,
		Glyphs: make(map[rune]Glyph, len(m.Glyphs)),
	}
	for s, v := range m.Glyphs {
		for _, r := range s {
			a.Glyphs[r] = Glyph{Index: v[0], Width: v[1]}
			break
		}
	}
	return a, nil
}

// Measure 算一段文字的像素寬度。沒烘進圖集的字按全形算。
func (a *Atlas) Measure(s string) int {
	w := 0
	for _, r := range s {
		if g, ok := a.Glyphs[r]; ok {
			w += g.Width
		} else {
			w += a.Size
		}
	}
	return w
}
