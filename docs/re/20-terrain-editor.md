# 地形編輯器（SimCity Terrain Editor 1.0）— 拆解筆記

目標是把原版的地形編輯器重製進 remake（使用者定案 2026-09-03：**照原版重製**，
不是自己設計一個編輯模式）。這份記錄目前解到哪、卡在哪、下一個入口是什麼。

素材是軟體世界 1990 年重新打包的那兩片磁片，逐檔盤點見
[`../formats/00-e220-terrain-editor.md`](../formats/00-e220-terrain-editor.md)。

## 一、招牌畫面：`*TE.PPF`（已確認）

編輯器自己的開場畫面，五種顯示模式都解得開，用的是遊戲那套 `.PPF` 版面
（`simtool ppf`）：

| 檔案 | 模式 | 解出 |
|---|---|---|
| `CEGATE.PPF` | cega | 640×350 |
| `SEGATE.PPF` | sega | 320×200 |
| `TDYTE.PPF` | tdy | 320×200 |
| `MONOTE.PPF` | mono | 640×347 |
| `CGATE.PPF` | cga | 320×200 |

畫面是一台推土機加上 `SIM CITY / Terrain Editor` 字樣。**這是招牌不是工作介面。**

`MCGATE.PPF` 還沒解：它的調色盤要從 `MCGATED.PGF` 來，而那是**基本檔版面**
（沒有五位元組檔頭），現行的 `-pal` 只吃得下一般 `.PGF`。**未解，工具問題。**

## 二、`*TED.PGF` 裡沒有介面美術（已確認）

`CEGATED.PGF` 壓縮 63 010、解出 **139 434** 位元組。扣掉第 0 庫的
960 張 16×16 四平面圖塊（960 × 128 ＝ 122 880）之後只剩 16 554，
而且 122 880 起的位元組與遊戲的 `CEGADAT.PGF` **完全相同**（那是共用的
地圖縮圖與內建字型區塊）。`LoadPGFBase` 在後面找不到行內圖形庫表。

對照組：`CEGADAT.PGF` 解出 219 552，扣掉同樣的 122 880 還有 96 672
是工具盤、需求指標、統計圖按鈕、圖層圖示、色階這些介面美術。

**所以編輯器的介面不是貼圖貼出來的，是程式自己畫的。**
要知道它長什麼樣，只能跑起來看，或者反組譯。

## 三、`TERRAIN.CFG`：五個位元組（強證據）

編輯器直接跑會說 `FATAL ERROR: PROGRAM ABORTED / Terrain Editor
configuration file not found.`，設定檔要由磁片自己的 `INSTALL.EXE` 產生。
產生出來的 `TERRAIN.CFG` 只有五個位元組：

```
45 4E 49 31 91        E  N  I  1  ·
```

前四個對得上 `SIMCITY.CFG` 自帶的解碼表，順序也一樣：

| 位元組 | 值 | 對應 |
|---|---|---|
| 0 | `E` | Screen Mode ＝ Hires EGA Color |
| 1 | `N` | Joystick ＝ No Joystick |
| 2 | `I` | Sound ＝ Internal IBM Sound |
| 3 | `1` | Covox Channel ＝ 1 |
| 4 | `0x91` | **未解** |

也就是說遊戲存的是自我說明的純文字，編輯器存的是同樣四個選擇的緊湊版。

