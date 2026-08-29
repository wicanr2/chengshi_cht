# 11 — 十六相位主迴圈

**推論等級：已確認**（空城與 Dullsville 兩份逐 frame 對拍各 8000/8000，
見 [`12-tick-parity.md`](12-tick-parity.md)）。
日期 2026-08-29。接線：`internal/sim/simulate.go`。

## 一、一刻分成十六個相位

`SimFrame`（`s_sim.c:90`）每個 timer frame 呼叫一次 `Simulate(Fcycle & 15)`，
所以**一個 frame 只走一個相位**，十六個 frame 才是一刻。

| 相位 | 做什麼 |
|---:|---|
| 0 | `Scycle++`、初次評分、`CityTime++`、累計稅、每兩刻 `SetValves`、`ClearCensus` |
| 1–8 | 各掃八分之一張地圖（`MapScan`）|
| 9 | 每 4 刻普查、每 48 刻長期普查、每 48 刻收稅並評分 |
| 10 | 成長率與車流衰退、送訊息 |
| 11 | 電力掃描 |
| 12 | 汙染／地形／地價掃描 |
| 13 | 犯罪掃描 |
| 14 | 人口密度掃描 |
| 15 | 消防涵蓋掃描、災難判定 |

## 二、速度不只是快慢，還改變模擬精度

`SimFrame` 在速度 1 時每五個 frame 才做一次、速度 2 時每三個
（`s_sim.c:99-105`）。**速度 0 直接 return，整個模擬停住**——
對拍的暫停就靠這一條。

而相位 11–15 的掃描還各有自己的週期表（`s_sim.c:114-118`），索引是速度：

| 掃描 | 速度 0 | 1 | 2 | **3** |
|---|--:|--:|--:|--:|
| 電力 | 1 | 2 | 4 | **5** |
| 汙染／地價 | 1 | 2 | 7 | **17** |
| 犯罪 | 1 | 1 | 8 | **18** |
| 人口密度 | 1 | 1 | 9 | **19** |
| 消防 | 1 | 1 | 10 | **20** |

**快轉時模擬得比較粗**：地價每 17 個 `Scycle` 才更新一次。
這不是最佳化，是遊戲設計——而且它會改變結果，所以對拍時速度必須一致。

## 三、`Scycle` 與 `CityTime` 是兩個不同的時鐘

- `CityTime` 每刻加一，48 刻一年。稅收、普查、劇本排程看它。
- `Scycle` 也是每刻加一（相位 0），但**到 1023 就繞回 0**
  （原始碼註解寫 `this is cosmic`）。掃描週期與 `SetValves` 看它。

兩者不同步：`Scycle` 繞回時 `CityTime` 不會。

## 四、`DoSimInit` 的順序不能改

```
InitSimLoad == 2 → InitSimMemory()   （新城市）
InitSimLoad == 1 → SimLoadInit()     （載入城市）
SetValves → ClearCensus → MapScan(全圖) → DoPowerScan → NewPower = 1
→ PTLScan → CrimeScan → PopDenScan → FireAnalysis
→ TotalPop = 1 → DoInitialEval = 1
```

⚠ **`MapScan` 排在 `DoPowerScan` 前面**，所以第一次掃描用的是舊的電力圖。
載入城市時 `SimLoadInit` 先把 `PowerMap` 整張設成全 1 再 `DoNilPower()`，
就是為了讓那一次掃描把所有分區當成有電——否則載入後第一輪全部斷電。

⚠ **最後兩行是給評分用的假值**：`TotalPop = 1` 讓下一個相位 0 的
`CityEvaluation` 走完整路徑（而不是 `EvalInit`），
而那一次評分會消耗約 650 個亂數。這對整刻對拍是關鍵細節。

## 五、相位 10 的訊息系統

`SendMessages()` 實作在 `internal/sim/message.go`（機制在
[`14-messages.md`](14-messages.md)），包含 `CheckGrowth`（人口里程碑）
與劇本勝敗判定（`DoScenarioScore`）。
