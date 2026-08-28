# 03 — 地圖陣列與圖塊編碼

**推論等級：已確認**（讀 `headers/sim.h` 與 `s_alloc.c:150 initMapArrays`，
並用 oracle 的 `sim Tile` 實測解碼）。日期 2026-08-29。
接線：`internal/sim/world.go`、`internal/sim/tiles.go`。

## 一、世界尺寸與四種解析度

`headers/sim.h:150-167`：

```c
#define SimWidth  120
#define SimHeight 100
#define WORLD_X  SimWidth        /* 120 */
#define WORLD_Y  SimHeight       /* 100 */
#define HWLDX    (SimWidth >>1)  /*  60 */   /* half  */
#define HWLDY    (SimHeight>>1)  /*  50 */
#define QWX      (SimWidth >>2)  /*  30 */   /* quarter */
#define QWY      (SimHeight>>2)  /*  25 */
#define SmX      (SimWidth >>3)      /* 15 */   /* small */
#define SmY      ((SimHeight+7)>>3)  /* 13 */
```

⚠ **`SmY` 是 `(100+7)>>3 = 13`，不是 `100>>3 = 12`。** 只有這一個維度有進位；
其餘三對都是純右移。照抄成 `>>3` 會少一列，而且是最外圈那一列——
早期不會出錯，等到城市長到地圖底部才會。

模擬用的四種解析度（`s_alloc.c:150` 起）：

| 陣列 | 型別 | 維度 | 每格代表 | 內容 |
|---|---|---|---|---|
| `Map[x][y]` | `short` | 120 × 100 | 1 格 | **圖塊 ＋ 旗標**，見第二節 |
| `PopDensity` `TrfDensity` `PollutionMem` `LandValueMem` `CrimeMem` | `Byte` | 60 × 50 | 2 × 2 格 | 掃描結果 |
| `tem` `tem2` | `Byte` | 60 × 50 | 2 × 2 格 | 平滑用的暫存 |
| `TerrainMem` `Qtem` | `Byte` | 30 × 25 | 4 × 4 格 | 地形（地價用）|
| `RateOGMem` `FireStMap` `PoliceMap` `PoliceMapEffect` `FireRate` `ComRate` `STem` | `short` | 15 × 13 | 8 × 8 格 | 成長率、警消涵蓋 |

歷史統計：`ResHis` `ComHis` `IndHis` `MoneyHis` `PollutionHis` `CrimeHis`
各 `HISTLEN = 480` **bytes**（＝240 個 `short`），`MiscHis` `MISCHISTLEN = 240` bytes。

⚠ **`NewPtr(HISTLEN)` 配的是 480 個 byte，指標型別是 `short *`，
所以實際容量是 240 個項目**，不是 480 個。長度單位在原始碼裡是 byte，
不是元素數——這是最容易把陣列長度寫成兩倍的地方。

電力圖 `PowerMap`：位元圖，`POWERMAPROW = (120+15)/16 = 8`，
`POWERWORD(x,y) = (x>>4) + (y<<3)`。`(y<<3)` 就是 `y * POWERMAPROW`，兩者相等。

## 二、每一格是一個 16 位元字

`headers/sim.h:245-256`：

| 位元 | 常數 | 值 | 意義 |
|---|---|---|---|
| 15 | `PWRBIT` | 32768 | 這一格有電 |
| 14 | `CONDBIT` | 16384 | 這一格導電 |
| 13 | `BURNBIT` | 8192 | 可燃 |
| 12 | `BULLBIT` | 4096 | 可推平 |
| 11 | `ANIMBIT` | 2048 | 這一格會動畫 |
| 10 | `ZONEBIT` | 1024 | **這一格是分區的中心** |
| 9–0 | `LOMASK` | 1023 | 圖塊編號 |

`ALLBITS = 64512` 是上面六個旗標的遮罩。組合常數：
`BLBNBIT = BULLBIT+BURNBIT`、`BNCNBIT = BURNBIT+CONDBIT`、
`BLBNCNBIT = BULLBIT+BURNBIT+CONDBIT`。

