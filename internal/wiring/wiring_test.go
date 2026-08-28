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

// 表格列：| [`NN-xxx.md`](NN-xxx.md) | 主題 | 狀態 | 引用點／理由 |
var rowRe = regexp.MustCompile(`^\|\s*\[` + "`" + `([0-9A-Za-z._-]+\.md)` + "`" + `\]\([^)]*\)\s*\|([^|]*)\|([^|]*)\|`)

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
			if strings.Contains(body, "docs/re/"+note) {
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

	// 反向：docs/re/ 底下的每份筆記都要在表上（00-wiring-status.md 自己除外）。
	entries, err := os.ReadDir(filepath.Join(root, "docs", "re"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".md") || n == "00-wiring-status.md" {
			continue
		}
		if !listed[n] {
			t.Errorf("docs/re/%s 沒有登記在接線表上", n)
		}
	}
}
