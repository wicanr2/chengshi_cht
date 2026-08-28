package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/sim"
	"github.com/wicanr2/chengshi_cht/internal/game"
)


// 正常玩家路徑：蓋電廠、拉電線、鋪路、劃分區，然後讓城市自己長。
//
// 這是唯一會同時走過工具層、接線層、電力、交通、分區成長、普查、預算與
// 訊息系統的測試。單元測試各自綠著、城市卻長不起來——那是最常見的
// 「零件都對、整台不動」，只有端到端跑一次才抓得到。
func TestPlaythroughCityGrows(t *testing.T) {
	w := sim.NewWorld(12345)
	w.GenerateMap(12345, sim.DefaultTerrainParams())
	w.DoSimInit()
	w.NoDisasters = true // 成長測試不要被隨機災難干擾
	w.SimSpeed = 3

	// 找一塊夠大的可建地。地形是隨機的，寫死座標會在換種子時莫名其妙失敗。
	ox, oy, ok := findFlatArea(w, 20, 16)
	if !ok {
		t.Fatal("種子 12345 的地圖上找不到夠大的可建地 —— 地形產生可能壞了")
	}
	t.Logf("在 (%d,%d) 建城", ox, oy)

	build := func(tool sim.Tool, x, y int) {
		t.Helper()
		if r := w.ApplyTool(tool, x, y); r != sim.ToolOK {
			t.Fatalf("%v 蓋在 (%d,%d) 失敗，回傳 %d（資金 %d）", tool, x, y, r, w.TotalFunds)
		}
	}

	// 佈局。三件事必須同時成立，分區才會長：
	//
	//   1. **通電** —— 要有一條 CONDBIT 連通的路徑接到發電廠。
	//      分區本身有 CONDBIT，所以分區碰分區也能導電。
	//   2. **有路** —— MakeTraf 要在分區周長上找得到路（FindPRoad）。
	//   3. **有目的地** —— 住宅要走得到商業或工業，交通才算通。
	//
	// 少任何一件，城市就是「蓋好了但完全不動」，而且沒有任何錯誤訊息。
	//
	//   x0   電線縱列（接到電廠）
	//   x0+1..x0+3  住宅區（左緣貼電線、右緣貼道路）
	//   x0+4 道路＋電線縱列（在路上拉電線會變成導電的路面）
	//   x0+5..x0+7  商業區
	//   x0+8 道路＋電線縱列
	//   x0+9..x0+11 工業用地
	x0, y0 := ox+5, oy+2
	const zoneRows = 4

	// 發電廠：4×4，點在 (px,py) 時佔 (px-1..px+2, py-1..py+2)，
	// 所以右緣要剛好貼著電線縱列。
	build(sim.ToolCoalPower, x0-3, y0+1)

	for y := y0; y < y0+zoneRows*3; y++ {
		build(sim.ToolWire, x0, y)
		build(sim.ToolRoad, x0+4, y)
		build(sim.ToolWire, x0+4, y) // 路面上再拉電線 → 導電的路
		build(sim.ToolRoad, x0+8, y)
		build(sim.ToolWire, x0+8, y)
	}
	for i := 0; i < zoneRows; i++ {
		cy := y0 + 1 + i*3
		build(sim.ToolResidential, x0+2, cy)
		build(sim.ToolCommercial, x0+6, cy)
		build(sim.ToolIndustrial, x0+10, cy)
	}

	spent := 20000 - w.TotalFunds
	if spent <= 0 {
		t.Fatal("蓋了一堆東西卻沒花錢 —— 成本沒有被扣")
	}
	t.Logf("建設花了 $%d，剩 $%d", spent, w.TotalFunds)

	// 跑三十年。
	//
	// ⚠ 普查用的計數器（ResPop、PwrdZCnt…）**每一刻都會被 ClearCensus
	// 歸零再重數**，所以不能跑完之後直接讀——收在相位 0 之後量到的是 0，
	// 看起來像「城市沒長」。這裡取整段的峰值。
	const years = 30
	peak := struct{ res, com, ind, pwrd, pop, score int }{}
	for i := 0; i < years*48*16; i++ {
		w.Frame()
		peak.res = max(peak.res, w.ResPop)
		peak.com = max(peak.com, w.ComPop)
		peak.ind = max(peak.ind, w.IndPop)
		peak.pwrd = max(peak.pwrd, w.PwrdZCnt)
		peak.pop = max(peak.pop, w.Eval.CityPop)
		peak.score = max(peak.score, w.CityScore)
	}

	t.Logf("%d 年後：人口 %d（住 %d 商 %d 工 %d）資金 $%d 評分 %d",
		years, peak.pop, peak.res, peak.com, peak.ind, w.TotalFunds, peak.score)
	t.Logf("  通電分區 %d；地價 %d 污染 %d 犯罪 %d",
		peak.pwrd, w.LVAverage, w.PolluteAverage, w.CrimeAverage)

	if peak.pwrd == 0 {
		t.Error("沒有任何分區通電 —— 電力傳導或電線接線壞了")
	}
	if peak.res == 0 {
		t.Error("三十年後住宅區還是空的 —— 分區成長沒有在跑")
	}
	if peak.pop == 0 {
		t.Error("人口是 0 —— 普查沒有在跑")
	}
	if peak.score == 0 {
		t.Error("城市評分是 0 —— 評估沒有在跑")
	}
}

