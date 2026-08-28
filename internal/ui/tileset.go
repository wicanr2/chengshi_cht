package ui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/chengshi_cht/internal/assets"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// TileSet 是一組畫得出來的圖塊，來自玩家自備的原版 .PGF。
//
// **本專案不散布原版素材**，所以這裡只負責讀取，不內嵌任何圖形。
// 找不到資料時要給玩家看得懂的中文訊息，不是 panic。
type TileSet struct {
	Style string // 風格顯示名，例如「Ancient Asia」
	Size  int    // 一格的邊長（CEGA 16、MCGA 8）
	Tiles []*ebiten.Image
	// Sprites 是精靈圖形庫，索引與 .PGF 的圖形庫編號一致。
	Sprites [][]*ebiten.Image
}

// 圖形檔的挑選順序。CEGA 是 EGA 640×350、圖塊 16×16，細節最多，
// 所以優先；找不到才退到 8×8 的模式。
var graphicsDirs = []struct {
	dir  string
	ext  string
	tile int
}{
	{"CEGA", ".PGF", 16},
	{"cega", ".pgf", 16},
	{"MONO", ".PGF", 16},
	{"sega", ".pgf", 8},
	{"mcga", ".pgf", 8},
}

// LoadTileSet 從 DOS 1.10 的目錄讀一組圖形。
//
// style 是六個前綴之一（asia／medi／west／fusa／feur／moon）。
func LoadTileSet(dataDir, style string) (*TileSet, error) {
	var lastErr error
	for _, g := range graphicsDirs {
		dir := filepath.Join(dataDir, g.dir)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		// 檔名是 <風格><模式>.PGF，例如 ASIACEGA.PGF、asiamcga.pgf。
		// 大小寫在兩批發行裡不一致，所以逐一比對而不是拼字串。
		ents, err := os.ReadDir(dir)
		if err != nil {
			lastErr = err
			continue
		}
		for _, e := range ents {
			n := strings.ToLower(e.Name())
			if !strings.HasPrefix(n, strings.ToLower(style)) ||
				!strings.HasSuffix(n, ".pgf") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				lastErr = err
				continue
			}
			pgf, err := assets.ParsePGF(raw)
			if err != nil {
				lastErr = err
				continue
			}
			return buildTileSet(pgf)
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("讀取風格 %q 的圖形失敗：%w", style, lastErr)
	}
	return nil, fmt.Errorf("在 %s 底下找不到風格 %q 的 .PGF —— "+
		"請確認那是解開的 SIMCITY 1.10 目錄（裡面應該有 CEGA/、mcga/、DATA/）",
		dataDir, style)
}

func buildTileSet(g *assets.PGF) (*TileSet, error) {
	if len(g.Banks) == 0 {
		return nil, fmt.Errorf("圖形檔裡沒有圖形庫")
	}
	b0 := g.Banks[0]
	if len(b0.Images) != sim.TILE_COUNT {
		return nil, fmt.Errorf("第 0 庫有 %d 張圖，應為 %d —— 這不是地圖圖塊庫",
			len(b0.Images), sim.TILE_COUNT)
	}
	ts := &TileSet{Style: g.Name, Size: b0.Width}
	pal := make([]color.RGBA, 256)
	for i, c := range g.Palette {
		pal[i] = color.RGBA{c.R, c.G, c.B, 255}
	}
	for i := range b0.Images {
		ts.Tiles = append(ts.Tiles, imageFrom(&b0, i, pal))
	}
	// 其餘圖形庫原樣收著，精靈與 UI 面板都在裡面。
	for bi := 1; bi < len(g.Banks); bi++ {
		b := g.Banks[bi]
		var imgs []*ebiten.Image
		for i := range b.Images {
			imgs = append(imgs, imageFrom(&b, i, pal))
		}
		ts.Sprites = append(ts.Sprites, imgs)
	}
	return ts, nil
}

// imageFrom 把一張調色盤圖轉成 Ebiten 影像。
//
// ⚠ 色號 0 當透明。原版靠遮罩庫做去背，但地圖圖塊是滿版的，
// 用 0 當透明對它沒有影響；精靈則需要，否則會拖著一塊黑底走。
func imageFrom(b *assets.PGFBank, i int, pal []color.RGBA) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, b.Width, b.Height))
	px := b.Images[i].Pixels
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			v := px[y*b.Width+x]
			c := pal[v]
			if v == 0 {
				c.A = 0
			}
			img.Set(x, y, c)
		}
	}
	return ebiten.NewImageFromImage(img)
}

// TileImage 回傳一個圖塊編號對應的圖。超出範圍時回第 0 張，
// 不是 nil——畫面缺一格比整個崩潰好處理。
func (t *TileSet) TileImage(n int) *ebiten.Image {
	if n < 0 || n >= len(t.Tiles) {
		return t.Tiles[0]
	}
	return t.Tiles[n]
}

// styleNameZH 是六種城市風格的中文名。
//
// **軟體世界說明書沒有收這六個名字**（它只講基本玩法），所以這是本專案
// 新譯，標記見 translations/glossary.md。原名寫在 .PGF 的檔頭裡。
//
// 電腦玩家那篇回顧提到資料片系列叫「古城風情系列」與「回到未來系列」，
// 那是**資料片的商品名**不是各風格的名字，不能拿來當譯名。
var styleNameZH = map[string]string{
	"Ancient Asia":   "古代亞洲",
	"Medieval Times": "中世紀",
	"Wild West":      "西部拓荒",
	"Future USA":     "未來美國",
	"Future Europe":  "未來歐洲",
	"Moon Colony":    "月球殖民地",
}

// StyleNameZH 回傳風格的中文名；沒收錄就回原名。
func StyleNameZH(s string) string {
	if z, ok := styleNameZH[s]; ok {
		return z
	}
	return s
}
