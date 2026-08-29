# 16 — DOS 原版當 oracle

Micropolis 回答得了「規則怎麼算」，回答不了「DOS 版按下去會怎樣」。
有些問題只有把 1991 年那支執行檔跑起來才有答案：八段音效各對應哪個事件、
防拷有沒有被拔掉、螢幕模式怎麼對到圖形檔的目錄。

工具：[`tools/dosbox.sh`](../../tools/dosbox.sh)（外層）
＋ [`tools/dosbox_inner.sh`](../../tools/dosbox_inner.sh)（容器內）
＋ [`docker/dosbox.Dockerfile`](../../docker/dosbox.Dockerfile)。

## 一、怎麼跑

```bash
docker build -f docker/dosbox.Dockerfile -t simcity-dosbox:x docker/
RUN=simcity ACTIONS="$PWD/tools/dosbox/act-disasters.txt" tools/dosbox.sh 8 dis
```

產物在 `workplace/dosbox/`：截圖、`.raw`（s16le 立體聲 22050 Hz 的全程錄音）、
`.marks`（每個動作的時間戳）、`.log`。

動作腳本一行一個動作：`key`／`click`／`drag`／`press`／`move`／`release`／
`wait`／`shot`／`mark`。**選單是按住式的**——`click` 打不開，要
`press` 在標題上、`move` 到項目、再 `release`。

座標是 **DOS 畫面座標**（640×350 的左上角是 0,0）。DOSBox 0.74 把視窗放在
螢幕左上角，兩者剛好一樣；**DOSBox-X 會把畫面置中**，差了將近兩百個像素。
腳本執行時用 `xdotool getwindowgeometry` 問一次視窗原點再加上去——
不問的話每一次點擊都落在畫面外，而且完全沒有錯誤訊息，只是「按了沒反應」。

三個設計上的決定：

- **音訊走 SDL 1.2 的 disk 驅動**（`SDL_AUDIODRIVER=disk`），不按 DOSBox 的
  錄音快速鍵。headless 底下少一個會掉的環節，而且整段連續錄，事後才切得開。
  `SDL_DISKAUDIODELAY` 要對上 `blocksize/rate`（1024 @ 22050 ≈ 46 ms）；
  設 1 的話 callback 會被用最快速度呼叫，25 秒錄出 15 分鐘的檔案。
- **`cycles` 寫死**。`auto` 會隨主機負載變動，同一段操作錄出來的長度就不一樣。
- **遊戲目錄是複製進容器的可寫副本**。SimCity 會寫 `SIMCITY.CFG` 與存檔，
  原始素材唯讀掛載。

## 二、防拷：查驗還在，一律判過

開新城市之後跳出的「Enter NAME of city ／ Page: ▢ ▬ ▢」就是手冊查驗
（要翻說明書某頁抄一個城市名）。直接按 Enter 送空白答案，回的是
`Congratulations, you passed.`。

所以 `read.me` 說的「移除防拷」是**改掉判斷結果**，不是把整段拿掉。
這定了 `CLAUDE.md` §2.1 的第 2 條：`SIMCITY.EXE` 確實被動過。

## 三、螢幕模式決定的是**目錄**，不是檔名

把 `SIMCITY.CFG` 的 `Screen Mode` 改成 `T`（Tandy Color）之後，遊戲噴：

```
FATAL ERROR: PROGRAM ABORTED
Cannot open graphics file:C:\tdy\WESTCEGA.pgf
```