**實測解碼**（2026-08-29，oracle 啟動畫面的地圖）：
`sim Tile 63 63` 回 `12322`。`12322 & 1023 = 34`（`TREEBASE 21 … LASTTREE 36` 之間，
是樹），`12322 >> 10 = 12` → `BULLBIT + BURNBIT`。樹可推平、可燃、不導電、
不是分區中心——與語意一致。

⚠ **圖塊編號只有 10 位元（0–1023），而 `TILE_COUNT = 960`。**
超過 1023 的圖塊編號放不進去；擴充圖塊時沒有空間可用。

## 三、圖塊編號區段

`headers/sim.h:285-423`。完整常數表照抄進 `internal/sim/tiles.go`；這裡只記分段：

| 範圍 | 內容 | 界標常數 |
|---|---|---|
| 0 | 空地 | `DIRT` |
| 2–4 | 水 | `RIVER` `REDGE` `CHANNEL` |
| 5–20 | 河岸 | `FIRSTRIVEDGE` … `LASTRIVEDGE` |
| 21–43 | 樹林 | `TREEBASE` `LASTTREE 36` `WOODS 37` `WOODS2…5` |
| 44–47 | 廢墟 | `RUBBLE` … `LASTRUBBLE` |
| 48–51 | 水災 | `FLOOD` … `LASTFLOOD` |
| 52 | 輻射 | `RADTILE` |
| 56–63 | 火 | `FIREBASE` … `LASTFIRE` |
| 64–206 | 道路與橋（含車流動畫的兩組 `LTRFBASE 80`、`HTRFBASE 144`）| `ROADBASE` … `LASTROAD` |
| 208–222 | 電線 | `POWERBASE` … `LASTPOWER` |
| 224–238 | 鐵路 | `RAILBASE` … `LASTRAIL` |
| 240–422 | 住宅區（`FREEZ 244`、`HOUSE 249`、`HOSPITAL 409`、`CHURCH 418`）| `RESBASE` |
| 423–611 | 商業區 | `COMBASE` `COMCLR 427` `CZB 436` |
| 612–692 | 工業區 | `INDBASE` `INDCLR 616` `IZB 625` |
| 693–708 | 海港 | `PORTBASE` `PORT 698` |
| 709–744 | 機場 | `AIRPORTBASE` `RADAR 711` `AIRPORT 716` |
| 745–760 | 燃煤電廠 | `COALBASE` `POWERPLANT 750` |
| 761–769 | 消防局 | `FIRESTBASE` `FIRESTATION 765` |
| 770–778 | 警察局 | `POLICESTBASE` `POLICESTATION 774` |
| 779–810 | 體育場 | `STADIUMBASE` `STADIUM 784` `FULLSTADIUM 800` |
| 811–826 | 核電廠 | `NUCLEARBASE` `NUCLEAR 816` `LASTZONE 826` |
| 827–959 | 特效與動畫 | 閃電、橋、雷達、噴泉、煙、爆炸、足球賽 |

`UNUSED_TRASH1…6`（38、39、53、54、55、223）是原始碼自己標的未用編號；
`ROADVPOWERH 239` 旁邊有一行原作註解 `/* bogus? */`。
**這兩類照抄，不要「整理掉」**——它們是原版的一部分，動了會讓對拍失準。

## 四、未解

| 項目 | 怎麼解 |
|---|---|
| ~~DOS 版是不是同一套圖塊編號與同一個 120×100~~ | **已解**：是。`.PSN` 解壓後的 27120 位元組能被 `ParseCityFile` 直接吃下去，八個劇本的每一格編號都 < `TILE_COUNT`。見 [`../formats/02-dos-lzss.md`](../formats/02-dos-lzss.md) §4 |
| `TILE_COUNT 960` 與圖形檔的圖塊數是否相同 | 解 `.PGF` 之後比對 |
