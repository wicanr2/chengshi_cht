package main

// DOS 原版與 remake 的抽樣對拍。結果與量法上的坑寫在 docs/re/18-dos-parity.md。
//
// 為什麼是「抽樣」而不是逐 tick：DOS 版載入時自己重設亂數種子，我們沒有
// 辦法把它的亂數狀態設成一樣，所以**逐 tick 完全一致在原理上做不到**
// （Micropolis 那一側做得到，因為 oracle 讓我們讀得到狀態，見 docs/re/12）。
// 能做的是：讓兩邊從**同一張地圖、同一個 CityTime** 出發，跑到同一刻，
// 再比一組對亂數不敏感的量。
//
// 取樣管道是 DOS 自己寫出來的 `.cty`：它帶著 CityTime，所以「跑到哪一刻」
// 不必猜——螢幕截圖比不出人口與資金，存檔可以。

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// metrics 是一組**對亂數不敏感**的量：地物的格數分布與幾個純量。
// 逐格比對在這裡沒有意義（兩邊的隨機事件不同），格數分布有。
func metrics(w *sim.World) map[string]int {
	n := map[string]int{}
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			t := int(w.Map[x][y] & sim.LOMASK)
			switch {
			case t >= sim.RIVER && t <= sim.LASTRIVEDGE:
				n["水"]++
			case t >= sim.TREEBASE && t <= sim.WOODS5:
				n["樹"]++
			case t >= sim.ROADBASE && t <= sim.LASTROAD:
				n["道路"]++
			case t >= sim.POWERBASE && t <= sim.LASTPOWER:
				n["電線"]++
			case t > sim.LASTPOWER && t <= sim.LASTRAIL:
				n["鐵軌"]++
			case t >= sim.RESBASE && t < sim.COMBASE:
				n["住宅"]++
			case t >= sim.COMBASE && t < sim.INDBASE:
				n["商業"]++
			case t >= sim.INDBASE && t < sim.PORTBASE:
				n["工業"]++
			case t >= sim.FIREBASE && t <= sim.LASTFIRE:
				n["火"]++
			case t >= sim.RUBBLE && t <= sim.LASTRUBBLE:
				n["廢墟"]++
			}
		}
	}
	n["資金"] = w.TotalFunds
	// ⚠ 人口用**唯讀重數**，不讀 `w.ResPop`。那些欄位是 MapScan 逐段
	// 累加的，取樣點落在掃描中途就會讀到半份（實測差到 100%）。
	res, com, ind := w.CountPops()
	n["住宅人口"] = res
	n["商業人口"] = com
	n["工業人口"] = ind
	// ＊開頭的量**不列入判準**：它們是逐段掃描累積出來的移動平均，
	// 讀到的值取決於「這一側跑過幾次掃描」，不是取決於城市長什麼樣。
	// 兩個實證：
	//   1. 剛載入的那一份地價是拿重心 (0,0) 算的（DoSimInit 的順序是
	//      PTLScan → PopDenScan，重心由後者算），劇本 1 實測 8；
	//      重心就位後同一張地圖是 107。
	//   2. 補掃一次也救不回來——PTLScan／CrimeScan 互相吃對方的 Mem
	//      陣列，不是冪等的。劇本 1 的犯罪補掃一次從 71 變 159。
	// 所以這三個量只印出來看趨勢，判準看地物格數與人口。
	n["＊地價均值"] = w.LVAverage
	n["＊汙染均值"] = w.PolluteAverage
	n["＊犯罪均值"] = w.CrimeAverage
	return n
}

// runScen 從第 n 個劇本出發，用指定的種子跑到 CityTime 為 until 那一刻。
//
// **種子要從外面給。** `game.LoadScenario` 照原版的行為用時鐘播種
// （DOS 版載入時也重播種），但那讓對拍結果每跑一次都不一樣——同一份
// 存檔實測跑三次得到 10224／10215／10223。要拿來寫進文件的數字必須
// 可重跑，所以這裡一律覆蓋種子。
func runScen(dir string, n int, seed uint32, until int) (*sim.World, error) {
	w, err := game.LoadScenarioSeed(dir, n, seed)
	if err != nil {
		return nil, err
	}
	// ⚠ 這兩個要跟 DOS 那一側的操作對齊（見 tools/dosbox/act-scen-run.txt）：
	//   Auto-Budget —— 不開的話原版每年跳預算對話框，模擬就停住了。
	//   No Disasters —— 不關的話比到的多半是「原版剛好起了一場火」。
	//     實測劇本 1 跑 14 年，原版燒掉 26 格工業區，工業格數差 113%。
	w.AutoBudget = true
	w.NoDisasters = true
	for w.CityTime < until {
		w.Frame()
	}
	return w, nil
}

// mapAgree 回傳兩張地圖相同的格數：完全相同（含旗標位元）與只看圖塊編號。
func mapAgree(a, b *sim.World) (full, low int) {
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			p, q := a.Map[x][y], b.Map[x][y]
			if p == q {
				full++
			}
			if p&sim.LOMASK == q&sim.LOMASK {
				low++
			}
		}
	}
	return
}

