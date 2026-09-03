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
| `PARAMETERS` | `Name & Level`／`Game Year`／`Sound On`（開關，字串本體是 ` Sound Off`）|

兩件事因此定案：

- **`Create Island` 在 `TERRAIN` 選單裡，而且是一個「開關」**：勾起來之後
  下一次「產生隨機地形」才造島，不是按下去就造一座島。反組譯的證據在 §十二。
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

解包版的字串把介面的文字全攤開了（工具盤那六個標籤是**畫**出來的，
不在字串表裡——見 §九）。

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
`smoothRiver`／`smoothTrees` 也在（`SmoothTerrain`）。

`CreateIsland` 在字串表裡找不到，是因為它是 `TERRAIN` 選單的一條
（` Create Island           CTRL-I`，`dseg:0x19A4`），而且是**開關**不是參數——
見 §十二。

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

## 九、主畫面的版面（實機量測，已確認）

量法同 §四之三：`E` 模式截圖裁出 640×350 的遊戲區，逐列逐行讀色
（`workplace/dosbox/tep-00-ui.png`、`tef-15-filled.png`）。

**結論先講：編輯器用的是遊戲本體那一套視窗系統，座標一模一樣。**
`docs/spec/ui-layout.md` §二量到的每一個數字都對得上——選單列 y 0–17、
編輯視窗外框 x 5–579／y 21–324、標題列 y 24–37、資金帶 y 38–54、
地圖區 x 64–575／y 55–310、工具帶 y 311–324、City Form 視窗 x 240–639／
y 21–347。所以 remake 不必為編輯器另量一套版面。

| 元件 | 編輯器 | 遊戲本體 |
|---|---|---|
| 選單列 | 三個標題（中心 x ＝ **159／347／551**）| 四個（112／250／402／554）|
| 編輯視窗標題列 | **只有年月**（實測 `Jan 1900`），沒有城市名 | 城市名置中 ＋ 年月靠右 |
| 資金／訊息帶 | **空的**（純深灰）| `Funds: $20,000` ＋ 訊息 |
| 工具盤 | 六個 52×24 的文字按鈕，x 8–63、y 55–210 | 庫 2 的美術，2 欄 × 7 列 |
| 需求指標 | 沒有 | 庫 3，(12,237) |
| 工具帶 | 目前工具的名稱（`Dirt`／`Trees`）| 名稱 ＋ 造價 |
| 右邊的視窗 | `City Map`，**沒有圖層圖示也沒有色階** | City Form，九個圖層圖示 ＋ 色階 |

### 工具盤的六格

```
外框     x 8–63、y 55–210（白 2 像素、內一圈黑）
按鈕     x 11–62（寬 52），第 i 個 y ＝ 59 ＋ 25i，高 24
分隔     每個按鈕之間一列 (85,85,85)，上緣 y 57–58、下緣 y 208–209
選取     內縮的黃色 2 像素框（DIRT 選取時實測 y 59–60 與 80–81 是黃）
```

每一格的底是**對應地物的圖塊平鋪**——DIRT 是泥土、TREES 是樹林、
RIVER 是水面、CHANNEL 是水道；FILL 是灰底加藍斜線、UNDO 是紅底。
標籤是帶黑描邊的粗體字，顏色固定：DIRT 黃、TREES 白、RIVER 與 CHANNEL 青、
FILL 與 UNDO 黑。**選取狀態只由那一圈黃框表示，不是換字色。**
UNDO 沒得復原時整格鋪紅白棋盤（看起來被停用）。

⚠ 這解釋了 §二 的「`*TED.PGF` 裡沒有介面美術」：按鈕的底就是第 0 庫的
地圖圖塊，程式自己鋪的，不需要另一份美術。

## 十、六個工具（反組譯，已確認）

### 十之一 工具描述表

`sub_1EF36` 與 `sub_1F0C0` 都用 `18 × byte_595E0` 當索引去讀 `ds:0x2B42`
起的一張表，一列 18 位元組。把資料段的位元組印出來（dseg 在解包檔的
偏移是 **0x4B6E0**）：

