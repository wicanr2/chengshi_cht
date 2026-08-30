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

## 四、音效：模擬器問得出「哪個動作有聲音」，問不出「那是哪一段」

> 「哪一段是哪個事件」已經解出來了，答案在 §五之四，走的是反組譯呼叫端。
> 本節保留的是**模擬器這條路能問到什麼、問不到什麼**——那個界線沒有變。

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

**結論**：要答「哪一段是哪個事件」得看 DOS 程式碼的呼叫點。
那條路走通了（§五 脫殼 → §五之二 發聲常式 → §五之四 八段對應）。

模擬器這一側仍然聽不到這八段 PCM，而且原因與素材無關：

- `T` Tandy 走的是 `INT 1Ah AH=83h`（Tandy 音效 BIOS），DOSBox-X 沒實作，
  **拿到帶 `tdy\` 圖形檔的副本也一樣**（§五之二）；
- `S` Covox Sound Master 沒有模擬器支援。

所以「用耳朵驗收」這件事目前做不到。剩下能驗的是**結構**：
段長、對齊、4 位元 nibble 順序、以及呼叫端條件——都已經驗過。

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


## 五之二、發聲常式找到了——而且 Tandy 那條路在模擬器裡走不通

**推論等級：已確認**（讀到解開後映像的反組譯，位址是映像線性位移）。

發聲常式在 **0xC9EB**（`push bp / mov bp,sp / sub sp,6 / push di`，far 函式）。
參數：`[bp+6]` ＝ 指向 4 位元取樣資料的 far 指標、`[bp+0Ah]` ＝ 位元組長度。

它先配置 `2 × 長度` 的緩衝，把**每個位元組的兩個 nibble 攤成兩個位元組**，
nibble 放在高四位：

```
lodsb / mov ah,al / and al,0F0h / stosb          ; 高 nibble → 一個位元組
shl ah,1 ×4 / mov al,ah / stosb                  ; 低 nibble → 下一個位元組
```

**這獨立確認了 `.PSF` 是 4 位元取樣、高位 nibble 在前**
（[`../formats/05-psf-sound.md`](../formats/05-psf-sound.md) §三 原本是用
「相鄰取樣平均絕對差」量出來的，現在有程式碼佐證）。

接著依 `byte_29A7`（音效裝置編號）分三條路：

| 條件 | 走哪裡 |
|---|---|
| `== 3` | **DAC**：`sub_D5A8`／`sub_D956`／`sub_D53D`，把攤開的緩衝送出去 |
| 否則其一 | `mov dx,300h / mov ax,8307h / int 1Ah` ← **Tandy 音效 BIOS** |
| 否則 | PC 喇叭 PWM（`out 43h/42h/61h`，`in al,61h`）|

### 這解釋了四次 Tandy 實驗為什麼全是靜音

**Tandy 那條路不是寫 I/O 埠，是呼叫 `INT 1Ah` 的 `AH=83h`**——那是
Tandy／PCjr 的音效 BIOS 服務。DOSBox-X 的 `machine=tandy` **不提供這個
BIOS 服務**，所以呼叫直接落空，沒有任何埠寫入可錄。

這與執行檔裡那條 `Tandy sound BIOS not found.` 字串完全對得起來
（§六的字串表）。

**所以 Tandy 這條路在目前的模擬環境裡走不通，而且不是圖形檔的問題**
——就算拿到帶 `tdy\` 的副本也一樣。§4.1–4.2 那四次實驗的結論要照這個
修正：缺的不只是圖形檔，是**會實作 Tandy 音效 BIOS 的模擬器**。

### 五之三、far call 搜得到——`PlaySound(n)` 的 n 就是段編號

「節區值載入時才填」這個判斷**是錯的**。解開的映像就是載入時的記憶體
影像，而 `9A off seg` 裡的 `seg` 是**相對載入節區**的值，所以

```
目標線性位址 ＝ seg × 16 ＋ off
```

驗證：全映像 **2118 個 far call、405 個相異目標，其中 181 個正好落在
`55 8B EC` 函式序言上**——遠高於隨機命中。工具：
[`tools/dos_farcalls.py`](../../tools/dos_farcalls.py)。

於是：**0xC9EB 只有一個呼叫者，0xCF51**，而它就是 `PlaySound(n)`：

```
0CF51  push bp / mov bp,sp / push di
0CF55  mov  es, <音效開關的節區>
0CF59  cmp  byte ptr es:29B4h, 0     ; 音效關掉就直接 return
0CF61  cmp  word ptr [bp+6], 8       ; ⭐ 參數 ≥ 8 就 return
0CF65  jge  ret                      ;   → **n 的值域是 0–7，就是段編號**
0CF6B  mov  bx,[bp+6] / shl bx,1
0CF70  mov  di, es:[bx-759Ch]        ; 八個 word 的**長度表**
0CF75  or   di,di / jz ret           ; 長度 0 不放
0CF79  push di                       ; 參數 3：長度
0CF7E  mov  bx,[bp+6] / shl bx,1 / shl bx,1
0CF85  push word ptr es:[bx-781Eh]   ; 八個 far pointer 的**資料表**：segment
0CF8A  push word ptr es:[bx-7820h]   ;                              offset
0CF8F  call far 0xC9EB               ; PlaySample(far *data, len)
```

### 五之四、十個呼叫點——八段全部認出來了

`python3 tools/dos_farcalls.py workplace/ida/image16.bin 0xCF51`：

| 呼叫點 | n | 上下文 |
|---|---:|---|
| `0x0EB98` | 7 | 工具函式：`sub_1686F` 回 1 → 播 7 |
| `0x0EBAE` | 7 | 同一支：另一條失敗路徑 |
| `0x0EFED` | 7 | 工具主流程：`di != 1`（失敗）且 `[bp-2] == 0` → 播 7 |
| `0x0EFD6` | 6 | 工具主流程：`di == 1`（成功）且 `es:2B50h > 4` → 播 6 |
| `0x0EFDE` | 5 | 同上，`es:2B50h ≤ 4` → 播 5 |
| `0x1517A` | 3 | 訊息派送 `0x14F8F`：訊息屬性 ∈ {6, 7} 且訊息編號 ≠ 32 |
| `0x1E316` | 4 | `DoShipSprite` `0x1E2FD` |
| `0x1E74D` | 2 | `DoMonsterSprite` `0x1E5DE` |
| `0x1EC4F` | 1 | `DoExplosionSprite` `0x1EC32`（`frame == 1`）|
| `0x1F8C4` | 1 | `ExplodeSprite` `0x1F808` |
| `0x1EE62` | **0** | `DoCopterSprite` `0x1ED62`。常數是 `sub ax, ax` 壓的，所以 `mov ax,imm` 的掃描看不到它 |

> 最後一列是這一輪最實際的教訓：先前寫「段 0 十個呼叫點都沒用到」，
> 而它其實在表上——只是掃描器只認 `mov ax,imm/push ax`。
> **「掃描沒掃到」要先懷疑掃描器的形狀假設，再懷疑目標不存在。**

#### 八段對應表

**推論等級：已確認**（每一段都讀到呼叫端所在函式的完整反組譯，
並與 Micropolis 的對應函式逐項比對）。

| 段 | 事件 | DOS 函式 | 對得上的 Micropolis |
|---:|---|---|---|
| 0 | 交通壅塞（直升機回報）| `0x1ED62` | `w_sprite.c:768` `HeavyTraffic` |
| 1 | 爆炸 | `0x1EC32`、`0x1F808` | `w_sprite.c:1123`／`:1410` `Explosion-High` |
| 2 | 怪獸吼叫 | `0x1E5DE` | `w_sprite.c:1005` `Monster` |
| 3 | 警笛 | `0x14F8F` | `s_msg.c:331–354` `Siren` |
| 4 | 船笛 | `0x1E2FD` | `w_sprite.c:854` `HonkHonk-Low` |
| 5 | 工具成功，工具編號 ≤ 4 | `0x0EFDE` | 無（X11 版的成功音在介面層）|
| 6 | 工具成功，工具編號 > 4 | `0x0EFD6` | 無 |
| 7 | 工具失敗 | `0x0EB98`／`0x0EBAE`／`0x0EFED` | `w_tool.c:1554/1558` `UhUh`／`Sorry` |

#### 五個逐項對上的結構

認定不是靠名字像，是靠**同一段邏輯的數字逐項對上**：

| DOS | Micropolis | 對上的東西 |
|---|---|---|
| `0x1E5DE`：`d = (frame−1)/3`、`z = (frame−1)%3`、`d < 4`、兩張 4 筆表相隔 8 位元組（`4ED8h`／`4EE0h`）、`Rand(10)`、`Rand(1)` | `DoMonsterSprite` `w_sprite.c:990–1010`：同樣的除法、`ND1[4]={0,1,2,3}`／`ND2[4]={1,2,3,0}`、`!Rand(10)`、`Rand16()&1` | 逐行 |
| `0x1EC32`：`frame == 1` → 播音 → `SendMesAt(32, x, y)`；`frame` 超過就在五個位移點放火 | `DoExplosionSprite` `w_sprite.c:1117–1141`：`SendMesAt(32, …)`、五次 `StartFire` 位移 `(−8,+16) (−24,0) (+8,0) (−24,+32) (+8,+32)` | 訊息編號 ＋ 五個位移 |
| `0x1F808`：型別 3→24、4→25、1→26、2→27，然後播音 | `ExplodeSprite` `w_sprite.c:1383–1401`：`AIR`→−24、`SHI`→−25、`TRA`→−26、`COP`→−27 | **精靈型別編號與訊息編號兩邊都一樣** |
| `0x1ED62`：`x/2B52h/2`、界限 59／49、`bx = x*50 + y`、`TrfDensity[bx] > 0xA0`、`SendMesAt(41, …)` | `DoCopterSprite` `w_sprite.c:759–770`：`x>>5`、`WORLD_X>>1`、`TrfDensity[x][y] > 170`、`SendMesAt(−41, …)` | 索引算式 ＋ 門檻。**門檻 DOS 是 160、Micropolis 是 170**——正好是那句註解 `Don changed from 160 to 170 to shut the #$%#$% thing up!` 講的改動，所以 DOS 保留的是原值 |
| `0x1E2FD`：`z = (pem & 7) + 1`、`t = Map[x*100 + y] & 3FFh`、兩張 9 筆位移表 | `DoShipSprite` `w_sprite.c:864–872` | 方向算式 ＋ `LOMASK` |

