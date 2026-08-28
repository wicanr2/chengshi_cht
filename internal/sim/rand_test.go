package sim

import "testing"

// oracleRandSequence 是 2026-08-29 從活的 Micropolis oracle 取回的黃金樣本：
// 連續 24 次 `sim Rand`（無參數 → Rand16）。
// 取法見 tools/oracle/tcl/rand.tcl 與 docs/re/02-rng.md §5。
var oracleRandSequence = []int{
	32733, 41618, 1670, 36929, 17562, 6660, 35924, 51032,
	11849, 29924, 9930, 26204, 64069, 23392, 55460, 57177,
	57259, 52322, 5213, 47387, 27479, 51606, 63538, 6958,
}

// 驗收條件 1（docs/spec/rng.md）：從黃金樣本的前四個值反推狀態之後，
// 必須能逐項預測其餘 20 個值。這一條同時驗證了轉移式與取值式。
func TestRand16MatchesOracleSequence(t *testing.T) {
	const seen = 4
	state, ok := RecoverState(oracleRandSequence[:seen])
	if !ok {
		t.Fatalf("看了 %d 個輸出仍無法唯一反推狀態", seen)
	}
	r := &Rand{}
	r.SetState(state)
	for i, want := range oracleRandSequence[seen:] {
		if got := r.Rand16(); got != want {
			t.Fatalf("第 %d 個值：得到 %d，原版是 %d", seen+i, got, want)
		}
	}
}

// 反推要真的唯一：兩個輸出還不夠時不能謊報成功。
func TestRecoverStateNeedsEnoughOutputs(t *testing.T) {
	if _, ok := RecoverState(oracleRandSequence[:1]); ok {
		t.Error("只有一個輸出不該能唯一反推")
	}
	if _, ok := RecoverState(oracleRandSequence[:4]); !ok {
		t.Error("四個輸出應該足以唯一反推")
	}
}

// 種子語意：Seed 之後的第一個輸出等於 ((seed*A+C) mod 2^24) >> 8，
// 種子本身不會被輸出（docs/spec/rng.md 不變量 4）。
func TestSeedDoesNotEmitSeedItself(t *testing.T) {
	const seed = 1 // rand.c:40 的初值
	r := NewRand(seed)
	want := int(((seed*randA + randC) & randMask) >> 8)
	if got := r.Rand16(); got != want {
		t.Fatalf("種子 %d 之後第一個值：得到 %d，應為 %d", seed, got, want)
	}
}

// 驗收條件 2：Rand(n) 的值域是 [0, n] 閉區間——上界要取得到。
func TestRandIsInclusiveOfN(t *testing.T) {
	for _, n := range []int{0, 1, 2, 5, 99} {
		r := NewRand(1)
		seenMax, seenMin := false, false
		for i := 0; i < 200000; i++ {
			v := r.Rand(n)
			if v < 0 || v > n {
				t.Fatalf("Rand(%d) 回傳 %d，超出 [0,%d]", n, v, n)
			}
			if v == n {
				seenMax = true
			}
			if v == 0 {
				seenMin = true
			}
			if seenMax && seenMin {
				break
			}
		}
		if !seenMax {
			t.Errorf("Rand(%d) 取不到上界 %d", n, n)
		}
		if !seenMin {
			t.Errorf("Rand(%d) 取不到 0", n)
		}
	}
}

// 驗收條件 3：拒絕取樣真的會發生——Rand(n) 消耗的 Rand16 次數不固定。
// 這一條守著「兩邊呼叫次數必須一致」那個前提：如果哪天有人把拒絕取樣
// 改成取模，這個測試會紅。
func TestRandRejectsSamples(t *testing.T) {
	// n+1 = 7 不整除 65535 的倍數關係：maxMultiple = (65535/7)*7 = 65534，
	// 所以 65534 與 65535 兩個值會被拒絕。
	r := NewRand(1)
	counter := &countingRand{r: r}
	calls := 0
	for i := 0; i < 300000; i++ {
		before := counter.n
		counter.Rand(6)
		if counter.n-before > 1 {
			calls++
		}
	}
	if calls == 0 {
		t.Error("Rand(6) 從來沒有拒絕過取樣——拒絕取樣可能被改掉了")
	}
}

type countingRand struct {
	r *Rand
	n int
}

func (c *countingRand) Rand16() int {
	c.n++
	return c.r.Rand16()
}

func (c *countingRand) Rand(n int) int {
	n++
	maxMultiple := (randomRange / n) * n
	for {
		v := c.Rand16()
		if v < maxMultiple {
			return v % n
		}
	}
}

// Rand16Signed 照抄 32767-i，不是二補數轉換（docs/spec/rng.md 不變量 3）。
func TestRand16SignedIsNotTwosComplement(t *testing.T) {
	cases := map[int]int{65535: -32768, 32768: -1, 32767: 32767, 0: 0}
	for in, want := range cases {
		r := &Rand{}
		// 直接構造：讓下一個 Rand16 回傳 in
		r.SetState(uint32(in) << 8)
		got := r.rand16SignedFrom(in)
		if got != want {
			t.Errorf("Rand16Signed(%d) = %d，應為 %d", in, got, want)
		}
	}
}

// rand16SignedFrom 只在測試裡用：把 s_sim.c:1216 的轉換套在給定值上。
func (r *Rand) rand16SignedFrom(i int) int {
	if i > 32767 {
		i = 32767 - i
	}
	return i
}