| 列 | +0x00 | +0x02 旗標 | +0x04 造價 | +0x0C 圖塊 | +0x10 尺寸 | 是誰 |
|---:|---|---|---:|---:|---:|---|
| 1 | 01 | 0000 | 0 | 0 | 1 | **DIRT** |
| 2 | 01 | 3000 | 0 | 37 | 1 | **TREES** |
| 3 | 01 | 0000 | 0 | 3 | 1 | **RIVER** |
| 4 | 01 | 0000 | 0 | 4 | 1 | **CHANNEL** |
| 5 | 01 | 1000 | 10 | 40 | 1 | 遊戲的公園（編輯器沒用到）|
| 6–15 | 00 | … | 100–10000 | 240／423／612／770／761／779／811／693／709／745 | 3–6 | 遊戲的分區與建物 |

第 6–15 列與 Micropolis 的 `RESBASE 240`／`COMBASE 423`／`INDBASE 612`／
`PORTBASE 693`／`AIRPORTBASE 709`／`COALBASE 745` 逐項對得上，
造價也與訊息檔的 `Police station: $500`⋯一致——**這張表是遊戲與編輯器共用的**，
編輯器只用得到前四列。

寫入那一行是 `sub_1EF36`＋0x1F0A6：

```
Map[x][y] = 表[工具].圖塊 + 表[工具].旗標
```

所以四個畫筆寫的 16 位元字是 **0 ／ 0x3025 ／ 3 ／ 4**。
`0x3025` ＝ WOODS 37 加 `BURNBIT|BULLBIT`。

⚠ **RIVER 寫的是 3（REDGE）不是 2（RIVER）**。真正的河面圖塊要靠之後的
「平滑河流」算出來——這也是為什麼原版要把平滑放進選單。

### 十之二 一次一格，值一樣就不寫

`sub_1EF36` 的尾巴：

```
cmp [bx+di+4482h], ax   ; 現值 == 要寫的值？
jz  short loc_1F0B6     ; 一樣就直接回 1，不記復原也不寫
call sub_106BA          ; 記一格的復原
mov [bx+di+4482h], ax
```

界外（`x+size > 120` 或 `y+size > 100` 或負數）直接回 0。
**尺寸欄雖然參與界限判斷，寫入卻只有一格**——編輯器那四列的尺寸都是 1。

### 十之三 拖曳

`sub_1F0C0` 是按住不放時的迴圈：`sub_17C40` 回報還按著就一直呼叫
`sub_1EF36`，並在游標移出視窗時捲動地圖。表的 +0x00 是「可不可以拖曳」，
編輯器那四列都是 1（遊戲的分區是 0，所以蓋不出一排體育館）。

## 十一、FILL 是開關，UNDO 是動作（已確認）

`sub_22636(n)` 是編輯器自己的「選工具」，**不是單純設值**：

```
n == 6 → sub_10862()            復原（動作）
n == 5 → byte_59194 ^= 1        油漆桶（開關）
n <  5 → byte_595E0 = n         換畫筆
之後一律 sub_2268C(0) 重畫工具盤
```

所以工具盤那六格是「四個畫筆 ＋ 一個開關 ＋ 一個動作」，不是六個畫筆。

### 十一之一 油漆桶（`sub_229F0`）

油漆桶亮著時，`sub_22DAE` 把點擊送到 `sub_229F0(x, y, 目前畫筆)`。

**帶是由起點那一格的地物類別決定的，不是由畫筆決定的**
（`sub_229F0`＋0x22A94 起的三段比較，上界都是開區間）：

| 起點的圖塊編號 | 帶 | 什麼情況不做事 |
|---|---|---|
| < 2 | [0, 2) 空地 | 畫筆是 DIRT |
| < 21 | [2, 21) 水域與河岸 | 畫筆是 RIVER 或 CHANNEL |
| < 40 | [21, 40) 樹林 | 畫筆是 TREES |
| ≥ 40 | —— | **退化成單格畫筆**（`loc_22B64` 直接呼叫 `sub_1F0C0`）|