#### 段 3 的判準來自資料檔，不是硬編碼表

`0x14F8F` 是訊息派送常式。它拿訊息編號去查一張 **6 位元組**的記錄
（`{char far *text; u16 attr;}`），取出 `attr`，然後：

```
if ((attr == 6 || attr == 7) && firstTime && msg != 32) PlaySound(3);
```

那個 `attr` 就是 `.PTF` 第 0 段每一筆後面那兩個位元組的第二個
（[`../formats/04-ptf-messages.md`](../formats/04-ptf-messages.md) §二，
原本記成「疑似音效或嚴重度」）。從 `DATA/MESSAGE.PTF` 讀出來：

| attr | 訊息 |
|---:|---|
| 6 | 火警、怪獸、龍捲風、空難、船難、火車事故、直升機事故、**爆炸**、水災、爐心熔毀 |
| 7 | 大地震 |
| 8 | 交通壅塞、`Cannot bulldoze here.` |
| 9 | 錢不夠、要先推平、不能蓋在水上、這裡不能蓋 |
| 2–5 | 建議與里程碑（催分區、催路、停電、人口達標…）|

爆炸（訊息 32）被特例排除，因為爆炸精靈自己已經播了段 1；
不排除就會爆炸聲疊警笛。

**這個特例反過來釘死了記錄對齊。** `.PTF` 第 0 段是「文字、屬性」交替，
而屬性到底屬於前一筆還是後一筆，光看檔案分不出來——兩種讀法都自洽。
程式碼分得出來：只有「屬性屬於前一筆」時，訊息 32 才是 attr 6，
那行特例才有意義。

