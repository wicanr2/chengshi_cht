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
	// Sprites 是精靈圖形庫，索引 i 對應 .PGF 的第 i+1 庫。
	Sprites [][]*ebiten.Image
	// Mini 是地圖視窗（City Form）用的 960 張縮圖，一張 3×3（CEGA／MONO）、
	// 3×1（sega）或 1×1（mcga）。原版的地圖不是純色方塊，見 minimap.go。
	Mini *assets.MiniTiles
	// miniPal 是縮圖已經換好顏色的版本，避免每一格重查調色盤。
	miniPal [][]color.RGBA
	// UI 是介面美術，索引同 Sprites，但**色號 0 不透明**。
	//
	// ⚠ 兩份不能共用。精靈要拿色號 0 當透明（否則拖著一塊黑底走），
	// 而介面圖的黑色是圖本身的一部分——工具盤的格線、按鈕的陰影都是黑的，
	// 當成透明的話畫出來會缺一塊，而且缺得很像「圖解錯了」。
	UI [][]*ebiten.Image
}

// 介面美術在哪一庫。證據：docs/formats/03-pgf-graphics.md §5之二。
const (
	BankToolPalette = 2 // 57×182，2 欄 × 7 列的工具圖示
	BankDemand      = 3 // 46×39，需求指標底圖
	BankGraphBtns   = 4 // 51×102，統計圖視窗的按鈕
	BankMapIcons    = 5 // 26×226，地圖視窗左緣的圖層圖示
	BankRampA       = 6 // 24×72，色階圖例
	BankRampB       = 7 // 24×72，另一種配色
)

// UIImage 回傳第 bank 庫的第 i 張介面圖；沒有就回 nil。
func (t *TileSet) UIImage(bank, i int) *ebiten.Image {
	k := bank - 1
	if k < 0 || k >= len(t.UI) || i < 0 || i >= len(t.UI[k]) {
		return nil
	}
	return t.UI[k][i]
}

// 圖形檔的挑選順序。CEGA 是 EGA 640×350、圖塊 16×16，細節最多，
// 所以優先；找不到才退到 8×8 的模式。
var graphicsDirs = []struct {
	dir  string
	ext  string
	tile int
	bpp  int
}{
	{"CEGA", ".PGF", 16, 4},
	{"cega", ".pgf", 16, 4},
	{"MONO", ".PGF", 16, 1},
	{"sega", ".pgf", 8, 4},
	{"mcga", ".pgf", 8, 8},
}

// StyleBase 是「沒有資料片的原始城市外觀」。
//
// 它的圖形檔叫 <模式>DAT.PGF，版面與六個風格檔不一樣（沒有圖形庫表，
// 表是行內的），所以走另一條解析路徑。見 internal/assets/pgfbase.go。
const StyleBase = "base"

