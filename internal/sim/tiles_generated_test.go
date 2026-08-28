package sim

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tiles.go 是 tools/gen_tiles.py 從 Micropolis 的 headers/sim.h 重產的。
// 這個測試重跑一次工具，確認產物沒有被手改、也沒有跟封存脫節。
//
// 沒有封存時跳過——封存由使用者自備（CLAUDE.md §8），CI 上不會有。
func TestTilesGoMatchesGenerator(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	header := filepath.Join(root, "workplace", "ref", "micropolis",
		"micropolis-activity", "src", "sim", "headers", "sim.h")
	if _, err := os.Stat(header); os.IsNotExist(err) {
		t.Skip("沒有 Micropolis 封存，跳過（使用者自備）")
	}
	gen := filepath.Join(root, "tools", "gen_tiles.py")
	if _, err := os.Stat(gen); os.IsNotExist(err) {
		t.Skip("沒有 tools/gen_tiles.py")
	}
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("容器裡沒有 python3，跳過")
	}

	out, err := exec.Command(py, gen, header).Output()
	if err != nil {
		t.Fatalf("重產失敗：%v", err)
	}
	current, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "tiles.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != strings.TrimSpace(string(current)) {
		t.Error("internal/sim/tiles.go 與 tools/gen_tiles.py 的輸出不同 —— " +
			"產物被手改了，或封存換版了。改語意要改工具再重產，不要手改產物。")
	}
}