#### 包絡比對事後看是對的，只是當時不能定案

§4.4 那張表現在可以回頭判分。三段的第一名就是答案：

| 段 | 答案 | §4.4 的第一名 |
|---:|---|---|
| 0 | 交通壅塞 | `traffic` 0.76（第二名 0.40，差距大）|
| 1 | 爆炸 | `expl-low` 0.75／`expl-hi` 0.75 並列 |
| 4 | 船笛 | `honk-hi`／`honk-low`／`honk-med` 三個並列 0.94 |
| 2 | 怪獸 | 基本檔錯（`oop`），但 `MEDI` 檔第一名是 `monster` 0.87 |
| 3 | 警笛 | **錯**（`expl-low` 0.79，`siren` 沒進前三）|

> 教訓寫成規則：**包絡比對能排序，不能定案。** 五段裡三段第一名正確，
> 但當時無從分辨哪三段是對的——「並列」與「跨檔不一致」把訊號吃掉了。
> 需要定案的問題要找能一次釘死的證據（這裡是呼叫端），
> 排序法只能拿來事後交叉檢查。
## 五之五、取樣率：量得出約 5.4 kHz，還沒從程式碼直讀

八段的事件對應定了之後，§4.3 那條「用長度比反推取樣率」的路重新成立——
當時失敗是因為不知道哪一段對哪一個 `.au`，46 個候選裡怎麼配都湊得出比值。
現在配對是已知的，比值只剩一個自由度。

