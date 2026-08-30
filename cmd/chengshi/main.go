// chengshi 是《城市》的遊戲執行檔。
//
// 原版素材不隨本專案散布，玩家要自備一份合法的 SimCity 1.10（DOS）
// 並解開到某個目錄，用 -data 指過去。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/sim"
	"github.com/wicanr2/chengshi_cht/internal/ui"
)

// version 由 tools/release.sh 在連結時填（-X main.version=…）。
// 從原始碼直接跑的話會是 dev。
var version = "dev"

// 城市風格。前綴是原版檔名用的，顯示名寫在 .PGF 的檔頭裡。
// 中文名說明書沒有收，所以先用原名——見 translations/glossary.md 的待補。
//
// base 是「沒有資料片的原始外觀」，圖形檔叫 <模式>DAT.PGF，
// 版面與六個資料片不一樣（見 internal/assets/pgfbase.go）。
var styles = map[string]string{
	"base": "基本",
	"asia": "Ancient Asia",
	"medi": "Medieval Times",
	"west": "Wild West",
	"fusa": "Future USA",
	"feur": "Future Europe",
	"moon": "Moon Colony",
}

// audioUsable 開一個子行程試開音效裝置。
//
// 為什麼不在本行程裡試：`oto.NewContext` 沒有 Close，試完裝置就被佔住，
// Ebiten 自己那個環境反而開不成。子行程結束時作業系統會收回。
func audioUsable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-audio-probe")
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(firstLine(msg))
	}
	return nil
}

// firstLine 只留第一行：ALSA 失敗時會列出十幾個裝置名，那對玩家沒有意義。
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}