`INSTALL.EXE` 的流程：來源磁碟（A:／B:）→ 目的地 `C:\SIMCITY\` →
顯示驅動六選一（CGA／Tandy／Hercules／EGA 三種／VGA-MCGA 兩種）→
搖桿 → 音效 → 複製。裝完只放六個檔：
`TERRAIN.EXE`、`TERRAIN.CFG`、`CEGATE.PPF`、`CEGATED.PGF`、
`MONOTED.PGF`、`SOUNDDAT.PSF`。

裝好的副本留在 `workplace/te-installed/`，之後不必重裝
（`tools/dosbox.sh` 的 `EXTRA` 會把它疊進遊戲目錄）。

## 四、怎麼在 DOSBox-X 裡跑起來（已確認）

**三件事都要對，少一件就跑不起來：**

1. **執行檔要先解 LZEXE。** 打包版（83 373 位元組，偏移 0x1C 有 `LZ91`）在
   DOSBox-X 上是 `ERROR CPU:Illegal Unhandled Interrupt Called 6` 無限迴圈。
   `unlzexe` 解出來的 325 728 位元組版本跑得動。
   ⚠ `unlzexe` 的輸出檔名緩衝區只有 12 個字元，給長路徑會被截斷成別的檔名
   而且**不報錯**。要 `cd` 到目標目錄再用短檔名。
2. **圖形資料檔要跟執行檔在同一個目錄。** 少了對應模式的 `*TED.PGF`，
   程式印的是
   ```
   FATAL ERROR: PROGRAM ABORTED
   256K of VGA/EGA memory
   Couldn't load VGA/EGA blocks!
   ```
   那則訊息**只是它自己組的一句話**（`dseg:0x2163` 的 `\n%dK of VGA/EGA memory\n`
   `Couldn't load VGA/EGA blocks!`），`%d` 是從 BIOS 資料區 `0000:0487` 的
   bit 5–6 查表得到的診斷數字，不是失敗原因。真正的失敗在 `sub_167E3`
   回 0，而它只有三條回 0 的路：`malloc(0x5000)` 失敗、
   `open(<模式>ted.pgf)` 回的 handle ≤ 0、以及解壓縮讀不滿 0x5000。
   **檔名是小寫寫死在 `dseg:0xE90` 的 `cegated.pgf`**。
3. **`TERRAIN.CFG` 的第 0 個位元組要選一個模擬得出來的顯示模式。**
   `H`（Hercules）與 `T`（Tandy）在 `machine=svga_s3` 下會停在
   `Hercules not detected.  Continue anyway [y/N]?` 等輸入。

實測七種模式（`tools/te_modes.sh`）：

| `TERRAIN.CFG[0]` | 模式 | 資料檔 | 結果 |
|---|---|---|---|
| `E` | Hires EGA Color 640×350 | `CEGATED.PGF` | **進得了介面** |
| `M` | Hires EGA Mono | `MONOTED.PGF` | 進得了介面（黑白）|
| `e` | Lores EGA Color 320×200 | `SEGATED.PGF` | 進得了介面 |
| `C` | CGA | `CGATED.PGF` | 進得了介面 |
| `2` | 256 Color VGA/MCGA | `MCGATED.PGF` | 進得了介面 |
| `H` | Hercules | — | 停在「未偵測到」的詢問 |
| `T` | Tandy Color | — | 停在「未偵測到」的詢問 |

跑法：`EXTRA=workplace/te-modes RUN=terrain ACTIONS=<絕對路徑> tools/dosbox.sh`。

## 四之二、介面全貌（已確認，實機截圖）

**編輯器是一個完整的繪圖程式，不只是一個參數對話框。**
畫面是 640×350（`E` 模式），最上面是選單列，左邊是編輯視窗與直立的工具盤，
右邊預設開著 `City Map` 全市地圖視窗，底下是目前工具的狀態列。

**工具盤（六個，由上而下）**：`DIRT`／`TREES`／`RIVER`／`CHANNEL`／`FILL`／
`UNDO`（`UNDO` 平常是灰的）。

**三個選單**（按住式，與遊戲本體一致）：

| 選單 | 項目 |
|---|---|
| `SYSTEM` | `About TERRAIN`／`Print`／`Start New City`／`Load City Ctrl-L`／`Save City as ...`／`Save City Ctrl-S`／`Exit Ctrl-X` |
| `TERRAIN` | `Clear Map Ctrl-C`／`Clear Unnatural Objects`／`Create Random Terrain CTRL-T`／`Smooth Trees`／`Smooth Rivers`／`Smooth Everything CTRL-A`／`Create Island CTRL-I` |
| `PARAMETERS` | `Name & Level`／`Game Year`／`Sound On` |

兩件事因此定案：

- **`Create Island` 有介面**，在 `TERRAIN` 選單裡，是一個**動作**不是一個滑桿。
- **參數對話框掛在 `TERRAIN` → `Create Random Terrain`**，不在 `PARAMETERS` 選單裡；
  `PARAMETERS` → `Name & Level` 開的是遊戲本體那個「市名 ＋ 技術等級 ＋ OK」對話框。

截圖：`workplace/dosbox/tep-*.png`、`ter-20-random-terrain.png`（不入版控）。

## 四之三、參數對話框的版面（實機量測，已確認）

反組譯推出來的欄列與實機量到的完全一致，另外補上原本未解的絕對位置與配色。

