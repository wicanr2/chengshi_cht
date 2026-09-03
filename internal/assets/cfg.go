package assets

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GraphicsStyles 是六個資料片圖形集的代號，順序照發行：
// 古城風情三個、回到未來三個。基本集不在內（它沒有前綴）。
//
// **一代的主題圖形集就這六個**，官方只出過兩片資料片；逐檔盤點與
// 兩份 `UPDATE.DAT` 的線索見 docs/formats/00-addon-graphics-sets.md。
var GraphicsStyles = []string{"asia", "medi", "west", "fusa", "feur", "moon"}

// StyleFromConfig 讀原版設定檔 `SIMCITY.CFG` 的 `Graphics Set:`，
// 回傳圖形集代號（小寫）。讀不到、認不得就回空字串，由呼叫端決定退路。
//
// **原版就是這樣決定開機時用哪一套圖形的**：`SETTINGS.EXE` 把選擇寫進
// `SIMCITY.CFG`，遊戲啟動時照著載入；進遊戲之後還能從
// SYSTEM → `Load Graphics` 換（原版截圖 `workplace/dosbox/g-ui-01-system.png`）。
//
// 欄位值同時編了圖形集與顯示模式，例如 `WESTCEGA` ＝ Wild West 的
// EGA 高解析版（`CLAUDE.md` §2.1）。這裡只取前四個字元當圖形集，
// 對不上六個前綴就當成基本集——**不硬猜基本集的值長什麼樣**，
// 手上的樣本只有 `WESTCEGA` 一種。
func StyleFromConfig(dataDir string) string {
	f, err := os.Open(filepath.Join(dataDir, "SIMCITY.CFG"))
	if err != nil {
		// 大小寫在兩批發行裡不一致，換小寫再試一次。
		f, err = os.Open(filepath.Join(dataDir, "simcity.cfg"))
		if err != nil {
			return ""
		}
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		rest, ok := strings.CutPrefix(line, "Graphics Set:")
		if !ok {
			continue
		}
		v := strings.ToLower(strings.TrimSpace(rest))
		if len(v) < 4 {
			return ""
		}
		for _, s := range GraphicsStyles {
			if strings.HasPrefix(v, s) {
				return s
			}
		}
		return ""
	}
	return ""
}
