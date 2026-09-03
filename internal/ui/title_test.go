package ui

import (
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
)

func TestTitleButtonLabelsAreLocalizedAndFit(t *testing.T) {
	font, err := LoadFont()
	if err != nil {
		t.Fatal(err)
	}
	wantHant := [4]string{"建立新城市", "載入城市", "選擇劇本", "地形編輯器"}
	for _, lang := range i18n.Langs {
		catalog, err := i18n.LoadLang("west", lang)
		if err != nil {
			t.Fatalf("LoadLang(%s): %v", lang, err)
		}
		g := &Game{font: font, txt: catalog}
		labels := g.titleButtonLabels()
		if lang == i18n.ZhHant && labels != wantHant {
			t.Fatalf("繁中招牌選項 = %q，want %q", labels, wantHant)
		}
		for i, label := range labels {
			if label == "" {
				t.Fatalf("%s 的第 %d 個招牌選項是空字串", lang, i)
			}
			maxWidth := (titleButtons[i].Dx() - 6) * UIScale
			if got := font.Measure(label); got > maxWidth {
				t.Fatalf("%s 的招牌選項 %q 寬 %d，超過文字安全寬 %d", lang, label, got, maxWidth)
			}
		}
	}
}
