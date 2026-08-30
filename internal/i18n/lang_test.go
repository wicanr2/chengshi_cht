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
	for _, l := range []Lang{ZhHant, ZhHans} {
		if n := c.CountLang(l); n < 200 {
			t.Errorf("%s 只有 %d 筆，基本檔應該是全譯的", l, n)
		}
	}
	// 日文只做了標籤那幾段（工具、月份、選單、地圖圖層、地物名…），
	// 訊息與圖片文字走退路。這個數字是**現況**，往上調不要往下調。
	if n := c.CountLang(Ja); n < 150 {
		t.Errorf("日文只有 %d 筆，標籤那幾段應該有 150 筆以上", n)
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