`SOUNDDAT.V4` 的取樣數（段長 × 2）除以 X11 對應 `.au` 的取樣數
（8000 Hz µ-law）：

| 段 | 事件 | DOS 取樣 | `.au` | `.au` 取樣 | 比值 | 推得取樣率 |
|---:|---|---:|---|---:|---:|---:|
| 0 | 交通壅塞 | 11264 | `traffic` | 16616 | 0.678 | 5423 Hz |
| 1 | 爆炸 | 8704 | `expl-hi` | 12858 | 0.677 | 5415 Hz |
| 4 | 船笛 | 5120 | `honk-low` | 7603 | 0.673 | 5387 Hz |
| 3 | 警笛 | 13440 | `siren` | 18719 | 0.718 | 5744 Hz |
| 2 | 怪獸 | 10752 | `monster` | 5918 | 1.817 | — |

**推論等級：強證據。** 三段落在 0.673–0.678（彼此差 0.8% 以內），
指向 **5400 Hz 上下**。這三段正好也是 §4.4 包絡比對第一名正確的三段。

⚠ 但**這三個數字不是三份獨立證據**——它們共用「兩邊是同一份素材」這個前提，
而下一節的頻譜比對顯示那個前提只有段 1 成立。

不採信另外兩段，理由分別寫清楚：

- 段 2（怪獸）**本來就不會是同一份錄音**：`.PSF` 第 2 段每個風格包都不一樣
  （[`../formats/05-psf-sound.md`](../formats/05-psf-sound.md) §四），
  古代的怪獸和未來的怪獸不同聲音，DOS 基本檔與 X11 沒有理由共用素材。
- 段 3（警笛）差 6%。警笛是可以任意剪長剪短的循環音，長度不構成證據。

⚠ 三個數字的不確定度不只 0.8%：`SOUNDDAT.V4` 的段長**補齊到 64 的倍數**
（同上 §二），最多可能多算 126 個取樣。對段 4（5120 取樣）那是 2.5%。
所以合理的區間是 **5300–5450 Hz**，不是「5408 Hz」。


### 第二個方法：頻譜形狀。同一份錄音只有段 1

長度比有一個沒被檢驗的前提：**兩邊是同一份素材，而且剪裁一致。**
段長補齊到 64 的倍數已經算進不確定度了，但「是不是同一份素材」沒有驗過——
§4.4 的包絡比對只能排序，不能定案（同上）。

