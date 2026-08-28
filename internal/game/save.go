package game

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// SaveCity 把城市存成原版格式的 `.cty`。
//
// 用原版格式而不是自創格式，是刻意的：存出來的檔案可以拿去餵原版
// SimCity 或 Micropolis，反過來也一樣。remake 的存檔如果自成一格，
// 玩家的城市就被鎖在這個實作裡了。
func SaveCity(path string, w *sim.World) error {
	if filepath.Ext(path) == "" {
		path += ".cty"
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	b := w.ToCityFile().Bytes()
	if len(b) != sim.CityFileSize1x1 {
		return fmt.Errorf("打包出來是 %d 位元組，應為 %d", len(b), sim.CityFileSize1x1)
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadCity 讀一個原版格式的 `.cty`。
func LoadCity(path string) (*sim.World, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) != sim.CityFileSize1x1 {
		return nil, fmt.Errorf("%s 是 %d 位元組，不是 %d —— "+
			"這可能是壓縮過的 DOS 劇本（.PSN）而不是城市檔",
			filepath.Base(path), len(raw), sim.CityFileSize1x1)
	}
	cf, err := sim.ParseCityFile(raw)
	if err != nil {
		return nil, err
	}
	w := sim.NewWorld(sim.RandomSeed())
	w.LoadCityFile(cf)
	w.InitSimLoad = 1
	w.DoSimInit()
	return w, nil
}
