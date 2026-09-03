# CONTEXT — 模擬城市重製與繁中化（chengshi_cht）

全專案的單一現況入口。接手時先讀本檔，再讀 [`CLAUDE.md`](CLAUDE.md)
與目標規格。

最後核對：2026-09-03。

## 1. 目前狀態

**可以玩，也出得了發行包。** 規則層、呈現層、中文化都接起來了：新城市或
八個劇本開得起來，基本外觀加六種資料片風格切得動，工具蓋得下去，四個視窗
打得開，存檔用的是原版 **DOS 存檔格式**（128 位元組檔頭 ＋ 27120 檔身），
城市名存在檔頭裡；讀取端三種都吃得下（裸檔身、DOS 存檔、解壓後的 `.PSN`）。
Linux tar.gz／AppImage、Windows、macOS 三個平台的發行包都打得出來，正常玩家路徑有實機驗證。
正式版號固定使用 `v.<主版>.<次版>.<修訂版>-YYYYMMDD`；Git tag、GitHub Release、
程式版本字串、`dist-all/<版本>/` 與 checksum 必須完全一致。目前重建目標是
`v.1.0.0-20260901`，舊 `20260901` Release 與 tag 已依使用者決定刪除，不得復用。

**逐次元對拍收斂了，精靈也在內。** 四份實驗、每份 8000 個 frame（400 個的
短版一份）全部逐 frame 完全一致：空城 13 954 次抽樣、**Dullsville 劇本
122 314 次**、**Tokyo 劇本 955 206 次**（大城，火車、船、飛機、直昇機、
怪獸與爆炸全員上場）。每個 frame 比的是抽樣次數（規則層與精靈分開算）、
亂數狀態、`Scycle`、需求閥門、城市評估的分數與問題表、**場上每一隻精靈的
十八個欄位**，以及**整張 12 000 格地圖的雜湊**；終點地圖與資金也相同。
做法是給 oracle 加單步與觀測指令，見
[`docs/re/12-tick-parity.md`](docs/re/12-tick-parity.md)。

**也對到 DOS 執行檔本身了，但只到抽樣的程度。** DOSBox 裡跑原版、載入劇本、
打開 Auto-Budget 讓它自己跑一段再存檔，remake 從同一份劇本跑到同一個
`CityTime` 再逐格比。逐 tick 對拍對 DOS 版在原理上做不到（它載入時自己重播種），
所以這條路量的是「同一起點、同一刻、地圖有多像」。結果與兩個量法上的坑寫在
[`docs/re/18-dos-parity.md`](docs/re/18-dos-parity.md)。

**操作介面是原版 DOS 的版面，而且逐格對過**，量測與現況在
[`docs/spec/ui-layout.md`](docs/spec/ui-layout.md)（主版面已 READY）。
畫布 1920×1050 ＝ 原版 640×350 × 3；介面美術直接用原版 `.PGF` 的庫 2–7。
已對拍並實作：選單列 ＋ 四個下拉選單、編輯視窗（含標題列的城市名與年月）、
City Form 視窗（外框、標題列、九個圖層圖示、色階圖例、網點）、
三個資料視窗、查詢面板、視窗疊放與搬動、`Ctrl-C／H／E／R`、五段速度、
工具盤十四格 ＋ 發電廠副選單、**招牌與劇本選單兩幕**。

判準是**逐格逐位元**，不是「看起來像」：

| 比什麼 | 結果 |
|---|---|
| 關閉 City Form 後的完整編輯區 512 格 | **502 格逐位元相同**；10 格全是兩側不同的游標覆蓋 |
| City Form 完整區域 | 最新重跑 **130 581／131 600 個像素相同**（99.226%）；地圖本體 **107 834／108 300**（99.570%）；游標取樣會造成小幅變動 |
| 劇本選單 | 223 867／224 000（99.941%），差的是滑鼠游標那個 16×15 的方塊 |
| 招牌 | 221 602／224 000；差異的 93.7% 是中文化的三個開場按鈕，框外 151 個像素是游標。⚠ 2026-09-04 招牌加了**第四顆按鈕**（remake 自加的地形編輯器入口，放在量出來沒有原版美術的那一塊），下一次重量會多出它的面積 |

**發行的 AppImage 也對過了，不只工作樹的原始碼。** `v.1.0.0-20260901` 完整版
AppImage 與同一份原始碼現建的執行檔，在十四幕玩家畫面上無可分辨差異，其中十三幕
有逐位元完全相同的截圖；四項原版逐格與逐像素數字改用 AppImage 當 remake 側重量，
與前一份收據一致。所以包裝層（發行旗標、內嵌字型與圖集、AppImage runtime、
資料路徑解析）沒有改變任何一幕。見
[`docs/playtest/appimage-parity-2026-09-01.md`](docs/playtest/appimage-parity-2026-09-01.md)。

⚠ 同一份收據記下一件會影響所有畫面量測的性質：**這個遊戲的畫面不是逐次可重播的**。
`game.LoadCity` 走 `LoadCitySeed(path, sim.RandomSeed())`，而 `RandomSeed()` 是
`time.Now().UnixNano()`；`-seed` 只接到「開新城市」那條路，`-load` 與 `-scenario`
都沒有。所以同一顆二進位讀同一份城市檔，兩次啟動的畫面就會差幾十到幾百個像素。
拿截圖比對下結論之前要先量自己跟自己的雜訊底線。

這條線抓到的錯全部是「編得過、測得過、玩得動」那種——最重的是城市檔的
檔頭長度讀錯 16 個位元組（地圖整張平移八列，而純量、地物格數、
存檔 round-trip 全部正常），以及 `.PPF` 的位元平面組反（版面分毫不差、
字讀得出來，只有顏色整組錯位）。

**1990 年那支地形編輯器整支重製完了。** 原版 `TERRAIN.EXE` 是軟體世界那片
磁片附的獨立程式，LZEXE 解包之後反組譯：三個選單的十七條命令、六格工具盤
（四個畫筆、油漆桶、復原）、五千格的復原環加四份全圖快照、參數對話框、
年份輸入全部照原版做出來。**它用的是遊戲本體那一套視窗系統**，座標一模一樣，
所以 remake 不必為它另量一套版面。出口也照原版：編輯器只存檔，回遊戲自己讀。
規則層在 `internal/sim/editor.go`（headless、有單元測試），
規格 [`docs/spec/terrain-editor.md`](docs/spec/terrain-editor.md)（READY）。