頻譜形狀可以驗。同一份錄音用不同取樣率播，頻譜會沿頻率軸整個縮放：

```
ν_dos = ν_x11 × 8000 / R
```

所以把 X11 的對數頻譜沿頻率軸拉伸一個倍率去對 DOS 的，相關係數最高的倍率
就給出 R。這個方法**完全不受剪裁影響**——它敏感的誤差來源跟長度比不重疊，
不是把同一件事再算一次。

工具是 [`tools/snd_rate_fit.py`](../../tools/snd_rate_fit.py)（純標準庫）。
**先做正對照**：把 X11 那份人工降到已知取樣率、加上 DOS 那種 4 位元量化，
餵給同一支估計器，看它還不還得回真值。

```
真值 4800 → 估 4800（r=0.984）    真值 6000 → 估 6000（r=0.991）
真值 5400 → 估 5395（r=0.989）    真值 7000 → 估 7005（r=0.990）
```

四個真值都還原到 0.1% 內，峰值 0.01 內的區間寬約 ±1%。**尺會量。**

拿去量八段 × 46 個 `.au` 的全部組合，只有一格超過 0.9：

| 段 | `.au` | 相關係數 | 估得取樣率 | 峰值 0.01 內 |
|---:|---|---:|---:|---|
| 1 | `expl-hi` | **0.966** | **5365 Hz** | 5320–5410 |

第二名是段 1 對 `road` 的 0.879，而且落在 7355 Hz——不是同一個答案，
只是寬頻噪音彼此都有點像。段 1 那一格是孤立的尖峰。

**所以取樣率有兩個對不同誤差敏感的方法給出同一個答案**：
長度比 5415 Hz、頻譜形狀 5365 Hz，差 0.9%。這比原本「三段長度比互相印證」強，
因為那三段其實不是三份獨立證據（見下）。

#### 段 0、3、4 跟 X11 不是同一份波形

同一支估計器對其他段全部落空，而且不是因為方法不適用——**逐對做過正對照**：

| 段 ↔ `.au` | 正對照（人工降到 5400） | 實測 |
|---|---|---|
| 0 ↔ `traffic` | 5385 Hz，r=0.981 | 3000 Hz，r=0.155 |
| 1 ↔ `expl-hi` | 5395 Hz，r=0.989 | **5365 Hz，r=0.966** |
| 3 ↔ `siren` | 5405 Hz，r=0.983 | 3890 Hz，r=0.065 |
| 4 ↔ `honk-low` | 5395 Hz，r=0.914 | 3135 Hz，r=0.301 |

把掃描範圍拉到 800–24000 Hz，段 0／3／4 在**全域都沒有峰**（r 始終 ≤ 0.33）。
也試過「DOS 那邊是最近鄰抽取樣、沒有抗鋸齒，所以高頻摺回來把形狀吃掉」這個
假設——正對照打掉了它，最近鄰抽取樣照樣還原（r=0.915–0.986）。

所以段 0、3、4 與 X11 同名檔**長度對得上、包絡像、波形不同**。
合理的解釋是兩邊出自同一套音效母帶、剪在同樣的長度，但 DOS 這邊另外算過一次。

**這不動搖事件對應**（那是從 `PlaySound(n)` 的呼叫點釘死的，見上），
但它改寫長度比那三個數字的分量：它們證明的是**兩份發行的素材共用同一組時間長度**，
不是「同一份錄音」。長度比仍然給得出取樣率（同樣長度的聲音，取樣數比 ＝ 取樣率比），
只是「三段互相印證」要降級成「一段有波形佐證、兩段只有長度」。

> 教訓寫成規則：**幾個數字彼此吻合，不等於它們是幾份獨立證據。**
> 先問它們是不是共用同一個沒被檢驗的前提——這裡共用的是「同一份素材」，
> 而那個前提只有其中一段成立。

### 為什麼還沒從程式碼直讀

發聲那一路（`byte_29A7 == 3`）走的是 `sub_D53D`／`sub_D5A8`／`sub_D956`，
它們屬於映像裡 segment `24ABh` 的一套音效卡驅動。讀進去看到的是：

