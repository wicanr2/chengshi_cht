package sim

import "testing"

// 劇本城市是用原版工具蓋出來的，所以它的路、鐵路、電線形狀正是
// fixSingle 的輸出。對每一格重跑一次，**形狀**應該完全不變。
//
// ⚠ 比的是形狀不是整個字。fixSingle 寫回去的是
// `表[adj] | BULLBIT | BURNBIT`（電線多一個 CONDBIT），
// 所以它會順手清掉車流量與 PWRBIT——C 也一樣。在一座跑過的城市上
// 逐位元比較一定會紅，而那是正常行為，不是錯誤。
//
// 這是免費的強驗證：接線表的索引位元順序、neutralizeRoad 的範圍、
// 三種線路各自的排除清單，任何一個弄錯都會讓某些格子換形狀。
// 黃金資料是 oracle 的 `sim LoadScenario 1` 倒出來的 12000 格。
// 原版地圖裡本來就有極少數「形狀過期」的格子。
//
// 成因是**摧毀不重接**：火災（DoFire）、爆炸（Destroy）、洪水與分區成長
// （ZonePlop）都直接覆寫格子，不會回頭對鄰居跑 fixSingle。所以一條路或
// 鐵路可能保留著指向某個方向的接頭，而那個方向的東西早就沒了。
//
// 這個解釋有分布上的證據：26 格裡漢堡（大轟炸劇本）佔 12 格、
// 里約（洪水劇本）佔 8 格，兩個災難劇本就佔了四分之三；
// 而東京、底特律各只有 1 格。逐格檢查也一致——差異全都是
// 「接頭指向空地或樹木」。
//
// 門檻是回歸護欄：比現況更多就代表接線表或位元順序壞了。
const fixSingleStaleBudget = 26

func TestFixSingleIsIdempotentOnScenarioCity(t *testing.T) {
	cities := []string{
		"scenario1-dullsville", "scenario2-sanfrancisco", "scenario3-hamburg",
		"scenario4-bern", "scenario5-tokyo", "scenario6-detroit",
		"scenario7-boston", "scenario8-rio",
	}
	totalChecked, totalChanged := 0, 0
	for _, name := range cities {
		m := loadGoldenMap(t, "testdata/"+name+".csv")
		w := NewWorld(0)
		w.Map = m

		checked, changed := 0, 0
		byKind := map[string]int{}
		for x := 0; x < WorldX; x++ {
			for y := 0; y < WorldY; y++ {
				before := neutralizeRoad(int(w.Map[x][y]))
				isLine := (before >= 66 && before <= 76) ||
					(before >= 226 && before <= 236) ||
					(before >= 210 && before <= 220)
				if !isLine {
					continue
				}
				checked++
				w.fixSingle(x, y)
				after := neutralizeRoad(int(w.Map[x][y]))
				if after == before {
					continue
				}
				changed++
				switch {
				case before <= 76:
					byKind["道路"]++
				case before >= 226:
					byKind["鐵路"]++
				default:
					byKind["電線"]++
				}
				if changed <= 3 {
					t.Logf("  %s (%d,%d) 形狀 %d → %d", name, x, y, before, after)
				}
			}
		}
		totalChecked += checked
		totalChanged += changed
		if changed != 0 {
			t.Logf("%s：%d/%d 格形狀不同（%v）", name, changed, checked, byKind)
		}
	}
	t.Logf("八座劇本城市共檢查 %d 格線路，%d 格形狀不同（%.2f%%）",
		totalChecked, totalChanged, 100*float64(totalChanged)/float64(totalChecked))
	if totalChecked < 10000 {
		t.Fatalf("只檢查到 %d 格，黃金地圖可能不對", totalChecked)
	}
	if totalChanged > fixSingleStaleBudget {
		t.Errorf("%d 格形狀不同，超過現況 %d —— 接線表或位元順序可能壞了",
			totalChanged, fixSingleStaleBudget)
	}
}

