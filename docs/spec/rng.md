# 規格 — 亂數產生器　**READY**

證據：[`docs/re/02-rng.md`](../re/02-rng.md)。
一手出處：`rand.c:42 sim_rand()`、`s_sim.c:1192-1225`、`headers/mac.h:69`。
封存 SHA-256 見 [`docs/re/00-source-map.md`](../re/00-source-map.md)。

## 狀態

單一 24 位元無號整數 `next`。初值 `1`。

## 轉移與取值

```
next ← (next × 1103515245 + 12345)  mod 2²⁴
out  ← next >> 8                       // 0 … 65535
```

推導：原版的 `next` 是 `unsigned long`，但取值只用 `next mod 2²⁴`，
而模 2²⁴ 的乘加封閉，所以低 24 位元的演化與整數寬度無關。**已確認**。

## 介面

| 函式 | 語意 | 出處 |
|---|---|---|
| `Seed(u uint32)` | `next ← u mod 2²⁴` | `rand.c:50 sim_srand` |
| `Rand16() int` | 回傳一次取值，`0…65535` | `s_sim.c:1209` |
| `Rand16Signed() int` | `i := Rand16(); if i > 32767 { i = 32767 - i }; return i` | `s_sim.c:1216` |
| `Rand(n int) int` | 回傳 `0…n`（**含 n**）| `s_sim.c:1195` |

`Rand(n)` 的完整語意：

```
n ← n + 1
maxMultiple ← (65535 / n) × n          // 整數除法
loop { r ← Rand16(); if r < maxMultiple { return r mod n } }
```

## 不變量（測試要守住）

1. `Rand(n)` 的值域是 `[0, n]` 閉區間。
2. `Rand(n)` **會拒絕取樣**，所以每次呼叫消耗的 `Rand16()` 次數不固定。
   逐 tick 對拍時兩邊的呼叫次數必須一致。
3. `Rand16Signed()` 照抄 `32767 - i`，不得改寫成二補數轉換。
4. `Seed` 之後的第一個輸出必須等於 `((seed×A + C) mod 2²⁴) >> 8`
   —— 種子本身不會被輸出。

## 刻意不照做

| 原版 | 本專案 | 為什麼 |
|---|---|---|
| `next` 用 `unsigned long`（x86-64 上 64 位元）| `uint32`，只留低 24 位元 | 輸出序列證明相同（見上）。模擬 64 位元寬度是照抄實作細節，不是照抄語意 |
| `RandomlySeedRand()` 用 `gettimeofday` | 明確傳入種子 | 不決定性的東西不進規則層。呈現層要隨機開局時自己給種子 |

## 未解

無。
