package ui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 玩家看得到的字不准寫死在原始碼裡。
//
// 這一條沒有測試釘著的話，**在繁體中文底下完全看不出來**——字是對的、
// 版面是對的、所有測試都綠。要切到日文或英文、而且剛好走到那一行，
// 才會在畫面上看到一句中文。2026-09-04 一次抓到二十幾處：劇本簡介的
// 按鈕、資金帶的「模擬速度：」、預算視窗的欄標題、查詢面板的程度詞、
// 系統選單的圖形集與顯示模式名、兩個關於頁。
//
// 判準是**送進畫面的那三個出口**：`setMessage`（資金帶）與
// `font.Draw`／`font.DrawCentered`。錯誤訊息（`fmt.Errorf`）與寫進報告檔的
// 字不算，那些不會出現在遊戲畫面上。
func TestNoHardcodedChineseOnScreen(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("找不到原始碼：%v", err)
	}
	// 送進畫面的三個出口，後面第一個參數是字面字串的話就抓出來。
	call := regexp.MustCompile(`(?:setMessage|font\.Draw|font\.DrawCentered)\((?:dst, )?"([^"]*)"`)

	// 正對照：這支測試最可能的失效方式是**判準抓不到東西**（改了呼叫寫法、
	// 正規式的分支順序吃掉較長的那個），而那會讓它永遠綠。所以先拿三個
	// 真的出過事的寫法餵它一次。
	for _, sample := range []string{
		`		g.setMessage("模擬速度：" + g.speedName(n))`,
		`	g.font.Draw(dst, "繼續", (buttonX+3)*UIScale, y, colDlgLine)`,
		`	g.font.DrawCentered(dst, "市名欄", ncX*UIScale, y, w, colDlgLine)`,
	} {
		m := call.FindStringSubmatch(sample)
		if m == nil || !hasCJK(m[1]) {
			t.Fatalf("判準抓不到已知的違規寫法，這支測試等於沒在測：%s", sample)
		}
	}

	n := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, m := range call.FindAllStringSubmatch(line, -1) {
				if !hasCJK(m[1]) {
					continue
				}
				n++
				t.Errorf("%s:%d 把中文寫死送上畫面：%q —— 譯文要放進 messages/ui.tsv 再用 g.txt.UI 取",
					f, i+1, m[1])
			}
		}
	}
	if n == 0 {
		t.Log("畫面出口沒有寫死的中文")
	}
}

func hasCJK(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
