# 01 — Micropolis oracle：建置、驅動與第一批已確認常數

**推論等級：已確認**（實際建置、實際跑、實際讀出值）。日期 2026-08-29。
接線狀態：見 `00-wiring-status.md`。

## 一、為什麼要做這個

`rulebook/60`：先建可重跑的 pass／fail 迴路，再寫規則。
模擬城市的狀態是一張格子陣列加上幾十個純量，全部可序列化——所以「Go 版對不對」
這件事可以變成**逐 tick 的機械比對**，不必靠眼睛看畫面。

前提是原版要跑得起來、而且能被腳本驅動。兩件事都成立了。

## 二、建置（`tools/oracle/build.sh`）

Micropolis 是 1993 年的 K&R C ＋ 自帶的 Tcl 7／Tk 3／TclX。現代 gcc 與 glibc
會在三個地方擋下來，全部用**外部注入**解決，封存的原始碼一個 byte 都不動：

| 症狀 | 原因 | 處置 |
|---|---|---|
| `'DOMAIN' undeclared`、`'OVERFLOW' undeclared` | SVID `matherr` 的例外代碼在 glibc 2.27 之後從 `<math.h>` 移除 | `docker/compat.h` 補定義 |
| `invalid use of undefined type 'struct exception'` | 同上，`struct exception` 也被移除 | `docker/compat.h` 補結構 |
| `yacc: No such file or directory` | TclX 的日期解析器要 yacc | image 裝 `byacc` |

注入方式是 **gcc 包裝器**：`/usr/local/bin/gcc` 轉呼叫真正的 gcc 並補上
`-std=gnu89 -fcommon -fno-strict-aliasing -w -include /opt/compat.h`。
之所以不用環境變數，是因為**各層 makefile 自己寫死 `CFLAGS`，`CFLAGS=` 傳不進去**——
第一次嘗試就是這樣靜默失效的（看起來像旗標沒作用，其實是根本沒被讀到）。

產物：`workplace/ref/micropolis/micropolis-activity/res/sim`（約 1 MB）。

## 三、驅動（`tools/oracle/drive.sh` ＋ `drive.py`）

`sim` 內嵌一個 Tcl 直譯器，並註冊了一個 `sim` 指令與 **128 個子指令**
（`w_sim.c:1567` 起）。這就是狀態存取介面。

### ⚠ 一定要 pty，不能用管線

`sim -t` 只有在 **`isatty(0)` 為真**時才註冊 stdin 的 Tcl 讀取器
（`sim.c:629` 的 `sim_tty = isatty(0)`，`w_tk.c:808` 的 `Tk_CreateFileHandler`）。
stdin 是管線時 `sim_tty` 留 0，**REPL 根本不會開，而且不印任何錯誤**——
症狀是「指令送進去沒反應」，很容易誤判成指令名寫錯。
`drive.py` 用 `pty.openpty()` 解決。

協定：tty 模式下每執行完一條指令會印一行 `sim:`（`w_tk.c:541`），以它切分結果。

### 對拍用得到的子指令

| 類別 | 子指令 |
|---|---|
| 地圖 | `Tile x y`（讀寫單格）、`Fill`、`ClearMap`、`ClearUnnatural` |
| 純量 | `Funds` `TaxRate` `Year` `CityName` `GameLevel` `Speed` `Skips` `Skip` `Delay` |
| 統計 | `LandValue` `Traffic` `Crime` `Unemployment` `Fires` `Pollution` `Votes` `Dollars` |
| 極值座標 | `PolMaxX/Y` `TrafMaxX/Y` `CrimeMaxX/Y` `MeltX/Y` `FloodX/Y` `CrashX/Y` |
| 流程 | `Pause` `Resume` `InitGame` `GameStarted` `Update` |
| 載入 | `LoadScenario` `LoadCity` `SaveCityAs` `GenerateNewCity` `GenerateSomeCity` |
| 災難 | `MakeFire` `MakeFlood` `MakeTornado` `MakeEarthquake` `MakeMonster` `MakeMeltdown` `MakeAirCrash` `FireBomb` |
| 亂數 | `Rand` |
| 撥款 | `FireFund` `PoliceFund` `RoadFund` `AutoBudget` |

## 四、第一批已確認常數

冒煙測試（`tools/oracle/tcl/smoke.tcl`）讀出來、並回原始碼對照過：

| 常數 | 值 | 出處 |
|---|---|---|
| 版本 | `4.0` | `sim Version` |
| **地圖寬 × 高** | **120 × 100** | `headers/sim.h:156-157` `#define SimWidth 120` / `SimHeight 100`；`sim WorldX`／`WorldY` 實測 |
| 起始年 | 1900 | `sim.c:180 StartingYear = 1900` |
| **起始稅率** | **7 %** | `sim.c:182 CityTax = 7` |
| 起始資金 | 20000 | `sim Funds` 實測 |
| 起始 `CityTime` | 50 | `sim.c:183` |
| 起始 `SimSpeed` | 3 | `sim.c:194` |
| 自動推平／自動預算 | 兩者預設開 | `sim.c:188-189` |
| 城市存檔大小 | 27120 bytes | `s_fileio.c:207 case 27120: /* Normal city */`；24 個 `.cty` 全部是這個大小 |

### 這批數字推翻了兩條二手說法

1. **地圖不是 128 × 100，也不是 128 × 128。** Tony Chen 2002 年的規格表寫
   「建設範圍 128 × 128」（`docs/research/player-voices.md` 1-1），
   README 初版照抄了。一手是 **120 × 100**。
   存檔大小佐證：`120 × 100 × 2 bytes = 24000`，加上歷史統計等於 27120。
   > DOS 版是不是同一個尺寸還沒證實——要從 `.PSN` 劇本檔或 `SIMCITY.EXE` 自證。
   > 在那之前不得把 120×100 寫成「DOS 版的尺寸」。
2. **「7 % 稅率」不是純迷信，它就是遊戲的預設值。** 手冊寫的是
   「optimum for fast growth is between 5 and 7%」（區間），
   而原始碼把初始值設成 7。玩家記得的 7 % 同時是**預設值**與**手冊區間的上緣**。

## 五、已知限制

- oracle 是 **Micropolis（X11 版血緣）**，不是 DOS 1.10。規則層可以信，
  畫面、訊息文字、劇本編號與 DOS 版的差異要另外查。
  已知一例：啟動畫面寫 **Tokyo 1967**，IBM 手冊寫 **TOKYO, JAPAN 1957**。
- `sim ReallyQuit` 之後行程仍需 `terminate()`，`drive.py` 已處理。
- 容器內 `/bin/sh` 是 dash，`sim` 啟動時會噴兩行
  `sh: 1: Syntax error: Bad fd number`——來自它自己呼叫 shell 的地方，不影響 REPL。