- `sub_D956(8, 8)` 把兩個 4 位元值併成一個位元組、以 `AH=0Eh` 送出——**音量**
  （左右各 0–15，上限存在 `word_24AF5`）。不是取樣率。
- `sub_D53D(far *buf, u32 len)` → `sub_D637`，而 `sub_D637` 那一帶寫的是
  `out 0Ah`（DMA 遮罩）、DMA page register、`out 21h`（PIC 遮罩）、
  `out 20h`（EOI），並掛 `int 15h AX=91F0h`（OS HOOK，等待 I/O 完成）。
  **這是一張 DMA 驅動的音效卡**，取樣率由卡上的計時器決定，
  透過 `sub_DD83`（`AH` ＝ 命令、`AL` ＝ 資料）設定。

所以取樣率的常數在**初始化**那一段，不在播放那一段。

初始化找到了，在映像 `0xCC20`：

```
mov ah,81h / int 1Ah        ; 問 Tandy 音效 BIOS
cmp ax,0C4h / jnz ...       ; 答得出來 → 走 Tandy，回傳裝置碼 2
...
call sub_D338               ; 裝 Sound Master 驅動
mov ax,14h / push ax / call sub_D622    ; ← 唯一一個速率型參數：20
mov ax,1  / push ax / call sub_D5A8
mov ax,0Ch / push ax / push ax / call sub_D956   ; 音量 12／12
mov ax,3 / retf             ; 回傳裝置碼 3
```

這同時解釋了 `PlaySample` 裡的 `byte_29A7`：**3 ＝ Sound Master、2 ＝ Tandy**，
值就是這支初始化的回傳碼。

`sub_D622(0x14)` → `sub_D79B`，而 `sub_D79B` 把 `dx` 拆成兩半，
以命令 `AH=04h`（低位元組）與 `AH=05h`（高位元組）送給卡：
**一個 16 位元參數，遊戲設 20。** 掃過整套驅動，遊戲只設三樣東西——
裝置腳位（`sub_D5D2`：1 → DMA 2／IRQ 3／page port 83h，3 → DMA 6／IRQ 7／page port 82h）、
音量、以及這個 20。**所以 20 就是取樣率那個參數**，剩下的是它的單位。

單位查不出來：那是第三方音效卡的命令集，手上沒有規格。
如果 20 是除數，配上量到的 5300–5450 Hz 反推卡上時脈約 108 kHz。
**這一步不要用猜的往下寫。** 下一個入口是 Covox Sound Master 的
程式設計文件，或一個會模擬它的環境。

另外兩件事順帶釘死：

- **整份執行檔只有 `out 43h`／`out 42h`／`out 61h` 兩處**
  （`0x7212`、`0xCAE4`），全部是計時器通道 2 ＝ PC 喇叭。
  **通道 0（`out 40h`）一次都沒有**——DAC 那條路不改系統計時器，
  所以「用 PIT 中斷餵取樣」這個假設可以排除。
- `PlaySample` 走 Tandy 那條路時送的是 `AX = 8307h`。
  `AL = 07h` 是待查的線索：Tandy 音效 BIOS 的 `AH=83h` 服務，
  `AL` 很可能就是取樣率代碼。**還沒查證，不要當結論用。**

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

## 七、原版顯示的年份比劇本自己的日期早 51 年（已確認）

`CONTEXT.md` §5.5 掛了很久的一項：**DOS 原版狀態列顯示的年份與存檔裡的
`CityTime` 對不上**。查清楚了。

### 量法

載入劇本 → **`Ctrl-C` 關掉 City Form 視窗**（不關的話它蓋住編輯視窗標題列
右半，日期看不到，而畫面上完全沒有提示）→ 截標題列。
腳本：[`../../tools/dosbox/act-year2.txt`](../../tools/dosbox/act-year2.txt)
（達斯維利）與 `act-year3.txt`（漢堡）。

### 四個資料點，公式吻合到個位數

