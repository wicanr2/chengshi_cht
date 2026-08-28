# 規格 — 地形產生　**READY**

證據：[`docs/re/04-terrain-generation.md`](../re/04-terrain-generation.md)。
一手出處：`s_gen.c`。

## 介面

```go
func (w *World) GenerateMap(seed uint32, p TerrainParams)
```

`TerrainParams` 四個欄位初值皆為 −1（`s_gen.c:76-79`）。
**−1 ＝「用預設的隨機量」，0 ＝「完全不做」**，兩者不可混。

## 流程

見機制筆記 §2。實作必須逐句照抄控制流，包括：

1. 造島分支命中後**提前 return**。
2. `DoRivers` 第三段只重設 `LastDir`，`Dir` 沿用。
3. `DoBRiv`／`DoSRiv` 的兩個轉向 `if` 是並列，不是 `else if`。
4. `SmoothRiver`／`SmoothTrees` 用區域游標，不動全域。
5. `PutOnMap` 的非零判斷。
6. `WOODS_HIGH = 39`（`UNUSED_TRASH2`）。

## 驗收條件

1. 種子 5、7、12345、4242 四張地圖，**每一格的完整 16 位元字**都與
   `internal/sim/testdata/terrain-seed*.csv` 相同。
2. 種子 5 必須走造島分支（水面格數遠多於一般地圖）。
3. 同一顆種子跑兩次結果相同。
4. 每一格的圖塊編號 < `TILE_COUNT`。

## 刻意不照做

| 原版 | 本專案 | 為什麼 |
|---|---|---|
| `GenerateMap` 結尾 `RandomlySeedRand()` | 不做 | 不決定性的東西不進規則層；呼叫端自己給種子 |
| `GenerateSomeCity` 順帶做的 `InitWillStuff`／`ResetMapState`／`DoSimInit`／UI 通知 | 不做 | 那些是遊戲流程與呈現層，不是地形產生 |

## 未解

見機制筆記 §6（非預設旋鈕值、`SmoothWater`、DOS 版）。
