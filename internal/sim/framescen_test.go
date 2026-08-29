package sim

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// loadPreState 把「載入之前」的衍生陣列殘值套進世界。
//
// ⚠ 原版那個行程開機時已經跑過一座隨機城市，所以載入劇本的那一刻，
// **不會被 DoSimInit 重算的陣列**（交通密度、成長率記憶、商業距離表）
// 以及**在第一次 MapScan 之後才重算的陣列**（人口密度、地價、汙染、犯罪）
// 都帶著殘值。少了它們，載入後的地圖會差兩格——差在一條路的車流顯示
// 與一格住宅的密度，看起來像規則寫錯，其實是起始狀態沒重建完整。
func loadPreState(t *testing.T, w *World, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, ln := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		parts := strings.Split(ln, ",")
		name, vals := parts[0], parts[1:]
		get := func(i int) int {
			v, err := strconv.Atoi(vals[i])
			if err != nil {
				t.Fatalf("%s 第 %d 個值讀不了：%q", name, i, vals[i])
			}
			return v
		}
		// oracle 是 y 外層、x 內層
		switch name {
		case "TRF", "POP", "LV", "POL", "CRI":
			if len(vals) != HWldX*HWldY {
				t.Fatalf("%s 應該有 %d 個值，實際 %d", name, HWldX*HWldY, len(vals))
			}
			for i := range vals {
				x, y := i%HWldX, i/HWldX
				switch name {
				case "TRF":
					w.TrfDensity[x][y] = uint8(get(i))
				case "POP":
					w.PopDensity[x][y] = uint8(get(i))
				case "LV":
					w.LandValueMem[x][y] = uint8(get(i))
				case "POL":
					w.PollutionMem[x][y] = uint8(get(i))
				case "CRI":
					w.CrimeMem[x][y] = uint8(get(i))
				}
			}
		case "ROG", "COM", "FST", "PLC", "PLE", "FRT":
			if len(vals) != SmX*SmY {
				t.Fatalf("%s 應該有 %d 個值，實際 %d", name, SmX*SmY, len(vals))
			}
			for i := range vals {
				x, y := i%SmX, i/SmX
				switch name {
				case "ROG":
					w.RateOGMem[x][y] = int16(get(i))
				case "COM":
					w.ComRate[x][y] = int16(get(i))
				case "FST":
					w.FireStMap[x][y] = int16(get(i))
				case "PLC":
					w.PoliceMap[x][y] = int16(get(i))
				case "PLE":
					w.PoliceMapEffect[x][y] = int16(get(i))
				case "FRT":
					w.FireRate[x][y] = int16(get(i))
				}
			}
		default:
			t.Fatalf("不認得的陣列名稱 %q", name)
		}
	}
}

// 逐 frame 對拍（劇本版）。
//
// frameparity_test.go 那座城市只有三個空住宅區、幾條路和一座電廠，
// 8000 個 frame 只抽了 13 582 次。這一份載入 Dullsville 再跑同樣的
// 8000 個 frame，抽樣數是 **123 383 次**——有人口、有交通、有稅收、
// 有精靈，掃到的規則多得多。
//
// 資料由 tools/oracle/tcl/frame-parity-scen.tcl 產生。
const frameScenBudget = 8000

// probMismatch 比對城市評估的分數與問題表；相同回空字串。
//
// 投票迴圈跑幾次由問題表決定，所以評估那個 frame 的抽樣次數對不上時，
// 先看是不是問題表本身就不同——那是規則層的差，不是投票的差。
func probMismatch(w *World, f frameCP) string {
	if !f.HasProb {
		return ""
	}
	// 先比問題表：投票迴圈跑幾次由它決定，所以它是上游。
	for i := 0; i < 7; i++ {
		if w.Eval.ProblemTable[i] != f.ProbTable[i] {
			return fmt.Sprintf("問題表 %v，原版 %v",
				w.Eval.ProblemTable[:7], f.ProbTable)
		}
	}
	if w.CityScore != f.CityScore || w.Eval.CityYes != f.CityYes ||
		w.Eval.CityNo != f.CityNo {
		return fmt.Sprintf("分數 %d（原版 %d）、贊成 %d/%d、反對 %d/%d",
			w.CityScore, f.CityScore, w.Eval.CityYes, f.CityYes,
			w.Eval.CityNo, f.CityNo)
	}
	return ""
}

