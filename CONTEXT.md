# CONTEXT — 模擬城市重製與繁中化（chengshi_cht）

全專案的單一現況入口。接手時先讀本檔，再讀 [`CLAUDE.md`](CLAUDE.md)
與目標規格。

最後核對：2026-08-29。

## 1. 目前狀態

**可以玩，也出得了發行包。** 規則層、呈現層、中文化都接起來了：新城市或
八個劇本開得起來，基本外觀加六種資料片風格切得動，工具蓋得下去，四個視窗
打得開，存讀檔用的是原版 `.cty` 格式（拿去餵 Micropolis 也讀得起來）。
Linux／Windows／macOS 三個平台的發行包都打得出來，正常玩家路徑有實機驗證。

**逐次元對拍收斂了，精靈也在內。** 四份實驗、每份 8000 個 frame（400 個的
短版一份）全部逐 frame 完全一致：空城 13 954 次抽樣、**Dullsville 劇本
122 314 次**、**Tokyo 劇本 955 206 次**（大城，火車、船、飛機、直昇機、
怪獸與爆炸全員上場）。每個 frame 比的是抽樣次數（規則層與精靈分開算）、
亂數狀態、`Scycle`、需求閥門、城市評估的分數與問題表、**場上每一隻精靈的
十八個欄位**，以及**整張 12 000 格地圖的雜湊**；終點地圖與資金也相同。
做法是給 oracle 加單步與觀測指令，見
[`docs/re/12-tick-parity.md`](docs/re/12-tick-parity.md)。

還沒收完的一件事：

- **音效**：容器格式解開了，但這八段 PCM **只走 DAC**，而手上的環境
  兩條 DAC 路都走不通（Covox Sound Master 沒有模擬器、Tandy 缺 `tdy\`
  圖形檔）。內建喇叭放的是程式自己合成的嗶聲，不是這些資料——換一份
  音效檔錄同一組動作，聲音逐取樣相同。所以聲音沒有接進遊戲。
  見 [`docs/re/16-dos-oracle.md`](docs/re/16-dos-oracle.md) §4。

### 已盤點的素材

| 素材 | 狀態 |
|---|---|
| Micropolis 原始碼（EA 2008 GPL-3.0）| 已封存在 `workplace/ref/micropolis/`，是規則層的第一手依據 |
| DOS 版 1.10（69 檔）| 已盤點。`.PGF`／`.PTF`／`.PSN`／`.PSF` 全部解得開，是呈現層與中文化語料的來源 |
| DUX X11 版（SGI／SunOS／Solaris）| 已列內容：30 個 Tcl、154 個 XPM、46 個 au、23 個 `.cty`，無 C 原始碼 |
| 軟體世界珍藏版 29 說明書 | 56 張跨頁掃描已解開，操作手冊部分已轉錄成 `docs/manual-cht/` |

### 驗證入口

```bash
tools/go.sh test ./...              # 全部測試（docker，含接線檢查與字型覆蓋率）
tools/playtest.sh [種子]            # 正常玩家路徑實機驗證（真視窗、真鍵盤、真滑鼠）
tools/release.sh ＋ tools/verify_release.sh   # 打發行包並驗包本身
tools/build-mac.sh                  # macOS universal（osxcross）＋ 靜態驗收
tools/screenshot.sh [秒] [檔名]     # 單張截圖，GAME_ARGS 帶遊戲參數
tools/font.sh                       # 重烘點陣字圖集（改過譯文或註解後）
tools/i18n.sh                       # 重新合併七份訊息檔的譯文
tools/dosbox.sh <秒> <前綴>         # 跑 DOS 原版當 oracle（截圖 ＋ 錄音 ＋ 動作腳本）
tools/oracle/build.sh               # 建 Micropolis oracle
tools/oracle/drive.sh <tcl> <json>  # 用 pty 驅動 oracle 取狀態
```

### 已確認的事實（可引用）

- 手上這份 DOS 副本的防拷**查驗還在，但一律判過**：開新城市後跳出
  「Enter NAME of city ／ Page:」，送空白答案回的是
  `Congratulations, you passed.`（`docs/re/16-dos-oracle.md` §2）。
- 螢幕模式決定的是**圖形檔的目錄**不是檔名：`Screen Mode: T` 會去找
  `C:\tdy\WESTCEGA.pgf`（同上 §3）。

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
| [`LICENSE`](LICENSE) | PolyForm Noncommercial 1.0.0 全文（逐字未改）＋ 附註：原版素材排除、商標、規格參考揭露 |
| `WORKLOG.md` | 逐輪工作紀錄（尚未建立）|
| [`docs/research/`](docs/research/) | 查證筆記：玩家與評論對一代的說法（二手彙整，每條標來源）|
| `docs/re/` | 機制筆記（尚未建立）|
| `docs/spec/` | 可實作規格，只有 READY 的能動工（尚未建立）|
| `docs/manual-cht/` | 軟體世界說明書逐頁轉錄（尚未建立）|
| `translations/glossary.md` | 譯名表，唯一真相（尚未建立）|

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


## 5.5 未解（有證據但還沒定案）

- **維修費**：《參考手冊》p.63 寫「道路 $1、橋 $4、鐵路 $4、隧道 $10」，
  Micropolis 算出來是道路 1、橋 5、鐵路 2、隧道不存在。這一項沒有 DOS
  資料檔可以裁判（維修費不寫在 `.PTF` 裡）。規則層照 Micropolis。
  見 `docs/manual-cht/p23-58-operations.md` 末節。

| 問題 | 現況 | 要怎麼定案 |
|---|---|---|
| `.PGF` 第 0 庫後面那塊共用資料（CEGA 11 523、MCGA 9 155、MONO 5 699 位元組）| 風格檔與基本檔之間逐位元組相同，第一張圖的表頭讀出來是 4×45，但整塊沒有逐張分界 | 反組譯繪圖常式，看它從哪個位移開始讀 |
| 兩份 `SOUNDDAT.PSF` 哪一份被讀 | 1991 與 2012 兩個版本並存 | 反組譯檔名字串，或 DOSBox 追檔案開啟 |

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
21. ~~正常玩家路徑試玩驗收~~ **完成**：`tools/playtest.sh`。在 Xvfb 裡真的
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
25. 聲音：**容器格式已解**（`docs/formats/05-psf-sound.md`、`internal/assets/psf.go`）——
    九份檔案各切成八段 4 位元 PCM。**還缺事件對應與取樣率**，而且
    **DOSBox 這條路已經走到底、問不出答案**：`Sound: I`（內建喇叭）放的是
    程式合成的嗶聲，不是這八段資料——換成 MOON 的音效檔（第 4、5 段長度
    差 1.7～2 倍）錄到的聲音逐取樣相同（相關 0.99／0.89／0.96）。
    八段 PCM 只走 DAC，而 Covox Sound Master 沒有模擬器、Tandy 缺 `tdy\`
    圖形檔。見 `docs/re/16-dos-oracle.md` §4。剩下的路：帶 `tdy\` 的副本、
    會模擬 Sound Master 的環境，或脫殼反組譯 `SIMCITY.EXE`（檔案裡有
    明文符號表可以對函式名）。**在對出來之前不接進遊戲**。
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

## 7. 現行驗證入口

見第 1 節「驗證入口」。
