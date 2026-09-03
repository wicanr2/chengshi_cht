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
		if err := Save(p, lang, "dos", ""); err != nil {
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

// 舊設定檔沒有 save_format 這一欄，不能因此整份被判為不支援——
// 那會連玩家的語言選擇一起掉。空值由呼叫端當成預設。
func TestLoadAcceptsSettingsWithoutSaveFormat(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(p, []byte(`{"version":1,"language":"ja"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(p)
	if err != nil {
		t.Fatalf("舊設定檔讀不起來：%v", err)
	}
	if f.Language != i18n.Ja {
		t.Errorf("語言讀成 %q", f.Language)
	}
	if f.SaveFormat != "" {
		t.Errorf("沒有那一欄時應為空字串，得到 %q", f.SaveFormat)
	}
}

// 不認得的存檔格式要擋下來，不能默默寫壞玩家的存檔。
func TestLoadRejectsUnknownSaveFormat(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(p,
		[]byte(`{"version":1,"language":"ja","save_format":"nonsense"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Error("不認得的存檔格式應該回錯")
	}
}
