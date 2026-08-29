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

所以**音效不接進遊戲**。要解開，需要下列其中一項：

- 一份帶 `tdy\` 圖形檔的 1.10 副本（Tandy 畫面 ＋ Tandy DAC 同時成立）；
- 一個會模擬 Covox Sound Master 的環境；
- 反組譯 `SIMCITY.EXE` 的發聲程式。⚠ 這比想像中麻煩：進入點在
  `CS:0x1ea0`（檔尾附近）、重定位表是空的，而整個檔案裡找不到任何一個
  操作元合理的 `out 40h/42h`——真正的程式碼是打包的，要先脫殼。
  好消息是檔案裡有一份**明文的執行期符號表**（49 個名字，含
  `_InitSounds`、`_SoundOff`、`_SoundWait`、`_RemoveSound`、`_soundMode`、
  `_DoBudget`、`_MoveObjects`、`_Randomize`），脫殼後可以直接對上函式。
