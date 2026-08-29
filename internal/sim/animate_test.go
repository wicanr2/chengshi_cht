package sim

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// anitab.go 是 tools/gen_anitab.py 從 Micropolis 的 headers/animtab.h 重產的。
// 同 tiles.go：重跑一次工具，確認產物沒有被手改、也沒有跟封存脫節。
func TestAniTabMatchesGenerator(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	header := filepath.Join(root, "workplace", "ref", "micropolis",
		"micropolis-activity", "src", "sim", "headers", "animtab.h")
	if _, err := os.Stat(header); os.IsNotExist(err) {
		t.Skip("沒有 Micropolis 封存，跳過（使用者自備）")
	}
	gen := filepath.Join(root, "tools", "gen_anitab.py")
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("容器裡沒有 python3，跳過")
	}
	out, err := exec.Command(py, gen, header).Output()
	if err != nil {
		t.Fatalf("重產失敗：%v", err)
	}
	current, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "anitab.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != strings.TrimSpace(string(current)) {
		t.Error("internal/sim/anitab.go 與 tools/gen_anitab.py 的輸出不同 —— " +
			"產物被手改了，或封存換版了。改語意要改工具再重產，不要手改產物。")
	}
}

// TestAnimateTiles 檢查三件事：只動帶 ANIMBIT 的格子、旗標位元原樣保留、
// 火的八格會繞回原點。
func TestAnimateTiles(t *testing.T) {
	w := NewWorld(1)
	// 不帶 ANIMBIT 的格子不動。
	w.Map[0][0] = ROADBASE
	// 火：56…63 循環，aniTile[56] = 57、aniTile[63] = 56。
	w.Map[1][0] = uint16(FIRE) | ANIMBIT | BURNBIT
	// 旗標要原樣留著。
	w.Map[2][0] = uint16(FIRE+7) | ANIMBIT | BULLBIT | ZONEBIT

	w.AnimateTiles()

	if w.Map[0][0] != ROADBASE {
		t.Errorf("沒有 ANIMBIT 的格子被動了：%d", w.Map[0][0])
	}
	if got, want := w.Map[1][0], uint16(FIRE+1)|ANIMBIT|BURNBIT; got != want {
		t.Errorf("火的下一格：%d，應該是 %d", got, want)
	}
	if got, want := w.Map[2][0], uint16(FIRE)|ANIMBIT|BULLBIT|ZONEBIT; got != want {
		t.Errorf("火的最後一格要繞回開頭：%d，應該是 %d", got, want)
	}

	// 走完八次應該回到原點。
	w.Map[3][0] = uint16(FIRE) | ANIMBIT
	for i := 0; i < 8; i++ {
		w.AnimateTiles()
	}
	if got, want := w.Map[3][0], uint16(FIRE)|ANIMBIT; got != want {
		t.Errorf("八格一循環：走完是 %d，應該回到 %d", got, want)
	}
}