然後 `sub_106BA(-1,-1)` 記一份全圖快照，再做掃描線填色：沿 x 軸走、
上下兩列找新的種子，落在帶裡的格子一律寫成

```
畫筆 1 → 0        畫筆 2 → 0x3025
畫筆 3 → 3        畫筆 4 → 4
```

（`loc_22D02`／`loc_22D28`／`loc_22D40`／`loc_22D58` 四個 case，
寫的是**原始字**不是「圖塊＋旗標」，值與畫筆那一組相同。）

倒完之後 `loc_22D98` 呼叫 `sub_22636(5)` 把油漆桶**熄掉**——一次性的。

### 十一之二 復原（`sub_106BA` 記、`sub_10862` 還原）

環形緩衝區 **5000 格**，一格四個位元組 `{x, y, 舊值}`，
折返靠 `sub_1C9C2(0, idx±1, 0x1387)`（小於下限折到上限、大於上限折到下限）。

- `sub_106BA(x, y)`：把 `Map[x][y]` 的現值寫進環，head 前進；
  head 撞上 tail 就把 tail 也推一格（最舊的那一步被擠掉）。
- `sub_106BA(-1, -1)`：**全圖快照**。快照緩衝區最多四份（`cmp word_4BFC2, 4`），
  滿了就把環的 tail 推到最舊那個快照標記的後面、緩衝區整批往前搬一格，
  再把 12 000 格複製進去。快照在環裡也佔一格，x 與 y 都寫 `0xFF`。
- `sub_10862`（復原）：`head == tail` 就發**第 7 號音效**（工具失敗）並收工；
  否則 head 退一格，`x == 0xFF` 就從最後一份快照還原整張圖，
  否則只還原那一格。

會記快照的動作：清除地圖、清除人造物、產生隨機地形、三個平滑、油漆桶。

## 十二、三個選單的命令碼（已確認）

字串表在 `dseg:0x1950`（SYSTEM，11 列）、`0x1980`（TERRAIN，10 列）、
`0x19CC`（PARAMETERS，4 列），三張表的位置又列在 `dseg:0x19E0`。
標題在 `dseg:0x1940`：`0x170A` SYSTEM／`0x1711` TERRAIN／`0x1719` PARAMETERS。

**命令碼就是 `(選單編號 << 4) | 列號`**，直接當成 `sub_10A0A` 的參數。
證據是分隔線：SYSTEM 的分隔線在第 1／3／5／9 列，而 IDA 把
`sub_10A0A` 的 case 1、3、5、9 標成 default——四個都對得上，
`0x0B`–`0x0F`（SYSTEM 只有 11 列）也是 default。

| 碼 | 選單／列 | 動作 |
|---|---|---|
| `0x00` | SYSTEM 0 About TERRAIN | `sub_1E5FC()` 開關於視窗 |
| `0x02` | SYSTEM 2 Print | `sub_1D078()` 然後 return |
| `0x04` | SYSTEM 4 Start New City | 確認框標題 `NEW GAME`（`ds:0xEF`）→ `sub_1CEA0(0)` |
| `0x06` | SYSTEM 6 Load City | `sub_1FA6A(0,0,0)` ＋ 兩次 `sub_2190A` |
| `0x07` | SYSTEM 7 Save City as | `sub_1FE44(0)`（沒帶檔名 → 問）|
| `0x08` | SYSTEM 8 Save City | `sub_1FE44(目前檔名)` |
| `0x0A` | SYSTEM 10 Exit | 確認框標題 `EXIT`（`ds:0xCD`）→ `sub_102DA("MAXIS SimCity Terrain Editor")` |
| `0x10` | TERRAIN 0 Clear Map | 快照 → `sub_119EC`（SeedRand）→ `sub_109E0` 重畫 |
| `0x11` | TERRAIN 1 Clear Unnatural Objects | 見 §十三 |
| `0x13` | TERRAIN 3 Create Random Terrain | 參數對話框 → 20×5 的「Now terraforming」→ 產生流程（§七）|
| `0x15` | TERRAIN 5 Smooth Trees | 16×5 的「Smoothing...」→ `SmoothTrees` ×2 |
| `0x16` | TERRAIN 6 Smooth Rivers | 同上 → `sub_11A24` ＋ `SmoothRiver` |
| `0x17` | TERRAIN 7 Smooth Everything | 兩個位元都設：**先河後樹** |
| `0x19` | TERRAIN 9 Create Island | `byte_52E72 ^= 1` ＋ `sub_17E9A` 重畫選單 |
| `0x20` | PARAMETERS 0 Name & Level | `sub_1C9F8()` ＋ `sub_230E6()` |
| `0x21` | PARAMETERS 1 Game Year | `sub_111E4()`，見 §十四 |
| `0x23` | PARAMETERS 3 Sound | `byte_59444 ^= 1` ＋ `sub_17E9A` 重畫選單 |

