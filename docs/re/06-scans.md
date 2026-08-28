# 06 — 四個逐格掃描

**推論等級：已確認**（讀 `s_scan.c`／`s_zone.c` ＋ 在 oracle 上現搭受控地圖，
比對收斂後的三個平均值）。日期 2026-08-29。接線：`internal/sim/scan.go`。

## 一、四個掃描與它們的解析度

| 掃描 | 位置 | 讀 | 寫 | 解析度 |
|---|---|---|---|---|
| `PopDenScan` | `s_scan.c:93` | `Map` 的分區中心 | `PopDensity`、`CCx/CCy`、`ComRate` | 半（60×50）|
| `PTLScan` | `s_scan.c:167` | `Map`、`TerrainMem`、`PollutionMem`、`CrimeMem` | `PollutionMem`、`LandValueMem`、`TerrainMem`、`Qtem` | 半 ＋ 四分之一 |
| `CrimeScan` | `s_scan.c:300` | `LandValueMem`、`PopDensity`、`PoliceMap` | `CrimeMem`、`PoliceMapEffect` | 半 |
| `FireAnalysis` | `s_scan.c:77` | `FireStMap` | `FireRate` | 八分之一（15×13）|

**它們互相回饋**：`PTLScan` 的地價要用上一輪的汙染與犯罪，`CrimeScan` 的犯罪要用
這一輪的地價。地圖不變時會收斂到固定點——驗收就是驗那個固定點。

## 二、兩個平滑核不一樣，不可互換

| 函式 | 核 | 用在 |
|---|---|---|
| `DoSmooth` / `DoSmooth2`（`s_scan.c:407`、`:455`）| `(四鄰居和 + 自己) >> 2` | 人口密度、汙染 |
| `SmoothFSMap` / `SmoothPSMap`（`s_scan.c:486`、`:507`）| `((四鄰居和 >> 2) + 自己) >> 1` | 消防、警察涵蓋 |
| `SmoothTerrain`（`s_scan.c:365`）| `(uint8)((四鄰居和>>2) + 自己) >> 1` | 地形 |

三件事要注意：

1. **`DoSmooth` 是除以 4 不是除以 5**（四個鄰居加自己共五項）。
   邊界上少算的鄰居也不補，所以邊緣天生偏低。這不是 bug，是原版的行為。
2. **`SmoothTerrain` 的截斷點在括號裡面**：
   `(unsigned char)((z >> 2) + Qtem[x][y]) >> 1`
   —— 先加、**先截成 8 位元**、再右移。寫成先右移再截斷，超過 255 的格子就不一樣。
3. `DoSmooth`／`DoSmooth2`／`SmoothTerrain` 各有一條 `DonDither` 的替代分支
   （蛇行掃描 ＋ 誤差擴散）。`DonDither` 預設 0（`s_scan.c:73`），
   Tcl 有 `sim DonDither` 可以打開。**本專案只實作非 dither 分支**，
   dither 分支列在未解。

## 三、地價公式（`s_scan.c:200`）

只有格子裡有 `>= ROADBASE` 的圖塊（也就是人造物）才算地價：

```
dis  = 34 − 到重心的曼哈頓距離（上限 32）
dis <<= 2
dis += TerrainMem[x>>1][y>>1]
dis -= PollutionMem[x][y]
if CrimeMem[x][y] > 190: dis -= 20
夾限到 1…250
```

沒有人造物的格子地價是 **0**，而且不計入平均。

## 四、汙染貢獻表（`s_scan.c:257 GetPValue`）

| 圖塊範圍 | 值 |
|---|---:|
| 壅塞車流 `HTRFBASE` 以上（且 < `POWERBASE`）| 75 |
| 稀疏車流 `LTRFBASE` 以上 | 50 |
| 火 `FIREBASE` 以上（且 < `ROADBASE`）| 90 |
| 輻射 `RADTILE` 以上（且 < `ROADBASE`）| **255** |
| 工業 `LASTIND`＋1 … `PORTBASE`−1 | 50 |
| 海港／機場／電廠 `PORTBASE` … `LASTPOWERPLANT` | 100 |
| 其餘 | 0 |

原始碼在輻射那一行留了註解 `XXX: Why negative pollution from radiation?`，
旁邊還有被註解掉的舊值 `-40`。**照抄現行值 255。**

## 五、城市重心的單位不一致

```c
if (Ztot) { CCx = Xtot / Ztot; CCy = Ytot / Ztot; }
else      { CCx = HWLDX;       CCy = HWLDY; }      /* s_scan.c:130 */
```

有分區時 `CCx/CCy` 是**全解析度**座標（分區中心的平均）；
沒有分區時卻指派**半解析度**的常數 `HWLDX`(60)／`HWLDY`(50)。
單位不一致，但那是原版行為，照抄。

（順帶一提，全解析度的中心其實也是 60／50，所以這個不一致在 120×100 的地圖上
看不出來——換個尺寸就會露餡。）

## 六、平手時擲骰：兩對最大值座標不是決定性的

`PTLScan` 與 `CrimeScan` 找最大值時：

```c
if ((z > pmax) || ((z == pmax) && (!(Rand16() & 3))))
```

**平手有四分之一機率換位置**。所以 `PolMaxX/Y`（怪獸的目標）與 `CrimeMaxX/Y`
不是決定性的，不能拿來對拍。實測同一張地圖跑兩次：
三個平均值完全相同，但 `PolMaxX` 從 1192 變成 1224。

> Tcl 的 `sim PolMaxX` 回傳的是**像素座標** `(PolMaxX << 4) + 8`
> （`w_sim.c:1106`），不是格子座標。

## 七、驗收紀錄

受控實驗（`tools/oracle/tcl/scan-experiment.tcl`）：一張**完全不含 `ZONEBIT`
分區**的地圖，這樣重心退化成固定值、警消涵蓋圖全零，其餘輸入完全由地圖決定。
用的圖塊（650 工業美術、700 港區、電線）都不會被 `MapScan` 改動——
跑完倒回來的地圖與寫進去的完全一樣，證實了這一點。

| 值 | oracle | Go 版 |
|---|---:|---:|
| 地價平均 `LVAverage` | 25 | 25 |
| 汙染平均 `PolluteAverage` | 144 | 144 |
| 犯罪平均 `CrimeAverage` | 102 | 102 |

三個值各由 3000 個半解析度格子算出來。oracle 端跑兩次結果相同。

## 八、未解

| 項目 | 怎麼解 |
|---|---|
| `DonDither` 的蛇行 ＋ 誤差擴散分支 | Tcl 有 `sim DonDither`，打開後重跑同一個實驗即可 |
| 逐格的中間值（不只是平均）| 那些陣列沒有 Tcl 存取器；要嘛截小地圖的畫面解色，要嘛等整個 tick 對拍 |
| `PoliceMap`／`FireStMap` 的填入 | 在 `DoSPZone` 裡，等 `w_tool.c`／`s_zone.c` 那一份 |
