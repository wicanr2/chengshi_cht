# 01 — 城市檔格式（`.cty` / `snro.*`）

**推論等級：已確認**（讀 `s_fileio.c` ＋ 32 個檔案逐位元組 round-trip ＋ 對 oracle 載入後的狀態比對）。
日期 2026-08-29。接線：`internal/sim/cityfile.go`、`internal/sim/scenario.go`。

## 一、版面

固定 **27120 位元組**，沒有檔頭、沒有魔數、沒有版本欄位。
所有 16 位元值都是 **big-endian**。

| 位移 | 長度 | 內容 |
|---:|---:|---|
| 0 | 480 | `ResHis[240]` 住宅人口歷史 |
| 480 | 480 | `ComHis[240]` 商業 |
| 960 | 480 | `IndHis[240]` 工業 |
| 1440 | 480 | `CrimeHis[240]` 犯罪 |
| 1920 | 480 | `PollutionHis[240]` 汙染 |
| 2400 | 480 | `MoneyHis[240]` 資金 |
| 2880 | 240 | `MiscHis[120]` 雜項 ＋ 遊戲純量 |
| 3120 | 24000 | `Map[120][100]`，**x 外層、y 內層** |

`s_fileio.c:207-215` 的 `switch` 另外認得 99120（2×2）與 219120（3×3），
那是 MEGA 版的大地圖；**本專案不支援，遇到就拒絕**。

### 位元組序怎麼確定的

`s_fileio.c:69` 的 `NOOP_ON_BE` 巨集：

```c
#define NOOP_ON_BE  { int test = 1; if (!(*(unsigned char*)(&test))) return; }
```

小端機器上 `*(unsigned char*)&test` 是 1，`!1` 為假 → **不 return，執行交換**。
大端機器上第一個位元組是 0 → return，不交換。
所以**檔案裡是大端**，交換是為了轉成本機序。

## 二、`MiscHis` 裡的遊戲純量

| 索引 | 型別 | 內容 |
|---|---|---|
| 8–9 | int32 | `CityTime`（遊戲刻）|
| 50–51 | int32 | `TotalFunds`（市庫）|
| 52 | int16 | `autoBulldoze` |
| 53 | int16 | `autoBudget` |
| 54 | int16 | `autoGo` |
| 55 | int16 | `UserSoundOn` |
| 56 | int16 | `CityTax` |
| 57 | int16 | `SimSpeed` |
| 58–59 | int32 | `policePercent × 65536` |
| 60–61 | int32 | `firePercent × 65536` |
| 62–63 | int32 | `roadPercent × 65536` |

32 位元值在原始碼裡是 `*(QUAD*)(MiscHis + n)` 配 `HALF_SWAP_LONGS`
（交換兩個半字），那是為了對付小端機器上 `long` 的半字順序。
**反推到檔案上，結果就是一個單純的大端 32 位元整數**：
`MiscHis[n]` 是高半字，`MiscHis[n+1]` 是低半字。

## 三、三件會讓實作「自洽但錯」的事

### 1. 載入有兩層，劇本走的是下面那層

```
_load_file(name, dir)   只讀七個陣列
loadFile(name)          = _load_file + 把 MiscHis 的純量套進遊戲狀態
LoadCity(name)          → loadFile
LoadScenario(s)         → 先設劇本表寫死的純量，再直接呼叫 _load_file
```

**`LoadScenario` 跳過 MiscHis 那一層。** 所以 `snro.111` 裡殘留的
`CityTime = 1716` **從來沒有生效過**——劇本表寫死的 `(1900-1900)*48+2 = 2` 才算數。
oracle 實測 `sim Year` 回 1900、`sim Funds` 回 5000，與劇本表一致。

把兩層合併成一個「載入城市檔」函式，劇本的年份與市庫就會錯。

### 2. 撥款百分比是死欄位

`loadFile` 對三個百分比各做了兩次賦值：

```c
l = *(QUAD *)(MiscHis + 58);
HALF_SWAP_LONGS(&l, 1);
policePercent = l / 65536.0;          /* 正確 */
...
policePercent = (*(QUAD*)(MiscHis + 58)) / 65536.0;   /* 漏了 HALF_SWAP —— 錯的 */
```