`Sound` 那一列的字串是 ` Sound Off`（`dseg:0x18CF`），程式把結尾三個字
換成 `dseg:0x19F4` 的 `On`／`Off`，所以畫面上看到的是 `Sound On`。

### ⚠ 這一節推翻了 §四之二 的一句話

`Create Island` **不是一個動作**，是一個**開關**：它只翻轉 `byte_52E72`，
下一次「產生隨機地形」走到 `if (byte_52E72) call sub_11CD4` 才會造島
（`sub_10A0A`＋0x10C6B）。原本寫成「是一個動作不是一個滑桿」只講對了一半。

## 十三、Clear Unnatural Objects（已確認）

`sub_10A0A` 的 case 0x11（`loc_10B39`）：逐格取低十位元，
**大於 37（WOODS）就把整個 16 位元字寫成 0**——旗標一起清掉，不是只換圖塊。

```
for x in 0..119:
  for y in 0..99:
    if (Map[x][y] & 0x3FF) > 37: Map[x][y] = 0
```

## 十三之二、Smooth Rivers 的前置（`sub_11A24`，已確認）

「平滑河流」不是直接呼叫 `SmoothRiver`，前面還有一支
**Micropolis 沒有的**函式：把每一個**四鄰裡有非水面**的水格打回 `REDGE`。

```
水面 ＝ 圖塊編號在 [2, 20]
for x in 0..119, y in 0..99:
  if 不是水面: 下一格
  if x > 0   且 左鄰不是水面 → 寫 3
  if x < 119 且 右鄰不是水面 → 寫 3
  if y > 0   且 上鄰不是水面 → 寫 3
  if y < 99  且 下鄰不是水面 → 寫 3
```

⚠ **地圖最外圈那一側的鄰居不檢查**（`or di,di / jle`、`cmp si,77h / jge` 那幾條），
所以貼著邊的水面不會被打回 REDGE。

為什麼需要它：`SmoothRiver` 只改寫**本來就是 REDGE** 的格子，
而畫筆與油漆桶畫出來的水塊內部全是 REDGE 或 CHANNEL，沒有這一步算不出岸線。

## 十四、Game Year（`sub_111E4`，已確認）

視窗是 `sub_1C010(&win, 0x12, 5)` ＝ **18 欄 × 5 列**，
標題 `Enter Game Year:`（`dseg:0x127`），欄位預填 `sprintf("%4d", CityTime/48 + 1900)`
（`dseg:0x137` 是那個 `%4d`）。

收下的規則（`sub_111E4`＋0x11365 起）：

```
if strlen(輸入) != 4: 發第 7 號音效，丟掉
t = atol(輸入) * 48 - 0x16440      ; 0x16440 = 91200 = 1900 × 48
if t <= 0: 丟掉
CityTime = t
```

**所以年份必須是四位數而且要大於 1900**，換算是 `CityTime = (年 − 1900) × 48`。

⚠ 這裡的基準是 **1900**，與 `docs/re/16-dos-oracle.md` §七 量到的
「遊戲狀態列顯示 `1849 + CityTime/48`」不同。編輯器與手冊、劇本簡介同一個基準，
狀態列才是那個對不上的。（實測編輯器的標題列是 `Jan 1900`，不是 `Jan 1849`。）

## 十五、關於畫面（實機截圖，已確認）

