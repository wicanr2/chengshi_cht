# 04 — 地形產生

**推論等級：已確認**（讀 `s_gen.c` ＋ 四顆種子逐格對拍，48000 格全部相同）。
日期 2026-08-29。接線：`internal/sim/terrain.go`。

## 一、為什麼從這裡開始

`GenerateMap(int r)` 一進來就 `SeedRand(r)`（`s_gen.c:129`），所以**同一個種子
一定產生同一張地圖**。Tcl 介面直接暴露 `sim GenerateSomeCity <r>`，
於是「Go 版對不對」這件事變成 12000 格的機械比對，不必看畫面。

這是整個專案第一個真正的 parity 驗收，也是所有後續規則的地基：
地形錯了，之後的地價、汙染、交通全部沒有意義。

## 二、流程（`s_gen.c:127 GenerateMap`）

```
SeedRand(r)
if CreateIsland < 0 and Rand(100) < 10:      # 一成機率
        MakeIsland(); return                 # ← 提前 return，下面全部不做
if CreateIsland == 1: MakeNakedIsland()
else:                 ClearMap()
GetRandStart()
if CurveLevel != 0: DoRivers()
if LakeLevel  != 0: MakeLakes()
SmoothRiver()
if TreeLevel  != 0: DoTrees()
RandomlySeedRand()                           # ← 本專案不照做，見下
```

四個旋鈕的初值都是 **−1**（`s_gen.c:76-79`）。
**−1 與 0 不是同一件事**：−1 是「用預設的隨機量」，0 是「完全不做」。

## 三、六個會產生「自洽但錯」結果的地方

這一節是本篇的重點。下面每一條單獨看都很小，但錯任何一條，**產出的地圖仍然
看起來像一張正常的地圖**——只是跟原版不一樣，而且從第一個分歧點之後全錯。

1. **提前 return 的造島分支。** `Rand(100) < 10` 命中時 `MakeIsland()` 之後直接
   `return`，`GetRandStart`／`DoRivers`／`MakeLakes` 全部不執行。
   把它寫成 `if…else` 而不是 `return`，一成的種子會走錯路。
   > 種子 5 走這條，種子 7／12345／4242 走一般路徑。測試四顆都要有。

2. **`DoRivers` 的第三段沒有重設 `Dir`。**（`s_gen.c:395`）
   ```c
   LastDir = Rand(3); Dir = LastDir; DoBRiv();     /* 第一條大河 */
   MapX=XStart; MapY=YStart;
   LastDir = LastDir ^ 4; Dir = LastDir; DoBRiv(); /* 第二條大河，反向 */
   MapX=XStart; MapY=YStart;
   LastDir = Rand(3);            /* ← 只設 LastDir */
   DoSRiv();                     /* ← Dir 沿用第二條大河結束時的值 */
   ```
   看起來像漏寫，但它決定了小河的初始方向。照抄。

3. **`DoBRiv` 的轉向一定取兩次亂數。**
   ```c
   if (Rand(r1) < 10) Dir = LastDir;
   else { if (Rand(r2) > 90) Dir++;  if (Rand(r2) > 90) Dir--; }
   ```
   兩個 `if` 是並列不是 `else if`——第一個成立時第二個照樣取一次亂數。
   合併成 `else if` 會少消耗一次亂數，之後全部錯開。

4. **`SmoothRiver` 與 `SmoothTrees` 用 `register short MapX, MapY` 遮蔽了全域。**
   （`s_gen.c:318`、`:367`）所以這兩個函式**不會**移動河流繪製留下的游標。
   把它們寫成操作全域游標，後面的樹會從錯誤的位置開始長。

5. **`PutOnMap` 的 `if (temp = Map[Xloc][Yloc])` 是賦值兼判斷。**（`s_gen.c:476`）
   只有原本非零的格子才進入「水面讓路」判斷；原本是 `DIRT`（0）的格子直接覆蓋。
   讀成比較（`==`）會讓每一格都跑讓路判斷，河的形狀就變了。

6. **`WOODS_HIGH` 是 `UNUSED_TRASH2`（39）。**（`s_gen.c:71`）
   樹的判定上界是一個被原始碼自己標成「未用」的編號。
   看起來像 bug，但 `IsTree` 與 `SmoothTrees` 都吃它。照抄。

另外兩個較小但一樣會錯的地方：

- `SmoothTrees` 的 `if (temp != WOODS) if ((MapX+MapY)&1) temp -= 8;`
  ——只有非 `WOODS` 的邊界圖塊才做棋盤格變體。
- `BRivPlop`／`SRivPlop` 的矩陣是 `Matrix[y][x]`，但位移傳 `(x, y)`。
  兩個矩陣剛好對稱所以結果相同，但照抄比較安全。

## 四、`RandomlySeedRand()` 刻意不照做

`GenerateMap` 的最後一行是 `RandomlySeedRand()`（`s_gen.c:153`），
用 `gettimeofday` 重新播種。**本專案不做**：不決定性的東西不進規則層。
呼叫端要接著模擬時自己給種子。

這也是為什麼對拍腳本要在 `GenerateSomeCity` 之後**立刻**倒地圖——
再往後就進入非決定性區域了。

## 五、驗收紀錄

| 種子 | 路徑 | 水面格數 | 結果 |
|---|---|---:|---|
| 5 | 造島 | 6246 | 12000 格全部相同 |
| 7 | 一般 | 1939 | 12000 格全部相同 |
| 12345 | 一般 | 1764 | 12000 格全部相同 |
| 4242 | 一般 | 2006 | 12000 格全部相同 |

黃金資料在 `internal/sim/testdata/terrain-seed*.csv`，取法見
`tools/oracle/tcl/terrain-*.tcl`。**這是執行原版程式得到的輸出，
不是原版的資料檔**，所以進版控。

種子 12345 另外驗過**同一顆種子跑兩次結果相同**，確認 oracle 那一端沒有隱藏的
不決定性。

## 六、未解

| 項目 | 怎麼解 |
|---|---|
| `TreeLevel`／`LakeLevel`／`CurveLevel`／`CreateIsland` 非預設值的行為 | Tcl 有 `sim TreeLevel` 等存取器，可以直接設了再產生對拍 |
| `SmoothWater()`（`s_gen.c:522`）沒有被 `GenerateMap` 呼叫 | 它是 Tcl 暴露的獨立指令，用途待查 |
| DOS 版的地形產生是否相同 | 需要 DOS 版自證；目前無證據 |
