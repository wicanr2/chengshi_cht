package sim

import "testing"

// 整刻對拍。
//
// oracle 沒有「走一刻」的指令（SimFrame 每個 timer frame 只走一個相位，
// 而 Tcl 只能設速度），所以不能直接比對第 N 刻。改用**亂數狀態當時鐘**：
//
//  1. 暫停 → 讀四次 sim Rand → 反推內部狀態 S0（那是開跑瞬間的狀態）
//  2. 倒出起始地圖
//  3. 跑一段時間 → 暫停 → 再讀四次 → 反推並倒退四步得到結束狀態 S1
//  4. 倒出結束地圖
//  5. Go 版從起始地圖 ＋ S0 出發，**逐相位**推進，找出一個相位數 n 使得
//     亂數狀態等於 S1 **而且** 12000 格全部相同
//
// 亂數狀態能對上，代表每一個消耗亂數的分支都走得一樣；地圖能對上，
// 代表每一次寫入都一樣。兩者同時成立就是逐指令等價的強證據。
//
// 起始的 Fcycle／Scycle 不可觀測（GenerateSomeCity 之後到暫停之間可能跑了
// 幾個 frame），所以在小範圍內搜尋。
// 目前的對拍水準：Go 版重現原版 108 格變化中的 101 格。
//
// ⚠ **這還不是逐指令等價。** 亂數狀態始終對不上，代表某處的抽樣次數與原版不同，
// 而抽樣次數一旦差一次，之後的數列全部錯開。已知還沒實作的東西（精靈系統、
// 訊息系統）都會影響抽樣次數，所以在它們補上之前不預期能對齊。
//
// 這個測試是**回歸護欄**：差異格數不得比現況更差。修好一處就把門檻調緊，
// 直到亂數狀態也能對上為止。進度記在 docs/re/12-tick-parity.md。
const tickParityBudget = 8

func TestTickParityBestEffort(t *testing.T) {
	s0, ok := RecoverState([]int{38231, 17264, 16134, 55346})
	if !ok {
		t.Fatal("反推不出起始亂數狀態")
	}
	s1raw, ok := RecoverState([]int{32494, 14428, 51627, 30048})
	if !ok {
		t.Fatal("反推不出結束亂數狀態")
	}
	s1 := rewindRand(s1raw, 4) // 四次 sim Rand 是讀取動作，不算模擬消耗

	start := loadGoldenMap(t, "testdata/tick-parity-map0.csv")
	end := loadGoldenMap(t, "testdata/tick-parity-map1.csv")

	changed := 0
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			if start[x][y] != end[x][y] {
				changed++
			}
		}
	}

	best, bestPhase, bestStep := 1<<30, -1, -1
	rngHit := -1
	for cand := 0; cand < 16*48; cand++ {
		startPhase, ct := cand/48, cand%48
		w := newTickParityWorld(start, s0, 20000, 0, true)
		w.CityTime = ct
		w.Fcycle = startPhase
		for n := 0; n <= 3000; n++ {
			if w.Rand.State() == s1 && rngHit < 0 {
				rngHit = n
				t.Logf("★ 亂數狀態對上：起始相位 %d、ct%%48=%d、第 %d 步（%.1f 刻）",
					startPhase, ct, n, float64(n)/16)
			}
			d := 0
			for x := 0; x < WorldX; x++ {
				for y := 0; y < WorldY; y++ {
					if w.Map[x][y] != end[x][y] {
						d++
					}
				}
			}
			if d < best {
				best, bestPhase, bestStep = d, startPhase, n
			}
			if best == 0 {
				break
			}
			w.Frame()
		}
		if best == 0 {
			break
		}
	}

	t.Logf("原版改變了 %d 格；Go 版最接近時差 %d 格（起始相位 %d、第 %d 步，%.1f 刻）—— 重現 %.1f%%",
		changed, best, bestPhase, bestStep, float64(bestStep)/16,
		100*float64(changed-best)/float64(changed))
	if rngHit < 0 {
		t.Logf("亂數狀態在 16 個相位 × 8000 步內都沒對上")
	}

	if best > tickParityBudget {
		t.Errorf("差異 %d 格，超過現況門檻 %d —— 有東西退步了", best, tickParityBudget)
	}
	if best < tickParityBudget {
		t.Errorf("差異只有 %d 格，比門檻 %d 好 —— 請把 tickParityBudget 調到 %d",
			best, tickParityBudget, best)
	}
}

