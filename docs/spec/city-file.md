# 規格 — 城市檔讀寫　**READY**

證據：[`docs/formats/01-city-file.md`](../formats/01-city-file.md)。
一手出處：`s_fileio.c`。

## 分層

| 層 | 型別／函式 | 責任 |
|---|---|---|
| 序列化 | `ParseCityFile([]byte) (*CityFile, error)`、`(*CityFile).Bytes()` | 只做位元組 ↔ 結構。**逐位元組互逆** |
| 純量視圖 | `(*CityFile).CityTime()` 等 | `MiscHis` 上的具名讀寫，不改變資料 |
| 套用 | `(*World).LoadCityFile(cf)` | ＝ `loadFile()`：套用 MiscHis 純量 ＋ 三個夾限 ＋ 撥款百分比歸 1.0 |
| 套用（劇本）| `(*World).LoadScenarioFile(cf, s)` | ＝ `LoadScenario()`：**不套用 MiscHis 純量**，改用劇本表 |

分層的理由是 CLAUDE.md §4 的「改寫不是重建」：未解的位元組要原樣寫回去，
所以 `MiscHis` 整塊保留，語意欄位只是它上面的視圖。

## 夾限（`s_fileio.c:286-291`）

```
if CityTime < 0            → 0
if CityTax > 20 or < 0     → 7
if SimSpeed < 0 or > 3     → 3
```

## 劇本表（`s_fileio.c:406-447`）

| 編號 | 名稱 | 檔案 | 年份 | `CityTime` | 起始市庫 |
|---:|---|---|---:|---:|---:|
| 1 | Dullsville | `snro.111` | 1900 | 2 | **5000** |
| 2 | San Francisco | `snro.222` | 1906 | 290 | 20000 |
| 3 | Hamburg | `snro.333` | 1944 | 2114 | 20000 |
| 4 | Bern | `snro.444` | 1965 | 3122 | 20000 |
| 5 | Tokyo | `snro.555` | **1957** | 2738 | 20000 |
| 6 | Detroit | `snro.666` | 1972 | 3458 | 20000 |
| 7 | Boston | `snro.777` | 2010 | 5282 | 20000 |
| 8 | Rio de Janeiro | `snro.888` | 2047 | 7058 | 20000 |

`CityTime = (年份 − 1900) × 48 + 2`，一年 48 刻。
編號超出 1…8 夾成 1。載入後稅率 7、速度 3。

> 東京是 **1957**。啟動畫面縮圖的美術上印 1967，但程式碼、
> `res/micropolis.tcl:451` 的訊息字串與 IBM 官方手冊都是 1957。

## 驗收條件

1. 封存裡的 32 個城市檔全部逐位元組 round-trip 相同。
2. `snro.111` 解出的地圖與 oracle 載入後的地圖，**扣掉 `PWRBIT` 差異與
   帶 `ANIMBIT` 的格子後**，零差異。
3. 劇本表八列與原始碼相同。
4. 三個夾限行為正確。
5. 大小不是 27120 就拒絕（含原版認得的 99120 / 219120）。

## 刻意不照做

| 原版 | 本專案 | 為什麼 |
|---|---|---|
| 從檔案讀撥款百分比（讀兩次，第二次是 bug）| 載入後一律 1.0 | `InitFundingLevel()` 隨即覆蓋，那三個欄位從來沒生效過 |
| 支援 2×2／3×3 大地圖 | 拒絕 | MEGA 版的尺寸巨集不在本專案範圍 |
| `loadFile` 順帶做的 `DoSimInit`／UI 失效通知 | 不做 | 那是遊戲流程與呈現層 |
