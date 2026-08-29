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

### 為什麼還是對不出「哪一段」

拿八段參考音效的包絡線去比（`tools/snd_ident.py`），只有一個對得起來：
**0.115 秒那個聲音對到第 5 段，相關係數 0.90–0.95**，而且要假設取樣率
在 8000 上下（第 5 段 926 個取樣 ÷ 0.115 秒 ≈ 7700）。

其餘三個對不上任何一段（相關係數掉到 0.25），而且長度湊不出一個一致的
取樣率：

- 0.115 秒 ↔ 第 5 段 ⇒ 約 7700 Hz
- 1.90 秒的最長候選是第 0 段（11 526 取樣）⇒ 約 6070 Hz
- 0.279 秒在這兩個取樣率下都落在兩段之間

⚠ 差異是系統性的**偏短**，形狀像是 DOSBox 的喇叭通道把尾巴截掉，
不像是取樣率猜錯。要分辨這兩種可能，得有一個會完整播完的發聲路徑——
而三條路（Covox 卡沒人模擬、Tandy 缺圖形檔、PC 喇叭被截）目前都不是。

所以**音效不接進遊戲**。剩下的路有兩條：找一份帶 `tdy\` 圖形檔的
1.10 副本走 Tandy DAC，或反組譯 `SIMCITY.EXE` 的發聲程式直接讀出
索引與取樣率。

### 兩個會騙人的地方

- **SDL 的 disk 音訊驅動要節流**。`SDL_DISKAUDIODELAY` 沒對上
  `blocksize/rate`（1024 @ 22050 ≈ 46 ms）的話，callback 會被用最快速度
  呼叫，25 秒錄出 15 分鐘的檔案。
- **DOSBox 放完聲音會把喇叭停在一個固定電位**，音量看起來很大但完全沒有
  內容。用音量當門檻切事件，量出來每一段都剛好等於兩個動作之間的間隔
  ——一個看起來很合理、其實是假的數字。判準要用每 5 毫秒的**標準差**。
- **錄音的時間軸和動作的時間戳差一個固定偏移**（實測約 2 秒），
  對事件時要先用幾個明確的動作把偏移量校出來，不能直接相減。

## 五、順帶確認的事

- 災難選單觸發得了，而且**風格會改寫災難名稱**：西部拓荒版的怪獸叫
  `Tumbleweed`、空難叫 `Balloon Crash`。跳出來的整頁訊息與
  [`04-ptf-messages.md`](../formats/04-ptf-messages.md) 第 2 段解出來的
  文字對得上——兩條獨立的證據互相印證。
- 這份副本的 `SIMCITY.CFG` 預設是 `WESTCEGA`，所以直接跑起來是**西部拓荒**
  風格，不是基本風格。
