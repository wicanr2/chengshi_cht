// Package sim 是模擬規則層。它不認識畫面、不認識 Ebiten，也不做 I/O。
//
// 規格來源一律標在用到結論的地方，格式是 `檔名:行號 函式名`，指向 Micropolis
// 封存的原始檔。封存身分（commit 與逐檔 SHA-256）記在機制筆記裡，不在程式碼裡
// —— 那是索引，不是規則，寫進來會讓接線檢查誤判成「這份索引已經接上了」。
package sim

import "time"

// Rand 是原版的亂數產生器。
//
// 規格：docs/spec/rng.md（READY）／證據：docs/re/02-rng.md
// 一手出處：rand.c:42 sim_rand()、s_sim.c:1192-1225
//
// 原版的狀態變數是 unsigned long（headers/mac.h:69 的 QUAD），在 x86-64 上是
// 64 位元；但取值只用 next mod 2^24，而模 2^24 的乘加封閉，所以低 24 位元的
// 演化與整數寬度無關。這裡只留低 24 位元。
type Rand struct {
	next uint32 // 只有低 24 位元有意義

	// Watch 不是 nil 的時候，每抽一次就被呼叫一次。
	//
	// 只給逐次元對拍的診斷用（見 docs/re/12-tick-parity.md §5）：
	// 「這一刻比原版多抽了一次」要知道多的是哪一個呼叫點，
	// 而那件事沒辦法從外面觀察——狀態只看得到結果，看不到來源。
	// 平常是 nil，一個 nil 檢查的代價換掉整輪的猜測。
	Watch func()
}

const (
	randA    = 1103515245 // rand.c:45
	randC    = 12345      // rand.c:45
	randMask = 1<<24 - 1  // (SIM_RAND_MAX+1)<<8 - 1，rand.c:46

	// s_sim.c:1192 #define RANDOM_RANGE 0xffff
	randomRange = 0xffff
)

// NewRand 建立一個以 seed 起始的產生器。
// 對應 rand.c:50 sim_srand()。原版的初值是 1（rand.c:40）。
func NewRand(seed uint32) *Rand {
	return &Rand{next: seed & randMask}
}

// Seed 重設種子。對應 rand.c:50 sim_srand()。
func (r *Rand) Seed(seed uint32) { r.next = seed & randMask }

// State 回傳目前的內部狀態（低 24 位元）。測試與對拍用。
func (r *Rand) State() uint32 { return r.next & randMask }

// SetState 直接設定內部狀態。用於把 Go 版對齊到 oracle 的當前狀態
// （原版沒有這個介面；Tcl 也沒有 SeedRand 子指令，見 docs/re/02-rng.md §4）。
func (r *Rand) SetState(s uint32) { r.next = s & randMask }

// Rand16 取一次值，回傳 0…65535。
// rand.c:42 sim_rand() ＋ s_sim.c:1209 Rand16()
//
// 這是**唯一**推進內部狀態的地方，Rand／Rand16Signed／ERand 都走這裡。
// 對拍要問「這一刻多抽了哪一次」時，掛 Watch 就能逐次記下呼叫者。
func (r *Rand) Rand16() int {
	r.next = (r.next*randA + randC) & randMask
	if r.Watch != nil {
		r.Watch()
	}
	return int(r.next >> 8)
}

// Rand16Signed 對應 s_sim.c:1216。
//
// 注意那個式子是 32767 - i，不是二補數轉換：i = 65535 得到 -32768、
// i = 32768 得到 -1。照抄，不要「修正」。
func (r *Rand) Rand16Signed() int {
	i := r.Rand16()
	if i > 32767 {
		i = 32767 - i
	}
	return i
}

// Rand 回傳 0…n 的閉區間亂數。s_sim.c:1195 Rand(short range)
//
// 原版一進來就 range++，所以上界包含 n。它用拒絕取樣去掉模數偏差，
// 因此**每次呼叫消耗的 Rand16 次數不固定**——逐 tick 對拍時兩邊的呼叫次數
// 必須一致，否則之後的數列全部錯開。
func (r *Rand) Rand(n int) int {
	n++
	maxMultiple := (randomRange / n) * n
	for {
		v := r.Rand16()
		if v < maxMultiple {
			return v % n
		}
	}
}

// ERand 取兩次 Rand(limit) 並回傳較小者，偏向小值。
// s_gen.c:115 ERand(short limit)
func (r *Rand) ERand(limit int) int {
	z := r.Rand(limit)
	x := r.Rand(limit)
	if z < x {
		return z
	}
	return x
}

// RecoverState 從一串連續的 Rand16 輸出反推內部狀態。
//
// 輸出只給出 next 的第 8–23 位元，低 8 位元看不見，所以一個輸出對應 256 個
// 候選狀態；每多一個輸出候選數約除以 256，實測四個輸出就唯一
// （docs/re/02-rng.md §5）。
//
// 回傳的是**產生 outs 最後一個值之後**的狀態，也就是接下來呼叫 Rand16
// 會從這個狀態往前走。ok 為 false 表示候選不唯一或無解。
func RecoverState(outs []int) (state uint32, ok bool) {
	if len(outs) < 2 {
		return 0, false
	}
	var cands []uint32
	for lo := 0; lo < 256; lo++ {
		cands = append(cands, uint32(outs[0])<<8|uint32(lo))
	}
	for _, want := range outs[1:] {
		var next []uint32
		for _, c := range cands {
			n := (c*randA + randC) & randMask
			if int(n>>8) == want {
				next = append(next, n)
			}
		}
		cands = next
		if len(cands) == 0 {
			return 0, false
		}
	}
	if len(cands) != 1 {
		return 0, false
	}
	return cands[0], true
}

// RandomSeed 產生一個隨機種子。
//
// 原版是 `time.tv_usec ^ time.tv_sec ^ sim_rand()`（s_sim.c:1233
// RandomlySeedRand）。這裡用時間就夠——種子只影響地形，
// 不影響任何需要重現的東西；要重現就用固定種子。
func RandomSeed() uint32 {
	return uint32(time.Now().UnixNano()) & randMask
}