**remake 加了三樣原版沒有的東西**，都標明是加的、不是還原：
**縮小**（`-`／`=`／滾輪，1 → 1/2 → 1/4）、**四種語言**
（繁中／简中／日文／English，SYSTEM→設定；會保存到使用者設定檔，`-lang` 只覆蓋
本次啟動）、
**背景音樂**（播玩家自己準備的 `.ogg`／`.wav`）。
原版**沒有音樂**這件事查清楚了，見
[`docs/re/19-no-music.md`](docs/re/19-no-music.md)。

**2026-08-31 語系設定決策：** 使用者確認在 SYSTEM 原版項目之後加分隔線與
「設定」，排除隱藏快捷鍵與修改招牌。原版 0–12 列索引與動作不變；設定檔格式、
路徑、命令列優先序及失敗即關閉退路見 [`docs/spec/settings.md`](docs/spec/settings.md)。
四語設定視窗已以正式字型、1920×1050 畫布目視驗收，文字與目前語言反白均無裁切。

**2026-08-31 音樂 polish 決策：** 使用者指定從其自備的
`SimCity_2000_Special_Edition_DOS.zip` 擷取 General MIDI 組並轉成 OGG，供本機
`music/` 播放。這是《模擬城市 2000》的跨作品曲目，不是 1989 DOS 原版音樂；
所有衍生音訊只留本機、不進 Git 或公開 `release/`。來源、工具版本與驗收見
[`docs/research/sc2000-music-import.md`](docs/research/sc2000-music-import.md)。
播放器已改為情境式選曲：平時只在曲末依 `CityClass` 切換兩首一組的曲池；災難立即
插播固定主題，完整播完一次再回平時曲池。`M` 可暫停／恢復、`[`／`]` 可手動換曲；
一般情境不打斷手選曲，災難仍有最高優先序，`-mute` 同時停用音效與音樂。

逐災難固定表為：火災 `10004`、水災 `10000`、空難 `10002`、龍捲風 `10003`、
地震 `10007`、怪獸 `10013`、空襲 `10012`、核熔毀 `10008`。只有 `10004` 有
SC2000 災難用途資料支持；其他都是使用者確認的 remake 產品配對。完整平時曲池、
缺檔退路、優先序與測試結果見
[`docs/spec/adaptive-music.md`](docs/spec/adaptive-music.md)（CONFORMED）。

Docker／Xvfb 正常玩家路徑已用 `Alt-D` 觸發火災並切到 `SC2000-10004.ogg`；另以
ALSA 檔案裝置確認 OGG 經 Ebiten／oto 送出非零音訊。這是技術驗收，不冒稱為人耳
曲風或長時間疲勞評鑑。SC2000 音樂只進本機 `full/`，公開 `release/` 明確排除。

還沒收完的事：

- **自動玩家的劇本通關率是 30/40**（八個劇本 × 五顆種子，六個劇本全過）。
  達斯維利與舊金山全掛，原因量到底了。達斯維利：把資金補滿、積極鋪路、
  每年蓋滿，三十年飽和在 **52 000–62 000**（門檻十萬），再放寬上限數字
  逐位元不變——是這一類策略的天花板。同一套策略給一百二十年是 101 720，
  所以門檻不是不可達，是需要約三倍的時間。
  舊金山差 6%：結算時 72 000–94 000，門檻是加權人口十萬。地震斷線要修四年，
  修完只剩一年可以長。（不出手的話是 43 000，所以策略確實有用，只是不夠。）
  **七種介入全部沒有提升通關數**，所以這是局部最佳解：要再往上得重新
  設計策略（按目標與剩餘年數切換擴張／守成），不是調參數。
  2026-08-30 修掉的是另一件事：自動玩家先前完全沒回應
  `ResCap`／`ComCap`／`IndCap`（沒有體育場／海港／機場就把對應需求壓成 0），
  那是真缺陷（天花板測試 27 460 → 101 720），但不是那兩個劇本的綁定條件。
  見 `internal/autoplay/demandcap.go` 與 `TestDemandCapsLifted`。


- **音效**：八段的事件對應解開了（0 交通壅塞、1 爆炸、2 怪獸、3 警笛、
  4 船笛、5／6 工具成功、7 工具失敗，見
  [`docs/re/16-dos-oracle.md`](docs/re/16-dos-oracle.md) §五之四），
  **也接進遊戲了**（`internal/audio`、`internal/ui/sound.go`、
  規格 [`docs/spec/sound.md`](docs/spec/sound.md)）。
  **取樣率是暫代值 5400 Hz**：兩個量測方法都指到它（強證據，§五之五）——
  長度比 5300–5450、頻譜形狀 5320–5410（`tools/snd_rate_fit.py`，附正對照），
  但沒有從程式碼直讀。下一個入口是音效卡驅動命令 `AH=04h/05h` 那個 16 位元參數
  （遊戲設 20）的單位。
  用原版的聲音對這件事目前做不到：Tandy 走 `INT 1Ah AH=83h`（DOSBox-X 沒實作）、
  Covox Sound Master 沒有模擬器，而內建喇叭放的是程式自己合成的嗶聲。
  所以驗收只到「錄下真正送到音效裝置的位元組、量長度」（`tools/audio_capture.sh`）。

- **多語的覆蓋率不平均**，而且四種語言的來歷不一樣（`translations/README.md`）：
  繁中是全譯、简中是繁中的**字面**轉換（不是用語在地化）、
  **日文只做了標籤那幾段**（155 筆：工具、月份、選單、地圖圖層、地物名、
  評估欄位），訊息與圖片文字走退路顯示繁中；英文是執行時從玩家自己那份
  `.PTF` 讀的，沒有原版就沒有英文。另外 `internal/ui` 還有一批寫死的
  中文字串沒進 `ui.tsv`（多半是錯誤訊息），切到別的語言時那幾句仍是中文。

- **數值圖層的第五級門檻是暫代值**。原版的密度類圖層實測是**五級**
  （三個 EGA 色 ＋ 兩段網點），而 Micropolis 的 `GetCI` 只有四級
  （50／100／150／200）。多出來那一級的門檻沒量到，先照等距外推填 250。
  要定案得讓 oracle 吐出密度陣列再與 DOS 畫面逐格對，見 §5.5。

### 已盤點的素材

