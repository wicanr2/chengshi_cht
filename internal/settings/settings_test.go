package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/i18n"
)

func TestSaveLoadLanguages(t *testing.T) {
	for _, lang := range i18n.Langs {
		p := filepath.Join(t.TempDir(), "nested", "settings.json")
		if err := Save(p, lang); err != nil {
			t.Fatalf("Save(%s): %v", lang, err)
		}
		got, err := Load(p)
		if err != nil {
			t.Fatalf("Load(%s): %v", lang, err)
		}
		if got.Language != lang || got.Version != Version {
			t.Fatalf("Load = %+v，預期 version=%d language=%s", got, Version, lang)
		}
		if info, err := os.Stat(p); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("設定檔權限 = %v, %v，預期 0600", info, err)
		}
	}
}

func TestLoadRejectsInvalidSettings(t *testing.T) {
	for name, body := range map[string]string{
		"broken":   `{`,
		"version":  `{"version":2,"language":"ja"}`,
		"language": `{"version":1,"language":"kl"}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(p); err == nil {
				t.Fatal("不合法設定應該失敗")
			}
		})
	}
}

func TestDefaultPathUsesUserConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "chengshi", "settings.json")
	if p != want {
		t.Fatalf("DefaultPath = %q，預期 %q", p, want)
	}
}
