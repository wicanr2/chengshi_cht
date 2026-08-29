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
| `machine=tandy` `Sound: T` `TDY_FROM=sega` | 卡在對話框過不去 | 0 |
| 同上 ＋ 加長等待、Return 與點擊各送兩輪 | 游標確實落在按鈕上了，對話框還是不關；**而且對話框後面那片地圖是純灰的——城市根本沒生成** | 0 |

最後那一列是決定性的：問題**不只是「文字讀不出來」**。
替代圖形能讓程式跑起來、把視窗框畫出來，但遊戲本身沒有進入可玩狀態
（地圖沒畫、對話框關不掉）。**換一套圖形集不足以讓 Tandy 模式可用。**

**四次 Tandy 實驗，四次各自因為不同的理由失敗，沒有一次是因為聲音。**
靜音到目前為止仍然什麼都證明不了
（`~/diagnosis-notes/docs/03-silence-is-not-success/`）。

所以缺的東西沒有變：**一份帶真正 `tdy\` 圖形檔的 1.10 副本**。
有了它，畫面讀得出來、驅動得動，Tandy DAC 那條路才問得出答案。

### 4.3 換一條路：拿 X11 版的具名音效對長度——**不行**

X11 版（`Rare simcity.zip`）的 `res/` 底下有 **46 個具名 `.au`**
（`expl-hi`、`siren`、`bulldoze`、`monster`…，8000 Hz、8 位元 µ-law）。
想法是：DOS 的八段如果就是其中八個的另一份編碼，那
「DOS 取樣數 ÷ au 取樣數」在八段上應該是**同一個常數**（＝取樣率比），
這樣既解出對應、也順便解出 DOS 的取樣率。

**量下來不成立。** 基本檔八段最多只有四段能落在同一個比值上，
而且每一段都對得上好幾個候選（長度相近的 `.au` 太多）。

原因清楚：**基本檔的段長全部是 64 的倍數**——那是補齊，不是錄音長度
（[`../formats/05-psf-sound.md`](../formats/05-psf-sound.md) §二）。
風格檔的段長沒補齊，但單靠長度在 46 個候選裡仍然分不出來。

### 4.4 比波形包絡——做了，**也認不出來**

[`tools/sound_envelope_match.py`](../../tools/sound_envelope_match.py)：
兩邊都轉成 48 點的能量包絡（DOS 的 4 位元 PCM 取 `|x−8|`、`.au` 的 µ-law
解碼後取絕對值），正規化之後算相關係數。包絡先攤成固定點數，長度與
取樣率就被除掉了，剩下形狀。

基本檔（`SOUNDDAT.V4`）的結果：

| 段 | 前三名 |
|---|---|
| 0 | traffic 0.76、uhuh 0.40、coal 0.34 |
| 1 | expl-low 0.75、expl-hi 0.75、road 0.52 |
| 2 | oop 0.50、sorry 0.44、wire 0.42 |
| 3 | expl-low 0.79、expl-hi 0.78、road 0.67 |
| 4 | honk-hi 0.94、honk-low 0.94、honk-med 0.94 |
| 5 | o 0.51、airport 0.33、build 0.33 |
| 6 | monster 0.31、build 0.05、honk-med −0.00 |
| 7 | whip 0.54、aaah 0.31、res 0.28 |

**認不出來，理由有三個：**

1. **同分。** 段 4 的三個 honk 都是 0.94——包絡分不出它們，而那正是
   「哪一段是哪個事件」要分的粒度。
2. **跨風格不一致。** 同一個段號在不同風格檔對到不同的名字
   （段 2：基本檔 oop、`MEDI` monster 0.87、`ASIA` oop 0.82）。
   不同風格本來就是不同錄音，所以這不算矛盾——但也就等於**沒有資訊**。
3. **相關值太低。** 除了段 4，最高只有 0.79。48 點包絡在 46 個候選裡
   0.8 上下的相關到處都是。

一個附帶確認（與長度表對得起來）：`ASIA`／`MEDI`／`WEST` 的**段 3–7 完全
相同**（相關值逐位一模一樣），`base`／`MOON` 是另一組。

**所以還是回到同一個結論**：要答「哪一段是哪個事件」得看 DOS 程式碼的
呼叫點，而那卡在容器結構（[`18-dos-parity.md`](18-dos-parity.md) §6.5）。

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

容器本身**已經解對了**（[`18-dos-parity.md`](18-dos-parity.md) §6.5）：
六塊、175 011 位元組、607 個函式序言，逐塊對上自己宣稱的長度。
所以現在缺的只剩「模組編號 → 映像位址」這一層。

試過的捷徑：假設模組編號就是相對載入基底的節區值（線性 ＝ 模組×16 ＋ 位移），
拿 22 個程式符號去對映像裡的 `55 8B EC` 找一致的位移量。**不成立**
（最好的位移只有 4 票／22）。合理——六塊各自載到不同的節區，
不是一段連續映像，所以位移不會是單一常數。

**已經排掉的**：

- 「模組編號 ＝ 相對載入基底的節區值」——四種變體（線性 ＝ 模組×16＋位移、
  找一致的位移量、假設符號指向 `EA`／`E9`／`9A` 開頭的 thunk、
  用同一模組內多個符號的**相對**距離去找基底）全部不成立。
- 想從字串位移反推資料節區基底再回推程式碼位置：四個音效字串的位移確實
  在一個 0x600 的視窗裡「同時出現」，但**回頭逐位元組看，那些位置根本
  不是立即數**（`0x18343` 那兩個位元組是 `d3 e8` ＝ `shr ax,cl`）。
  巧合，不是訊號。

**目前能說的**：模組編號的值域是 `0x01`–`0x68` 這種小數字，
所以它**是模組表的索引，不是節區值**。要往下走得先找到那張模組表。

順帶記一個工具面的坑：把解開的映像丟進 IDA 要**先把節區設成 16 位元**
（`ida_segment.set_segm_addressing(seg, 0)`），否則 raw binary 預設 32 位元
定址，`create_insn` 會把整段解成 `dq`。做法在
[`tools/ida/dump16.py`](../../tools/ida/dump16.py)。

排掉的一個誤會：那些 `_InitSounds`、`_MoveObjects` 的名字**不在遊戲的
程式碼裡**，而在載入器帶的這份表裡——所以「檔案裡看得到符號名」
不代表程式碼是明文的。


## 六、執行檔裡的硬編碼字串——中文化的第三個來源

`CLAUDE.md` §3.2 列了三個文字來源：`.PTF` 訊息檔、X11 版的 Tcl 腳本、
**DOS 執行檔內硬編碼的選單／按鈕／數字格式**。第三個一直拿不到，
因為執行檔是打包的。**現在解得開了**
（[`18-dos-parity.md`](18-dos-parity.md) §6.5）：

```bash
python3 tools/unpack_simcity_exe.py "…/SIMCITY.EXE" workplace/ida/image.bin
python3 tools/dos_exe_strings.py workplace/ida/image.bin
```

**497 條**。抽樣：

| 類別 | 例 |
|---|---|
| 對話框按鈕 | `Continue`、`Cancel`、`Retry`、`%s: Are you sure?` |
| 列印 | `Print City`、` 8-page poster`、` 1-page map`、`Printing %s`、`  0%% done`、`Press [ESC] to abort`、`Abort printing` |
| 存讀檔 | `Save changes before loading` ／ `another city?`、`SIMCITY city name:` |
| 開新城市 | `Game Play Level`、`Now terraforming`、`HERESVILLE`（預設城市名）|
| 需求／評估的程度詞 | `Sparse`、`Medium`、`High!`、`Little`、`Severe`、`Rapid` |
| 圖形／裝置 | `Tandy`、`Hercules`、`Mono EGA`、`MCGA/VGA Color`、`MCGA/VGA mono`、`Classic`、`Loading %s graphics` |
| 錯誤 | `FATAL ERROR: PROGRAM ABORTED`、`Cannot open graphics file:%s`、`Not enough memory to load graphics`、`No parallel port found. Can't print` |
| 檔名樣板 | `sounddat.psf`、`message.ptf`、`%sntro.ppf`、`%sscen.ppf`、`monodat.pgf`、`tdydat.pgf` |
| 製作名單 | `SimCity the city simulator, E1.10`、`Concept & design:Will Wright`、`IBM programming: Daniel Goldman`、`IBM artwork:     Don Bayless` |
| 編譯器 | `MS Run-Time Library - Copyright (c) 1990, Microsoft Corp` ← **這份是 Microsoft C 6.0 編的** |

三件順帶確認的事：

- `tdydat.pgf` 在字串表裡——**Tandy 的圖形檔名規則與其他模式一致**，
  這份副本只是沒附那個目錄（§4.1）。
- 那些 `%s`／`%d`／`%c` 樣板就是 §3.2 說的「模板要能重排參數順序」的實例，
  中文語序不同，不能寫死成前綴＋數字＋後綴。
- 製作名單裡的 `E1.10` 與檔案版本對得上。

⚠ 位移是**解壓後映像**的線性位移，不是檔案位移，也不是執行時的
segment:offset。引用時要連同位移與映像來源（SHA-256 `66457cc4…`）一起寫。