| 素材 | 狀態 |
|---|---|
| Micropolis 原始碼（EA 2008 GPL-3.0）| 已封存在 `workplace/ref/micropolis/`，是規則層的第一手依據 |
| DOS 版 1.10（69 檔）| 已盤點。`.PGF`／`.PTF`／`.PSN`／`.PSF` 全部解得開，是呈現層與中文化語料的來源 |
| DUX X11 版（SGI／SunOS／Solaris）| 已列內容：30 個 Tcl、154 個 XPM、46 個 au、23 個 `.cty`，無 C 原始碼 |
| 軟體世界珍藏版 29 說明書 | 56 張跨頁掃描已解開；本文 p.1–82 逐頁轉錄成 `docs/manual-cht/`，另含《模擬城市參考附表》。只有附贈的密碼表不轉錄 |

### 驗證入口

```bash
tools/go.sh test ./...              # 全部測試（docker，含接線檢查與字型覆蓋率）
tools/playtest.sh [種子]            # 正常玩家路徑實機驗證（真視窗、真鍵盤、真滑鼠）
tools/package_all.sh v.1.0.0-20260901       # 建 dist-all/<版本>/full|release|promo
tools/verify_package_all.sh v.1.0.0-20260901 # 驗 AppImage 入口、滑鼠選單、權利、存檔與音訊
tools/build-mac.sh                  # macOS universal（osxcross）＋ 靜態驗收
tools/screenshot.sh [秒] [檔名]     # 單張截圖，GAME_ARGS 帶遊戲參數
tools/font.sh                       # 重烘點陣字圖集（改過譯文或註解後）
tools/i18n.sh                       # 比對原版 .PTF，列出 TSV 缺的鍵（預設空跑）
tools/screen_parity.sh [劇本] [風格] # 畫面逐格對拍（自己跑一次 DOSBox 產基準）
tools/appimage_parity.sh [版本]     # 發行 AppImage 的畫面對拍（同源證明 ＋ 原版抽查）
python3 tools/gen_city_masks.py     # 畫台北／台中／台南的粗胚地形 → tools/maps/
tools/go.sh run ./tools/citymap <粗胚.txt> <輸出.cty> <城市名>  # 粗胚轉城市檔
tools/dosbox.sh <秒> <前綴>         # 跑 DOS 原版當 oracle（截圖 ＋ 錄音 ＋ 動作腳本）
MODE=run tools/dos_parity.sh        # DOS 原版 vs remake 的抽樣對拍（八個劇本）
tools/go.sh run ./cmd/simtool play all   # 自動玩家把八個劇本各玩一次
tools/oracle/build.sh               # 建 Micropolis oracle
tools/oracle/drive.sh <tcl> <json>  # 用 pty 驅動 oracle 取狀態
python3 tools/unpack_simcity_exe.py <SIMCITY.EXE> out.bin   # 解開執行檔的自解壓外殼
tools/ida.sh analyze SIMCITY.EXE    # IDA Pro 9.4 headless（反組譯是退路，見 CLAUDE.md §2.4）
```

### 已確認的事實（可引用）

- 手上這份 DOS 副本的防拷**查驗還在，但一律判過**：開新城市後跳出
  「Enter NAME of city ／ Page:」，送空白答案回的是
  `Congratulations, you passed.`（`docs/re/16-dos-oracle.md` §2）。
- 螢幕模式決定的是**圖形檔的目錄**不是檔名：`Screen Mode: T` 會去找
  `C:\tdy\WESTCEGA.pgf`（同上 §3）。
- **DOS 1.10 的六個汙染權重全部是 Micropolis 註解裡的舊值**：壅塞車流 25、
  稀疏車流 10、火災 60、輻射 −40、海港／機場／電廠 60、工業 50。
  從執行檔解開後的位元組讀出來（`docs/re/18-dos-parity.md` §6.3）。
  Micropolis 2008 把它們調成 75／50／90／255／100。
  **remake 照 Micropolis**——DOS 1.10 是同一份引擎的較早版本。
- `SIMCITY.EXE` 是**打包過的**，而且進入點是**破解程式的 stub**（掛 INT 21h
  把防拷判斷蓋掉），原版進入點在 `載入段 + 0xE0 : 0`。壓縮法是資料檔那支
  LZSS **多一道 `ror 1`**。解法：`tools/unpack_simcity_exe.py`。
  ⚠ 解壓有位元級瑕疵，**單一次解壓的輸出不可信**；可信的是「多個起點解出來
  一致的那些位元組」（工具會自動交叉檢查）。

- 軟體世界代理的是**英文版遊戲 ＋ 中文說明書**，一代沒有中文版遊戲。
  （來源：封面「珍藏版 29／NT 180／2 片裝」＋ 骨灰集散地規格表「中文版本：無」）
- 八個劇本檔存在且與說明書的八個「悲情城市」對得上：`BERN`、`BOSTON`、
  `DETROIT`、`DULLSVIL`、`HAMBURG`、`RIO`、`SANFRAN`、`TOKYO`。
- 說明書第 13–14 頁的譯名樣本：悲情城市（SCENARIOS）、達斯維利（DULLSVILLE）、
  明日之都、市鑰。
- DOS 版有**八種螢幕模式**（Hercules／CGA 單色／Tandy Color／Hires EGA 單色／
  Lores EGA Color／Hires EGA Color／單色 VGA-MCGA／256 色 VGA-MCGA），
  圖形檔名規則是 `<圖形集><模式>`（`SIMCITY.CFG` 的 `Graphics Set: WESTCEGA`）。
  **中文化的畫布尺寸不能只假設一種版面。**
- 劇本的城市名來自 `.PSN` 檔頭，不隨風格改；劇本簡介則隨風格改寫。
  所以中世紀風格玩劇本 3 會同時看到「漢堡　1944」與「史邦美樂，1535」，
  原版就是這個組合。

## 2. 證據優先序

見 [`CLAUDE.md`](CLAUDE.md) §1.1。摘要：
Micropolis 原始碼 > DOS 1.10 資料檔 > X11 Tcl／XPM > DOS 反組譯／DOSBox 實跑 >
軟體世界中文說明書（譯名）> 官方英文手冊 > 社群。

## 3. 文件導覽