// 鋪一條直路，形狀要隨鄰居長出來：單格是 ROADS，接上去之後變成
// 橫向連接，中間那格是十字。
func TestLayRoadShapes(t *testing.T) {
	w := NewWorld(0)
	w.TotalFunds = 10000

	if got := w.ConnecTile(10, 10, ConnRoad); got != ToolOK {
		t.Fatalf("鋪第一格失敗：%d", got)
	}
	if got := int(w.Map[10][10]) & LOMASK; got != roadTable[0] {
		t.Fatalf("孤立的一格應為 %d，得到 %d", roadTable[0], got)
	}

	w.ConnecTile(11, 10, ConnRoad)
	// 左右相連：(10,10) 應該只有右邊(位元 2)，(11,10) 只有左邊(位元 8)
	if got := int(w.Map[10][10]) & LOMASK; got != roadTable[2] {
		t.Errorf("(10,10) 應為 %d，得到 %d", roadTable[2], got)
	}
	if got := int(w.Map[11][10]) & LOMASK; got != roadTable[8] {
		t.Errorf("(11,10) 應為 %d，得到 %d", roadTable[8], got)
	}

	// 十字路口：上下左右都有路
	w.ConnecTile(12, 10, ConnRoad)
	w.ConnecTile(11, 9, ConnRoad)
	w.ConnecTile(11, 11, ConnRoad)
	if got := int(w.Map[11][10]) & LOMASK; got != roadTable[15] {
		t.Errorf("十字路口應為 %d（roadTable[15]），得到 %d", roadTable[15], got)
	}
}

// 位元順序：上=1、右=2、下=4、左=8。方向弄錯的話對稱的圖形看不出來，
// 所以特意用不對稱的 L 形。
func TestConnectBitOrder(t *testing.T) {
	w := NewWorld(0)
	w.TotalFunds = 10000
	// 只有上與右：一個往右上的彎道
	w.ConnecTile(10, 10, ConnRoad)
	w.ConnecTile(10, 9, ConnRoad)  // 上
	w.ConnecTile(11, 10, ConnRoad) // 右
	want := roadTable[1|2]
	if got := int(w.Map[10][10]) & LOMASK; got != want {
		t.Fatalf("上＋右應為 roadTable[3]=%d，得到 %d —— 位元順序可能反了", want, got)
	}
}

// neutralizeRoad：有車流的路段在判斷連通時要被還原成空路。
func TestNeutralizeRoad(t *testing.T) {
	cases := []struct{ in, want int }{
		{66, 66},   // 空路，不變
		{66 + 64, 66},  // 輕度車流
		{66 + 128, 66}, // 重度車流
		{240, 240}, // 不在 64..207 的範圍，不變
		{0, 0},
	}
	for _, c := range cases {
		if got := neutralizeRoad(c.in); got != c.want {
			t.Errorf("neutralizeRoad(%d) = %d，應為 %d", c.in, got, c.want)
		}
	}
}

// 推土機：推掉水上的橋要還原成水，推掉陸上的東西還原成空地。
func TestBulldozeBridgeBecomesWater(t *testing.T) {
	w := NewWorld(0)
	w.TotalFunds = 10000
	w.Map[10][10] = HBRIDGE | BULLBIT
	if got := w.Bulldoze(10, 10); got != ToolOK {
		t.Fatalf("推橋失敗：%d", got)
	}
	if got := int(w.Map[10][10]) & LOMASK; got != RIVER {
		t.Errorf("拆橋之後應該是水（%d），得到 %d —— 憑空多了一塊陸地", RIVER, got)
	}

	w.Map[11][10] = ROADS | BULLBIT | BURNBIT
	w.Bulldoze(11, 10)
	if got := int(w.Map[11][10]) & LOMASK; got != DIRT {
		t.Errorf("拆路之後應該是空地，得到 %d", got)
	}
}

// 錢不夠時不能施工，而且不能把錢扣成負的。
func TestToolsRespectFunds(t *testing.T) {
	w := NewWorld(0)
	w.TotalFunds = 4 // 電線要 5
	if got := w.ConnecTile(10, 10, ConnWire); got != ToolNoMoney {
		t.Errorf("錢不夠應回 %d，得到 %d", ToolNoMoney, got)
	}
	if w.TotalFunds != 4 {
		t.Errorf("失敗的施工不該扣錢，剩 %d", w.TotalFunds)
	}
	if w.Map[10][10] != 0 {
		t.Error("失敗的施工不該改地圖")
	}
}
