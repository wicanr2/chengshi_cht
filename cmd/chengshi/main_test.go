package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
	"github.com/wicanr2/chengshi_cht/internal/settings"
)

func TestResolveLanguagePrecedence(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := settings.Save(p, i18n.Ja, "dos"); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveLanguage("", p); err != nil || got != i18n.Ja {
		t.Fatalf("設定語言 = %s, %v，預期 ja", got, err)
	}
	if got, err := resolveLanguage("en", p); err != nil || got != i18n.En {
		t.Fatalf("命令列覆蓋 = %s, %v，預期 en", got, err)
	}
	// 覆蓋本次啟動不應改寫檔案。
	if saved, err := settings.Load(p); err != nil || saved.Language != i18n.Ja {
		t.Fatalf("-lang 改寫了持久設定：%+v, %v", saved, err)
	}
}

func TestResolveLanguageFallbacks(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.json")
	if got, err := resolveLanguage("", missing); err != nil || got != i18n.ZhHant {
		t.Fatalf("缺檔 = %s, %v，預期繁體且無錯", got, err)
	}
	broken := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(broken, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveLanguage("", broken); err == nil || got != i18n.ZhHant {
		t.Fatalf("壞檔 = %s, %v，預期繁體並回報錯誤", got, err)
	}
	if got, err := resolveLanguage("klingon", missing); err == nil || got != i18n.ZhHant {
		t.Fatalf("未知命令列語言 = %s, %v", got, err)
	}
}