// LoadTileSet 從 DOS 1.10 的目錄讀一組圖形。
//
// style 是六個資料片前綴之一（asia／medi／west／fusa／feur／moon），
// 或 StyleBase（基本外觀）。
func LoadTileSet(dataDir, style string) (*TileSet, error) {
	var lastErr error
	for _, g := range graphicsDirs {
		dir := filepath.Join(dataDir, g.dir)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		if style == StyleBase {
			raw, err := readAnyCase(dir, g.dir+"dat"+g.ext)
			if err != nil {
				lastErr = err
				continue
			}
			pgf, err := assets.LoadPGFBase(raw, g.tile, g.bpp)
			if err != nil {
				lastErr = err
				continue
			}
			return buildTileSet(pgf)
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
	// ⚠ 地圖圖塊要**不透明**：色號 0 是真正的黑（道路標線、建築輪廓），
	// 不是透明。當成透明會讓桌面灰透出來——畫面照樣看得懂，只是每一塊
	// 有黑色的圖塊都被打了洞。
	//
	// 量法：原版截圖的地圖格網上 512 格裡有 504 格**逐位元**等於第 0 庫的
	// 某一張圖塊（`tools/shot_tilescan.py`），所以原版是原樣貼上去的。
	// 改成透明版之後 remake 只剩一半的格子對得上，差的點全部是
	// 「原版 (0,0,0)、remake (170,170,170)」。
	for i := range b0.Images {
		ts.Tiles = append(ts.Tiles, imageFromOpaque(&b0, i, pal))
	}
	// 其餘圖形庫原樣收著，精靈與介面美術都在裡面。兩份，透明處理不同。
	masks := maskBanks(g)
	for bi := 1; bi < len(g.Banks); bi++ {
		b := g.Banks[bi]
		var imgs, opaque []*ebiten.Image
		for i := range b.Images {
			imgs = append(imgs, imageFrom(&b, i, pal, masks[bi]))
			opaque = append(opaque, imageFromOpaque(&b, i, pal))
		}
		ts.Sprites = append(ts.Sprites, imgs)
		ts.UI = append(ts.UI, opaque)
	}
	if g.Mini != nil {
		ts.Mini = g.Mini
		per := g.Mini.Width * g.Mini.Height
		ts.miniPal = make([][]color.RGBA, sim.TILE_COUNT)
		for i := range ts.miniPal {
			src := g.Mini.Tile(i)
			if len(src) != per {
				continue
			}
			row := make([]color.RGBA, per)
			for j, v := range src {
				row[j] = pal[v]
			}
			ts.miniPal[i] = row
		}
	}
	return ts, nil
}

// MiniColors 回傳一個圖塊編號對應的縮圖顏色，長度 Mini.Width*Mini.Height。
// 沒有縮圖（或編號超出範圍）回 nil，呼叫端要退回純色。
func (t *TileSet) MiniColors(n int) []color.RGBA {
	if n < 0 || n >= len(t.miniPal) {
		return nil
	}
	return t.miniPal[n]
}

// maskBanks 找出「哪一庫是哪一庫的遮罩」。
//
// 判準（證據見 docs/formats/03-pgf-graphics.md §4之二）：
// **精靈從第 10 庫起兩兩成對**，後面那一庫是單平面（旗標 0x0100）、
// 尺寸與張數與前一庫相同。8 位元的 mcga 沒有遮罩——256 色可以逐位元組
// 比對色號 0，用不著。
//
// ⚠ 起點是 10 不是 1。第 6、7 庫也是兩個同尺寸的單平面庫（20×70），
// 但它們是**兩張色階圖例**，不是一對美術與遮罩——照尺寸配會配錯。
func maskBanks(g *assets.PGF) map[int]*assets.PGFBank {
	out := map[int]*assets.PGFBank{}
	for i := 10; i+1 < len(g.Banks); {
		a, m := &g.Banks[i], &g.Banks[i+1]
		if m.Flags&pgfSinglePlaneFlag != 0 &&
			a.Width == m.Width && a.Height == m.Height &&
			len(a.Images) == len(m.Images) {
			out[i] = m
			i += 2
			continue
		}
		i++
	}
	return out
}

// pgfSinglePlaneFlag 是「這一庫只有一個平面」的旗標。
const pgfSinglePlaneFlag = 0x0100

// imageFrom 把一張調色盤圖轉成 Ebiten 影像。
//
// ⚠ **有遮罩就用遮罩，不要拿色號 0 當透明。** 精靈裡有真正的黑色——
// 直升機的旋翼、怪獸的輪廓線都是色號 0；把 0 當透明會在精靈身上開洞。
// 原版正是因為這樣才另外存一份遮罩：實測七對裡「遮罩是 1 而美術不是 0」
// 的比例是 0%–4%，而「遮罩是 0 而美術是 0」有 0%–27%——後面那一群就是
// 精靈內部的黑色。
//
// 遮罩的 1 是**透明**。沒有遮罩的庫（地圖圖塊、8 位元的精靈）才退回
// 「色號 0 當透明」。
func imageFrom(b *assets.PGFBank, i int, pal []color.RGBA, mask *assets.PGFBank) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, b.Width, b.Height))
	px := b.Images[i].Pixels
	var mp []uint8
	if mask != nil && i < len(mask.Images) {
		mp = mask.Images[i].Pixels
	}
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			j := y*b.Width + x
			v := px[j]
			c := pal[v]
			if mp != nil {
				if mp[j] != 0 {
					c.A = 0
				}
			} else if v == 0 {
				c.A = 0
			}
			img.Set(x, y, c)
		}
	}
	return ebiten.NewImageFromImage(img)
}

// imageFromOpaque 同 imageFrom，但**不做去背**。介面美術用這個。
func imageFromOpaque(b *assets.PGFBank, i int, pal []color.RGBA) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, b.Width, b.Height))
	px := b.Images[i].Pixels
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			img.Set(x, y, pal[px[y*b.Width+x]])
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
	"基本":             "基本",
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

// readAnyCase 不分大小寫地讀一個檔。兩批 DOS 發行的檔名大小寫不一致
// （`CEGADAT.PGF` 與 `mcgadat.pgf`），拼字串比對會漏掉一半。
func readAnyCase(dir, name string) ([]byte, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range ents {
		if strings.EqualFold(e.Name(), name) {
			return os.ReadFile(filepath.Join(dir, e.Name()))
		}
	}
	return nil, fmt.Errorf("%s 底下沒有 %s", dir, name)
}
