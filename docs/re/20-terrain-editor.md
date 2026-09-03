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

## 四、卡住的地方：`TERRAIN.EXE` 在 DOSBox-X 跑不起來

設定檔補齊之後，程式會啟動、不報錯、也不回到提示符，畫面停在文字模式，
DOSBox 的 log 一路刷：

```
ERROR CPU:Illegal Unhandled Interrupt Called 6
```

INT 6 是無效指令。試過兩組設定，四十五秒內都沒有進圖形模式：

| 設定 | 結果 |
|---|---|
| `machine=svga_s3　cputype=386　memsize=16　cycles=fixed 20000` | INT 6 迴圈 |
| `machine=ega　cputype=286　memsize=1　cycles=fixed 3000` | INT 6 迴圈 |

⚠ **同一套環境跑得動遊戲本體**（`SIMCITY.EXE` 一路正常），所以不是容器或
X 的問題，是這支程式與這個模擬器的組合。

執行檔本身是好的，而且知道是怎麼包的：

```
MZ 檔頭：頁數 163、最後頁 429 → 映像 83 373 位元組 ＝ 實際檔案大小（沒有截斷）
重定位項 0、檔頭 32 位元組、CS:IP = 13a1:000e
偏移 0x1C 有 "LZ91" → LZEXE 0.91 打包
```

`LZ91` 也解釋了為什麼字串是碎的（481 條裡幾乎都是壓縮資料的碎片）——
與 1.10 的 `SIMCITY.EXE`、資料片的 `UPDATE.DAT` 同一類，不能直接反組譯。

## 四之二、解包之後：錯誤訊息變得有意義，但還是跑不起來

`unlzexe`（`mywave82/unlzexe` 的 C 原始碼，容器內 `gcc` 編）一次就解開：

```
file 'TERRAIN.EXE' is compressed by LZEXE Ver. 0.91
```

| | 位元組 | 可讀字串 |
|---|---:|---:|
| 打包版 | 83 373 | 481（幾乎都是壓縮資料的碎片）|
| **解包版** | **325 728** | **751（真的字串）** |

⚠ `unlzexe` 的輸出檔名緩衝區只有 12 個字元（DOS 時代的遺跡），
給長路徑會被截斷成別的檔名而且**不報錯**。要 `cd` 到目標目錄再用短檔名。

解包版拿去跑，錯誤訊息從「INT 6 無限迴圈」變成明確的一行：

```
FATAL ERROR: PROGRAM ABORTED
256K of VGA/EGA memory
Couldn't load VGA/EGA blocks!
```

`machine=svga_s3` 與 `machine=ega` 兩種都一樣。**所以打包殼確實是 INT 6 的成因，
但底下還有第二個問題**：它載不進 VGA/EGA 的圖形區塊。還沒解。

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

## 九、下一個入口（依序試，不要跳）

1. **`vmemsize`／`machine=vgaonly`。** 錯誤訊息點名 VGA/EGA 記憶體，
   下一個要動的就是這個設定，不是 CPU。
2. **換原版 DOSBox。** 容器裡只有 `dosbox-x`，要改 image。
3. **找編輯器自己的說明書。** 版面如果有印出來就不必靠跑。

這三項是為了「眼睛確認」，不是實作的必要條件：介面文字、版面、參數語意與換算
都已經是一手證據。

## 十、規則層其實已經解完了

編輯器要寫的東西 remake 早就有：

- 地形圖塊編號與河岸／樹林的邊緣規則 → `internal/sim/terrain.go` 的
  `SmoothTerrain()`（`s_gen.c` 的 `smoothRiver`／`smoothTrees`）。
- 城市檔的讀寫與逐位元組 round-trip → `internal/game/save.go`、
  [`../formats/01-city-file.md`](../formats/01-city-file.md)。
- 從遮罩產生地圖的路徑 → `tools/citymap`（本專案畫台北台中台南用的就是它）。

差的只有樹叢數量那一式，已經補進 `TerrainParams.TreeAmountDOS`。
