package i18n

// 譯名表與畫面文字的接線檢查。
//
// 有一批玩家看得到的字**不在訊息檔裡**：預算與評估視窗的欄位標題、按鈕、
// 程度詞。它們硬編碼在 `internal/ui/*.go`（`CLAUDE.md` §3.2 說的第三個
// 翻譯來源，出自 DOS 執行檔）。訊息檔那批有 `TestGlossaryTerms` 釘著，
// 這批**沒有任何東西釘著**——改了不會有測試變紅，而譯名表會悄悄變成謊言。
//
// 這支測試讀 `translations/glossary.md` 的「畫面上實際用的字」欄，
// 逐條確認那個字串真的出現在 `internal/ui` 的原始碼裡。
//
// 為什麼那一欄存在：說明書 p.43 的表是**原文對照的說明**不是螢幕標籤
// （「實際撥給金額（＝支付百分比 × 維護需求額）」整串顯然是解釋），
// 而預算視窗的欄標題只放得下四個字。取捨的理由寫在譯名表那段引文裡。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGlossaryScreenTermsAppearInUI(t *testing.T) {
	md, err := os.ReadFile("../../translations/glossary.md")
	if err != nil {
		t.Skipf("讀不到譯名表：%v", err)
	}
	src, err := filepath.Glob("../ui/*.go")
	if err != nil || len(src) == 0 {
		t.Fatalf("找不到 internal/ui 的原始碼：%v", err)
	}
	var all strings.Builder
	for _, f := range src {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		all.Write(b)
	}
	// 有些畫面用字在訊息檔裡（預算視窗的「稅率」「交通」出自 `.PTF` 第 3 段），
	// 不在 Go 原始碼裡。兩邊都算。
	for _, f := range []string{"", "asia", "medi", "west", "fusa", "feur", "moon"} {
		c, err := Load(f)
		if err != nil {
			t.Fatalf("%q：%v", f, err)
		}
		for sec := 0; sec <= 21; sec++ {
			for idx := 0; c.Has(sec, idx); idx++ {
				all.WriteString(c.S(sec, idx))
				all.WriteByte('\n')
			}
		}
	}
	code := all.String()

	// ⚠ 只掃**表頭寫著「畫面上實際用的字」的那張表**。譯名表裡還有別的
	// 四欄表，第四欄是備註不是螢幕文字——不限定範圍的話會拿備註去 grep 原始碼。
	body := ""
	for _, block := range strings.Split(string(md), "\n\n") {
		if strings.Contains(block, "畫面上實際用的字") {
			body = block
			break
		}
	}
	if body == "" {
		t.Fatal("譯名表裡找不到帶「畫面上實際用的字」的表")
	}
	row := regexp.MustCompile(`(?m)^\|([^|\n]+)\|([^|\n]+)\|([^|\n]+)\|([^|\n]+)\|\s*$`)
	n := 0
	for _, m := range row.FindAllStringSubmatch(body, -1) {
		screen := strings.TrimSpace(strings.ReplaceAll(m[4], "**", ""))
		if screen == "" || screen == "畫面上實際用的字" || strings.HasPrefix(screen, "-") {
			continue
		}
		want := screen
		if want == "同左" {
			want = strings.TrimSpace(strings.ReplaceAll(m[2], "**", ""))
		}
		if want == "" {
			continue
		}
		n++
		// 畫面上的字可能被拆成兩行（預算視窗的欄標題就是兩行兩字），
		// 所以整串找不到時，再試「拆成前後兩半」。
		half := len([]rune(want)) / 2
		r := []rune(want)
		split := len(r)%2 == 0 && half > 0 &&
			strings.Contains(code, `"`+string(r[:half])+`"`) &&
			strings.Contains(code, `"`+string(r[half:])+`"`)
		if !strings.Contains(code, want) && !split {
			t.Errorf("譯名表說畫面上用「%s」（原文 %s），但 internal/ui 裡找不到這個字串",
				want, strings.TrimSpace(m[1]))
		}
	}
	if n == 0 {
		t.Error("譯名表裡沒有帶「畫面上實際用的字」的四欄表，這支測試等於沒在測")
	}
	t.Logf("檢查了 %d 條畫面用字", n)
}