檔名還是 `WESTCEGA.pgf`，變的是目錄 `tdy\`。所以 `Graphics Set: WESTCEGA`
是**完整的檔名主幹**，目錄才是由螢幕模式決定的：
`E → CEGA\`、`2 → mcga\`、`V → MONO\`、`e → sega\`、`T → tdy\`。

這也再次確認手上這份副本缺 Tandy 與 CGA 的資料檔。

## 四、音效：問得出「哪個動作有聲音」，問不出「那是哪一段」

`SIMCITY.CFG` 的 `Sound` 欄有四個值：`I` 內建 PC 喇叭、`S` Covox、
`T` Tandy、`N` 無聲。三條路各自的結果：

| 設定 | 結果 |
|---|---|
| `S` Covox | **遊戲自己回報「`Sound Master not found. Using internal speaker`」**。所以 `S` 指的是 **Covox Sound Master**（一張獨立音效卡），不是 LPT 上的 Covox DAC；DOSBox 與 DOSBox-X 都不模擬它。這條路是死的，不是設定沒調好 |
| `T` Tandy | 開不起來：Tandy 模式要 `tdy\` 目錄的圖形檔，這份副本沒有（`Cannot open graphics file:C:\tdy\WESTCEGA.pgf`）|
| `I` PC 喇叭 ＋ DOSBox 0.74 | 只錄得到單頻方波。實測某次事件是 0.2 秒內過零 462 次（約 1155 Hz），取樣值只有 ±5000 兩種——那是 DOSBox `pcspeaker` 的固定音量常數 |
| `I` PC 喇叭 ＋ DOSBox-X 的 `pcspeaker=impulse` | **錄得到有內容的聲音**。四種可重現的長度，對得到具體動作 |

### 四種聲音

用 `tools/dosbox/act-pcm.txt` 與 `act-pcm2.txt` 跑兩輪，動作彼此隔十到
十五秒，事後用每 5 毫秒的標準差切事件（`tools/snd_events.py`）：

| 長度 | 觸發的動作 |
|---|---|
| 0.030 秒 | 用推土機推一格 |
| 0.115 秒 | 鋪路成功、蓋在樹上、關掉整頁訊息 |
| 0.279 秒 | 鋪到水裡、用查詢工具點一格 |
| 1.90 秒 | 災難的整頁訊息（火災與怪獸兩次都是）|

兩輪之間「鋪路 0.115 秒」重現，所以這些長度不是雜訊。

### 內建喇叭放的**不是**資料檔裡的 PCM

上面那組長度看起來像在放某一段取樣，但不是。兩個實驗把這件事釘死：

**一、換掉音效檔，聲音一模一樣。** 同一份動作腳本跑兩輪，一輪用
`Graphics Set: WESTCEGA`（載 `DATA/WEST_SND.PSF`），一輪用 `MOONCEGA`
（載 `DATA/MOON_SND.PSF`），兩份檔案的段長差很多：

| 段 | WEST 取樣數 | MOON 取樣數 |
|---:|---:|---:|
| 4 | 3540 | 7184 |
| 5 | 926 | 1536 |

如果放的是資料，長度應該跟著變成兩倍左右。實測四個事件的長度**完全一樣**
（0.115／0.279／0.284／0.030 秒），逐取樣的波形相關係數是
**0.99／0.89／0.96**——不是「像」，是同一個聲音。

**二、改 CPU 速度，長度不變。** `cycles=fixed 20000` 與 `fixed 5000`
錄到同一組長度，所以發聲是計時器驅動的，不是忙等迴圈；那組長度是真的，
不是被 DOSBox 的節流拉出來的。

所以 `Sound: I` 這條路放的是**程式自己合成的嗶聲**，八段 4 位元 PCM
只給 DAC 用。先前「0.115 秒對到第 5 段、相關 0.90–0.95」是巧合——
它在 MOON 那一輪同樣是 0.115 秒，而 MOON 的第 5 段長度差了 1.66 倍。

工具是 `tools/snd_fit.py`（切事件、以長度反推取樣率、比包絡線）。

### 三條發聲路徑的現況

| 路徑 | 現況 |
|---|---|
| `I` 內建喇叭 | 通，但放的是合成嗶聲，與 `.PSF` 無關 |
| `S` Covox Sound Master | 遊戲開機就跳「`Sound Master not found. Using internal speaker`」。沒有模擬器支援 |
| `T` Tandy | **遊戲接受，不跳錯誤框**（同一時間點與 `S` 的對照截圖是決定性的），但在 `machine=svga_s3` 下 DOSBox-X 根本沒建 Tandy 音效裝置（log 裡一行 tandy 都沒有），錄到的是**振幅恆為 0** 的全靜音——遊戲把樣本寫進了沒被模擬的埠。改 `machine=tandy` 則卡在 `Please wait - loading graphics`，因為這份副本缺 `tdy\` 圖形檔 |

「Tandy 被接受、而且寫出去的東西沒人接」本身是**正面證據**：PCM 那條路
就是 DAC。但我們放不出來。

### 4.1 用替代圖形讓 Tandy 模式跑起來（2026-08-30）

`machine=tandy` 原本卡在 `Please wait - loading graphics`，因為這份副本沒有
`tdy\` 目錄。**繞過去了**：螢幕模式決定的是目錄不是檔名（§3），所以

```bash
MACHINE=tandy CFG_SOUND=T CFG_SCREEN=T CFG_GFX=WESTSEGA TDY_FROM=sega \
  RUN=simcity ACTIONS=… tools/dosbox.sh 6 tdy
