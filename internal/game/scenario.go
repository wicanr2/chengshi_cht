// Package game 是**組裝層**：把 internal/assets 讀到的原版資料接到
// internal/sim 的規則層。
//
// 它不相依 Ebiten，所以測試在無頭環境跑得起來——這一點是刻意的：
// 「劇本載得起來嗎」「起始城市長得起來嗎」這種問題和繪圖無關，
// 不該為了測它們去架 X server。
package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/assets"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// scenarioFile 是八個劇本的檔名，順序與 sim.Scenario 的編號一致
// （1 起算，對應原版訊息檔第 6 段列出的檔名）。
var scenarioFile = [...]string{
	"DULLSVIL.PSN", "SANFRAN.PSN", "HAMBURG.PSN", "BERN.PSN",
	"TOKYO.PSN", "DETROIT.PSN", "BOSTON.PSN", "RIO.PSN",
}

// LoadScenario 從玩家自備的 DOS 目錄讀第 n 個劇本（1–8）。
//
// ⚠ 劇本**不套用檔案裡的純量**（CityTime、資金…）。原版的 LoadScenario
// 只讀七個陣列，其餘由劇本表寫死——所以 `snro.111` 裡殘留的 CityTime
// 從來沒有生效過。細節見 docs/formats/01-city-file.md。
func LoadScenario(dataDir string, n int) (*sim.World, error) {
	if n < 1 || n > len(scenarioFile) {
		return nil, fmt.Errorf("劇本編號 %d 超出範圍（1–8）", n)
	}
	dir := filepath.Join(dataDir, "SCENARIO")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("讀不到 %s：%w —— "+
			"請確認 -data 指向解開的 SIMCITY 1.10 目錄", dir, err)
	}
	want := strings.ToLower(scenarioFile[n-1])
	var path string
	for _, e := range ents {
		if strings.ToLower(e.Name()) == want {
			path = filepath.Join(dir, e.Name())
			break
		}
	}
	if path == "" {
		return nil, fmt.Errorf("在 %s 底下找不到 %s", dir, scenarioFile[n-1])
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	psn, err := assets.LoadPSN(raw)
	if err != nil {
		return nil, fmt.Errorf("%s：%w", scenarioFile[n-1], err)
	}
	cf, err := sim.ParseCityFile(psn.Body)
	if err != nil {
		return nil, fmt.Errorf("%s：%w", scenarioFile[n-1], err)
	}
	w := sim.NewWorld(sim.RandomSeed())
	w.LoadScenarioFile(cf, sim.Scenario(n))
	w.InitSimLoad = 1
	w.DoSimInit()
	return w, nil
}

// ScenarioNameZH 回傳劇本的中文名。譯名來自軟體世界說明書 p.13–14。
func ScenarioNameZH(n int) string {
	names := [...]string{
		"達斯維利　1900", "舊金山　1906", "漢堡　1944", "伯恩　1965",
		"東京　1957", "底特律　1972", "波士頓　2010", "里約熱內盧　2047",
	}
	if n < 1 || n > len(names) {
		return ""
	}
	return names[n-1]
}