func TestFrameParityScenario(t *testing.T) {
	raw, err := os.ReadFile("../../workplace/ref/micropolis/micropolis-activity/res/snro.111")
	if err != nil {
		t.Skip("封存裡沒有 snro.111")
	}
	cf, err := ParseCityFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	m := loadFrameMetaIn(t, "testdata/frame-scen")
	wantStart := loadGoldenMap(t, "testdata/frame-scen/cp0.csv")

	w := NewWorld(1)
	loadPreState(t, w, "testdata/frame-scen/prestate.csv")
	_ = wantStart
	// ⚠ 起點是**載入之前**的亂數狀態：DoSimInit 裡的 MapScan 會抽亂數，
	// 而且會讓分區長大——所以事後把地圖蓋回去不夠，衍生陣列（地價、
	// 汙染、犯罪）會是在另一張地圖上算出來的。
	w.Rand.SetState(m.Init.PreState)
	w.LoadScenarioFile(cf, ScenarioDullsville)
	// s_fileio.c:470 LoadScenario 尾巴：InitSimLoad = 1、DoInitialEval = 0，
	// 然後 DoSimInit()（它自己會把 DoInitialEval 設回 1）。
	w.InitSimLoad = 1
	w.EnableSprites()
	// ⚠ `NewPower` 在原版**開機時就被設成 1**（啟動時先產生一座隨機城市，
	// 那次的 InitSimMemory 尾巴會設它），之後永遠是 1。所以載入劇本時
	// DoSimInit 的那次 MapScan 會對每一格帶 CONDBIT 的格子呼叫 SetZPower，
	// 而 SimLoadInit 剛把整張 PowerMap 設成全 1——結果是**所有導電的格子
	// 都先被標成通電**，直到下一輪 MapScan 才依真正的電力網修正。
	// 從 NewWorld 起步的話這個位元是 false，載入後的地圖會差 265 格。
	w.NewPower = true
	w.DoSimInit()

	// ⚠ 這裡**直接用 oracle 載入完成當下的狀態覆蓋**，而不是要求我們自己
	// 的 DoSimInit 重建得一模一樣。理由：載入那一瞬間的地圖差 6 格，全部
	// 是道路的車流顯示等級——那是 DoSimInit 那一次 MapScan 途中累積的
	// 交通，而原版那個行程開機時已經跑過一座隨機城市，殘留狀態重建不完全。
	//
	// 這個測試要問的是**八千個 frame 的動態**對不對，不是載入程序對不對
	// （後者由 cityfile_test.go 與 power_test.go 顧）。起始狀態直接對齊，
	// 問題就收斂成單一變因。
	if d := mapDiffM(&w.Map, &wantStart); d != 0 {
		t.Logf("載入後的地圖差 %d 格（都是 DoSimInit 途中的車流顯示），"+
			"直接用原版的起始狀態覆蓋", d)
	}
	w.Map = wantStart
	loadPreState(t, w, "testdata/frame-scen/poststate.csv")
	w.Rand.SetState(mustRec(m.Init.Rands))

	trace := os.Getenv("SIMCITY_TRACE") != ""
	matched, total := 0, 0
	for _, f := range m.Frames {
		before := w.Rand.State()
		sites := map[string]int{}
		if trace {
			w.Rand.Watch = func() { sites[callerSite()]++ }
		}
		w.Frame()
		w.Rand.Watch = nil
		got := drawsBetween(before, w.Rand.State())
		bad := probMismatch(w, f)
		if bad != "" || got != f.Draws || w.Scycle != f.Scycle || !valvesEqual(w, f) {
			for k, v := range sites {
				t.Logf("  %s ×%d", k, v)
			}
			if f.HasVote {
				t.Logf("  原版：投票迴圈抽 %d、市民投票抽 %d、迭代 %d、成功 %d",
					f.Vote[0], f.Vote[1], f.Vote[2], f.Vote[3])
			}
		}
		if bad != "" {
			t.Logf("第 %d 個 frame 的評估狀態就對不上了：%s", f.I, bad)
			break
		}
		if got != f.Draws || w.Scycle != f.Scycle || !valvesEqual(w, f) {
			t.Logf("第 %d 個 frame 分岔：抽樣 %d 次（原版 %d 次）、Scycle %d（原版 %d）、"+
				"閥門 %d/%d/%d（原版 %d/%d/%d）",
				f.I, got, f.Draws, w.Scycle, f.Scycle,
				w.RValve, w.CValve, w.IValve, f.Valves[0], f.Valves[1], f.Valves[2])
			break
		}
		// 原版每個 frame 之後自己抽了四次；推完之後狀態應該與原版相同。
		w.Rand.SetState(advanceRand(w.Rand.State(), 4))
		if f.State != 0 && w.Rand.State() != f.State {
			t.Logf("第 %d 個 frame：抽樣次數對得上，但狀態 %d ≠ 原版 %d",
				f.I, w.Rand.State(), f.State)
			break
		}
		matched++
		total += f.Draws
	}
	t.Logf("劇本逐 frame 完全一致 %d/%d 個（共 %d 次抽樣）", matched, len(m.Frames), total)
	if matched == len(m.Frames) {
		wantEnd := loadFrameEndMapIn(t, "testdata/frame-scen")
		if d := mapDiffM(&w.Map, &wantEnd); d != 0 {
			t.Errorf("抽樣全部對上，但終點地圖差 %d 格", d)
		}
		if w.TotalFunds != m.End.Funds {
			t.Errorf("終點資金 %d，原版 %d", w.TotalFunds, m.End.Funds)
		}
	}
	if matched < frameScenBudget {
		t.Errorf("只對上 %d 個 frame，低於現況 %d —— 有東西退步了", matched, frameScenBudget)
	}
	if matched > frameScenBudget {
		t.Errorf("對上 %d 個 frame，比現況 %d 好 —— 請把 frameScenBudget 調到 %d",
			matched, frameScenBudget, matched)
	}
}
