package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// 地圖視窗（City Form）的畫法。
//
// 原版**不是**一格一個純色方塊，而是一格一張 3×3 的縮圖——泥土有顆粒、
// 樹林是網點、道路是一條線。那 960 張縮圖躺在 .PGF 宣告的圖形庫之後，
// 見 internal/assets/pgfmini.go 與 docs/formats/03-pgf-graphics.md §7。
//
// ⚠ **資料圖層也是畫在縮圖上面的**：原版先畫整張 City Form（`drawAll`），
// 再把值夠高的格子蓋上色階（`maybeDrawRect`，值太低就 return）。
// 所以人口分佈圖上看得到底下的泥土、樹林與水，不是一片純色。

// minimap 是畫好的全市地圖，重畫一次上傳一次。
//
// 逐格 DrawFilledRect 要 12 000 次繪製呼叫；改成先在 CPU 上把
// 360×300 個像素填好、再一次上傳，繪製呼叫只剩一次。
type minimap struct {
	img *ebiten.Image
	buf *image.RGBA
	w   int // 一格幾個像素
	h   int
}

// minimapImage 重畫並回傳全市地圖。回傳的圖每一格佔 mw×mh 像素。
func (g *Game) minimapImage() (*ebiten.Image, int, int) {
	mw, mh := 1, 1
	if g.tiles != nil && g.tiles.Mini != nil {
		mw, mh = g.tiles.Mini.Width, g.tiles.Mini.Height
	}
	if g.mini == nil || g.mini.w != mw || g.mini.h != mh {
		g.mini = &minimap{
			buf: image.NewRGBA(image.Rect(0, 0, sim.WorldX*mw, sim.WorldY*mh)),
			img: ebiten.NewImage(sim.WorldX*mw, sim.WorldY*mh),
			w:   mw, h: mh,
		}
	}
	m := g.mini
	stride := m.buf.Stride
	px := m.buf.Pix
	hasMini := g.tiles != nil && g.tiles.Mini != nil
	for ty := 0; ty < sim.WorldY; ty++ {
		for tx := 0; tx < sim.WorldX; tx++ {
			var src []color.RGBA
			var flat color.RGBA
			// 色階蓋不到的格子畫縮圖，蓋得到的畫純色——與原版
			// 「先 drawAll 再 maybeDrawRect」同一個結果。
			if c, ok := g.overlayColor(tx, ty); ok {
				flat = c
			} else if hasMini {
				src = g.tiles.MiniColors(g.world.TileNum(tx, ty))
			}
			if src == nil && flat == (color.RGBA{}) {
				flat = g.layerColor(tx, ty)
			}
			for y := 0; y < mh; y++ {
				o := (ty*mh+y)*stride + tx*mw*4
				for x := 0; x < mw; x++ {
					c := flat
					if src != nil {
						c = src[y*mw+x]
					}
					px[o] = c.R
					px[o+1] = c.G
					px[o+2] = c.B
					px[o+3] = 0xff
					o += 4
				}
			}
		}
	}
	m.img.WritePixels(px)
	return m.img, mw, mh
}

// drawMinimap 把全市地圖畫到 dst 上。x／y／cell 都是**螢幕**像素，
// cell 是一格佔幾個螢幕像素。
func (g *Game) drawMinimap(dst *ebiten.Image, x, y, cell int) {
	img, mw, mh := g.minimapImage()
	// ⚠ 兩個軸要分開算。sega 的縮圖是 3×1（640×200 放不下 3 列），
	// 用同一個倍率會把整張地圖壓成三分之一高。
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(cell)/float64(mw), float64(cell)/float64(mh))
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(img, op)
}