```

`TDY_FROM` 把 `sega\`（EGA 低解析度，320×200，與 Tandy 同尺寸）複製成
`tdy\`，`CFG_GFX` 把圖形集改成 `WESTSEGA` 讓檔名對得上。**遊戲跑起來了**
——選單列、對話框、按鈕都畫得出來。

兩個中間狀態，都是照順序踩出來的：

| 設定 | 結果 |
|---|---|
| 不建 `tdy\` | `Cannot open graphics file: C:\tdy\WESTCEGA.pgf` |
| `tdy\` 放 `CEGA`（640×350 的資料）| `Not enough memory to load graphics` |
| `tdy\` 放 `sega` ＋ 圖形集改 `WESTSEGA` | **進得去** |

### 4.2 正對照建起來了，但 Tandy 那一側還是驅動不到觸發點

**Tandy 與 MCGA 都是 320×200，版面相同**——所以可以在讀得出畫面的 MCGA
量座標，再拿去驅動讀不出畫面的 Tandy。座標寫進
[`tools/dosbox/act-quake-320.txt`](../../tools/dosbox/act-quake-320.txt)：

```
標題畫面 START NEW CITY (207,223)
手冊查驗：Return ×2 之後 Continue (304,215)
DISASTERS 選單標題 (327,11)；Earthquake (320,103)
```

⚠ **320×200 的視窗是 640×400（2 倍縮放），座標 ＝ 遊戲像素 × 2**，
與 640×350 的 EGA 模式不同。

**正對照成立**：同一支腳本，`machine=svga_s3` ＋ `Sound: I`（內建喇叭）
在 MCGA 下觸發地震，錄到**最大振幅 9997、63 234 個非零樣本**。
所以腳本確實會觸發發聲，錄音管線也確實會收到。

**但 Tandy 那一側還是到不了觸發點**：

| 設定 | 到哪裡 | 錄到 |
|---|---|---|
| `machine=svga_s3` `Sound: I` MCGA | 進遊戲、觸發地震 | 振幅 9997 ✅ |
| `machine=tandy` `Sound: T` `TDY_FROM=mcga` | **黑畫面**（mcga 資料在 Tandy 下載不起來）| 0 |
| `machine=tandy` `Sound: T` `TDY_FROM=sega` | 卡在防拷對話框，Continue 點不到 | 0 |

最後那一列的原因不是聲音，是**游標飄移**：DOS 的滑鼠驅動吃相對位移，
遊戲自己搬過游標之後絕對座標就對不齊了（`tools/dosbox_inner.sh` 的
`goto` 註解記過同一個坑）。畫面讀得出來的時候可以用畫面校正，
Tandy 這一側讀不出來，所以校正不了。

**三次 Tandy 實驗，三次各自因為不同的理由失敗，沒有一次是因為聲音。**
靜音到目前為止仍然什麼都證明不了
（`~/diagnosis-notes/docs/03-silence-is-not-success/`）。

所以缺的東西沒有變：**一份帶真正 `tdy\` 圖形檔的 1.10 副本**。
有了它，畫面讀得出來、驅動得動，Tandy DAC 那條路才問得出答案。

所以**音效不接進遊戲**。要解開，需要下列其中一項：

- 一份帶 `tdy\` 圖形檔的 1.10 副本（Tandy 畫面 ＋ Tandy DAC 同時成立）；
- 一個會模擬 Covox Sound Master 的環境；
- 反組譯 `SIMCITY.EXE` 的發聲程式。**進度見 §5。**

## 五、脫殼與符號表（2026-08-30）

`CS:0x1EA0` 那個進入點是**破解程式的 stub**，不是原版進入點；原版在
`載入段 + 0xE0 : 0`，而且是自解壓的。脫殼的方法、LZSS 多的那一道
`ror 1`、以及解壓瑕疵怎麼交叉檢查，全部寫在
[`18-dos-parity.md`](18-dos-parity.md) §6.5，工具是
[`tools/unpack_simcity_exe.py`](../../tools/unpack_simcity_exe.py)。
第一個成果是汙染權重（同文件 §6.3）。

**符號表解出來了**（[`tools/dos_symbols.py`](../../tools/dos_symbols.py)）。
它在載入器區（明文，`0x1000` 起），紀錄格式是

```
[4 位元組 far 指標][種類 word][模組 word][位移 word][…][長度][名字][00]
種類 0x0003 = 程式、0x0103 = 資料
```

**30 個符號**（不是先前記的 49——那是把帶雜訊的字串也數進去了）。
音效相關的三支都在**模組 0x23**：

| 符號 | 模組:位移 |
|---|---|
| `_SoundOff` | `0x23:0x0194` |
| `_RemoveSound` | `0x23:0x01DE` |
| `_InitSounds` | `0x23:0x02E4` |
| `_ReadConfig` | `0x23:0x065E` |

⚠ **模組編號還沒對應到解壓後映像的位址**，所以還讀不到那三支的內容。

試過一條捷徑，**失敗了，記在這裡免得下次重試**：假設模組編號就是相對於
載入基底的節區值（線性位址 ＝ 模組×16 ＋ 位移），拿 22 個程式符號去對
解壓輸出裡的 `55 8B EC` 函式序言，找一個一致的位移量。結果三個解壓起點
的最佳位移分別只有 3、1、1 票——那是雜訊，不是訊號。

**原因是解壓品質，不是假設錯**：長串流解出來 189 KB 裡只有 29 個函式序言，
真正的 C 程式應該有數百個。GetPValue 那一段能用是因為它靠前、而且有三份
交叉檢查（[`18-dos-parity.md`](18-dos-parity.md) §6.3）；整份映像不能用。

所以下一步不是「換算位址」，而是**先把容器結構解對**：載入器的區塊鏈
（`word[2C]` ＝ 6 個區塊、每個 `[長度][?]` 開頭、由 `dword[38]` 走鏈）
還沒解出來，現在的工具是拿兩個已知起點硬解，跨區塊之後就是垃圾。

排掉的一個誤會：那些 `_InitSounds`、`_MoveObjects` 的名字**不在遊戲的
程式碼裡**，而在載入器帶的這份表裡——所以「檔案裡看得到符號名」
不代表程式碼是明文的。
