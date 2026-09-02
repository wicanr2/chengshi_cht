// citymap 把一張 120×100 的 ASCII 粗胚地形轉成原版格式的城市檔。
//
// 為什麼不直接寫圖塊編號：岸線與林緣有一整套邊界圖塊規則，寫在
// `s_gen.c` 的 smoothRiver／smoothTrees 裡。工具只填「這一格是水／陸／林」，
// 邊界交給引擎自己那兩支跑（internal/sim 的 SmoothTerrain），
// 產出的地形因此與原版的地形產生器同一套外觀，不是另寫一份會漂移的複製品。
//
// 粗胚的字元：
//
//	.  空地（可蓋）
//	~  水面
//	T  樹林
//
// 用法：
//
//	go run ./tools/citymap <粗胚.txt> <輸出.cty> <城市名>
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "用法：citymap <粗胚.txt> <輸出.cty> <城市名>")
		os.Exit(2)
	}
	src, dst, name := os.Args[1], os.Args[2], os.Args[3]

	rows, err := readMask(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 種子固定：地形本身完全由粗胚決定，這個世界不靠亂數。
	w := sim.NewWorld(1)
	var land, water, trees int
	for y := 0; y < sim.WorldY; y++ {
		for x := 0; x < sim.WorldX; x++ {
			switch rows[y][x] {
			case '~':
				// 先全部填成河岸；smoothRiver 會把內部換成 RIVER、
				// 邊緣換成對應的十六種岸線圖塊。
				w.Map[x][y] = sim.REDGE
				water++
			case 'T':
				w.Map[x][y] = sim.WOODS + sim.BLBNBIT
				trees++
			default:
				w.Map[x][y] = sim.DIRT
				land++
			}
		}
	}
	w.SmoothTerrain()
	w.CityName = name

	if err := game.SaveCityAs(dst, w, game.SaveWithHeader); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s → %s（%s）：陸地 %d、水面 %d、樹林 %d，共 %d 格\n",
		src, dst, name, land, water, trees, land+water+trees)
}

// readMask 讀粗胚並檢查尺寸。尺寸不對就直接失敗——沉默補齊會讓地圖悄悄
// 平移或截斷，而那種錯在畫面上看起來完全正常。
func readMask(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rows []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) != sim.WorldX {
			return nil, fmt.Errorf("%s 第 %d 列有 %d 個字元，應為 %d",
				path, len(rows)+1, len(line), sim.WorldX)
		}
		rows = append(rows, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(rows) != sim.WorldY {
		return nil, fmt.Errorf("%s 有 %d 列，應為 %d", path, len(rows), sim.WorldY)
	}
	return rows, nil
}
