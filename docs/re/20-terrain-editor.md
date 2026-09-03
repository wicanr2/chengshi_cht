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

## 五、下一個入口（依序試，不要跳）

1. **把 LZEXE 0.91 解開。** 這條同時解決兩件事：解開後是乾淨的反組譯目標
   （對照組：1.03 的 `SIMCITY.EXE` 沒打包，859 條可讀字串），而且解開的
   執行檔常常在模擬器上跑得起來——INT 6 有可能就是打包殼與 DOSBox-X 的
   組合問題。格式是公開的，寫一支解包器是有界的工作。
2. **換原版 DOSBox。** 容器裡只有 `dosbox-x`；1990 年的程式在 vanilla
   DOSBox 上的相容性經常比較好。要改 image。
3. **再掃模擬器設定**（`machine=vgaonly`、`cputype=386_prefetch`、
   `core=simple`、更低的 cycles）。便宜但是亂槍打鳥，**排在前兩項之後**。
4. **找編輯器自己的說明書。** 版面與工具清單如果有印出來，就不必靠跑。

## 六、規則層其實已經解完了

編輯器要寫的東西 remake 早就有：

- 地形圖塊編號與河岸／樹林的邊緣規則 → `internal/sim/terrain.go` 的
  `SmoothTerrain()`（`s_gen.c` 的 `smoothRiver`／`smoothTrees`）。
- 城市檔的讀寫與逐位元組 round-trip → `internal/game/save.go`、
  [`../formats/01-city-file.md`](../formats/01-city-file.md)。
- 從遮罩產生地圖的路徑 → `tools/citymap`（本專案畫台北台中台南用的就是它）。

**缺的只有介面**：原版的版面、工具清單、操作方式。所以第五節那四條全都是
為了同一個問題——把介面看清楚。
