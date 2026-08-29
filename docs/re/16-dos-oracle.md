# 16 — DOS 原版當 oracle

Micropolis 回答得了「規則怎麼算」，回答不了「DOS 版按下去會怎樣」。
有些問題只有把 1991 年那支執行檔跑起來才有答案：八段音效各對應哪個事件、
防拷有沒有被拔掉、螢幕模式怎麼對到圖形檔的目錄。

工具：[`tools/dosbox.sh`](../../tools/dosbox.sh)（外層）
＋ [`tools/dosbox_inner.sh`](../../tools/dosbox_inner.sh)（容器內）
＋ [`docker/dosbox.Dockerfile`](../../docker/dosbox.Dockerfile)。

## 一、怎麼跑

```bash
docker build -f docker/dosbox.Dockerfile -t simcity-dosbox:0.74 docker/
RUN=simcity ACTIONS="$PWD/tools/dosbox/act-disasters.txt" tools/dosbox.sh 8 dis
```

產物在 `workplace/dosbox/`：截圖、`.raw`（s16le 立體聲 22050 Hz 的全程錄音）、
`.marks`（每個動作的時間戳）、`.log`。

動作腳本一行一個動作：`key`／`click`／`drag`／`press`／`move`／`release`／
`wait`／`shot`／`mark`。**選單是按住式的**——`click` 打不開，要
`press` 在標題上、`move` 到項目、再 `release`。

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

## 四、音效：這條路目前走不通，原因很具體

`SIMCITY.CFG` 的 `Sound` 欄有四個值：`I` 內建 PC 喇叭、`S` Covox（LPT 上的
DAC）、`T` Tandy、`N` 無聲。

| 設定 | 結果 |
|---|---|
| `I` ＋ DOSBox 0.74 | 只錄得到**單頻方波**。實測某次事件是 0.2 秒內過零 462 次（約 1155 Hz），取樣值只有 ±5000 兩種——那是 DOSBox `pcspeaker` 的固定音量常數，不是遊戲的取樣 |
| `S` ＋ `disney=true` | **完全無聲**。DOSBox 0.74 的 disney 裝置只認 Disney Sound Source 的 FIFO 交握，Covox 是直接對 LPT 資料埠寫值，它不接 |
| `T` ＋ `machine=tandy` | 開不起來：Tandy 模式要 `tdy\` 目錄的圖形檔，這份副本沒有 |

還有一個會騙人的地方：**遊戲放完聲音之後，DOSBox 0.74 的喇叭通道會停在
一個固定電位**，音量看起來很大但完全沒有內容。用音量當門檻切事件，量出來
每一段都剛好等於兩個動作之間的間隔——一個看起來很合理、其實是假的數字。
判準要用**每 5 毫秒的標準差**（`tools/snd_events.py` 就是這樣做的）。

所以四位元 PCM 那八段**沒有真的被播出來過**，錄到的只有簡單的嗶聲。
要對出「哪一段是哪個事件」，下一步是換一個**支援 Covox／LPT DAC 的
DOSBox**（DOSBox-X 或 DOSBox Staging，Debian bookworm 的套件庫沒有，
要自己編）。在那之前音效不接進遊戲。

## 五、順帶確認的事

- 災難選單觸發得了，而且**風格會改寫災難名稱**：西部拓荒版的怪獸叫
  `Tumbleweed`、空難叫 `Balloon Crash`。跳出來的整頁訊息與
  [`04-ptf-messages.md`](../formats/04-ptf-messages.md) 第 2 段解出來的
  文字對得上——兩條獨立的證據互相印證。
- 這份副本的 `SIMCITY.CFG` 預設是 `WESTCEGA`，所以直接跑起來是**西部拓荒**
  風格，不是基本風格。
