package assets

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStyleFromConfig 釘住「開機時用哪一套圖形由 SIMCITY.CFG 決定」這條，
// 那是原版的行為（見 StyleFromConfig 的說明）。
func TestStyleFromConfig(t *testing.T) {
	for _, c := range []struct{ name, body, want string }{
		{"手上那份原版設定檔", "Screen Mode: E\nJoystick: N\nGraphics Set: WESTCEGA \n", "west"},
		{"另一個圖形集與模式", "Graphics Set: ASIAMCGA\n", "asia"},
		{"小寫也要認得", "graphics set: MOONTDY\nGraphics Set: MOONTDY\n", "moon"},
		{"基本集：對不上六個前綴就回空", "Graphics Set: CEGADAT\n", ""},
		{"沒有這個欄位", "Screen Mode: E\n", ""},
		{"欄位在但值是空的", "Graphics Set:\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "SIMCITY.CFG"), []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := StyleFromConfig(dir); got != c.want {
				t.Errorf("讀到 %q，應為 %q", got, c.want)
			}
		})
	}
	if got := StyleFromConfig(t.TempDir()); got != "" {
		t.Errorf("沒有設定檔時應回空字串，卻是 %q", got)
	}
}
