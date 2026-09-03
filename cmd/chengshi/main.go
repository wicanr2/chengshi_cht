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
	"github.com/wicanr2/chengshi_cht/internal/settings"
	"github.com/wicanr2/chengshi_cht/internal/sim"
	"github.com/wicanr2/chengshi_cht/internal/assets"
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
	style := flag.String("style", "", "城市風格：base 基本／asia／medi／west／fusa／feur／moon"+
		"（留空 ＝ 照原版 SIMCITY.CFG 的 Graphics Set）")
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
	win := flag.String("window", "", "啟動時開啟的視窗：maps／graphs／budget／eval／about／saveas／newcity／load／language")
	layer := flag.Int("layer", 0, "地圖視窗的圖層編號（0–10）")
	langFlag := flag.String("lang", "", "本次啟動語言：zh-Hant 繁體／zh-Hans 简体／ja 日本語／en English（空白時讀取玩家設定）")
	musicDir := flag.String("music", "", "背景音樂目錄（.ogg／.wav）；不給就找存檔目錄底下的 music/")
	saveFmt := flag.String("save-format", "",
		"存檔版面：dos（128 位元組檔頭，存得住城市名）／bare（27120 裸檔身，餵得進 Micropolis）；空白時讀玩家設定")
	freezeTest := flag.Int("freeze-test", 0,
		"內部用：跑滿這麼多秒之後故意讓主迴圈卡住，驗收卡死偵測")
	watchdog := flag.Int("watchdog", 10,
		"主迴圈停住幾秒就把現場寫成報告（0 ＝ 關閉）")
	showVer := flag.Bool("version", false, "印出版本後結束")
	flag.Parse()

	// 風格沒指定就照原版設定檔。**原版就是這樣決定開機時用哪一套圖形**
	// （`SETTINGS.EXE` 寫進 `SIMCITY.CFG`），進遊戲之後從
	// SYSTEM → 讀取圖形集 再換。留空而不是預設 "base"，是為了讓
	// 「玩家在原版設定過古城風情」這件事被沿用。
	styleSource := "（-style 指定）"
	if *style == "" {
		if s := assets.StyleFromConfig(*data); s != "" {
			*style, styleSource = s, "（照 SIMCITY.CFG 的 Graphics Set）"
		} else {
			*style, styleSource = ui.StyleBase, "（預設）"
		}
	}

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
	fmt.Printf("圖形集：%s（%s）%s\n", *style, ts.Style, styleSource)
	font, err := ui.LoadFont()
	if err != nil {
		fmt.Fprintln(os.Stderr, "字型載入失敗：", err)
		os.Exit(1)
	}
	// 文字跟著風格走：古代亞洲的發電廠叫「水井」、鐵路叫「人力車道」，
	// 那是原版的設計，不是翻譯自由發揮。
	settingsPath, settingsPathErr := settings.DefaultPath()
	lang, langErr := resolveLanguage(*langFlag, settingsPath)
	if langErr != nil {
		fmt.Fprintf(os.Stderr, "%v（使用繁體中文）\n", langErr)
	}
	txt, err := i18n.LoadLang(*style, lang)
	if err != nil {
		fmt.Fprintln(os.Stderr, "文字載入失敗：", err)
		os.Exit(1)
	}

	var w *sim.World
	cityDir := findCityDir()
	if *load != "" {
		// 相對路徑在 AppImage 底下解析不到（工作目錄不是 AppDir），
		// 所以找不到時回頭到隨附的地圖目錄再找一次。
		if _, err := os.Stat(*load); err != nil && cityDir != "" {
			if alt := filepath.Join(cityDir, filepath.Base(*load)); fileExists(alt) {
				*load = alt
			}
		}
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
		// 快轉。**要先把速度撐到最快再跑**：`SimFrame` 在 `SimSpeed == 0` 時
		// 直接返回，速度 1 與 2 也只讓五分之一、三分之一的畫格生效
		// （`internal/sim/simulate.go:297`）。所以載進來的城市檔存著什麼速度，
		// 就決定了 `-demo N` 實際跑幾年——`KAOHSIUN.CTY` 存的是暫停，
		// 快轉整段是空轉，開起來還停在 1900 年，而畫面看起來完全正常。
		// 跑完把速度還原成檔案裡的值，快轉是工具動作，不改玩家的狀態。
		speed := w.SimSpeed
		w.SimSpeed = 3
		for i := 0; i < *demo*48*16; i++ {
			w.Frame()
		}
		w.SimSpeed = speed
	}

	g := ui.NewGame(w, ts, font, txt)
	if *demo > 0 {
		g.LookAt(demoX+6, demoY+6)
	}
	g.SetSavePath(*save)
	g.SetBundledCityDir(cityDir)
	g.SetVersion(version)
	if *cam != "" {
		var cx, cy int
		if _, err := fmt.Sscanf(*cam, "%d,%d", &cx, &cy); err != nil {
			fmt.Fprintf(os.Stderr, "-cam 要寫成 x,y：%v\n", err)
			os.Exit(2)
		}
		g.SetCamera(cx, cy)
	}
	g.SetLang(lang)
	sf, sfErr := resolveSaveFormat(*saveFmt, settingsPath)
	if sfErr != nil {
		fmt.Fprintf(os.Stderr, "%v（使用 DOS 存檔版面）\n", sfErr)
	}
	g.SetSaveFormat(sf)
	// 卡死偵測：主迴圈停住就把所有 goroutine 的堆疊與最後的狀態寫成報告。
	// 報告放存檔目錄——那是玩家找得到、而且一定可寫的地方。
	g.StartWatchdog(filepath.Dir(*save), time.Duration(*watchdog)*time.Second)
	g.SetFreezeTest(time.Duration(*freezeTest) * time.Second)
	if settingsPathErr == nil {
		g.SetPrefsSaver(func(l i18n.Lang, f string) error {
			return settings.Save(settingsPath, l, f)
		})
	}
	// 系統選單要靠這兩個才換得了劇本與圖形集（Alt-S）。
	// ⚠ 這一行也負責把**英文原文**從玩家那份 `.PTF` 讀進來，
	// 所以要在 SetLang 之後——英文那一層是疊在語言表上面的。
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
			fmt.Fprintf(os.Stderr, "音訊沒開起來（遊戲照跑）：%v\n", err)
		} else {
			if err := g.EnableSound(*data, *style); err != nil {
				fmt.Fprintf(os.Stderr, "音效沒開起來（遊戲照跑）：%v\n", err)
			} else if *sndTest >= 0 {
				g.PlaySoundOnce(*sndTest)
			}
			// 背景音樂：原版沒有，這是 remake 加的。目錄裡有曲目就自動播放；
			// 沒有音訊裝置或用了 -mute 時完全不建立播放器。
			if err := g.EnableMusic(ui.FindMusicDir(*musicDir, *save)); err != nil {
				fmt.Fprintf(os.Stderr, "音樂沒開起來（遊戲照跑）：%v\n", err)
			}
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

// resolveLanguage 的優先序是命令列 → 玩家設定 → 繁體預設。
// 命令列只影響本次啟動；持久化只由遊戲內設定選單觸發。
func resolveLanguage(flagValue, settingsPath string) (i18n.Lang, error) {
	if flagValue != "" {
		if lang, ok := i18n.ParseLang(flagValue); ok {
			return lang, nil
		}
		return i18n.ZhHant, fmt.Errorf("不認得的語言 %q", flagValue)
	}
	if settingsPath == "" {
		return i18n.ZhHant, nil
	}
	saved, err := settings.Load(settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		return i18n.ZhHant, nil
	}
	if err != nil {
		return i18n.ZhHant, fmt.Errorf("設定檔無法讀取：%w", err)
	}
	return saved.Language, nil
}

// resolveSaveFormat 的優先序與語言相同：命令列 → 玩家設定 → 預設。
// 預設是 DOS 版面，因為城市名唯一的容身處就是那個檔頭。
func resolveSaveFormat(flagValue, settingsPath string) (game.SaveFormat, error) {
	if flagValue != "" {
		f, ok := game.ParseSaveFormat(flagValue)
		if !ok {
			return game.SaveWithHeader, fmt.Errorf("不認得的存檔版面 %q", flagValue)
		}
		return f, nil
	}
	if settingsPath == "" {
		return game.SaveWithHeader, nil
	}
	saved, err := settings.Load(settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		return game.SaveWithHeader, nil
	}
	if err != nil {
		return game.SaveWithHeader, fmt.Errorf("設定檔無法讀取：%w", err)
	}
	f, ok := game.ParseSaveFormat(saved.SaveFormat)
	if !ok {
		return game.SaveWithHeader, fmt.Errorf("設定檔的存檔版面 %q 不認得", saved.SaveFormat)
	}
	return f, nil
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
// findCityDir 找隨附的地圖目錄。
//
// 為什麼需要：AppImage 把整包掛在暫存目錄，執行檔在 `usr/bin/`，而 `cities/`
// 在 AppDir 根目錄。玩家的工作目錄是別的地方，所以 `-load cities/X.CTY`
// 這種相對路徑解析不到，遊戲內的讀檔清單也只掃存檔目錄——地圖檔明明在包裡，
// 玩家卻看不到也讀不到。
//
// AppRun 會設 `APPDIR`，那是最可靠的一條；其餘照 findDataDir 那幾種相對位置。
func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func findCityDir() string {
	var cands []string
	if d := os.Getenv("APPDIR"); d != "" {
		cands = append(cands, filepath.Join(d, "cities"))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		cands = append(cands,
			filepath.Join(dir, "cities"),
			// AppImage：usr/bin/chengshi → AppDir 根目錄
			filepath.Join(dir, "..", "..", "cities"),
			// macOS：.app/Contents/MacOS/chengshi → .app 旁邊
			filepath.Join(dir, "..", "..", "..", "cities"))
	}
	cands = append(cands, "cities")
	for _, c := range cands {
		ents, err := os.ReadDir(c)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".cty") {
				return c
			}
		}
	}
	return ""
}

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
