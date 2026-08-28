# 13 — 精靈系統：八種會動的東西

**推論等級：已確認**（一手出處 `w_sprite.c`、`s_sim.c`、`s_disast.c`）。
日期 2026-08-29。接線：`internal/sim/sprite.go`、`sprite_move.go`、
`sprite_effects.go`。

## 一、精靈不是模擬的一部分

`MoveObjects()` 在 `sim_loop`（`w_sim.c`）裡，和 `SimFrame()` **平行**呼叫：

```c
if (doSim) SimFrame();
MoveObjects();
```

`SimFrame` 會依 `SimSpeed` 決定要不要跑（速度 1 只有五分之一的 frame 會跑），
**`MoveObjects` 每個 frame 都跑**。所以精靈的動作頻率與模擬速度無關——
慢速時城市長得慢，火車照樣同一個速度在跑。

Go 這邊對應 `World.Frame()`：先 `SimFrame()` 再 `spriteSys.MoveObjects()`。

## 二、八種精靈

| 型別 | 產生於 | 消失條件 |
|---|---|---|
| 火車 `TRA` | `DoRail`，1/256 機率 | 走到非鐵軌 |
| 直昇機 `COP` | `DoAirport`，`Rand(12)==0` | 燃料耗盡（`count` 歸零）|
| 飛機 `AIR` | `DoAirport`，`Rand(5)==0` | 飛出地圖 |
| 船 `SHI` | `DoPort` / 隨機 | 撞岸或走到非水面 |
| 怪獸 `GOD` | `MakeMonster` | 停留夠久後自行離開 |
| 龍捲風 `TOR` | `MakeTornado` | `count` 歸零 |
| 爆炸 `EXP` | `MakeExplosion`、碰撞 | 動畫播完 |
| 巴士 `BUS` | `GenerateBus`（**被註解掉**）| — |

巴士在原版**永遠不會出現**：`DoRoad` 裡呼叫 `GenerateBus` 的那行是註解狀態。
程式碼、圖形、移動邏輯全都在，只是沒有人叫它。照抄——不要「順手修好」。

## 三、六個會產生「自洽但錯」結論的地方

### 3.1 `GetDir` 的第二個修正分支永遠不會成立

```c
if ((absDist = (dispX < 0 ? -dispX : dispX) + (dispY < 0 ? -dispY : dispY)) > 0) {
  ...
  else if ((dispY << 1) < dispY) z--;
```

`(dispY << 1) < dispY` 只有 `dispY < 0` 時成立，但這一段已經在
`dispY >= 0` 的分支裡。**那個 `z--` 永遠不會執行。**
照抄可以，但不要根據它去解釋任何行為。

### 3.2 `absDist` 是跨 frame 的殘留值

`GetDir` 把距離寫進全域 `absDist`，呼叫端在別的地方讀它。
**如果這個 frame 沒有呼叫 `GetDir`，讀到的是上一次的值。**
移植成區域變數會改變行為。

### 3.3 `MakeMonster` 的判斷式反了

```c
if (!done == 0) MonsterHere(60, 50);
```

`!done` 是 0 或 1，`== 0` 等於 `done != 0`。所以**找到河的時候才強制把怪獸
放到 (60,50)**，三百次都沒找到河反而不會有怪獸——和意圖相反。
結果是怪獸幾乎總是出現在 (60,50)。

### 3.4 `MakeAirCrash` 在官方建置裡被編掉

`micropolis-activity/src/sim/makefile` 帶 `-DNO_AIRCRASH`。
程式碼在，但官方二進位檔裡沒有。Go 這邊實作了但預設不啟用。

### 3.5 `Destroy` 的下界是 `TREEBASE`，不是 0

```c
if (z < TREEBASE) return;   /* 水面、空地不會被摧毀 */
```

所以爆炸不會在水面或空地上留下瓦礫。看畫面推不出這件事——
在空地上爆炸看起來「什麼都沒發生」，很容易被解釋成別的原因。

### 3.6 `oFireZone` 四個方向都會擲骰

```c
if (!(Rand16() & 3)) { ... }   /* 上 */
if (!(Rand16() & 3)) { ... }   /* 右 */
if (!(Rand16() & 3)) { ... }   /* 下 */
if (!(Rand16() & 3)) { ... }   /* 左 */
```

**四次都會擲**（除非中途 return）。少擲一次會讓亂數數列整條錯開——
這正是 [`12-tick-parity.md`](12-tick-parity.md) 那一類錯誤。

## 四、碰撞

`CheckSpriteCollision` 只檢查會動的對象，而且**距離判定用的是曼哈頓距離**
（`GetDis`），不是歐氏距離。飛機撞飛機、飛機撞直昇機都會爆炸。

直昇機的目標選擇有一條寫在原始碼註解裡的設計意圖：
「被怪獸與龍捲風吸引，so it blows up more often」——那是**故意的**，
不是 bug。

## 五、對拍狀態

精靈在 [`12-tick-parity.md`](12-tick-parity.md) 的對拍實驗裡沒有被觸發
（那座城市沒有機場、港口、鐵路，也沒有災難），所以目前**沒有逐次元證據**。
要驗證得另外設計一個有機場或鐵路的實驗。