`SYSTEM → About TERRAIN` 開一個黃框視窗（實測外框 x 168–471、y 28–307），
藍白棋盤底、深藍字。全文轉錄（保存用）：

```
SimCity Terrain Editor
Copyright Maxis 1989

Concept & design:Will Wright
IBM programming :Paul Schmidt
            and :Daniel Goldman
City artwork    :Don Bayless
Title screens   :Richard Payne
  and icons
Documentation   :Michael Bremer

For more information contact:
  Maxis
  1042 Country Club Drive, Suite C
  Moraga, CA 94556

  Tel: (415) 376-6434
  FAX: (415) 376-1823
```

（截圖：`workplace/dosbox/ted2-03-about.png`，不入版控。）

**remake 不照抄這一頁**：職稱表是史料，留著；1989 年的地址、電話與傳真
不放進會跑起來的程式裡——那等於拿別人的聯絡方式當自己的。
remake 的關於頁寫在 `internal/ui/terrain_draw.go` 的 `teAboutLines`。

## 十六、remake 做到哪裡

**整個編輯器都接上了**（`internal/sim/editor.go`、`internal/ui/terrain_screen.go`、
`internal/ui/terrain_draw.go`）：三個選單的十七條命令、六格工具盤、
畫筆與拖曳、油漆桶、五千格的復原環加四份全圖快照、參數對話框、
年份輸入、市名與難度、City Map 視窗、狀態列。

| 原版 | remake | 備註 |
|---|---|---|
| 四個畫筆 ＋ 油漆桶 ＋ 復原 | 全做 | 寫的 16 位元字與原版表相同 |
| `TERRAIN` 的七條 | 全做 | 造島是**開關**，照原版 |
| `SYSTEM` 的讀寫城市檔 | 沿用遊戲本體那一套 | |
| `SYSTEM` → `Print` | 沿用遊戲的「存成 PNG」 | **remake 自訂的對應物** |
| `SYSTEM` → `Exit` | 離開編輯器回遊戲 | **remake 自訂**：原版是離開程式 |
| `PARAMETERS` 三條 | 全做 | 年份的四位數與 > 1900 判斷照原版 |
| `About TERRAIN` | 只轉錄職稱表 | 見 §十五 |
| `Now terraforming`／`Smoothing...` | 照原版的字元格畫，留幾格畫格 | remake 的動作是瞬間完成的 |

還沒解的：

- **確認框（`sub_1C4B2`）的版面**。原版在「開新地圖」與「離開」前會問一次，
  標題分別是 `NEW GAME` 與 `EXIT`，但框的尺寸與按鈕位置沒有量。
  remake 先沿用參數對話框那一套配色與 70×20 的按鈕，**標成自訂**。
- **`Now terraforming`／`Smoothing...` 與年份框的絕對位置**。
  欄列數是從 `sub_1C010` 的參數讀出來的（20×5、16×5、18×5），
  但視窗原點的公式只有 36×10 那一個實際量過，其餘是外推（假說）。
- **`TERRAIN.CFG` 第 5 個位元組 `0x91`**（§三）。
- **`MCGATE.PPF` 的調色盤**（§一）。

## 十七、規則層與 Micropolis 的關係

編輯器要寫的東西大部分 remake 早就有：

- 地形圖塊編號與河岸／樹林的邊緣規則 → `internal/sim/terrain.go` 的
  `smoothRiver`／`smoothTrees`（`s_gen.c`）。
- 城市檔的讀寫與逐位元組 round-trip → `internal/game/save.go`、
  [`../formats/01-city-file.md`](../formats/01-city-file.md)。
- 從遮罩產生地圖的路徑 → `tools/citymap`（本專案畫台北台中台南用的就是它）。

**但有三條是 Micropolis 沒有、只在 `TERRAIN.EXE` 裡的**，全部寫進
`internal/sim/editor.go`：

1. 樹叢數量 `3 × pct`（§七）。
2. 平滑河流之前的 `ResetRiverEdges`（§十三之二）——`s_gen.c` 沒有這一步。
3. 「清除人造物」的 `> 37 就整格寫 0`（§十三）。
