package i18n

import "testing"

// 七個風格（含基本檔）都要載得起來，而且譯文筆數一致。
// 少了一份就代表合併工具漏跑，遊戲切到那個風格會整片空白。
func TestAllCatalogsLoad(t *testing.T) {
	styles := []string{"", "asia", "medi", "west", "fusa", "feur", "moon"}
	base := 0
	for _, s := range styles {
		c, err := Load(s)
		if err != nil {
			t.Fatalf("風格 %q 載入失敗：%v", s, err)
		}
		if base == 0 {
			base = c.Count()
		}
		if c.Count() != base {
			t.Errorf("風格 %q 有 %d 條譯文，基本檔有 %d 條 —— 合併可能漏跑",
				s, c.Count(), base)
		}
	}
	if base < 200 {
		t.Errorf("只有 %d 條譯文，太少", base)
	}
	t.Logf("每個風格 %d 條譯文", base)
}

// 風格包要真的換掉口吻。全部沿用基本檔的話就白做了，
// 而且**畫面上看起來完全正常**——只是少了原版的趣味。
func TestStylesOverrideVoice(t *testing.T) {
	base, _ := Load("")
	cases := map[string]string{
		"asia": "主上",
		"medi": "陛下",
		"west": "咱們",
		"moon": "巨蛋",
	}
	for style, want := range cases {
		c, err := Load(style)
		if err != nil {
			t.Fatal(err)
		}
		s := c.S(SecStatus, 0)
		if s == base.S(SecStatus, 0) {
			t.Errorf("風格 %q 的第一句和基本檔一樣 —— 覆寫沒生效", style)
		}
		if !contains(s, want) {
			t.Errorf("風格 %q 的第一句是 %q，應含 %q", style, s, want)
		}
	}
}

// 圖片訊息是多行的。跳脫序列沒處理的話會擠成一行，
// 在畫面上是「一長條跑出視窗」而不是報錯。
func TestPictureMessagesKeepLineBreaks(t *testing.T) {
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	s := c.S(SecPicture, 0)
	if s == "" {
		t.Fatal("第一則圖片訊息是空的")
	}
	if !contains(s, "\n") {
		t.Errorf("圖片訊息沒有換行：%q", s)
	}
}

// 幾個關鍵的鍵要對得上譯名表（說明書的譯法）。
func TestGlossaryTerms(t *testing.T) {
	c, _ := Load("")
	want := map[[2]int]string{
		{SecMonth, 0}:     "一月",
		{SecMonth, 11}:    "十二月",
		{SecDisaster, 3}:  " 龍捲風",
		{SecTileName, 24}: "體育館",
		{SecTileName, 19}: "海港",
		{SecClass, 5}:     "超級都會",
		{SecSpeed, 4}:     " 暫停  0",
	}
	for k, v := range want {
		if got := c.S(k[0], k[1]); got != v {
			t.Errorf("%d.%d = %q，應為 %q", k[0], k[1], got, v)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
