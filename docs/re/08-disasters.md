# 08 — 災難

**推論等級：已確認（讀出來的部分）／未解（要靠精靈的三種）**。
日期 2026-08-29。接線：`internal/sim/disaster.go`。

## 一、隨機災難的機率

`DoDisasters`（`s_disast.c:74`）每一刻擲一次 `Rand(DisChance[GameLevel])`：

| 難度 | 期望間隔 | 換算 |
|---|---:|---|
| Easy | 480 刻 | 約十年一次 |
| Medium | 240 刻 | 約五年一次 |
| Hard | 60 刻 | 約一年三次 |

命中之後 `Rand(8)` 挑種類：

| 值 | 災難 | 機率 |
|---|---|---:|
| 0, 1 | 火災 | 2/9 |
| 2, 3 | 水災 | 2/9 |
| 4 | 空難 | 1/9 |
| 5 | 龍捲風 | 1/9 |
| 6 | 地震 | 1/9 |
| 7, 8 | 怪獸（**汙染平均要 > 60**）| 2/9 |

⚠ `Rand(8)` 的值域是 **0…8（九個值）**，因為 `Rand(n)` 含上界。
所以是九分之一而不是八分之一，而且 case 7 與 case 8 共用怪獸。

⚠ `NoDisasters` 的 `return` 排在 `Rand()` **之前**（`s_disast.c:88`），
所以關掉災難時**一個亂數都不會消耗**。這對整刻對拍很重要
（[`12-tick-parity.md`](12-tick-parity.md) 的微實驗靠它）。

## 二、劇本災難是另一條路

`ScenarioDisaster`（`s_disast.c:117`）與隨機災難互不相干，
由 `DisasterEvent`／`DisasterWait` 排程（`s_sim.c:333` 的 `DisTab`）：

| 劇本 | 等待 | 事件 |
|---|---:|---|
| 舊金山 1906 | 2 | 倒數到 1 時地震 |
| 漢堡 1944 | 10 | **每一刻**都投彈 |
| 東京 1957 | 20 | 倒數到 1 時出怪獸 |
| 波士頓 2010 | 5 | 倒數到 1 時核熔毀 |
| 里約 2047 | 96 | 每 24 刻淹一次 |
| 達斯維利／伯恩／底特律 | — | 沒有災難（問題是慢性的）|

## 三、放火有兩套條件，不是同一件事

| 函式 | 觸發 | 條件 | 重試 |
|---|---|---|---|
| `SetFire`（`s_disast.c:205`）| 隨機災難 | 不是分區中心，且圖塊在 `LHTHR`…`LASTZONE` 之間 | **只試一次** |
| `MakeFire`（`s_disast.c:225`）| 玩家主動 | 不是分區中心、**要有 `BURNBIT`**，且圖塊在 21…`LASTZONE` | 最多四十次 |

下界不同（`LHTHR` 249 vs 21）：**玩家放火燒得到樹，隨機火災燒不到。**

## 四、地震只毀「建物本體」

`Vunerable`（`s_disast.c:246`）：圖塊要在 `RESBASE`…`LASTZONE` 之間
**而且不是分區中心**。所以地震不會直接消滅一個分區，只會把它周圍的建物打碎；
道路與自然地形也不受影響。

震動次數是 `Rand(700) + 300`，每次擲一點；四分之三變廢墟、四分之一變火。

## 五、靠精靈的三種災難

空難、龍捲風、怪獸本身就是精靈，實作在 `internal/sim/sprite_effects.go`
（`MakeAirCrash`／`MakeTornado`／`MakeMonster`），由 `disaster.go` 的
`makeAirCrash`／`makeTornado`／`makeMonster` 轉呼叫。沒有精靈系統時
（`World.Sprites` 是 nil）這三個走 `noSprites` 空實作，什麼都不做——
**這會改變抽樣次數**，所以做對拍實驗一定要 `EnableSprites()`。

熔毀（`DoMeltdown`）與水災（`MakeFlood`／`DoFlood`）不需要精靈。
