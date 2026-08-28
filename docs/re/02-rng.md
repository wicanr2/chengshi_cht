# 02 — 亂數產生器

**推論等級：已確認**（讀原始碼 ＋ 對活的 oracle 驗證公式）。日期 2026-08-29。
接線：`internal/sim/rand.go`。

## 一、遊戲只用一個亂數源，而且很小

封存裡有兩份亂數程式碼，**只有一份被遊戲用到**：

| 檔案 | 內容 | 遊戲有沒有用 |
|---|---|---|
| `rand.c` | 一個 24 位元的線性同餘產生器 | **有**。`Rand16()` 直接呼叫 `sim_rand()` |
| `random.c` | BSD `random()` 的完整移植（TYPE_3 三項式、`randtbl` 31 個字）| **沒有**。全檔沒有任何遊戲程式碼呼叫 `sim_random()` |

`grep -rn 'sim_random' *.c` 只命中 `random.c` 自己。**這推翻了 `docs/re/00-source-map.md`
初版的說法**（該處寫「亂數是封存自帶的 BSD `random()`」）——BSD 那份是死碼，
真正在跑的是 `rand.c` 那 12 行。結論不變（不依賴系統 libc，所以可重現），
但依據換了，那一列已改正。

## 二、核心：`rand.c:42` `sim_rand()`

```c
#define SIM_RAND_MAX 0xffff
static unsigned QUAD next = 1;

int sim_rand() {
    next = next * 1103515245 + 12345;
    return ((next % ((SIM_RAND_MAX + 1) << 8)) >> 8);
}

void sim_srand(u_int seed) { next = seed; }
```

`QUAD` 在非 OSF1 平台是 `long`（`headers/mac.h:69`），所以在 x86-64 上 `next` 是
**64 位元**。

> ⚠ **這一點看起來會壞事，其實不會，而理由要寫下來。**
> 取值只用 `next % 2²⁴`，而 `(n·A + C) mod 2²⁴` 只依賴 `n mod 2²⁴`——
> 模 2²⁴ 的乘加是封閉的。所以 32 位元與 64 位元的 `unsigned long` 產生**同一串輸出**。
> Go 版用 `uint32` 存低 24 位元即可，不必模擬 64 位元寬度。
> **不要因為「原版是 64 位元」就去模擬 64 位元**——那會是照抄實作細節而不是照抄語意。

輸出範圍：`next mod 2²⁴` 是 0…2²⁴−1，右移 8 位後是 **0…65535**。

## 三、遊戲層介面（`s_sim.c:1195` 起）

```c
#define RANDOM_RANGE 0xffff          /* s_sim.c:1192 */

int Rand16(void)       { return sim_rand(); }

int Rand16Signed(void) { int i = sim_rand(); if (i > 32767) i = 32767 - i; return i; }

Rand(short range) {                   /* 回傳 0..range（含）*/
    int maxMultiple, rnum;
    range++;
    maxMultiple = RANDOM_RANGE / range;
    maxMultiple *= range;
    while ((rnum = Rand16()) >= maxMultiple) continue;
    return rnum % range;
}
```

三件容易寫錯的事：

1. **`Rand(range)` 回傳 `0..range` 含端點**，不是 `0..range-1`。函式一進來就 `range++`。
2. **它會拒絕取樣**（`while … continue`），所以**消耗的亂數個數不固定**。
   逐 tick 對拍時，兩邊的 RNG 呼叫次數必須完全一致，否則之後全部錯開。
3. `Rand16Signed()` 的 `32767 - i`（不是 `i - 65536`）會把 32768…65535 映到
   −32768…−1，**但不是常見的二補數轉換**：`i = 65535` 得到 −32768，`i = 32768` 得到 −1。
   照抄這個式子，不要「修正」成看起來比較對的版本。

## 四、種子從哪裡來

| 位置 | 呼叫 | 決定性？ |
|---|---|---|
| `s_init.c:73` `InitWillStuff()` | `RandomlySeedRand()` | **否**：`gettimeofday` ⊕ 目前狀態 |
| `s_gen.c:129` `GenerateMap(int r)` | `SeedRand(r)` | **是**：地形產生吃明確種子 |
| `s_gen.c:153` `GenerateMap` 結尾 | `RandomlySeedRand()` | **否**：產完地形立刻重新亂數播種 |
| `s_gen.c:84` `GenerateNewCity()` | `GenerateSomeCity(Rand16())` | 否（種子取自目前狀態）|

**對拍的入口是 `sim GenerateSomeCity <r>`**：它把 `r` 直接餵給 `SeedRand`，
所以同一個 `r` 一定產生同一張地形。這是本專案第一個可機械驗收的標的。

⚠ **`GenerateMap` 收尾會重新亂數播種**，所以「產完地形之後」的模擬不是決定性的。
Tcl 介面**沒有** `SeedRand` 子指令，不能直接設種子。可行的替代做法：
連續讀四次 `sim Rand`，從輸出反推內部狀態（見下節），再讓 Go 版對齊。

## 五、狀態可從輸出反推（測試用得上）

輸出是 `next` 的第 8–23 位元，低 8 位元看不到。所以：

- 一個輸出 → 256 個候選狀態
- 每多一個輸出，候選數約除以 256
- **四個輸出足以唯一決定狀態**（實測如此）

驗證紀錄（2026-08-29，活的 oracle，`tools/oracle/tcl/rand.tcl`）：

```
32733 41618 1670 36929 17562 6660 35924 51032 11849 29924 9930 26204
64069 23392 55460 57177 57259 52322 5213 47387 27479 51606 63538 6958
```

用前四個值反推狀態後，**後 20 個值全部預測正確**。公式因此不是讀出來的推測，
是對活的原版驗證過的。

## 六、Go 版的驗收條件

1. `Rand16()` 的數列與上節的黃金樣本一致（從反推的狀態出發）。
2. `Rand(n)` 的分布端點正確：能取到 `n`，取不到 `n+1`。
3. `Rand(n)` 在 `n+1` 不整除 65535 時**會拒絕取樣**，消耗的 `Rand16()` 次數與原版一致。
   測法：記錄呼叫次數，與 oracle 用同一段種子跑同一串 `Rand` 比對。
