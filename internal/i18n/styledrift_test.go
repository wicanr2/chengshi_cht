package i18n

// 風格包漏改檢查。
//
// 六個圖形集的訊息檔對同一個鍵講同一件事，但**用詞不同**：古代亞洲的
// `Tsunami` 對上基本檔的 `Flood`、未來美國的 `Auto-Disintegrate` 對上
// `Auto-Bulldoze`。翻譯是照鍵複製再逐條改的，所以會出現一種特定的漏改：
//
//	原文與基本檔不同，譯文卻和基本檔一模一樣
//
// 這種漏改**看不出來**——檔案齊、鍵齊、沒有空字串，測試全綠，
// 玩家要切到那個風格、打開那個選單才會發現「怪獸」寫成別的東西。
//
// 2026-08-30 第一次跑這支檢查，古代亞洲抓到六條：
// `Tsunami` 譯成「水災」、`Typhoon` 譯成「龍捲風」、`Dragon Blast` 譯成
// 「空難」、`Water Wheel` 譯成「核能發電廠」、`Auto-Plow` 譯成「全自動整地」、
// 以及棒球比分沒換成日式隊名。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// 允許的差異：只差標點、縮寫或客套字，中文本來就會一樣。
var sameMeaning = regexp.MustCompile(`[ !.,'’]|Cannot|Can't|You |'doze|'Doze`)

func normEN(s string) string {
	return strings.ToLower(sameMeaning.ReplaceAllString(s, ""))
}

// 已判定「原文不同但中文相同是對的」的例外，每一條要寫理由。
var driftOK = map[string]string{
	"west/20.3": "Twister 與 Tornado 是同一種天氣，中文都是龍捲風",
	"moon/18.0": "Auto-'Doze 是 Auto-Bulldoze 的縮寫，中文都是全自動整地",
	"feur/0.86": "只多了 You，語意相同",
	"feur/0.88": "只多了 You，語意相同",
	"moon/0.90": "Can't 'doze 是 Cannot bulldoze 的口語縮寫，語意相同",
}

func TestStyleFilesDoNotInheritBaseWording(t *testing.T) {
	driftCheck(t, ZhHant)
}

// 日文同一道閘門。日文是後補的，補的方式是「照鍵複製再逐條改」——
// 和當初中文一樣，所以會犯一樣的漏改。沒有這一支的話，`asia` 的
// 「水車」在日文會靜靜地退回基本檔的「原子力発電所」，而檔案是齊的、
// 沒有空字串、測試全綠。
func TestStyleFilesDoNotInheritBaseWordingJa(t *testing.T) {
	driftCheck(t, Ja)
}

func driftCheck(t *testing.T, lang Lang) {
	t.Helper()
	dir := os.Getenv("SIMCITY_DATA")
	if dir == "" {
		dir = "../../workplace/dos110/SIMCITY 1.10"
	}
	data := filepath.Join(dir, "DATA")
	baseEN, err := ptfText(filepath.Join(data, "MESSAGE.PTF"))
	if err != nil {
		t.Skipf("沒有原版資料，跳過：%v", err)
	}
	base, err := LoadLang("", lang)
	if err != nil {
		t.Fatal(err)
	}
	for style, file := range styleFile {
		en, err := ptfText(filepath.Join(data, strings.ToUpper(style)+"_MSG.PTF"))
		if err != nil {
			t.Fatalf("%s：%v", style, err)
		}
		c, err := LoadLang(style, lang)
		if err != nil {
			t.Fatalf("%s（%s）：%v", style, file, err)
		}
		for key, text := range en {
			b, ok := baseEN[key]
			if !ok || normEN(text) == normEN(b) {
				continue
			}
			sec, idx := splitKey(key)
			if !c.Has(sec, idx) || !base.Has(sec, idx) {
				continue
			}
			got, bgot := strings.TrimSpace(c.S(sec, idx)), strings.TrimSpace(base.S(sec, idx))
			if got == "" || got != bgot {
				continue
			}
			if _, ok := driftOK[style+"/"+key]; ok {
				continue
			}
			t.Errorf("%s %s（%s）：原文從「%s」換成「%s」，但譯文還是基本檔的「%s」",
				style, key, lang, strings.TrimSpace(b), strings.TrimSpace(text), got)
		}
	}
}

func ptfText(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	secs, err := assets.LoadPTF(raw)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, m := range assets.TextMessages(secs) {
		out[itoa(m.Section)+"."+itoa(m.Index)] = m.Text
	}
	return out, nil
}

func splitKey(k string) (int, int) {
	i := strings.IndexByte(k, '.')
	return atoi(k[:i]), atoi(k[i+1:])
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// TestNoFormatSpecifiersInMessages：譯文不准出現格式符。
//
// 七份 `.PTF` 訊息檔**一個格式符都沒有**（2026-08-30 掃過，唯一的命中是
// 月球那則通關訊息裡的 `50% ownership`，被 `% o` 誤判）。帶變數的模板全在
// 執行檔裡（`%d Fiscal Budget`、`Funds:%-9s` 之類），不走這個目錄。
//
// 所以這裡的不變式很簡單：**訊息的譯文裡不該有 `%`**。
// 譯文裡多一個 `%s` 不會被任何既有測試抓到，但遊戲裡會原樣印出來。
func TestNoFormatSpecifiersInMessages(t *testing.T) {
	fmtRe := regexp.MustCompile(`%[-+ #0]*[0-9]*(?:\.[0-9]+)?[a-zA-Z]`)
	for style := range styleFile {
		check(t, style, fmtRe)
	}
	check(t, "", fmtRe)
}

func check(t *testing.T, style string, re *regexp.Regexp) {
	t.Helper()
	c, err := Load(style)
	if err != nil {
		t.Fatalf("%q：%v", style, err)
	}
	for sec := 0; sec <= 21; sec++ {
		for idx := 0; c.Has(sec, idx); idx++ {
			if m := re.FindString(c.S(sec, idx)); m != "" {
				t.Errorf("%q %d.%d 譯文含格式符 %q：%s", style, sec, idx, m, c.S(sec, idx))
			}
		}
	}
}
