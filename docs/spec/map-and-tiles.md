# 規格 — 地圖陣列與圖塊編碼　**READY**

證據：[`docs/re/03-map-and-tiles.md`](../re/03-map-and-tiles.md)。
一手出處：`headers/sim.h:150-167`、`:245-256`、`:285-423`、`s_alloc.c:150 initMapArrays`。

## 尺寸

```
WORLD_X = 120   WORLD_Y = 100
HWLDX   =  60   HWLDY   =  50     // WORLD >> 1
QWX     =  30   QWY     =  25     // WORLD >> 2
SmX     =  15   SmY     =  13     // SmY = (100+7)>>3，有進位，不是 12
```

## 每格的編碼

一個 `uint16`：低 10 位元是圖塊編號（`LOMASK = 1023`），高 6 位元是旗標
（`ZONEBIT` 1024、`ANIMBIT` 2048、`BULLBIT` 4096、`BURNBIT` 8192、
`CONDBIT` 16384、`PWRBIT` 32768）。

## 陣列

| 名稱 | 元素 | 維度 |
|---|---|---|
| `Map` | `uint16` | 120 × 100 |
| `PopDensity` `TrfDensity` `PollutionMem` `LandValueMem` `CrimeMem` `Tem` `Tem2` | `uint8` | 60 × 50 |
| `TerrainMem` `Qtem` | `uint8` | 30 × 25 |
| `RateOGMem` `FireStMap` `PoliceMap` `PoliceMapEffect` `FireRate` `ComRate` `STem` | `int16` | 15 × 13 |
| `ResHis` `ComHis` `IndHis` `MoneyHis` `PollutionHis` `CrimeHis` | `int16` | 240（`HISTLEN` 480 bytes ÷ 2）|
| `MiscHis` | `int16` | 120（`MISCHISTLEN` 240 bytes ÷ 2）|
| `PowerMap` | 位元圖 | `POWERMAPROW = 8` 個 word × 100 列 |

⚠ `HISTLEN` 是 **byte 數**，不是元素數。配成 480 個 `int16` 是兩倍。

## 索引順序

原版 `Map[i] = mapPtr + i * WORLD_Y`，也就是 **x 為外層、y 為內層**，
`Map[x][y]` 在記憶體裡連續的是 y。Go 版照這個順序，存檔的位元組順序才對得上。

## 不變量

1. `Tile(x,y) & LOMASK` 永遠 < `TILE_COUNT`（960）。
2. 寫入圖塊時只能動低 10 位元，旗標要另外設；反之亦然。
3. `SmY` 是 13。任何 `SimHeight>>3` 的寫法都是錯的。

## 刻意不照做

| 原版 | 本專案 | 為什麼 |
|---|---|---|
| 用 `NewPtr` 配一整塊再切成列指標 | Go 的二維陣列 | 沒有語意差別；指標算術是實作細節 |
| `short`（有號）存格子 | `uint16` | 旗標用到第 15 位元，有號型別在移位時會出事 |
| `UNUSED_TRASH1…6`、`ROADVPOWERH /* bogus? */` | **照抄保留** | 它們是原版的一部分 |

## 未解

DOS 版是否同一套編號與同一個尺寸（見機制筆記 §4）。