| 檔案 | 內容 |
|---|---|
| [`CLAUDE.md`](CLAUDE.md) | 方法論、四道閘門、證據優先序、授權立場 |
| [`README.md`](README.md) | 專案首頁：這是什麼、為何保存、現況 |
| [`LICENSE`](LICENSE) | RRSAL-1.0 全文（占位符已填、條款本文逐字未改）：非商業免費含修改再散布、實況分潤明示允許、貢獻回授、商業洽談 |
| [`WORKLOG.md`](WORKLOG.md) | 逐輪工作紀錄：日期、命令、失敗、negative result |
| [`docs/research/`](docs/research/) | 查證筆記：玩家與評論對一代的說法（二手彙整，每條標來源）|
| [`docs/re/`](docs/re/) | 機制筆記（讀 Micropolis 與反組譯的產物）。入口是 `00-master-index.md`、`00-function-index.md`、`00-wiring-status.md` |
| [`docs/formats/`](docs/formats/) | 資料格式規格：`.PGF`／`.PPF`／`.PSN`／`.PTF`／`.PSF`／`.cty`／DOS 存檔 |
| [`docs/spec/`](docs/spec/) | 可實作規格，只有標 READY 的能動工 |
| [`docs/manual-cht/`](docs/manual-cht/) | 軟體世界珍藏版 29 說明書**逐頁完整轉錄**（保存用，p.1–82 ＋ 參考附表）|
| [`docs/manual/`](docs/manual/) | 官方英文手冊（IBM PC 版）的繁中整理。**只整理機制章節並與原始碼逐節對照**，不做全書轉錄——理由（用途與授權）寫在該目錄的 README |
| [`docs/walkthrough/`](docs/walkthrough/) | 繁中攻略。**以讀出來的原始碼為根據**，不是翻譯社群攻略（理由寫在該目錄的 README）：成長迴圈、八個劇本的真實過關條件、七條常見錯誤說法的訂正 |
| [`translations/glossary.md`](translations/glossary.md) | 譯名表，唯一真相 |

## 4. 術語

| 詞 | 意思 |
|---|---|
| Micropolis | EA 於 2008 年以 GPL-3.0 釋出的 SimCity 原始碼，本專案規則層的一手證據 |
| DUX X11 版 | DUX Software 1993 的商業 Unix 移植，Micropolis 的直接前身 |
| 悲情城市 | 軟體世界對 SCENARIOS（八個劇本）的譯名 |
| 圖形集 | DOS 版的六組資料片美術：ASIA／MEDI／WEST／FUSA／FEUR／MOON |
| 逐 tick 對拍 | 同種子同操作餵進 Micropolis 與 Go 版，逐步比對 120×100 地圖與純量 |
| 圖形集 | DOS 版的六組資料片美術：ASIA／MEDI／WEST（古城風情）、FUSA／FEUR／MOON（回到未來）|
| 悲情城市 | 軟體世界對八個劇本（SCENARIOS）的譯名 |
| 遊戲刻 | `CityTime`。一年 48 刻；劇本的 `CityTime = (年份−1900)×48+2` |

## 5. 已被推翻的斷言

（目前沒有。這張表只放「已經知道錯、還不知道對」的；正確答案定案後刪掉該列。）

> 2026-08-30 一次推翻四項，四項都**已經有正確答案**，所以直接寫進正文，
> 這裡只留指標。四項的共同形狀是**沒有症狀**——編得過、測得過、玩得動、
> 目視也看不出來——而且四項都是同一個判準抓到的：
> **逐位元跟原版比，不是「看起來像」**。
>
> | 原本的斷言 | 正確答案 | 寫在哪 |
> |---|---|---|
> | 編輯視窗地圖區 `y 54–310` | `y 55–310`，54 那一列是白色外框 | `docs/spec/ui-layout.md` |
> | `.PGF` 的調色盤值就是螢幕顏色 | EGA 檔存的是 `0/80/160/240`，螢幕上是 `0/85/170/255` | 同上 |
> | 工具盤畫在 `(6,53)` | `(8,55)` | 同上 |
> | 地圖圖塊色號 0 是透明 | 是**真正的黑**（道路標線、建築輪廓）| 同上 |
> | DOS 存檔的 `MiscHis` 是 128 個 short、地圖在 3264、檔案短 16 位元組；`.PSN` 檔頭 144 | **兩種檔案都是 128 檔頭 ＋ 標準 27120 檔身**，地圖在 3248，什麼都沒缺 | `docs/formats/01-city-file.md` |
>
> 最後一項最值得記：它讓**整張地圖往下平移 8 列**，而純量、地物格數、
> 甚至 remake 與 DOS 存檔的逐格對拍全部正常——因為兩邊都經過同一個錯誤
> 的讀法，偏移互相抵銷。**只有跟原版畫面對拍才看得出來**
> （`tools/screen_parity.sh`）。


## 5.5 未解（有證據但還沒定案）

> 2026-08-30 移出一列：**DOS 1.10 的汙染權重比 Micropolis 低**，已從執行檔
> 讀到位元組定案（六個權重全部是 Micropolis 註解裡的舊值），見
> [`docs/re/18-dos-parity.md`](docs/re/18-dos-parity.md) §6.3。
> remake 照 Micropolis，不往 1991 年調。
> ⚠ **這個決定的後果量過了，而且是數量級的**：把權重換成 DOS 值再跑一次
> 八個劇本，里約從 25 620 變成 144 800（5.7 倍）、東京 73 880 → 124 760。
> 所以**拿網路上原版玩家的人口數字對照 remake 會對不起來**——那不是 bug，
> 是這個決定的已知後果。（同一個實驗也排除了「那兩關過不了是因為汙染太狠」：
> 換成舊值之後達斯維利離門檻還有 3.2 倍，而底特律反而掛了。見 WORKLOG 同日。）
>
> 2026-08-30 再移出一列：**八段音效各是什麼事件**，已從 `PlaySound(n)` 的
> 十一個呼叫點定案，見 [`docs/re/16-dos-oracle.md`](docs/re/16-dos-oracle.md)
> §五之四。取樣率仍未定案，留在下表。
>
> 2026-08-30 再移出一列：**DOS 原版顯示的年份與 `CityTime` 對不上**，
> 已量出關係：**DOS 顯示 `1849 + CityTime/48`，比劇本自己的日期早 51 年**
> （四個資料點吻合到個位數，見 [`docs/re/16-dos-oracle.md`](docs/re/16-dos-oracle.md) §七）。
> **這是原版自己前後不一致**：八個劇本的簡介與手冊全部對上 `1900 + CityTime/48`，
> 而狀態列一個都不中——玩家會先讀到「HAMBURG 1944」再看到狀態列寫 `Jan 1893`。
> **remake 維持 1900 基準，不重現這個偏移**，理由寫在該節。
>
> 2026-08-30 再移出一列：**`.PGF` 第 0 庫後面那塊資料是什麼**，已定案是
> 地圖視窗用的 960 張圖塊縮圖（CEGA／MONO 3×3、sega 3×1、mcga 1×1），
> 單色與 256 色後面還接著 128 字的 CP437 介面字型，見
> [`docs/formats/03-pgf-graphics.md`](docs/formats/03-pgf-graphics.md) §7。
> 同時推翻了「風格檔與基本檔逐位元組相同」——`MOONCEGA` 自己一份。