// cmdDosParityScen 拿 DOS 原版的一份劇本存檔當取樣點：remake 從**同一個
// 劇本**出發，跑到那份存檔的 CityTime，再比一組對亂數不敏感的量。
//
// 起點兩邊完全相同（同一份 `.PSN` 資料），所以差異全部來自這中間的模擬。
//
//	simtool dosparity-scen <1-8> <原版存檔.cty> [種子]
//	simtool dosparity-scen <1-8> <原版存檔.cty> sweep=<幾個種子>
//
// sweep 模式跑種子 1..K，印每個種子的相同格數與全距——單一種子的數字
// 說不出「這個差距是模擬的系統性差異，還是這次亂數剛好」。
func cmdDosParityScen(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "用法：simtool dosparity-scen <劇本編號 1-8> <原版存檔.cty> [種子|sweep=K]")
		os.Exit(2)
	}
	var n int
	if _, err := fmt.Sscanf(args[0], "%d", &n); err != nil || n < 1 || n > 8 {
		fmt.Fprintln(os.Stderr, "劇本編號要是 1–8")
		os.Exit(2)
	}
	seed, sweep := uint32(1), 0
	if len(args) >= 3 {
		if strings.HasPrefix(args[2], "sweep=") {
			if _, err := fmt.Sscanf(args[2][6:], "%d", &sweep); err != nil || sweep < 1 {
				fmt.Fprintln(os.Stderr, "sweep= 後面要是正整數")
				os.Exit(2)
			}
		} else if _, err := fmt.Sscanf(args[2], "%d", &seed); err != nil {
			fmt.Fprintln(os.Stderr, "種子要是整數")
			os.Exit(2)
		}
	}
	// 原版存檔這一側也要固定種子：`LoadCity` 會跑 `DoSimInit`，
	// 那一次 MapScan 同樣會擲亂數動到地圖。
	want, err := game.LoadCitySeed(args[1], seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dir := os.Getenv("SIMCITY_DATA")
	if dir == "" {
		dir = "workplace/dos110/SIMCITY 1.10"
	}

	if sweep > 0 {
		fmt.Printf("劇本 %d：原版存檔 CityTime %d（%d 年），掃 %d 個種子\n",
			n, want.CityTime, 1900+want.CityTime/48, sweep)
		fulls := make([]int, 0, sweep)
		lows := make([]int, 0, sweep)
		for s := 1; s <= sweep; s++ {
			w, err := runScen(dir, n, uint32(s), want.CityTime)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			f, l := mapAgree(want, w)
			fulls, lows = append(fulls, f), append(lows, l)
			fmt.Printf("  種子 %2d：含旗標 %5d／%d，只看圖塊 %5d／%d\n",
				s, f, sim.WorldX*sim.WorldY, l, sim.WorldX*sim.WorldY)
		}
		sort.Ints(fulls)
		sort.Ints(lows)
		fmt.Printf("含旗標  最少 %d 中位 %d 最多 %d\n", fulls[0], fulls[len(fulls)/2], fulls[len(fulls)-1])
		fmt.Printf("只看圖塊 最少 %d 中位 %d 最多 %d\n", lows[0], lows[len(lows)/2], lows[len(lows)-1])
		return
	}

	w, err := runScen(dir, n, seed, want.CityTime)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("劇本 %d：原版跑到 CityTime %d（%d 年），remake 從同一份劇本、種子 %d 跑到同一刻\n",
		n, want.CityTime, 1900+want.CityTime/48, seed)

	// 逐格比一次。地圖是兩邊最大的共同物件（12 000 格），而且它對亂數
	// 的敏感度比純量低——分區有沒有長出來看得見，長在哪一格才看得出差別。
	full, low := mapAgree(want, w)
	fmt.Printf("地圖：%d/%d 格完全相同（含旗標位元）；只看圖塊編號 %d/%d\n",
		full, sim.WorldX*sim.WorldY, low, sim.WorldX*sim.WorldY)
	fmt.Printf("圖塊相同但旗標不同的格數：%s\n\n", bitDiff(want, w))

	report(metrics(want), metrics(w))

	if cf, err := game.LoadCityFileRaw(args[1]); err == nil {
		reportMisc(cf, w)
	}
}

