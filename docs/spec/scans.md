# 規格 — 四個逐格掃描　**READY**

證據：[`docs/re/06-scans.md`](../re/06-scans.md)。
一手出處：`s_scan.c`、`s_zone.c:428-460`、`:605`。

## 介面

```go
func (w *World) PopDenScan()
func (w *World) PTLScan()
func (w *World) CrimeScan()
func (w *World) FireAnalysis()
```

四者互相回饋，地圖不變時收斂到固定點。呼叫順序照 `Simulate` 的相位：
12 → `PTLScan`、13 → `CrimeScan`、14 → `PopDenScan`、15 → `FireAnalysis`。

## 不變量

1. `DoSmooth` 是 `(四鄰居 + 自己) >> 2`，**除以 4 不是 5**，邊界不補。
2. `SmoothFSMap`／`SmoothPSMap` 是 `((四鄰居 >> 2) + 自己) >> 1`，與上面不同核。
3. `SmoothTerrain` 的 8 位元截斷在右移**之前**。
4. 沒有人造物（`>= ROADBASE`）的格子地價是 0，且不計入平均。
5. 無分區時 `CCx/CCy` 指派半解析度常數（單位不一致，照抄）。
6. `PolMaxX/Y` 與 `CrimeMaxX/Y` **不是決定性的**（平手擲骰），不得拿來對拍。

## 驗收條件

無分區的受控地圖收斂後：`LVAverage` = 25、`PolluteAverage` = 144、
`CrimeAverage` = 102（`testdata/scan-experiment.csv`）。

## 刻意不照做

| 原版 | 本專案 | 為什麼 |
|---|---|---|
| `DonDither` 的蛇行 ＋ 誤差擴散分支 | 只實作非 dither 分支 | 預設關閉；打開需要另一組驗收資料 |
| `NewMapFlags[...] = 1`（通知 UI 重畫）| 不做 | 呈現層的事 |

## 未解

見機制筆記 §8。
