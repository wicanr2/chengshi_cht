# 地形編輯器：參數對話框 — 規格

**狀態：READY**（版面與預設值來自反組譯，等級：已確認；未解項逐條標明）

證據來源：`workplace/e220/TERRAIN.EXE` 解 LZEXE 0.91 之後的 325 728 位元組版本，
IDA 資料庫 `workplace/ida/TERRAIN.EXE.i64`（輸入檔 SHA-256 前 16 碼 `9a07bfbd9b3eb136`）。
拆解過程見 [`../re/20-terrain-editor.md`](../re/20-terrain-editor.md)。

## 一、這個編輯器在做什麼

**是一個完整的繪圖程式。** 六個工具（`DIRT`／`TREES`／`RIVER`／`CHANNEL`／
`FILL`／`UNDO`）直接在地圖上畫，三個選單（`SYSTEM`／`TERRAIN`／`PARAMETERS`）
管城市檔、地形動作與遊戲參數。介面全貌與各選單的項目列在
[`../re/20-terrain-editor.md`](../re/20-terrain-editor.md) §四之二。

**這份規格只涵蓋其中一個對話框**：`TERRAIN` → `Create Random Terrain`
叫出來的「Terrain Creation Parameters」。玩家調三個百分比、按 Go，
程式產生地形、平滑化，然後回到編輯器。

`internal/sim` 已經有規則：`GenerateMap` ＋ `TerrainParams`
（`TreeLevel`／`LakeLevel`／`CurveLevel`，`s_gen.c:76-79`）與 `SmoothTerrain`
（`smoothRiver`／`smoothTrees`）。

## 二、對話框的版面（已確認）

視窗由 `sub_1C010(&win, 0x24, 0x0A)` 建立 —— **36 個字元寬、10 列高**。
之後所有座標都相對於視窗的左上角，單位是字元格：

```
011421  mov ax, 0Ah   ; push  → 10 列
011422  mov ax, 24h   ; push  → 36 欄
011426  lea ax, [bp+var_1C]  ; push ss:&win
01142B  call sub_1C010
```

結構前兩個字組是 `{列, 欄}`：`var_1C` ＝ 頂列、`var_1A` ＝ 左欄。
畫字用的像素座標由它們換算，**一個字元格寬 8 像素**：

```
X 像素 = 左欄 × 8 + 16          (011 4EA：shl 3 後 add 10h)
Y 像素 = (頂列 + 2) × 行高       (011 4FD：var_1C+2 乘 word_596F0)
```

`word_596F0` 是行高（位元組，執行期依顯示模式決定）。

### 標題與兩行標籤

| 元素 | 字串位移 | 位置 |
|---|---|---|
| 視窗標題 `Terrain Creation Parameters` | `ds:0x140` | 由 `sub_1C376` 設在標題列 |
| 第一行標籤 ` Number     Number     River  ` | `ds:0x15C` | X ＝ 上式＋8，Y ＝ 上式＋1 個行高 |
| 第二行標籤 `of Trees   of Lakes  Curviness` | `ds:0x17B` | 同 X，Y 再加 1 個行高 |

畫字的函式是 `sub_19D12(int x像素, int y像素, char far *s)`。

### 六個增減鍵與兩個按鈕（已確認）

建按鈕的函式是 **`sub_17C90(int 欄, int 列, char far *標籤, int 編號)`**
（cdecl 由右往左推，`add sp, 0Ah` ＝ 5 個字組）。

標籤是兩個單字元字串：`ds:0x19A` ＝ `0x11`（CP437 的 `◄`）、
`ds:0x19C` ＝ `0x10`（`►`）。**所以那六個按鈕是三個參數各一組減／增。**

| 參數 | `◄` 欄 | `►` 欄 | 列 | 編號（◄／►）|
|---|---:|---:|---:|---|
| Number of Trees | 左欄＋3 | 左欄＋10 | 頂列＋5 | 0x800 ／ 0x801 |
| Number of Lakes | 左欄＋14 | 左欄＋21 | 頂列＋5 | 0x802 ／ 0x803 |
| River Curviness | 左欄＋25 | 左欄＋32 | 頂列＋5 | 0x804 ／ 0x805 |

| 按鈕 | 標籤 | 欄 | 列 | 編號 |
|---|---|---:|---:|---|
| Go | `   Go   `（`ds:0x19E`）| 左欄＋3 | 頂列＋8 | 0x806 |
| Cancel | ` Cancel `（`ds:0x1A7`）| 左欄＋25 | 頂列＋8 | 0x807 |

兩個標籤都是**八個字元寬、前後補空白**，不是「置中的兩個字」。