- **維修費**：《參考手冊》p.63 寫「道路 $1、橋 $4、鐵路 $4、隧道 $10」，
  Micropolis 算出來是道路 1、橋 5、鐵路 2、隧道不存在。這一項沒有 DOS
  資料檔可以裁判（維修費不寫在 `.PTF` 裡）。規則層照 Micropolis。
  見 `docs/manual-cht/p23-58-operations.md` 末節。

| 問題 | 現況 | 要怎麼定案 |
|---|---|---|
| 兩份 `SOUNDDAT.PSF` 哪一份被讀 | 只差第 2 段（怪獸）：根目錄 7500 取樣、`DATA/` 10752（與 1991 的 `.V4` 一致）。remake 取 `DATA/`。**依據已從「字串相鄰」升級成同一條程式路徑的實測**：拿掉 `DATA/WEST_MSG.PTF` 之後遊戲印出它自己組的路徑 `C:\DATA\west_msg.ptf`——圖形檔頭存的是**裸檔名**，程式把 `DATA\` 補在前面；而音效檔名是同一組三欄位裡的第二欄，預設值後面就跟著兩個 `DATA`。音效那一欄本身沒被直接觀測到，等級仍是強證據 | **DOSBox 這條路已排除**：`SIMCITY.CFG` 的 `Sound:` 四個值（內建喇叭／Covox／Tandy／無）在模擬器裡都到不了 PCM，預設值 `I` 本來就是內建喇叭。只剩反組譯開檔那一段，或找出讓 `\nFound file %s` 印出來的除錯開關。見 [`docs/re/16-dos-oracle.md`](docs/re/16-dos-oracle.md) §八 |
| `.PSF` 八段的取樣率 | 兩個對不同誤差敏感的方法都指到 5400 上下：長度比 5300–5450、頻譜形狀 5320–5410（段 1 對 `expl-hi`，r=0.966，附正對照）。初始化找到了（映像 `0xCC20`）：遊戲只設腳位、音量、和**一個值為 20 的 16 位元參數**（命令 `AH=04h/05h`）——那就是速率參數，缺的是它的單位 | 要 Covox Sound Master 的程式設計文件，或一個會模擬它的環境。已排除 PIT（整份執行檔沒有 `out 40h`）|
| `.PGF` 第 9 庫的用途 | 12 張 16×16 單平面圖（方框切四象限的全部一／相鄰兩／三象限組合）。**兩條假說都排除了**：不是圖塊裁切遮罩（四種顯示模式下都是 16×16，而地圖圖塊在 MCGA 與低解析 EGA 是 8×8；四個 L 形也不是矩形交方格能產生的）；**十五個畫面狀態裡也沒有一個把它畫出來**（把它塗成全 1／全 0 各跑一次，地圖格網上「不是第 0 庫圖塊」的格子完全相同，都是那 8 格工具游標外框）。**有可能是沒被用到的資料** | 要下「完全沒用到」得反組譯找呼叫點（卡在分段沒重建）。沒走到的狀態還有災難進行中、載入／存檔對話框、列印、關於視窗，以及其他圖形集與螢幕模式 |
| `.PGF` 第 8 庫在什麼時候被畫出來 | 基本檔是 107×20 的一行日文「ウルトラ警備隊」（四種顯示模式各自獨立編碼、解出同一串字），風格包裡是空的。**十五個畫面狀態裡沒有一個把它畫出來**：風格檔的第 8 庫本來就空，塗成全 1 之後只要有被畫就會冒出 102×16 的實心方塊（≥1 632 像素），實測最大差異只有 869 像素、低於噪音底線 2 953，地圖格網的「不明格」也完全相同。**有可能是沒被用到的資料** | 同第 9 庫：要下「完全沒用到」得反組譯找呼叫點。是不是彩蛋仍是假說 |
| DOS 原版顯示的年份偏移的**成因** | **關係已解**（見下方移出紀錄），但 1900 − 1849 = 51 這個數字看不出是哪一種差一錯誤 | 要反組譯畫日期那一段。不影響任何事，純粹好奇 |

### 數值圖層的第五級門檻

原版的數值圖層（人口分佈、交通、污染、犯罪、地價、警消覆蓋）用
**三個 EGA 色 ＋ 兩段網點，共五級**，值太低的格子直接畫地形——
逐格分類 3000 個密度格量出來的（`workplace/dosbox/mi-04-icon3.png`）。

但 Micropolis 的 `GetCI`（`g_map.c:105`）只有**四級**：50／100／150／200。
DOS 版多出來的那一級門檻沒有量到，`internal/ui/windows.go` 的 `rampSteps`
先照等距外推填 250，**標成暫代值**。

要定案得把 remake 的密度陣列與原版同一時刻的畫面逐格對上，
而密度圖是載入後由掃描重算的，「同一時刻」不好對齊——下一個入口是
讓 oracle（Micropolis 那份）吐出密度陣列，再與 DOS 畫面對。

## 6. 現行工作清單

依序做，前一項沒完成不要跳下一項。

1. ~~取得 Micropolis 封存~~ **完成**：commit `c98f6b0`。
2. ~~核對原始碼地圖~~ **完成**：[`docs/re/00-source-map.md`](docs/re/00-source-map.md)。
   推翻了「`s_` ＝ 規則、`w_` ＝ 介面」的假說。
3. ~~建立可驅動的 oracle~~ **完成**：[`docs/re/01-oracle-harness.md`](docs/re/01-oracle-harness.md)。
   docker ＋ Xvfb 編得起來、跑得動，pty 驅動 128 個 Tcl 狀態存取子指令。
4. ~~DOS 1.10 素材盤點~~ **完成**：[`docs/formats/00-dos110-inventory.md`](docs/formats/00-dos110-inventory.md)。
5. ~~Go 骨架 ＋ 亂數~~ **完成**：[`docs/spec/rng.md`](docs/spec/rng.md) 標 READY，
   `internal/sim/rand.go` 實作，測試對活的 oracle 黃金樣本逐項相同，
   接線表與 `TestWiringStatus` 已建立（四道閘門第一次全部走完）。
6. ~~地圖陣列與圖塊編碼~~ **完成**：`docs/re/03` ＋ `docs/spec/map-and-tiles.md`，
   `internal/sim/world.go`、`tiles.go`（工具重產）。
7. ~~地形產生~~ **完成**：`docs/re/04` ＋ `docs/spec/terrain.md`，
   `internal/sim/terrain.go`。**四顆種子、48000 格逐格對拍全部相同**（含造島分支）。
8. ~~城市檔格式~~ **完成**：`docs/formats/01-city-file.md` ＋ `docs/spec/city-file.md`，
   `internal/sim/cityfile.go`、`scenario.go`。32 個檔案逐位元組 round-trip，
   劇本 1 對 oracle 零無法解釋的差異。
9. ~~電力傳導~~ **完成**：`docs/re/05-power-scan.md` ＋ `docs/spec/power.md`，
   `internal/sim/power.go`。受控實驗 12000 格逐格對拍，劇本 1 的 266 格
   `PWRBIT` 差異全部收掉。
10. ~~四個逐格掃描~~ **完成**：`docs/re/06-scans.md` ＋ `docs/spec/scans.md`，
    `internal/sim/scan.go`。收斂後的地價／汙染／犯罪平均值與原版相同。
11. ~~DOS 資料檔的共用壓縮~~ **完成**：`docs/formats/02-dos-lzss.md`，
    `internal/assets/`。一套 LZSS 打開 `.PGF`／`.PPF`／`.PSN`／`.PSF`／`.PTF`；
    七個訊息檔全部解出（中文化語料），八個 DOS 劇本能直接餵進模擬層。
12. ~~交通、分區、災難、普查、需求閥、預算、評分、十六相位主迴圈~~
    **已實作**：`docs/re/07`–`11`，`internal/sim/{traffic,zone,mapscan,disaster,census,eval,simulate}.go`。
    驗收：**住宅／商業／工業三種分區的微實驗都逐次元完全一致**
    （692.5／564.2／949.2 刻，地圖零差異）；整城逐 frame 對拍三份都 8000/8000
    （空城 13 954 次抽樣、Dullsville 122 314 次、Tokyo 955 206 次）。
    見 `docs/re/12-tick-parity.md`。
13. ~~精靈系統~~ **完成**：`docs/re/13-sprites.md`，
    `internal/sim/sprite.go`、`sprite_move.go`、`sprite_effects.go`。
    驗收用 **Tokyo 劇本**（劇本災難就是怪獸，是唯一會讓精靈全員上場的城市）：
    `TestFrameParityTokyo` **8000/8000**，判準包含每個 frame 場上每一隻精靈的
    十八個欄位與整張地圖的雜湊；`TestSpriteParity` 是同判準的 400 個 frame 短版。
    追這條線修掉七個真的錯，其中兩個會影響實際遊玩：
    **載入城市沒做 `InitWillStuff`**（讀第二座城市會留著前一座的地價、汙染、
    犯罪、交通與精靈）與 **`MoveObjects` 重建串列把爆炸蓋掉**（怪獸拆房子
    不會爆炸）。見 `docs/re/12-tick-parity.md` §6之七、§6之十。
13.5. ~~圖塊動畫~~ **完成**：`docs/re/17-tile-animation.md`，
    `internal/sim/animate.go` ＋ `anitab.go`（`tools/gen_anitab.py` 產生）。
    火在燒、煙在冒、車在跑、雷達在轉、噴泉在噴——原版靠一張 `aniTile[1024]`
    「下一格」表，每個畫格把帶 `ANIMBIT` 的格子換成下一格。
    ⚠ 它**會改地圖但不是模擬的一部分**：原版從畫編輯視窗的地方呼叫，
    暫停時不動。Go 版由 `internal/ui` 每個畫格呼叫一次，`SimFrame` 不碰，
    所以四份逐 frame 對拍不受影響。
14. ~~訊息系統~~ **完成**：`docs/re/14-messages.md`，`internal/sim/message.go`。
    含分區上限旗標、人口里程碑與八個劇本的勝敗條件。
15. ~~玩家工具~~ **完成**：`docs/re/15-tools.md`，
    `internal/sim/tool.go`、`internal/sim/connect.go`。
    自動接線用八座劇本城市驗證，15 447 格線路裡 99.83% 形狀一致。
16. ~~逐次元對拍~~ **完成**：三份 8000 個 frame 的對拍都完全一致
    （空城 13 954 次抽樣、Dullsville 122 314 次、Tokyo 955 206 次），
    終點地圖與資金零差異。見 `docs/re/12-tick-parity.md`。
17. ~~`.PGF` 圖形版面~~ **完成**：`docs/formats/03-pgf-graphics.md`，
    `internal/assets/pgf.go`。24 個風格圖形檔（4 種顯示模式 × 6 種風格）
    全部解開，第 0 庫一律 **960 張地圖圖塊**——與 Micropolis 的 `TILE_COUNT`
    對得上，是圖塊編號的獨立佐證。

18. ~~軟體世界說明書轉錄~~ **操作手冊與「進入本市」完成**：`docs/manual-cht/`。
    譯名表長出一手依據的詞（工具、災難、選單、地圖圖層、預算、評估、
    劇本難度與過關條件、編輯視窗元件）。剩下安裝步驟與參考手冊，見第 27 項。
19. ~~遊戲文字翻譯~~ **完成**：`internal/i18n/messages/`。
    基本檔 226 條 ＋ 六個風格包的覆寫（含各自改寫的圖片訊息與劇本簡介），
    合計 695 條。譯文來源在 `tools/i18n/`，`tools/i18n.sh` 合併。
20. ~~呈現層~~ **完成**：`internal/ui`（Ebiten）、`internal/game`（組裝層）。
    圖塊渲染、工具列、四個視窗、存讀檔、八個劇本、六種風格。
    按鍵照原版重排（`docs/spec/controls.md`）——`0`–`4` 是速度、
    `B`／`R`／`P`／`T`／`Q` 是原版的工具鍵。先前用 `1`–`0` 選工具，
    而遊戲內的速度副選單（照原版訊息檔翻的）正印著「暫停 0／慢速 1…」，
    等於在叫玩家按一組做別的事的鍵。
    另補上原版的兩個選單：**災難**（`Alt-D`，六種災難）與**系統**
    （`Alt-S`，換圖形集／換悲情城市／開新城市／讀檔／存檔／離開）。
    在系統選單接起來之前，換劇本與換圖形集只能靠命令列參數重開遊戲。
21. ~~正常玩家路徑試玩驗收~~ **完成**：`tools/playtest.sh`。
    另有 `TestAllScenariosReachVerdict`：八個劇本各載入後**不出手**跑到自己的
    判定時限，八個都在時限內送出判定，勝敗兩種結果都出得來（不出手 2/8 過）。
    ⚠ 那證明的是判定機制會觸發，**不是「玩得贏」**——沒有人真的從頭玩通
    任何一個劇本，這一項記在 README 的自評表。在 Xvfb 裡真的
    開視窗、真的敲鍵、真的點滑鼠，走完「開新城市 → 蓋發電廠／電線／道路／
    分區 → 四個視窗 → 查詢 → 捲動 → 存檔 → 離開 → 重開讀檔 → 劇本」，
    每一步截圖，最後用存檔內容做機械判定。
22. ~~發行包、README 更新~~ **完成**：`tools/release.sh` 打 Linux 與 Windows 兩個包，
    `tools/verify_release.sh` 驗包本身（解到乾淨目錄、從那裡執行、資料放別處）。
    README 改寫成現況並附四張畫面。
23. ~~逐刻對拍收斂~~ **完成**。關鍵是**給 oracle 加觀測指令**
    （`sim Frame N`／`Scycle`／`Fcycle`／`Valves`／`Mem`，
    `tools/oracle/patches/apply.py`，建在副本上、封存保持乾淨）。
    在那之前所有對拍都在跟看不到的內部狀態搏鬥，而那些搏鬥掩蓋了幾個
    真正的差異：`SetValves` 該用單精度與整數除法（整城對拍因此從差 8 格
    變成 0 格）、`DoPowerScan` 不該直接寫 `PWRBIT`、`ProblemTaken` 要跨
    評估保留、對拍腳手架自己多推了四步亂數，以及**原版的呈現層挑音效時
    和模擬共用亂數**（拿掉之後劇本版從 1512 跳到 8000/8000）。
    分段對拍（`segparity_test.go`）已被逐 frame 對拍取代並移除。
24. ~~基本風格（`CEGADAT.PGF`）的圖形庫表~~ **完成**：表是**行內**的，
    每個庫前面三個位元組是「平面數 ＋ u16 長度」，每張圖前面四個位元組是
    寬高。四個模式的檔案都解得開（`docs/formats/03-pgf-graphics.md` §8、
    `internal/assets/pgfbase.go`）。`-style base` 現在是預設值。
25. ~~聲音~~ **已接**（`docs/spec/sound.md`、`internal/audio`、
    `internal/ui/sound.go`、`internal/sim/sound.go`）：八段分別是交通壅塞、
    爆炸、怪獸、警笛、船笛、兩種工具成功、工具失敗。驗收錄下真正送到
    音效裝置的位元組（`tools/audio_capture.sh`），五段的長度都對得上。
    **剩下的是取樣率的定值**：程式取 5400 Hz，兩個方法量出的區間分別是
    5300–5450（長度比）與 5320–5410（頻譜形狀）（強證據），規格裡標成暫代。
    順帶查出段 0／3／4 與 X11 同名檔長度對得上但波形不同。
    下一個入口是驅動命令 `AH=04h/05h` 那個 16 位元參數
    （遊戲設 20）的單位。順帶釘死：整份執行檔沒有一次 `out 40h`，
    「用 PIT 中斷餵取樣」可以排除，那是一張 DMA 驅動的卡。
    ⚠ **沒有人拿原版的聲音對過**：Tandy 走 `INT 1Ah AH=83h`（DOSBox-X 沒實作）、
    Sound Master 沒有模擬器。驗的是長度不是音色。
26. ~~macOS 版~~ **完成**：`docker/osxcross.Dockerfile` ＋ `tools/build-mac.sh`，
    arm64 與 x86_64 各編一次再 lipo 成 universal，附靜態驗收（雙架構、
    arm64 的 ad-hoc 簽章、只相依系統庫、含得到中文字串）。`.app` 未簽名，
    首次開啟要右鍵 → 打開。**⚠ Linux 上執行不了 macOS binary，所以沒有
    實機試玩**——靜態全過只代表不會因結構問題開不起來。
27. ~~說明書轉錄~~ **完成**：說明書本文 p.1–82 逐頁轉錄（操作手冊全本 ＋
    《參考手冊》兩章），加上夾在裡面的**《模擬城市參考附表》**（掃描 0050–0051，
    完整按鍵對照 ＋ 中英對照的都市動力表）。56 張掃描全部有交代。
    唯一不轉錄的是**密碼表**（掃描 0052–0055）：它是附贈的防拷答案表，
    不是說明書內容。先前寫「密碼表這批掃描裡沒有」是錯的，已訂正。

28. ~~與原版**畫面**對拍~~ **已建立**：`tools/screen_parity.sh` ＋
    `tools/shot_diff_cells.py`，並且併進 `tools/playtest.sh` 第八段。
    判準是**逐格逐位元**：編輯視窗露出來的 176 格，現在 **174 格與跑起來的
    原版完全相同**，剩下兩格是滑鼠游標所在的位置。
    這條線抓到的錯全部是「編得過、測得過、玩得動」那種——最重的是
    **城市檔的檔頭長度讀錯 16 個位元組**（地圖整張往下平移 8 列，而純量、
    地物格數、存檔 round-trip 全部正常，因為兩邊都經過同一個錯誤的讀法）。
29. ~~招牌與劇本選單畫面~~ **完成**：`docs/formats/06-ppf-screen.md`，
    `internal/assets/ppf.go`、`internal/ui/title.go`。`.PPF` 是逐列交錯、
    **高位在前**的 EGA 四平面；順序組反會得到版面正確、顏色整組錯位的畫面。
    與 DOS 1.10 逐像素對拍：224000 個像素只差滑鼠游標那個 16×15 的方塊。
    remake 在這之前沒有這條路徑，一啟動就直接進城市。
30. **純鍵盤路徑已裁決**：`docs/spec/controls.md` §六。`Ctrl` ＋ 方向鍵確實
    捲動城市（實測），而方向鍵、數字鍵盤、`Ins`、`Del`、`ScrollLock`
    **在原版自己身上就沒有反應**——手冊說那條路是給沒有滑鼠的機器的。
    remake 只接 `Ctrl` ＋ 方向鍵並補上八個方向。

31. ~~`.PPF` 三種顯示模式~~ **完成**：`docs/formats/06-ppf-screen.md`。
    `sega` ＝ 320×200 四平面（招牌逐像素只差滑鼠游標那 128 點），
    `mcga` ＝ 320×199 每像素一位元組，調色盤取自同一個圖形集的 `.PGF`
    （37 個用到的色號全部相符，最大差 3 ＝ 六位元展八位元的算法差異）。
32. ~~City Form 的九個圖示對十一個圖層~~ **解開了**：
    `docs/spec/controls.md` §九。少的兩個（消防範圍、人口成長）藏在
    **按住式的小選單**裡，而訊息檔第 11 段那四筆就是這兩個小選單的內容。
    圖示欄、色階圖例（庫 6／7）、標題跟著圖層走，全部接上。
33. **remake 加了「縮小」**（原版沒有）：`-`／`=` 或滑鼠滾輪，
    1 → 1/2 → 1/4。規格與兩個實作陷阱寫在 `docs/spec/controls.md` §十。

34. ~~語言資料改成 TSV，支援四種語言~~ **完成**：`internal/i18n/messages/*.tsv`
    ＋ `ui.tsv`。一列一筆，四種語言並排；**TSV 是唯一真相**，
    沒有產生器會覆寫它。`-lang` 或系統選單切換。
    四種語言的來歷不一樣，寫在 `translations/README.md`：繁中是翻譯、
    简中是字面轉換、日文是本專案新譯（只做標籤那幾段）、
    **英文是執行時從玩家自己那份 `.PTF` 讀的**（不入版控）。
35. ~~查清楚原版有沒有音樂~~ **完成**：[`docs/re/19-no-music.md`](docs/re/19-no-music.md)。
    **沒有**——五條證據互相印證，最硬的一條是 148 秒實機錄音裡只有 1 秒
    不是靜音，而那 1 秒是對話框提示音（正對照）。
    remake 加了背景音樂，播的是玩家自己準備的檔案。

36. ~~地形編輯器~~ **完成（整支，不只對話框）**：
    [`docs/re/20-terrain-editor.md`](docs/re/20-terrain-editor.md)
    ＋ [`docs/spec/terrain-editor.md`](docs/spec/terrain-editor.md)（READY），
    規則層 `internal/sim/editor.go`、呈現層 `internal/ui/terrain_screen.go`
    ＋ `terrain_draw.go` ＋ `terrain_editor.go`，掛在系統選單。
    原版是 1990 年隨磁片附的獨立程式 `TERRAIN.EXE`；LZEXE 0.91 解包之後反組譯。
    **做完的東西**：三個選單的十七條命令、六格工具盤（四個畫筆 ＋ 油漆桶 ＋ 復原）、
    畫筆與拖曳、油漆桶、五千格的復原環加四份全圖快照、參數對話框、年份輸入、
    市名與難度、City Map 視窗、狀態列。
    幾個關鍵結論：**編輯器用的是遊戲本體那一套視窗系統，座標完全相同**；
    四個畫筆寫的 16 位元字來自遊戲與編輯器共用的工具描述表
    （`dseg:0x2B42`，`DIRT` 0／`TREES` 0x3025／`RIVER` **3（REDGE 不是 RIVER）**／
    `CHANNEL` 4）；**選單命令碼就是 `(選單 << 4) | 列號`**（分隔線佔號，
    正好對上 IDA 標的 default case）；**「造島」是開關不是動作**；
    「平滑河流」前面還有一支 Micropolis 沒有的 `ResetRiverEdges`；
    年份的換算是 `CityTime = (年 − 1900) × 48`，只收四位數且要大於 1900。
    三個百分比就是 `TreeLevel`／`LakeLevel`／`CurveLevel` 本身，
    只有樹叢數量是 `3 × pct`（接在 `TerrainParams.EditorDOS`）。
    出口照原版：**編輯器只存檔**，回遊戲之後自己讀那個檔（使用者定案 2026-09-03）。
    ⚠ 兩處還沒解：**確認框的版面**，以及**進度框與年份框的絕對位置**
    （欄列數讀得出來，視窗原點的公式只有 36×10 那一個量過）。
37. ~~兩片資料片（六個圖形集）的畫面對拍~~ **完成**：
    [`docs/playtest/style-parity-2026-09-03.md`](docs/playtest/style-parity-2026-09-03.md)。
    六個圖形集都是 498/512 格逐位元相同，不同的 14 格在六個集裡位置完全一樣
    （游標框，既有差異）。過程中修掉兩個缺陷：左欄介面圖畫在地圖之後蓋掉
    地圖左緣（庫 2 的尺寸每個圖形集不一樣），以及抓圖腳本的組合鍵靜默失效。
39. ~~交付 `v.1.2.0-20260903`~~ **完成**：四個平台 × 公開／完整兩個變體，
    `tools/verify_package_all.sh` 全過，GitHub release 已發布（附 `promo.mp4`）。
    推廣影片重出一版：57 秒、25 段，新增地形編輯器段落。
    ⚠ **release 的 tag 要指到建置那一個 commit**——`gh release create` 是拿
    遠端當時的 HEAD 建 tag 的，本機還沒推的 commit 不算。
38. **issue #1 第三項（雪花亂碼卡死）**：程式流程分析寫在
    [`docs/playtest/issue1-freeze-analysis-2026-09-03.md`](docs/playtest/issue1-freeze-analysis-2026-09-03.md)。
    找到並修掉音效播放器沒有關也沒有留參考的問題，壓力測試補上音效與縮放。
    **成因未定**——沒有重現過，判準交給卡死偵測器的現場。

## 7. 現行驗證入口

見第 1 節「驗證入口」。