| 城市 | `CityTime` | `÷48` | **DOS 標題列** | `1849+n` | `1900+n` | 劇本簡介與手冊 |
|---|---:|---:|---|---:|---:|---:|
| 新城市 | 0 | 0 | `1849`（評估視窗）| 1849 | 1900 | — |
| 達斯維利 | 2 | 0 | **`Feb 1849`** | 1849 | 1900 | 1900 |
| 漢堡 | 2114 | 44 | **`Jan 1893`** | 1893 | 1944 | 1944 |
| 東京 | 2739 | 57 | `Feb 1906` | 1906 | 1957 | 1957 |

（東京那一列是先前記錄的觀察，另外三列是這一輪實測。）

```
DOS 1.10 顯示的年份 = 1849 + CityTime / 48
本專案（照 Micropolis）  = 1900 + CityTime / 48
```

**差 51 年，而且是固定差。**

### 這是原版自己前後不一致，不是我們算錯

把八個劇本一起排開就看得很清楚：**`1900 + CityTime/48` 對上手冊與劇本簡介的
日期，八個全中；`1849 + CityTime/48` 一個都不中。**

| 劇本 | 簡介／手冊 | `1900+n` | DOS 狀態列 `1849+n` |
|---|---:|---:|---:|
| 達斯維利 | 1900 | 1900 ✓ | 1849 |
| 舊金山 | 1906 | 1906 ✓ | 1855 |
| 漢堡 | 1944 | 1944 ✓ | 1893 |
| 東京 | 1957 | 1957 ✓ | 1906 |
| 伯恩 | 1965 | 1965 ✓ | 1914 |
| 底特律 | 1972 | 1972 ✓ | 1921 |
| 波士頓 | 2010 | 2010 ✓ | 1959 |
| 里約 | 2047 | 2047 ✓ | 1996 |

所以原版的玩家會**先讀到「HAMBURG, GERMANY 1944」的劇本簡介，
按掉之後狀態列寫 `Jan 1893`**。那是原版自己的毛病。

### remake 怎麼做

**維持 `1900 + CityTime/48`，不重現這個偏移。** 理由：

- 劇本簡介（`.PTF` 第 2 段）與說明書都用 1900 基準，而簡介是玩家**幾秒前
  才讀過**的字。狀態列跟著錯會讓中文化的劇本簡介看起來像翻錯了。
- 存檔的 `CityTime` 本身就是照 1900 基準編碼的（`s_fileio.c:406-447`
  的劇本表，見 [`../spec/city-file.md`](../spec/city-file.md)）——
  1849 那個基準沒有出現在任何資料裡，只出現在畫面上。

⚠ **這是「不照原版」的少數幾處之一，所以要明寫。** 判準不是「哪個好看」，
是**原版的兩個輸出互相矛盾時，跟著資料檔走**（`CLAUDE.md` §1.1：
一手資料贏二手推論；這裡資料檔與手冊同一邊，畫面自己站另一邊）。

⚠ 偏移的**成因未解**：1900 − 1849 = 51，不是 50 也不是 48 的倍數，
看不出是哪一種常見的差一錯誤。不影響對拍（對拍讀的是 `CityTime` 欄位，
不是畫面上的字）。

### 追成因追到哪裡（卡點與下一個入口）

- **日期格式字串找到了**：`%3s %4d`，解壓映像的 `0x268E6`（月三字 ＋ 年四位，
  就是 `Feb 1849` 的版面）。月份名不在執行檔裡，出自 `.PTF` **第 4 段**。
- **卡在分段沒重建。** `workplace/ida/image16.bin.i64` 裡整個映像是**一個平坦的
  `seg000`、類別 CODE**（`tools/ida/segs.py` 印出來的），所以 IDA 解不出資料參照
  ——`%3s %4d` 與 `%d City Evaluation` 的 xref 都是空的。
  手算資料段基底也不成：把所有 `push imm16` 收集起來反推基底，
  沒有一個基底能同時對上兩個以上的字串。16 位元程式的字串參照不見得走
  `push imm16`，也可能是 `lea`／遠指標／`DS` 相對定址。