量法：`E` 模式的截圖裁出 640×350 的遊戲區，找白色客戶區的外接矩形與
藍色邊框，再把各元素的非白像素依列分群。

| 項目 | 量到的值 |
|---|---|
| 客戶區（白底）| x 180–459、y 102–233 → **280×132** |
| 邊框 | 藍 `(0,0,255)`，**三像素**，在客戶區外圈 |
| 視窗原點 | 反推 **左欄×8 ＝ 172、頂列×14 ＝ 95**；客戶區在原點右 8、下 7 |
| 標題 | 第 **1** 列（不是第 0 列）|
| 兩行標籤 | 第 3、4 列 |
| 三個數值與六個箭頭 | 第 5 列 |
| `Go`／`Cancel` | 第 8 列 |
| `◄`／`►` 的欄 | 3／10、14／21、25／32 —— **與反組譯推的六個欄號完全相同** |
| `◄`／`►` 的尺寸 | **14×20 的按鈕**：藍框、青底 `(0,170,255)`、中間一個藍三角形 |
| 三角形 | 九列，寬度 1-2-3-5-7-5-3-2-1，平邊固定在離框 4 像素處 |
| `Go`／`Cancel` 的尺寸 | **70×20**（八格標籤 64 ＋ 左右各三像素的框）|

⚠ **`(0,170,255)` 不在 EGA 預設十六色裡**。EGA 的 64 色盤每個通道兩位元
（0／85／170／255）都合法，所以那是編輯器自己重載過的調色盤暫存器。
拿遊戲本體的對話框配色套上去會整個偏色。

## 五、介面：字串全拿到了（已確認）

解包版的字串把整個介面攤開了。**編輯器不是畫筆工具，是參數式地形產生器。**

```
MAXIS SimCity Terrain Editor
NEW GAME                       EXIT
Terrain Creation Parameters
 Number     Number     River
of Trees   of Lakes  Curviness
   Go        Cancel
%3d%%   （六個，三個參數各兩個？未解）
Now terraforming
Smoothing...
Enter Game Year:
Easy   Medium   Hard
Cannot Picture file.
Please insert SimCity disk %c in drive %c
```

圖形檔名是照模式選的，六種都在：`cgated.pgf`／`tdyted.pgf`／`monoted.pgf`／
`mcgated.pgf`／`cegated.pgf`／`segated.pgf`，路徑用 `%s\%s` 組。

後面接的是地物名稱表（`Open land`／`Water`／`Forest`／`Park`／`Rubble`／`Flood`／
`Radio Active`／`Fire`／`Road`／`Power line`／`Rail`／`Residential`…），與遊戲共用。

### 這三個參數 remake 早就有

`Number of Trees`／`Number of Lakes`／`River Curviness` 正好對上
`internal/sim/terrain.go` 的 `TerrainParams`：

```go
type TerrainParams struct {
	TreeLevel    int // -1 => 隨機量
	LakeLevel    int // -1 => 隨機量；0 => 不造湖
	CurveLevel   int // -1 => 預設彎曲度；0 => 不造河
	CreateIsland int // -1 => 一成機率；0 => 不造島；1 => 一定造島
}
```

出自 `s_gen.c:76-79`，早就照著實作了（`GenerateMap`），`Smoothing...` 對應的
`smoothRiver`／`smoothTrees` 也在（`SmoothTerrain`）。**編輯器的規則層等於全解，
缺的只有那個對話框的版面。**

⚠ **`CreateIsland` 在原版編輯器的介面上沒有對應的字串。** 它是不是藏在別處、
或者編輯器根本不開放它，**未解**——不要因為 remake 有這個欄位就自己加一個滑桿上去。

## 六、反組譯：版面挖出來了（已確認）

解包版丟進 IDA（`tools/ida.sh analyze TERRAIN.EXE`，707 個函式）。
字串**沒有直接的程式碼 xref**——16-bit DOS 常見的形狀，它們是被 `dd` 指標表
與 16 位元立即數引用的。所以改成「拿字串在資料段裡的位移去掃全檔」，
一次就把畫對話框的函式反查出來：

| 字串 | 位移 | 命中的函式 |
|---|---|---|
| `Terrain Creation Parameters` | 0x140 | `sub_11402` … |
| `of Trees   of Lakes  Curviness` | 0x17B | **`sub_11402`** |
| `   Go   ` | 0x19E | **`sub_11402`** |
| ` Cancel ` | 0x1A7 | **`sub_11402`** |
| `MAXIS SimCity Terrain Editor` | 0x0D2 | `sub_10A0A` … |
| `Smoothing...` | 0x109 | **`sub_10A0A`**（唯一命中）|

