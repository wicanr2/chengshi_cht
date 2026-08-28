# 05 — 電力傳導

**推論等級：已確認**（讀 `s_power.c`／`s_zone.c`／`s_sim.c` ＋ 在 oracle 上現搭一張
受控地圖對拍 ＋ 劇本 1 端到端重算）。日期 2026-08-29。
接線：`internal/sim/power.go`。

## 一、它是什麼

從每一座電廠出發，沿著帶 `CONDBIT` 的格子做**堆疊式泛洪**，把走到的格子記進
`PowerMap` 位元圖；之後 `SetZPower` 再把位元圖攤回每一格的 `PWRBIT`。

分區要長大必須先有電（IBM 手冊：`All zones must be powered to develop.`），
所以這是分區成長的前提。

## 二、原版把它拆在三個地方

| 步驟 | 位置 | 做什麼 |
|---|---|---|
| 1 | `s_sim.c:1014 DoSPZone` | `MapScan` 掃到電廠時 `PushPowerStack()`，並累計 `CoalPop`／`NuclearPop` |
| 2 | `s_power.c:186 DoPowerScan` | 從堆疊泛洪，填 `PowerMap` |
| 3 | `s_zone.c:624 SetZPower` | 下一輪 `MapScan` 對每個帶 `CONDBIT` 的格子設或清 `PWRBIT` |

第 3 步有兩個入口：分區中心走 `DoZone`（`s_zone.c:71`），
其餘導電格走 `MapScan` 裡的 `if (NewPower && (CChr & CONDBIT)) SetZPower();`
（`s_sim.c:713`）。

本專案把三步收攏成一次 `World.DoPowerScan()`。**這是安全的**：地圖不變時原版會
收斂到同一個狀態，而驗收就是拿收斂後的原版地圖來比對（§5）。

## 三、四個會讓實作「自洽但錯」的地方

### 1. `TestForCond` 用的是 `MapScan` 留下的全域

```c
if ((Map[SMapX][SMapY] & CONDBIT) &&
    (CChr9 != NUCLEAR) && (CChr9 != POWERPLANT) && …)
```

`CChr9` **不是鄰居的圖塊**，是 `MapScan` 掃到最後一格非零圖塊時留下的全域
（`s_sim.c:701`）。原版把 `TestPowerBit()` 內聯進 `TestForCond` 時，
把它裡面那句「如果目前這格是電廠就當作已通電」一起帶了進來，
而「目前這格」指的是 `MapScan` 的游標，不是泛洪的游標。

後果：**如果 `MapScan` 掃到的最後一格剛好是電廠中心，整個泛洪會一格都走不動。**
實務上很難碰到（電廠中心要是全圖 x 最大、y 最大的非零格），但這是原版行為，
Go 版照樣把 `cChr9` 帶進去。

### 2. 供電上限一超過就整個中止，不是停在那一格

```c
if (++NumPower > MaxPower) { SendMes(40); return; }
```

`return` 直接離開 `DoPowerScan`，**堆疊裡還沒處理的分支全部丟掉**。
所以電力不足時不是「遠處的電線沒電」，而是「掃描中止之後那些格子連
`PowerMap` 的位元都沒設上」。寫成 `break` 會讓多出來的分支照樣通電。

容量：`MaxPower = CoalPop × 700 + NuclearPop × 2000`（`s_power.c:196`，
註解標 `post release`，是發行後才加的）。

### 3. 方向 4 是「原地不動」

`MoveMapSim(4)` 直接回 `TRUE` 而不移動（`s_power.c:109`）。
`DoPowerScan` 每次從堆疊取出位置後把 `ADir` 設成 4，
所以**第一步是讓取出的那一格自己先通電**。少了這個 case，電廠本身不會進 `PowerMap`。

### 4. `PowerStackX/Y` 是 `char`

`char PowerStackX[PWRSTKSIZE]`（`s_power.c:69`）。在 120×100 的地圖上座標最大 119，
塞得進有號 char。**這是尺寸相依的**：地圖一寬過 128 就會溢位。
Go 版用 `int`，並在筆記記下原版的這個限制。

另外 `TestForCond` 寫的是 `PowerWrd > PWRMAPSIZE`（應該是 `>=`）。
`PowerWrd` 的最大值是 `(119>>4) + 99×8 = 799`，而 `PWRMAPSIZE = 800`，
所以這個差一永遠不會被觸發——**照抄，但知道它為什麼無害**。

## 四、`PowerMap` 的版面

位元圖，每列 `POWERMAPROW = (120+15)/16 = 8` 個 16 位元 word，共 100 列 = 800 word。
`POWERWORD(x,y) = (x>>4) + (y<<3)`，位元是 `1 << (x & 15)`。
`(y<<3)` 只是剛好等於 `y × POWERMAPROW`（見 `docs/re/03-map-and-tiles.md`）。

## 五、驗收紀錄

### 受控實驗

在 oracle 上現搭一張地圖（`tools/oracle/tcl/power-experiment.tcl`）：
`sim Fill 0` 清空 → 寫一座 4×4 燃煤電廠 → 一條 57 格橫向電線 →
一條 34 格縱向分支 → 一段 15 格**刻意不相連**的電線 → 讓模擬跑 6 秒收斂 → 倒出全圖。

結果：橫線與分支全通、孤立線全不通、全圖 98 格帶 `PWRBIT`。

Go 版拿同一張地圖、把 `PWRBIT` 全部清掉重算：**12000 格逐格相同**。

### 端到端

劇本 1（Dullsville）載入後用 Go 版重算電力，與 oracle 載入後的地圖比對：
`docs/formats/01-city-file.md` §3.3 原本留下的 **266 格 `PWRBIT` 差異全部消掉**，
只剩 67 格帶 `ANIMBIT` 的動畫格（那是逐刻換幀，本來就不會相同）。

## 六、未解

| 項目 | 怎麼解 |
|---|---|
| `NewPower` 旗標的完整生命週期 | 它決定 `MapScan` 要不要順手更新非分區格的 `PWRBIT`；要等 `Simulate` 的相位實作出來才驗得到 |
| 電力不足時的 40 號訊息 | 等 `s_msg.c` 那一份 |
| `DoNilPower()`（載入城市時把 `PowerMap` 全設 1 再攤開）| 等 `DoSimInit` 的完整順序實作出來 |
