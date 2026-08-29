// Package integration 放跨層的驗收：assets 解出來的東西要能直接餵給 sim。
package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/assets"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

func dosDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(thisFile), "..", "..", "workplace", "dos110", "SIMCITY 1.10")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Skip("沒有解開的 DOS 1.10 資料，跳過（玩家自備）")
	}
	return p
}

// DOS 版的八個劇本解開之後，城市資料要能被 sim.ParseCityFile 直接吃下去。
//
// 這是「**DOS 版的地圖也是 120×100**」最直接的證據：
// docs/formats/01-city-file.md 原本把它列為未解，理由是 DOS 的 .PSN 只有
// 6–11 KB、與 Micropolis 的 27120 對不起來。解壓之後對上了——
// .PSN ＝ 144 位元組檔頭 ＋ 27120 位元組、與 Micropolis 完全相同的城市檔。
func TestDOSScenariosParseAsCityFiles(t *testing.T) {
	dir := dosDir(t)
	entries, err := os.ReadDir(filepath.Join(dir, "SCENARIO"))
	if err != nil {
		t.Skip("沒有 SCENARIO 目錄")
	}
	n := 0
	for _, e := range entries {
		if !strings.EqualFold(filepath.Ext(e.Name()), ".psn") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, "SCENARIO", e.Name()))
		if err != nil {
			t.Errorf("%s：%v", e.Name(), err)
			continue
		}
		sc, err := assets.LoadPSN(raw)
		if err != nil {
			t.Errorf("%s：%v", e.Name(), err)
			continue
		}
		cf, err := sim.ParseCityFile(sc.Body)
		if err != nil {
			t.Errorf("%s（%s）：城市檔解析失敗 %v", e.Name(), sc.Name, err)
			continue
		}
		// 每一格的圖塊編號都要合法——這是「同一套圖塊編號」的證據。
		bad := 0
		for x := 0; x < sim.WorldX; x++ {
			for y := 0; y < sim.WorldY; y++ {
				if int(cf.Map[x][y]&sim.LOMASK) >= sim.TILE_COUNT {
					bad++
				}
			}
		}
		if bad != 0 {
			t.Errorf("%s（%s）：%d 格的圖塊編號超過 TILE_COUNT", e.Name(), sc.Name, bad)
		}
		n++
	}
	if n != 8 {
		t.Errorf("只驗了 %d 個劇本，應為 8", n)
	}
}

// 載入 DOS 劇本之後跑電力傳導，不能爆掉，而且要真的找到電廠。
func TestDOSScenarioPowerScanRuns(t *testing.T) {
	dir := dosDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "SCENARIO", "DETROIT.PSN"))
	if err != nil {
		t.Skip("沒有 DETROIT.PSN")
	}
	sc, err := assets.LoadPSN(raw)
	if err != nil {
		t.Fatal(err)
	}
	cf, err := sim.ParseCityFile(sc.Body)
	if err != nil {
		t.Fatal(err)
	}
	w := sim.NewWorld(1)
	w.LoadScenarioFile(cf, sim.ScenarioDetroit)
	res := w.DoPowerScan()
	// ⚠ 原版的 DoPowerScan 只填 PowerMap；PWRBIT 是下一輪 MapScan 才寫回
	// 地圖的（見 power.go 的 ApplyPowerBits）。測試要看得到位元，就自己叫。
	res.Powered = w.ApplyPowerBits()
	if res.CoalPop+res.NuclearPop == 0 {
		t.Error("底特律應該有電廠，卻一座都沒找到")
	}
	if res.Powered == 0 {
		t.Error("有電廠卻沒有任何格子通電")
	}
	t.Logf("%s：燃煤 %d 座、核能 %d 座、通電 %d 格", sc.Name, res.CoalPop, res.NuclearPop, res.Powered)
}