func main() {
	data := flag.String("data", "", "解開的 SIMCITY 1.10 目錄（裡面要有 CEGA/、mcga/、DATA/）")
	style := flag.String("style", "base", "城市風格：base 基本／asia／medi／west／fusa／feur／moon")
	mute := flag.Bool("mute", false, "不要音效")
	cam := flag.String("cam", "", "起始鏡頭左上角的格子座標，例如 -cam 54,42（試玩腳本用）")
	probe := flag.Bool("audio-probe", false, "內部用：試開音效裝置後結束（0 ＝ 開得起來）")
	sndTest := flag.Int("sound-test", -1, "內部用：一開始就播第幾段音效（0–7），給錄音驗收")
	seed := flag.Int("seed", 0, "地形亂數種子（0 = 隨機）")
	scen := flag.Int("scenario", 0, "載入第幾個悲情城市（1–8，0 = 新城市）")
	load := flag.String("load", "", "讀取一個城市檔（.cty，原版格式）")
	save := flag.String("save", defaultSavePath(), "Ctrl-S 的存檔位置")
	scale := flag.Float64("scale", 1.0, "視窗縮放倍率")
	demo := flag.Int("demo", 0, "先蓋一座起始城市並快轉這麼多年再開始")
	win := flag.String("window", "", "啟動時開啟的視窗：maps／graphs／budget／eval／about／saveas")
	layer := flag.Int("layer", 0, "地圖視窗的圖層編號（0–10）")
	showVer := flag.Bool("version", false, "印出版本後結束")
	flag.Parse()

	// -audio-probe 是內部用的：只試開一次音效裝置就結束，回傳碼給
	// audioUsable 判斷。要放在所有其他初始化之前，它不需要任何資料。
	if *probe {
		if err := ui.ProbeAudio(); err != nil {
			// 寫 stdout：stderr 已經被圖形函式庫的警告佔滿了
			// （`XGB: Could not get authority info…`），混在一起讀不出原因。
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *showVer {
		fmt.Println("城市（chengshi_cht）", version)
		return
	}

	if *data == "" {
		*data = os.Getenv("CHENGSHI_DATA")
	}
	if *data == "" {
		*data = findDataDir()
	}
	if *data == "" {
		fmt.Fprintln(os.Stderr, `請用 -data 指向解開的 SimCity 1.10 目錄。

本專案不散布原版素材（圖形、音效、劇本檔），玩家必須自備一份合法的原版。
目錄裡應該看得到 CEGA/、mcga/、MONO/、sega/、DATA/、SCENARIO/。

例：chengshi -data "/path/to/SIMCITY 1.10" -style asia

也可以把路徑放進環境變數 CHENGSHI_DATA，或把整個 SIMCITY 1.10 目錄放到
下面任何一個位置，就不必每次都打：

  ./SIMCITY 1.10                                    （執行檔旁邊）
  ~/.local/share/chengshi/SIMCITY 1.10              （Linux）
  ~/Library/Application Support/chengshi/SIMCITY 1.10（macOS）`)
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

	var w *sim.World
	if *load != "" {
		w, err = game.LoadCity(*load)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("讀取城市檔：%s\n", *load)
	} else if *scen > 0 {
		w, err = game.LoadScenario(*data, *scen)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("載入悲情城市：%s\n", game.ScenarioNameZH(*scen))
	} else {
		s := uint32(*seed)
		if s == 0 {
			s = sim.RandomSeed()
		}
		w = sim.NewWorld(s)
		w.GenerateMap(s, sim.DefaultTerrainParams())
		w.DoSimInit()
	}

	var demoX, demoY int
	if *demo > 0 {
		var ok bool
		demoX, demoY, ok = game.BuildStarterCity(w)
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
	g.SetSavePath(*save)
	g.SetVersion(version)
	if *cam != "" {
		var cx, cy int
		if _, err := fmt.Sscanf(*cam, "%d,%d", &cx, &cy); err != nil {
			fmt.Fprintf(os.Stderr, "-cam 要寫成 x,y：%v\n", err)
			os.Exit(2)
		}
		g.SetCamera(cx, cy)
	}
	// 系統選單要靠這兩個才換得了劇本與圖形集（Alt-S）。
	g.SetDataDir(*data, *style)
	// 原版一啟動是招牌畫面（`CEGANTRO.PPF`），不是城市。只有在玩家沒有
	// 指定要玩哪一座城時才走那條路——命令列點名了劇本、存檔、示範城市或
	// 起始鏡頭，就是直接進去，試玩與截圖腳本靠這個。
	if *load == "" && *scen == 0 && *demo == 0 && *win == "" && *cam == "" && *seed == 0 {
		if err := g.LoadTitleScreens(*data); err != nil {
			fmt.Fprintln(os.Stderr, "招牌畫面讀不到（直接進城市）：", err)
		}
	}
	// 音效開不起來不算致命：印一行就繼續。
	//
	// ⚠ 但**必須先在子行程裡試開**，見 ui.ProbeAudio：Ebiten 把音效環境的
	// 錯誤當成致命錯誤，直接在遊戲行程裡開失敗的話，玩家會看到視窗閃一下
	// 就結束。沒有音效裝置的機器（伺服器、容器、有些 WSL）比想像中多。
	if !*mute {
		if err := audioUsable(); err != nil {
			fmt.Fprintf(os.Stderr, "音效沒開起來（遊戲照跑）：%v\n", err)
		} else if err := g.EnableSound(*data, *style); err != nil {
			fmt.Fprintf(os.Stderr, "音效沒開起來（遊戲照跑）：%v\n", err)
		} else if *sndTest >= 0 {
			g.PlaySoundOnce(*sndTest)
		}
	}
	g.ShowScenarioBrief()
	if *win != "" {
		if !g.OpenWindow(*win) {
			fmt.Fprintf(os.Stderr, "不認得的視窗 %q\n", *win)
			os.Exit(2)
		}
		g.SetLayer(*layer)
	}

	ebiten.SetWindowSize(int(float64(ui.CanvasW)**scale), int(float64(ui.CanvasH)**scale))
	ebiten.SetWindowTitle("城市 — 模擬城市繁體中文 remake")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// defaultSavePath 回傳 Ctrl-S 的預設存檔位置。
//
// 刻意不用工作目錄：發行包可能被解在唯讀的位置，玩家也可能從別的目錄
// 啟動。存檔寫不進去這件事，玩家通常是蓋了一小時城市之後按下 Ctrl-S
// 才會發現。改存到 XDG 的資料目錄，那裡一定寫得進去。
func defaultSavePath() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, ".local", "share")
		}
	}
	if dir == "" {
		return "city.cty" // 連家目錄都問不出來時的退路
	}
	return filepath.Join(dir, "chengshi", "city.cty")
}

// findDataDir 找玩家自備的原版目錄。
//
// 從 Finder 或桌面捷徑點開的時候沒有命令列參數，工作目錄也不是執行檔
// 所在的地方——只靠 -data 的話，macOS 的 .app 按下去就是「閃一下沒反應」，
// 而錯誤訊息寫在 stderr，玩家看不到。所以先找幾個約定的位置。
func findDataDir() string {
	var cands []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		cands = append(cands,
			filepath.Join(dir, "SIMCITY 1.10"),
			// .app/Contents/MacOS/chengshi → .app 旁邊
			filepath.Join(dir, "..", "..", "..", "SIMCITY 1.10"))
	}
	cands = append(cands, "SIMCITY 1.10")
	if home, err := os.UserHomeDir(); err == nil {
		cands = append(cands,
			filepath.Join(home, ".local", "share", "chengshi", "SIMCITY 1.10"),
			filepath.Join(home, "Library", "Application Support", "chengshi", "SIMCITY 1.10"))
	}
	for _, c := range cands {
		// 認得出是原版目錄才算：光是同名資料夾不夠。
		if _, err := os.Stat(filepath.Join(c, "DATA")); err == nil {
			return c
		}
	}
	return ""
}
