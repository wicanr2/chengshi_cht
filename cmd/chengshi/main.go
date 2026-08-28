// chengshi 是《城市》的遊戲執行檔。
//
// 原版素材不隨本專案散布，玩家要自備一份合法的 SimCity 1.10（DOS）
// 並解開到某個目錄，用 -data 指過去。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/sim"
	"github.com/wicanr2/chengshi_cht/internal/ui"
)

// 六種城市風格。前綴是原版檔名用的，顯示名寫在 .PGF 的檔頭裡。
// 中文名說明書沒有收，所以先用原名——見 translations/glossary.md 的待補。
var styles = map[string]string{
	"asia": "Ancient Asia",
	"medi": "Medieval Times",
	"west": "Wild West",
	"fusa": "Future USA",
	"feur": "Future Europe",
	"moon": "Moon Colony",
}

func main() {
	data := flag.String("data", "", "解開的 SIMCITY 1.10 目錄（裡面要有 CEGA/、mcga/、DATA/）")
	style := flag.String("style", "asia", "城市風格：asia／medi／west／fusa／feur／moon")
	seed := flag.Int("seed", 0, "地形亂數種子（0 = 隨機）")
	scale := flag.Float64("scale", 1.0, "視窗縮放倍率")
	demo := flag.Int("demo", 0, "先蓋一座起始城市並快轉這麼多年再開始")
	flag.Parse()

	if *data == "" {
		fmt.Fprintln(os.Stderr, `請用 -data 指向解開的 SimCity 1.10 目錄。

本專案不散布原版素材（圖形、音效、劇本檔），玩家必須自備一份合法的原版。
目錄裡應該看得到 CEGA/、mcga/、MONO/、sega/、DATA/、SCENARIO/。

例：chengshi -data "/path/to/SIMCITY 1.10" -style asia`)
		os.Exit(2)
	}
	if _, ok := styles[*style]; !ok {
		fmt.Fprintf(os.Stderr, "不認得的風格 %q\n", *style)
		os.Exit(2)
	}

	ts, err := ui.LoadTileSet(*data, *style)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	font, err := ui.LoadFont()
	if err != nil {
		fmt.Fprintln(os.Stderr, "字型載入失敗：", err)
		os.Exit(1)
	}
	// 文字跟著風格走：古代亞洲的發電廠叫「水井」、鐵路叫「人力車道」，
	// 那是原版的設計，不是翻譯自由發揮。
	txt, err := i18n.Load(*style)
	if err != nil {
		fmt.Fprintln(os.Stderr, "文字載入失敗：", err)
		os.Exit(1)
	}

	s := uint32(*seed)
	if s == 0 {
		s = sim.RandomSeed()
	}
	w := sim.NewWorld(s)
	w.GenerateMap(s, sim.DefaultTerrainParams())
	w.DoSimInit()

	var demoX, demoY int
	if *demo > 0 {
		var ok bool
		demoX, demoY, ok = ui.BuildStarterCity(w)
		if !ok {
			fmt.Fprintln(os.Stderr, "這張地圖上找不到夠大的平地，換個 -seed 試試")
			os.Exit(1)
		}
		for i := 0; i < *demo*48*16; i++ {
			w.Frame()
		}
	}

	g := ui.NewGame(w, ts, font, txt)
	if *demo > 0 {
		g.LookAt(demoX+6, demoY+6)
	}

	ebiten.SetWindowSize(int(float64(ui.CanvasW)**scale), int(float64(ui.CanvasH)**scale))
	ebiten.SetWindowTitle("城市 — 模擬城市繁體中文 remake")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
