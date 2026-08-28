# 規格 — 電力傳導　**READY**

證據：[`docs/re/05-power-scan.md`](../re/05-power-scan.md)。
一手出處：`s_power.c`、`s_zone.c:624`、`s_sim.c:1014`、`:713`。

## 介面

```go
func (w *World) DoPowerScan() PowerScanResult
```

一次呼叫做完原版分散在三處的事：找電廠並壓堆疊 → 泛洪填 `PowerMap` →
把 `PowerMap` 攤回每一格的 `PWRBIT`。

## 演算法

```
# 第 1 步：照 MapScan 的順序（x 外層、y 內層）掃全圖
for x, y:
    cChr = Map[x][y]; if cChr == 0: continue
    cChr9 = cChr & LOMASK
    記住 cChr9                     # 掃完留下最後一格非零圖塊
    if cChr9 < FLOOD: continue
    if cChr 沒有 ZONEBIT: continue
    if cChr9 == POWERPLANT: CoalPop++;    push(x, y)
    if cChr9 == NUCLEAR:    NuclearPop++; push(x, y)

# 第 2 步：泛洪
PowerMap 全部清 0
MaxPower = CoalPop*700 + NuclearPop*2000
NumPower = 0
while 堆疊非空:
    pull()
    ADir = 4                        # 4 ＝ 原地不動
    do:
        NumPower++
        if NumPower > MaxPower: 標記 OutOfPower; **整個 return**
        MoveMapSim(ADir)
        PowerMap 設起 (SMapX, SMapY) 那一位
        ConNum = 0; Dir = 0
        while Dir < 4 and ConNum < 2:
            if TestForCond(Dir): ConNum++; ADir = Dir
            Dir++
        if ConNum > 1: push()
    while ConNum != 0

# 第 3 步：攤回 PWRBIT
for 每個帶 CONDBIT 的格子:
    如果是 NUCLEAR/POWERPLANT，或 PowerMap 那一位是 1 → 設 PWRBIT
    否則 → 清 PWRBIT
```

`TestForCond(dir)`：往 dir 移一格（撞邊界回 false），檢查
`Map & CONDBIT` 且 `cChr9 != NUCLEAR` 且 `cChr9 != POWERPLANT` 且
（`word > PwrMapSize` 或 `PowerMap` 那一位是 0）；查完游標復原。

## 不變量

1. `MaxPower` 超過時是 **`return`**，不是 `break`——堆疊裡剩下的分支全部丟掉。
2. 方向 4 原地不動且回 true。少了它電廠本身不會通電。
3. `cChr9` 是 `MapScan` 掃完留下的全域，不是鄰居的圖塊。
4. 電廠自己永遠算有電，不看 `PowerMap`。

## 驗收條件

1. 受控實驗地圖（`testdata/power-experiment.csv`）清掉 `PWRBIT` 後重算，
   12000 格逐格相同。
2. 孤立電線不通電。
3. 一座燃煤電廠 + 超過 700 格電線 → `OutOfPower` 為真，且通電格數不超過容量太多。
4. 沒有電廠 → 通電 0 格。
5. 劇本 1 載入後重算，與 oracle 的差異只剩帶 `ANIMBIT` 的格子。

## 刻意不照做

| 原版 | 本專案 | 為什麼 |
|---|---|---|
| `char PowerStackX/Y` | `int` | 有號 char 在 120 寬的地圖剛好夠用，但那是尺寸相依的巧合 |
| 三個步驟散在 `MapScan`／`DoPowerScan`／`SetZPower` | 收攏成一次呼叫 | 地圖不變時原版會收斂到同一狀態；驗收用收斂後的地圖比對 |
| `SendMes(40)` 電力不足訊息 | 只回傳 `OutOfPower` 旗標 | 訊息系統還沒實作（`s_msg.c`）|

## 未解

見機制筆記 §6。