### 三個參數的預設值（已確認）

`dseg:0x0B6` 起三個 16 位元值，全部是 **50**：

```
059196  32 00 32 00 32 00      ← 50, 50, 50
```

顯示格式是 `%3d%%%%`（`dseg:0x1B0` 起連續六份，每份 8 位元組）——
**六份是格式字串不是六個標籤**：三個參數各兩份，初次畫用 `0x1B0`／`0x1B8`／`0x1C0`，
按了 `◄`／`►` 之後只重畫那一行、用 `0x1C8`／`0x1D0`／`0x1D8`。

## 三、Go 之後（已確認）

按 `Go` 產生地形、平滑化，然後**回到編輯器**——玩家可以繼續用六個工具改，
不是直接進遊戲。過程中會顯示 `Now terraforming` 與 `Smoothing...`。

`Enter Game Year:`（`dseg:0x137`，格式 `%4d`）與 `Easy`／`Medium`／`Hard`
（far pointer 表在 `dseg:0x116`）**不是** Go 之後的步驟：前者是
`PARAMETERS` → `Game Year`，後者在 `PARAMETERS` → `Name & Level` 的對話框裡
（那一個與遊戲本體的「市名 ＋ 技術等級 ＋ OK」是同一個版面）。
字串在資料段裡相鄰，不代表在流程上相鄰。

## 四、操作（已確認）

`sub_11402` 的訊息迴圈在 `0x11720`。

**滑鼠**：八個控制項的編號 `0x800`–`0x807`。`◄`／`►` 一次加減 **1**，
夾限在 **0–100**（`sub_113E4(0, 值, 100)`）。長按會自動重複：記下按下的時刻，
只要按鍵沒放開、且經過超過 5 個計時單位就再執行一次同一個動作，沒有加速段。

**鍵盤**：`+` 讓焦點往後輪一格（`(focus + 1) % 8`）、`-` 往前（0 折回 7）；
`Enter`／`G`／`g` 等同 Go，`Esc`／`C`／`c` 等同 Cancel。

## 五、百分比就是 `TerrainParams` 的值（已確認）

三個百分比直接是 `TreeLevel`／`LakeLevel`／`CurveLevel`，不需要換算：
產生流程 `sub_10A0A`＋`0x010C58` 與 `s_gen.c:127 GenerateMap()` 逐行對得上，
判空方式（`!= 0` 才做）與消費端的式子都一樣——除了一條。

| 用途 | TERRAIN.EXE | `s_gen.c` |
|---|---|---|
| 湖泊 | `Lim1 = pct / 2` | 相同（`s_gen.c:234`）|
| 河流彎曲 | `r1 = pct + 10`、`r2 = pct + 100` | 相同（`s_gen.c:422-423`）|
| 樹叢擴散 | `dis = Rand(2 × pct + 100) + 50` | 相同（`s_gen.c:279`）|
| **樹叢數量** | **`Amount = 3 × pct`** | `Amount = TreeLevel + 3`（`s_gen.c:301`）|

樹叢數量差三倍（50% ＝ 150 叢 vs 53 叢），實作要走 DOS 那一式：
`TerrainParams.TreeAmountDOS`。詳見 [`../re/20-terrain-editor.md`](../re/20-terrain-editor.md) §七。

原版在 `DoTrees` 之後**又多跑兩次** `SmoothTrees`（共四次）。

## 六、量自原版的版面（實機量測，已確認）

反組譯推出來的欄列與實機量到的一致；下列是原本未解、由量測補上的：

| 項目 | 值 |
|---|---|
| 視窗原點 | 左欄×8 ＝ **172**、頂列×14 ＝ **95**（水平置中，垂直略高於中央）|
| 客戶區 | 原點右 8、下 7，**280×132**，白底 |
| 邊框 | 藍 `(0,0,255)`，三像素 |
| 按鈕底色 | `(0,170,255)` —— **不在 EGA 預設十六色裡**，編輯器自己重載了調色盤 |
| 標題 | 第 1 列 |
| `◄`／`►` | **14×20 的按鈕**，三角形逐列寬度 1-2-3-5-7-5-3-2-1，平邊離框 4 像素 |
| `Go`／`Cancel` | **70×20** |

## 七、未解，不要自己補

- **`CreateIsland` 不進這個對話框。** 原版把它放在 `TERRAIN` 選單裡當成一個
  **動作**（`Create Island CTRL-I`），不是一個滑桿。
- 自動重複的「5 個計時單位」換算成毫秒是多少（`sub_18CA9` 的計時來源未讀）。