第二次覆蓋了第一次，而且**漏掉半字交換**，算出來的值差了 65536 倍。
但這個 bug 是死的：`loadFile` 稍後呼叫 `InitFundingLevel()`（`w_budget.c:83`），
把三個百分比全部設回 `1.0`。**檔案裡那三個欄位從來沒有影響過任何一局遊戲。**

本專案照最終行為做：載入後三個百分比一律 1.0，檔案裡的值只在 round-trip 時原樣寫回。

### 3. 載入後的狀態不等於檔案內容

拿 oracle 的 `sim LoadScenario 1` 之後的地圖去比對檔案，會有 333 格不同。
全部有解釋：

| 類別 | 格數 | 原因 |
|---|---:|---|
| 只差 `PWRBIT` | 266 | `DoSimInit()` 跑了一次電力掃描 |
| 帶 `ANIMBIT` 的格子 | 67 | 車流、煙囪、體育場每一刻換一幀 |
| **無法解釋** | **0** | |

**驗收要扣掉這兩類**，否則會誤判解析錯誤。

> 2026-08-29 更新：電力掃描已實作（[`docs/re/05-power-scan.md`](../re/05-power-scan.md)），
> **266 格 `PWRBIT` 差異已經全部收掉**——用 Go 版重算之後與原版逐格相同。
> 現在只剩 67 格動畫幀，那是逐刻換幀，本來就不會相同。

### 4. `sim Tile` 回傳有號值

Tcl 的 `sim Tile x y` 把 `short` 提升成 `int` 再印，所以帶 `PWRBIT`（第 15 位元）
的格子會印成負數。倒出來的黃金資料要用 `uint16()` 轉回去，不能直接當無號數讀。

## 四、驗收紀錄

| 檢查 | 結果 |
|---|---|
| 32 個城市檔（8 個 `snro.*` ＋ 24 個 `.cty`）逐位元組 round-trip | 全部相同 |
| `snro.111` 解出的地圖 vs oracle 載入後的地圖 | 扣掉 `PWRBIT` 與動畫幀後 0 差異 |
| 劇本表八列的年份與起始市庫 | 與 `s_fileio.c:406-447` 相同 |

## 五、未解

| 項目 | 怎麼解 |
|---|---|
| `MiscHis` 其餘 100 多個欄位的語意 | 目前只知道用到的那 14 個；其餘原樣 round-trip |
| ~~DOS 版的 `.PSN` 是不是同一個格式~~ | **已解**：是。解壓之後是 144 位元組檔頭 ＋ 完全相同的 27120。見 [`02-dos-lzss.md`](02-dos-lzss.md) §4 |
| 2×2 與 3×3 大地圖 | 本專案不支援；要支援得先處理 MEGA 版的尺寸巨集 |

## 附錄：存檔的反向驗證（原版讀得動我們寫的檔）

**推論等級：已確認**（實測）。日期 2026-08-29。

解析原版的檔案只證明了一半。另一半是：**我們寫出來的檔案，原版讀不讀得動？**

做法：

```bash
tools/go.sh run ./cmd/simtool save          # 產生 workplace/oracle/roundtrip.cty
tools/oracle/drive.sh tools/oracle/tcl/loadcity.tcl load.json
```

`loadcity.tcl` 在 Micropolis 裡跑 `sim LoadCity /out/roundtrip.cty`，
然後倒出 12000 格與資金、年份。

結果：**資金 18028、年份 1921 完全一致，12000 格地圖逐格相同。**

⚠ 比對時要先把 `sim Tile` 的回傳值還原成無號 16 位元。它宣告成 `short`，
所以設了 `PWRBIT`（0x8000）的格子會回負數——沒還原的話會看到「160 格
不同」，而那 160 格其實一模一樣（−7447 就是 58089）。這一點在本文件
前面已經記過，這裡再踩一次是因為**同一個陷阱換一個方向出現時，
不會自動被想起來**。

意義：存檔用原版格式而不是自創格式，玩家的城市可以在本專案、原版
SimCity 與 Micropolis 之間搬。remake 的存檔如果自成一格，城市就被鎖在
這個實作裡了。
