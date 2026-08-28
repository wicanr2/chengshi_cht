package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// 對拍測試共用的小工具。

// mapDiffM 回傳兩張地圖有幾格不同。
func mapDiffM(a, b *[WorldX][WorldY]uint16) int {
	d := 0
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			if a[x][y] != b[x][y] {
				d++
			}
		}
	}
	return d
}

// advanceRand 把 LCG 狀態往前推 n 步（rewindRand 的反向）。
func advanceRand(s uint32, n int) uint32 {
	for i := 0; i < n; i++ {
		s = (s*randA + randC) & randMask
	}
	return s
}

// drawsBetween 算兩個 LCG 狀態相距幾次抽樣。
//
// 亂數狀態能不能對上，等價於「消耗的次數對不對」——LCG 的狀態
// 由起點與步數唯一決定。所以對拍其實是在比對抽樣次數。
func drawsBetween(a, b uint32) int {
	n, s := 0, a
	for ; n < 5000000 && s != b; n++ {
		s = (s*randA + randC) & randMask
	}
	return n
}

func mustRec(v []int) uint32 { s, _ := RecoverState(v); return s }

func recoverOrDie(t *testing.T, v []int) uint32 {
	s, ok := RecoverState(v)
	if !ok {
		t.Fatalf("反推不出亂數狀態：%v", v)
	}
	return s
}

// segCP 是一個分段檢查點：原版在該點的亂數讀數、資金，以及從上一點
// 到這一點消耗的抽樣次數。由 tools/oracle/tcl/tick-parity-seg.tcl 產生。
type segCP struct {
	K     int   `json:"k"`
	Rands []int `json:"rands"`
	Funds int   `json:"funds"`
	Draws *int  `json:"draws"`
}

// loadSegMaps 讀出 23 個檢查點的地圖。
//
// 只有 cp0 存完整的 12000 格；其餘存成相對前一點的差異
// （`x,y,圖塊` 每行一格）。這批地圖幾乎完全一樣——存全量要 872 KB，
// 存差異只要 52 KB，而且看得出每段到底動了哪幾格。
func loadSegMaps(t *testing.T, n int) [][WorldX][WorldY]uint16 {
	out := make([][WorldX][WorldY]uint16, n)
	out[0] = loadGoldenMap(t, "testdata/seg/cp0.csv")
	for k := 1; k < n; k++ {
		out[k] = out[k-1]
		b, err := os.ReadFile(fmt.Sprintf("testdata/seg/cp%d.diff", k))
		if err != nil {
			t.Fatal(err)
		}
		for _, ln := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			if ln == "" {
				continue
			}
			var x, y, v int
			if _, err := fmt.Sscanf(ln, "%d,%d,%d", &x, &y, &v); err != nil {
				t.Fatalf("cp%d.diff 這行讀不了：%q", k, ln)
			}
			out[k][x][y] = uint16(v)
		}
	}
	return out
}

func loadSegMeta(t *testing.T) []segCP {
	b, err := os.ReadFile("testdata/seg/meta.json")
	if err != nil {
		t.Fatal(err)
	}
	var m []segCP
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
