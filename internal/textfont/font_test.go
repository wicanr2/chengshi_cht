package textfont

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 字型圖集是產物：字集由 tools/build_font.py 從 translations/ 與
// internal/ui/ 掃出來。這個測試擋住「翻譯加了新字卻忘了重烘」——
// 那種情形下遊戲裡會缺字，而**缺字不會報錯**，只會少一塊。
func TestAtlasCoversAllText(t *testing.T) {
	a, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	root := repoRoot(t)
	missing := map[rune]string{}
	for _, dir := range []string{"translations", "internal/i18n", "internal/ui"} {
		full := filepath.Join(root, dir)
		filepath.Walk(full, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			switch filepath.Ext(p) {
			case ".md", ".toml", ".go":
			default:
				return nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			for _, r := range string(b) {
				if r <= 0x7F || r == 0xFEFF {
					continue
				}
				if _, ok := a.Glyphs[r]; !ok {
					missing[r] = strings.TrimPrefix(p, root+"/")
				}
			}
			return nil
		})
	}
	if len(missing) != 0 {
		var b strings.Builder
		n := 0
		for r, p := range missing {
			b.WriteString(string(r) + "（" + p + "）")
			if n++; n >= 8 {
				break
			}
		}
		t.Errorf("圖集缺 %d 個字，例如 %s —— 跑 tools/font.sh 重烘",
			len(missing), b.String())
	}
}

// 半形與全形的寬度要分開：全部當全形，中英混排會鬆得難看；
// 全部當半形，中文會疊在一起。
func TestGlyphWidths(t *testing.T) {
	a, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Measure("AB"); got != a.Size {
		t.Errorf("兩個半形字寬 %d，應等於一個全形字寬 %d", got, a.Size)
	}
	if got := a.Measure("城市"); got != a.Size*2 {
		t.Errorf("兩個全形字寬 %d，應為 %d", got, a.Size*2)
	}
	// 格子是照原版的字元格訂的：原版一格 8×14 原版像素，畫布放大三倍。
	// 英數一格、中文兩格——所以 `Funds: $20,000` 這種純英數的欄位
	// 寬度與原版相同。改這三個數字等於改掉與原版的對齊，要先量原版。
	const scale = 3
	if a.Size != 16*scale {
		t.Errorf("全形寬 %d，應為 %d（原版兩格）", a.Size, 16*scale)
	}
	if a.Height != 14*scale {
		t.Errorf("字格高 %d，應為 %d（原版一格）", a.Height, 14*scale)
	}
	if got := a.Measure("A"); got != 8*scale {
		t.Errorf("半形寬 %d，應為 %d（原版一格）", got, 8*scale)
	}
}

// 字面必須是繁體。ttc 裡五個地區字面共用大部分字形，挑錯不會壞掉，
// 但「戶」「錄」這類有地區差異的字會出現簡體或日文寫法——
// 而且**看起來只是字型不同**，很難察覺。
func TestFaceIsTraditional(t *testing.T) {
	a, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a.Face, "TC") {
		t.Errorf("字面是 %q，不是繁體（TC）", a.Face)
	}
}

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
			t.Fatal("找不到 repo 根目錄")
		}
		dir = parent
	}
}