- **一條看起來很像但不可靠的線索**：`0x78E1` 有
  `mov word ptr [29A2], 076Ch`（＝**1900**），而 `0xC90F` 有 `add ax,[29A2]`，
  緊鄰一段除以 **48** 的碼（`mov ax,30h; cwd`）。合起來很像 `年 = CityTime/48 + 基準`。
  **但不能當結論**：`[29A2]` 另外有七處 `mov ax, es:[bx+3]; mov [29A2],ax`
  ——同一個位址被當成共用暫存寫來寫去，所以那個 1900 未必是年份基準。
- **下一個入口**：先把分段重建起來（找 `DS` 的實際值，或用 `unwrap.py`
  產生的中間檔對照原始 `.EXE` 的重定位表），字串 xref 一通，
  直接看誰把參數推給 `%3s %4d` 就結案。

## 八、「兩份 `SOUNDDAT.PSF` 讀哪一份」——DOSBox 這條路走不通（已確認）

`CONTEXT.md` §5.5 給這一項的下一步是「反組譯檔名字串，**或 DOSBox 追檔案開啟**」。
第二條路試過了，**結構上不可能成立**。

### 兩份差在哪

根目錄與 `DATA/` 各有一份 `SOUNDDAT.PSF`，**只差第 2 段（怪獸）**：

| 檔案 | 第 2 段 | 其餘七段 |
|---|---:|---|
| `SOUNDDAT.PSF`（根目錄，2012 重打包）| 3750 位元組 ／ 7500 取樣 | 相同 |
| `DATA/SOUNDDAT.PSF` | 5376 位元組 ／ 10752 取樣 | 相同 |
| `SOUNDDAT.V4`（未壓縮，手冊附錄有列）| 5376 ／ 10752 | 相同 |

所以 `DATA/` 那一份與 1991 年的 `.V4` 一致，根目錄那一份是後來換過的。

### 實驗與它為什麼失敗

給 [`../../tools/dosbox.sh`](../../tools/dosbox.sh) 加了 `PREP`：一段在**遊戲副本**上
跑的 shell，在 DOSBox 起來之前執行。用它把其中一份填成垃圾，看遊戲抱不抱怨。

| 情況 | 與基準的畫面差異 |
|---|---:|
| 弄壞根目錄的 `SOUNDDAT.PSF` | **0 像素** |
| 弄壞 `DATA/SOUNDDAT.PSF` | **0 像素** |
| **正對照**：弄壞 `DATA/MESSAGE.PTF` | **20 300 像素**（第一張截圖就看得出來）|

正對照證明這個方法**測得出壞掉的資料檔**，所以兩個零不是方法失靈。

原因在畫面上寫著（`workplace/dosbox/s-base-00-title.png`）：

```
Sound Master not found.
 Using internal speaker
```

**DOSBox 沒有 Covox Sound Master，遊戲退回內建喇叭**，而內建喇叭放的是程式自己
合成的嗶聲、不是那八段 PCM（§4 已經用「換掉音效檔再錄一次、聲音逐取樣相同」
證過）。**所以 `SOUNDDAT.PSF` 從頭到尾沒有被開啟過**，弄壞它當然沒有影響。

Tandy 那條路也一樣：`INT 1Ah AH=83h` 的 Tandy 音效 BIOS 在 DOSBox-X 裡沒有實作。
**兩條發聲路徑在模擬器裡都活不起來，所以「追檔案開啟」這個方法對這一題是死的。**

### 順帶確認的一件事

`Sound Master not found` 之後的行為是**退回內建喇叭**，不是靜音——
`Using internal speaker` 是同一個對話框的第二行。先前只知道有那個字串。

### 剩下的路

只剩**反組譯開檔那一段**。目前 remake 取 `DATA/` 那一份
（`internal/ui/sound.go` 的 `soundFile`），依據是執行檔字串表裡 `sounddat.psf`
與 `message.ptf`、`monodat.pgf`、`DATA` 相鄰——**那是相鄰性推論，不是實證**，
等級只到強證據。