`sub_11402`（1513 位元組）是參數對話框，`sub_10A0A`（1159 位元組）是主選單。

版面全部收攏進 [`../spec/terrain-editor.md`](../spec/terrain-editor.md)（**READY**）：
視窗 36×10 字元格、六個 `◄`／`►` 增減鍵的欄列、Go 與 Cancel 的位置、
三個參數的預設值都是 50。

⚠ **`ds:0x19A` 與 `ds:0x19C` 是單字元字串 `0x11` 與 `0x10`**，
也就是 CP437 的 `◄` 與 `►`。光看 `.asm` 只會看到兩個位移，
是把資料段的位元組印出來才認出來的——**位移要換算回位元組再看**。

## 七、參數的語意與換算（已確認）

三個百分比存在 dseg 的三個 16 位元全域，定義處就寫著初值：

```
059196  word_59196  dw 32h      ← Number of Trees，50
059198  word_59198  dw 32h      ← Number of Lakes，50
05919A  word_5919A  dw 32h      ← River Curviness，50
```

全檔掃過一遍，這三個全域只有 **26 處**引用（`TERRAIN.EXE.asm`，707 個函式），
分成四群：對話框顯示、`◄`／`►` 處理、產生流程的三個閘門、三支消費函式。
沒有第四種用途。

### 產生流程與 Micropolis 的 `GenerateMap` 逐行對得上

`sub_10A0A`（主選單）在 `0x010C58` 起的九個呼叫，對照 `s_gen.c:127 GenerateMap()`：

| `s_gen.c` | TERRAIN.EXE |
|---|---|
| `SeedRand(r)` | `0x010C58 call sub_119EC` |
| `ClearMap()` | `0x010C5D call sub_11EA4` |
| `MakeNakedIsland()`（`CreateIsland == 1`）| `if (byte_52E72) call sub_11CD4` |
| `if (CurveLevel != 0) DoRivers()` | `if (word_5919A) call sub_121EE` |
| `if (LakeLevel != 0) MakeLakes()` | `if (word_59198) call sub_11DEA` |
| `SmoothRiver()` | `0x010C8B call sub_11FC4` |
| `if (TreeLevel != 0) DoTrees()` | `if (word_59196) call sub_11ED8` |
| `SmoothTrees()` ×2 | `0x010C9C`／`0x010CA1 call sub_120F4` ×2 |

**三個百分比就是 `TreeLevel`／`LakeLevel`／`CurveLevel` 本身**，不是要再換算的中間量：
判空的方式（`!= 0` 才做）與傳進去的位置都一樣。

### 三支消費函式讀出來的式子

| 參數 | TERRAIN.EXE | 位址 | `s_gen.c` |
|---|---|---|---|
| 湖泊數量 | `Lim1 = pct / 2` | `sub_11DEA`＋`0x11DFB`（`cwd`／`sub ax,dx`／`sar ax,1`）| `Lim1 = LakeLevel / 2`（`s_gen.c:234`）**相同** |
| 河流彎曲 | `r1 = pct + 10`、`r2 = pct + 100` | `sub_12248`（DoBRiv）、`sub_122D4`（DoSRiv）| `s_gen.c:422-423`／`447-448` **相同** |
| 樹叢擴散距離 | `dis = Rand(2 × pct + 100) + 50` | `sub_11F30`＋`0x11F3C` | `dis = Rand(100 + TreeLevel * 2) + 50`（`s_gen.c:279`）**相同** |
| **樹叢數量** | **`Amount = 3 × pct`** | `sub_11ED8`＋`0x11F10`（`mov cx,ax`／`shl ax,1`／`add ax,cx`）| `Amount = TreeLevel + 3`（`s_gen.c:301`）**不同** |

樹叢數量是唯一不一致的一條。50% 在 DOS 編輯器是 150 叢，照 `s_gen.c` 只有 53 叢——
差三倍，不是捨入誤差。其餘三式連常數都一樣，所以這不是「兩份無關的實作」，
是同一份程式在這一行上的分歧。