// findFlatArea 找一塊 w×h 的可建地。
//
// ⚠ 不能只找全空地。地形產生器會撒滿樹林，種子 12345 的地圖上
// 連 24×16 的純空地都沒有。可建地的定義要跟著遊戲走：空地或樹林都算，
// 因為自動整地會把樹清掉（每格 $1）。水不算。
func findFlatArea(world *sim.World, w, h int) (int, int, bool) {
	for y := 2; y < sim.WorldY-h-2; y++ {
		for x := 2; x < sim.WorldX-w-2; x++ {
			if flat(world, x, y, w, h) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func flat(world *sim.World, x, y, w, h int) bool {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			t := world.TileNum(x+i, y+j)
			if t == 0 {
				continue
			}
			// 樹林（含 WOODS 系列）可以自動整地掉。
			if t >= sim.TREEBASE && t <= sim.WOODS5 {
				continue
			}
			return false
		}
	}
	return true
}

// 八個悲情城市都要載得起來，而且載進來的狀態要與劇本表一致。
//
// ⚠ 劇本**不套用檔案裡的純量**：原版的 LoadScenario 只讀七個陣列，
// CityTime 與起始資金由劇本表寫死。檔案裡殘留的值從來沒有生效過，
// 拿它去驗證會得到「自洽但錯」的結論。
func TestAllScenariosLoad(t *testing.T) {
	dir := dosDir(t)
	for n := 1; n <= 8; n++ {
		w, err := game.LoadScenario(dir, n)
		if err != nil {
			t.Errorf("第 %d 個劇本載入失敗：%v", n, err)
			continue
		}
		info := sim.Scenario(n).Info()
		if w.CityTime != info.CityTime {
			t.Errorf("%s：CityTime = %d，劇本表寫的是 %d",
				info.Name, w.CityTime, info.CityTime)
		}
		if w.TotalFunds != info.StartFunds {
			t.Errorf("%s：起始資金 = %d，劇本表寫的是 %d",
				info.Name, w.TotalFunds, info.StartFunds)
		}
		if w.CityTax != 7 {
			t.Errorf("%s：稅率 = %d，LoadScenario 設 7", info.Name, w.CityTax)
		}
		// 劇本城市一定有東西：地圖不能是空的。
		nonEmpty := 0
		for x := 0; x < sim.WorldX; x++ {
			for y := 0; y < sim.WorldY; y++ {
				if w.TileNum(x, y) != 0 {
					nonEmpty++
				}
			}
		}
		if nonEmpty < 1000 {
			t.Errorf("%s：地圖上只有 %d 格非空地，太少", info.Name, nonEmpty)
		}
		// 跑幾刻確認不會爆
		for i := 0; i < 48*16; i++ {
			w.Frame()
		}
		if zh := game.ScenarioNameZH(n); zh == "" {
			t.Errorf("第 %d 個劇本沒有中文名", n)
		}
	}
}

// 存檔 → 讀檔要回到同一個狀態，而且檔案是**原版格式**。
//
// ⚠ 打包時要先複製整個 MiscHis 再覆蓋純量。從零開始填會讓劇本編號、
// 城市等級、災難計時整批遺失——而**載入後看起來一切正常**，
// 只有跑一陣子才會發現劇本判定不觸發。
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	w := sim.NewWorld(4242)
	w.GenerateMap(4242, sim.DefaultTerrainParams())
	w.DoSimInit()
	if _, _, ok := game.BuildStarterCity(w); !ok {
		t.Skip("這張地圖上沒有夠大的可建地")
	}
	for i := 0; i < 20*48*16; i++ {
		w.Frame()
	}
	w.CityTax = 11
	w.RoadPercent = 0.75

	path := filepath.Join(dir, "test.cty")
	if err := game.SaveCity(path, w); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != sim.CityFileSize1x1 {
		t.Fatalf("存出 %d 位元組，原版格式是 %d", len(raw), sim.CityFileSize1x1)
	}

	w2, err := game.LoadCity(path)
	if err != nil {
		t.Fatal(err)
	}
	if w2.CityTime != w.CityTime {
		t.Errorf("CityTime %d ≠ %d", w2.CityTime, w.CityTime)
	}
	if w2.TotalFunds != w.TotalFunds {
		t.Errorf("資金 %d ≠ %d", w2.TotalFunds, w.TotalFunds)
	}
	if w2.CityTax != 11 {
		t.Errorf("稅率 %d ≠ 11", w2.CityTax)
	}
	diff := 0
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			if w2.Map[x][y] != w.Map[x][y] {
				diff++
			}
		}
	}
	if diff != 0 {
		t.Errorf("地圖有 %d 格對不上", diff)
	}
	// 存檔後 MiscHis 的其餘欄位不能被清掉
	for i := range w.MiscHis {
		if i >= 8 && i <= 9 || i >= 50 && i <= 63 {
			continue // 這些是存檔時覆蓋的純量
		}
		if w2.MiscHis[i] != w.MiscHis[i] {
			t.Errorf("MiscHis[%d] = %d，應為 %d —— 打包時把其餘欄位清掉了",
				i, w2.MiscHis[i], w.MiscHis[i])
			break
		}
	}
}
