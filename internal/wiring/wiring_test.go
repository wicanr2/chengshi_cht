// Package wiring 只有一個測試：守住 docs/re/00-wiring-status.md 與程式碼的一致。
//
// 它是 CLAUDE.md §0 的第四道閘門。前三道（讀懂、寫成規格、實作）各自都會綠，
// 而結論仍然可以躺在筆記裡沒人用——這個測試雙向都會紅。
package wiring

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 表格列：| [`docs/re/NN-xxx.md`](NN-xxx.md) | 主題 | 狀態 | 引用點／理由 |
// 第一個 capture 是**儲存庫根目錄起算的路徑**，連結目標不管。
var rowRe = regexp.MustCompile(`^\|\s*\[` + "`" + `([0-9A-Za-z._/-]+\.md)` + "`" + `\]\([^)]*\)\s*\|([^|]*)\|([^|]*)\|`)

// 要被接線表涵蓋的文件目錄。
var wiredDirs = []string{filepath.Join("docs", "re"), filepath.Join("docs", "formats")}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("找不到 repo 根目錄（沒有 go.mod）")
		}
		dir = parent
	}
}

// goSources 回傳 internal/ 與 cmd/ 底下所有 .go 檔的內容，以路徑為鍵。
func goSources(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, top := range []string{"internal", "cmd"} {
		base := filepath.Join(root, top)
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		}
		err := filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, p)
			out[rel] = string(b)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestWiringStatus(t *testing.T) {
	root := repoRoot(t)
	tablePath := filepath.Join(root, "docs", "re", "00-wiring-status.md")
	raw, err := os.ReadFile(tablePath)
	if err != nil {
		t.Fatalf("讀不到接線表：%v", err)
	}
	srcs := goSources(t, root)

	listed := map[string]bool{}
	rows := 0
	for _, line := range strings.Split(string(raw), "\n") {
		m := rowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		note, state := m[1], strings.TrimSpace(strings.ReplaceAll(m[3], "*", ""))
		listed[note] = true
		rows++

		var citing []string
		for path, body := range srcs {
			if strings.Contains(body, note) {
				citing = append(citing, path)
			}
		}

		switch state {
		case "已接":
			if len(citing) == 0 {
				t.Errorf("%s 標「已接」，但 internal/ 與 cmd/ 底下沒有任何 .go 引用 docs/re/%s", note, note)
			}
		case "未接":
			if len(citing) > 0 {
				t.Errorf("%s 標「未接」，卻被引用了：%v —— 改成已接", note, citing)
			}
		case "不適用":
			if len(citing) > 0 {
				t.Errorf("%s 標「不適用」，卻被 %v 引用 —— 它其實含可實作的規則，改成已接", note, citing)
			}
		default:
			t.Errorf("%s 的狀態欄是 %q，只能是 已接／未接／不適用", note, state)
		}
	}

	if rows == 0 {
		t.Fatal("接線表一列都沒解析到——格式壞了")
	}

	// 反向：docs/re/ 與 docs/formats/ 底下的每份文件都要在表上
	//（接線表自己除外）。
	for _, dir := range wiredDirs {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatalf("讀不到 %s：%v", dir, err)
		}
		for _, e := range entries {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".md") || n == "00-wiring-status.md" {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(dir, n))
			if !listed[rel] {
				t.Errorf("%s 沒有登記在接線表上", rel)
			}
		}
	}
}

// 試玩腳本自己算「格子座標 → 螢幕座標」，用的是寫死的地圖區原點與放大倍率。
// 那三個數字跟 `internal/ui/classic.go` 的常數是**兩份**，改一邊不會有人提醒
// 另一邊。這個測試比對兩邊的字面值。
//
// 2026-08-30 就這樣壞過一次：`editViewY` 從 54 訂正成 55（54 那一列是地圖區
// 的白色外框，圖塊從 55 開始），腳本沒跟著改，於是每一次點擊都往下偏一像素。
// **只有落在格線附近的那幾下會跨格**，所以症狀是「六格的道路只蓋出三格」，
// 不是「點不到」——而其他二十幾項檢查照樣通過。
//
// ⚠ 這裡用文字比對而不是直接引用常數，因為 `internal/ui` 匯入 Ebiten，
// 而 Ebiten 在沒有顯示裝置的環境會在 init 就 panic，那個套件放不了測試。
func TestPlaytestOriginMatchesLayout(t *testing.T) {
	root := repoRoot(t)
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("讀不到 %s：%v", p, err)
		}
		return string(b)
	}
	src := read(filepath.Join(root, "internal", "ui", "classic.go"))
	sh := read(filepath.Join(root, "tools", "playtest_inner.sh"))

	pick := func(text, pat, what string) string {
		m := regexp.MustCompile(pat).FindStringSubmatch(text)
		if m == nil {
			t.Fatalf("在 %s 裡找不到 %s（樣式 %s）", what, pat, what)
		}
		return m[1]
	}
	for _, c := range []struct{ name, goPat, shPat string }{
		{"UIScale／UIS", `UIScale\s*=\s*(\d+)`, `UIS=(\d+)`},
		{"editViewX／VIEWX", `editViewX\s+=\s+(\d+)`, `VIEWX=(\d+)`},
		{"editViewY／VIEWY", `editViewY\s+=\s+(\d+)`, `VIEWY=(\d+)`},
	} {
		g := pick(src, c.goPat, "internal/ui/classic.go")
		s := pick(sh, c.shPat, "tools/playtest_inner.sh")
		if g != s {
			t.Errorf("%s：Go 是 %s、試玩腳本是 %s —— 兩邊要一致，"+
				"否則試玩的每一次點擊都會偏掉", c.name, g, s)
		}
	}
}