⚠ **`SIMCITY.EXE` 走哪一式未查。** 遊戲本身沒有調這三個值的介面，
`TreeLevel` 一直是 -1（走 `Rand(100) + 50` 那一支），這條分歧在遊戲裡是死碼；
要回答得把整支 `SIMCITY.EXE` 反組譯，目前 `workplace/ida/image16.bin.asm`
只涵蓋 4 支函式。

### 樹林平滑跑幾次

`sub_11ED8` 自己結尾就呼叫 `sub_120F4` 兩次（與 `DoTrees()` 一樣），
而 `sub_10A0A` 在它之後**又呼叫兩次**。所以：

- 樹木百分比 > 0 → `SmoothTrees` 共四次
- 樹木百分比 ＝ 0 → 兩次（此時沒有樹可平滑，看不出差別）

`s_gen.c` 只有 `DoTrees()` 裡的兩次。

## 八、對話框的操作（已確認）

`sub_11402` 在 `0x11720` 起是訊息迴圈。

**鍵盤**（`sub_25DE6` 取字元）：

| 鍵 | 動作 |
|---|---|
| `+` | 焦點往後：`focus = (focus + 1) % 8`，再 `sub_19139(0x800 + focus)` 反白 |
| `-` | 焦點往前，0 折回 7 |
| `Enter`／`G`／`g` | 等同 Go |
| `Esc`／`C`／`c` | 等同 Cancel |

八個控制項的編號 `0x800`–`0x807` 與 §六的表一致，焦點就在這八個之間輪。

**滑鼠**（`0x117B3` 起的 switch，八個 case）：

| 編號 | 動作 |
|---|---|
| 0x800／0x801 | `word_59196 = clamp(值 ∓ 1, 0, 100)` |
| 0x802／0x803 | `word_59198` 同上 |
| 0x804／0x805 | `word_5919A` 同上 |
| 0x806 | 結果 ＝ 1（Go）|
| 0x807 | 結果 ＝ 0（Cancel）|

夾限是 `sub_113E4(0, 值, 100)`（cdecl 由右往左推 `100`、`值`、`0`）。
**一次只加減 1**，沒有加速段；長按由自動重複補：`0x118B0` 起記下按下的時刻
（`sub_18CA9` 取計時器），只要 `sub_17C40` 還回報按著、且經過的時間超過 **5** 個
計時單位，就跳回 `0x117B3` 再執行一次同一個 case。

改完值之後只重畫那一行：格式字串換成 `dseg:0x1C8`／`0x1D0`／`0x1D8`。
**這就是六份 `%3d%%%%` 的用途**——三個參數各兩份，初次畫用前三份、重畫用後三份，
不是「六個標籤」。

## 九、還沒做的

編輯器本體在 remake 裡**只做了參數對話框那一個**。原版是一個完整的繪圖程式，
下列都還沒有對應物：

| 原版 | remake |
|---|---|
| 六個工具 `DIRT`／`TREES`／`RIVER`／`CHANNEL`／`FILL`／`UNDO` | 沒有 |
| `TERRAIN` 選單的七個動作 | 只有「產生地形」那一條，而且是對話框按「開始」時做的 |
| `SYSTEM` 選單（讀寫城市檔、列印、離開）| 遊戲本體有對應物，編輯器沒有自己的入口 |
| `PARAMETERS` → `Game Year`／`Sound On` | 沒有 |
| `City Map` 全市地圖視窗 | 遊戲本體有 |

規則層的東西倒是齊的：`Clear Map`／`Smooth Trees`／`Smooth Rivers`／
`Create Island` 在 `internal/sim/terrain.go` 都有對應函式（`clearMap`、
`smoothTrees`、`smoothRiver`、`makeIsland`），缺的是介面與「畫筆改地圖」那一層。

## 十、規則層其實已經解完了

編輯器要寫的東西 remake 早就有：

- 地形圖塊編號與河岸／樹林的邊緣規則 → `internal/sim/terrain.go` 的
  `SmoothTerrain()`（`s_gen.c` 的 `smoothRiver`／`smoothTrees`）。
- 城市檔的讀寫與逐位元組 round-trip → `internal/game/save.go`、
  [`../formats/01-city-file.md`](../formats/01-city-file.md)。
- 從遮罩產生地圖的路徑 → `tools/citymap`（本專案畫台北台中台南用的就是它）。

差的只有樹叢數量那一式，已經補進 `TerrainParams.EditorDOS`。
