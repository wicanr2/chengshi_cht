package i18n

import "testing"

// 四種語言都要載得起來，而且**每一種都要真的有東西**。
//
// ⚠ 判準不是「載得起來」——退路會讓任何語言都回得出繁體中文，
// 所以只看 `S()` 有沒有值等於什麼都沒驗。要直接看那個語言自己那一層。
func TestEachLangHasText(t *testing.T) {
	c, err := LoadLang("base", ZhHant)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Lang{ZhHant, ZhHans, Ja} {
		if n := c.CountLang(l); n < 200 {
			t.Errorf("%s 只有 %d 筆，基本檔應該是全譯的", l, n)
		}
	}
	// 基本檔三種語言要**逐筆一樣多**。門檻寫成數字的話，漏掉一整段
	// （狀態訊息 46 筆、圖片訊息 20 筆）仍然過得了關；對齊繁體才擋得住。
	if n, want := c.CountLang(Ja), c.CountLang(ZhHant); n != want {
		t.Errorf("日文有 %d 筆、繁體有 %d 筆——基本檔要逐筆對齊", n, want)
	}
}

// 資料片的日文是**只填「用字和基本檔不同」的那些**，所以筆數本來就少於
// 繁體；判準是覆蓋率不是筆數。真正的閘門是
// `TestStyleFilesDoNotInheritBaseWordingJa`：原文換了字而譯文沒換就會紅。
func TestStyleFilesHaveJapaneseOverrides(t *testing.T) {
	for _, style := range []string{"asia", "medi", "west", "fusa", "feur", "moon"} {
		c, err := LoadLang(style, Ja)
		if err != nil {
			t.Fatalf("%s：%v", style, err)
		}
		if n := c.CountLang(Ja); n < 60 {
			t.Errorf("%s 的日文覆寫只有 %d 筆，六個風格各有六十筆以上的自有用字", style, n)
		}
	}
}

// 介面字串（remake 自己的字）四種語言都要齊。
//
// 這些是版面的骨架——欄位標題、按鈕、提示——缺一筆就會在畫面上留一個
// 鍵名。所以這裡的門檻是**逐筆相同**，不是「有就好」。
func TestUIStringsCompleteInAllLangs(t *testing.T) {
	n := UICount(ZhHant)
	if n < 50 {
		t.Fatalf("介面字串只有 %d 筆，看起來沒載到", n)
	}
	for _, l := range []Lang{ZhHans, Ja, En} {
		if got := UICount(l); got != n {
			t.Errorf("介面字串 %s 有 %d 筆，繁體有 %d 筆——四種語言要一樣齊", l, got, n)
		}
	}
}

// 資料片的覆寫要蓋得過基本檔，否則「水車」會被基本檔的「核能發電廠」蓋掉。
func TestStyleOverridesBeatBase(t *testing.T) {
	c, err := LoadLang("asia", ZhHant)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.S(14, 35); got != "水車" {
		t.Errorf("古代亞洲的第 14 段第 35 筆是 %q，應為「水車」", got)
	}
}

// 語言代號的寬鬆寫法。
func TestParseLang(t *testing.T) {
	for s, want := range map[string]Lang{
		"zh-Hant": ZhHant, "zh_TW": ZhHant, "cht": ZhHant,
		"zh-Hans": ZhHans, "zh_CN": ZhHans,
		"ja": Ja, "JP": Ja, "en": En, "English": En,
	} {
		if got, ok := ParseLang(s); !ok || got != want {
			t.Errorf("ParseLang(%q) = %v %v，應為 %v true", s, got, ok, want)
		}
	}
	if _, ok := ParseLang("kl"); ok {
		t.Error("不認得的代號應該回 false")
	}
}
