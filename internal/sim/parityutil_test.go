package sim

import "testing"

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