// newTickParityWorld 重建實驗的起始狀態。
//
// 順序很重要：
//  1. 先產生**寫入建物之前**的地形（GenerateSomeCity 12345），
//     因為 oracle 的衍生陣列（電力、汙染、地價、犯罪、人口密度）是
//     DoSimInit 在那張裸地形上算出來的。
//  2. 設定 DoSimInit 之後的純量。
//  3. 跑一次 DoSimInit 的掃描序列，把衍生陣列填成同樣的值。
//  4. **然後**才把 oracle 倒出來的起始地圖蓋上去（那是寫入建物之後的樣子）。
//  5. 最後才設亂數狀態——暖身的掃描可能消耗亂數，不能讓它污染 S0。
func newTickParityWorld(m [WorldX][WorldY]uint16, rngState uint32, funds, scycle int, initEval bool) *World {
	w := NewWorld(0)
	w.GenerateMap(12345, DefaultTerrainParams())

	w.TotalFunds = funds
	w.CityTime = 0
	w.CityTax = 7
	w.SimSpeed = 3
	w.GameLevel = 0
	w.NoDisasters = true // 實驗裡下了 sim Disasters 0
	w.InitSimLoad = 2
	w.EMarket = 6.0
	for i := range w.MoneyHis {
		w.MoneyHis[i] = 128
	}
	w.EvalInit()

	// DoSimInit 的掃描序列（s_sim.c:207）。
	w.SetValves()
	w.ClearCensus()
	w.MapScan(0, WorldX)
	w.DoPowerScan()
	w.NewPower = true
	w.PTLScan()
	w.CrimeScan()
	w.PopDenScan()
	w.FireAnalysis()
	w.TotalPop = 1
	w.DoInitialEval = initEval
	w.InitSimLoad = 0
	w.Scycle = scycle

	w.Map = m // 玩家（實驗腳本）寫進去的建物
	w.EnableSprites()
	w.Rand.SetState(rngState)
	return w
}

// rewindRand 把 LCG 狀態倒退 n 步。
//
// next' = (next·A + C) mod 2²⁴，A 是奇數所以模 2²⁴ 可逆：
// next = (next' − C)·A⁻¹ mod 2²⁴。
func rewindRand(state uint32, n int) uint32 {
	const mod = uint32(1) << 24
	inv := modInverse24(randA)
	s := state & randMask
	for i := 0; i < n; i++ {
		s = ((s - randC) & randMask) * inv % mod
		s &= randMask
	}
	return s
}

// modInverse24 用牛頓法求奇數在模 2²⁴ 下的乘法反元素。
func modInverse24(a uint32) uint32 {
	inv := uint32(1)
	for i := 0; i < 6; i++ {
		inv = inv * (2 - a*inv)
	}
	return inv & randMask
}

// 反元素本身要正確：a · a⁻¹ ≡ 1 (mod 2²⁴)。
func TestModInverse24(t *testing.T) {
	inv := modInverse24(randA)
	if (randA*inv)&randMask != 1 {
		t.Fatalf("A·A⁻¹ mod 2²⁴ = %d，應為 1", (randA*inv)&randMask)
	}
}

// 倒退再前進要回到原地。
func TestRewindRandRoundTrip(t *testing.T) {
	r := NewRand(12345)
	for i := 0; i < 7; i++ {
		r.Rand16()
	}
	want := r.State()
	for i := 0; i < 5; i++ {
		r.Rand16()
	}
	if got := rewindRand(r.State(), 5); got != want {
		t.Fatalf("倒退五步得到 %d，應為 %d", got, want)
	}
}