// reportMisc 比**原版自己記在存檔裡的純量**與 remake 同一刻的值。
//
// 這一組比 metrics 那一組硬：metrics 的原版側是我們拿原版的地圖重算的，
// 這一組是原版自己算完寫進 `MiscHis` 的。索引與 Micropolis 相同——
// 用 run1.cty 驗過：[2]=446 與唯讀重數的住宅人口相同、[9]=1029 是 CityTime、
// [51]=17216 是資金、[56]=7 是稅率，四個獨立錨點都對得上。
func reportMisc(cf *sim.CityFile, w *sim.World) {
	// ⚠ 人口用唯讀重數，不讀 w.ResPop——理由同 metrics（§五之二）。
	res, com, ind := w.CountPops()
	rows := []struct {
		name string
		idx  int
		got  int
	}{
		{"住宅人口", 2, res}, {"商業人口", 3, com}, {"工業人口", 4, ind},
		{"住宅需求", 5, w.RValve}, {"商業需求", 6, w.CValve}, {"工業需求", 7, w.IValve},
		{"犯罪坡度", 10, w.CrimeRamp}, {"汙染坡度", 11, w.PolluteRamp},
		{"地價均值", 12, w.LVAverage}, {"犯罪均值", 13, w.CrimeAverage},
		{"汙染均值", 14, w.PolluteAverage},
		{"城市等級", 16, w.CityClass}, {"城市評分", 17, w.CityScore},
	}
	fmt.Printf("\n原版自己記在 MiscHis 裡的值 vs remake 同一刻：\n")
	fmt.Printf("%-10s %10s %10s %10s\n", "量", "原版", "remake", "差")
	for _, r := range rows {
		a := int(cf.MiscHis[r.idx])
		fmt.Printf("%-10s %10d %10d %+10d\n", r.name, a, r.got, r.got-a)
	}
}

// bitDiff 把「圖塊編號相同、旗標不同」的格數依旗標拆開。
//
// 這一欄要單獨看：PWRBIT 是每一輪 DoPowerScan 重畫的，兩邊的掃描相位
// 不同就會整片差；ZONEBIT／BURNBIT 差才代表地物本身不一樣。
func bitDiff(a, b *sim.World) string {
	bits := []struct {
		name string
		mask uint16
	}{
		{"PWRBIT", sim.PWRBIT}, {"CONDBIT", sim.CONDBIT}, {"BURNBIT", sim.BURNBIT},
		{"BULLBIT", sim.BULLBIT}, {"ANIMBIT", sim.ANIMBIT}, {"ZONEBIT", sim.ZONEBIT},
	}
	cnt := make([]int, len(bits))
	for x := 0; x < sim.WorldX; x++ {
		for y := 0; y < sim.WorldY; y++ {
			p, q := a.Map[x][y], b.Map[x][y]
			if p&sim.LOMASK != q&sim.LOMASK || p == q {
				continue
			}
			for i, bt := range bits {
				if p&bt.mask != q&bt.mask {
					cnt[i]++
				}
			}
		}
	}
	out := ""
	for i, bt := range bits {
		if cnt[i] == 0 {
			continue
		}
		if out != "" {
			out += "、"
		}
		out += fmt.Sprintf("%s %d", bt.name, cnt[i])
	}
	if out == "" {
		out = "無"
	}
	return out
}

// report 印出兩邊的量與差。
func report(exp, got map[string]int) {
	keys := make([]string, 0, len(exp))
	for k := range exp {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	worst, worstKey := 0.0, ""
	fmt.Printf("%-10s %10s %10s %10s %8s\n", "量", "原版", "remake", "差", "相對")
	for _, k := range keys {
		a, b := exp[k], got[k]
		d := b - a
		rel := 0.0
		switch {
		case a != 0:
			rel = float64(d) / float64(a)
			if rel < 0 {
				rel = -rel
			}
		case d != 0:
			rel = 1
		}
		if rel > worst && !strings.HasPrefix(k, "＊") {
			worst, worstKey = rel, k
		}
		fmt.Printf("%-10s %10d %10d %+10d %7.1f%%\n", k, a, b, d, rel*100)
	}
	fmt.Printf("\n最大相對差 %.1f%%（%s）；＊開頭的量不列入判準（見 metrics 的註解）\n",
		worst*100, worstKey)
}

func cmdDosParity(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr,
			"用法：simtool dosparity <起點.cty> <取樣1.cty> [取樣2.cty …]\n"+
				"\n"+
				"全部都是 DOS 原版自己存出來的城市檔。第一個當起點，\n"+
				"remake 從它出發跑到每個取樣點的 CityTime，再比一組\n"+
				"對亂數不敏感的量。")
		os.Exit(2)
	}
	// 固定種子，理由同 cmdDosParityScen。
	w, err := game.LoadCitySeed(args[0], 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("起點 %s：CityTime %d（%d 年）資金 %d\n",
		args[0], w.CityTime, 1900+w.CityTime/48, w.TotalFunds)

	worst := 0.0
	for _, p := range args[1:] {
		want, err := game.LoadCitySeed(p, 1)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if want.CityTime < w.CityTime {
			fmt.Fprintf(os.Stderr, "%s 的 CityTime %d 比目前的 %d 還早，跳過\n",
				p, want.CityTime, w.CityTime)
			continue
		}
		// 跑到同一刻。一刻是 16 個 frame。
		for w.CityTime < want.CityTime {
			w.Frame()
		}
		fmt.Printf("\n== 取樣點 %s：CityTime %d（%d 年）==\n",
			p, want.CityTime, 1900+want.CityTime/48)
		report(metrics(want), metrics(w))
	}
	_ = worst
}
