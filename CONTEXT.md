# CONTEXT — 模擬城市重製與繁中化（chengshi_cht）

全專案的單一現況入口。接手時先讀本檔，再讀 [`CLAUDE.md`](CLAUDE.md)
與目標規格。

最後核對：2026-08-29。

## 1. 目前狀態

**模擬規則層大致完整，但整刻對拍還沒對齊。**
四道閘門走完的切片有六個（亂數：讀原始碼 → READY 規格 → Go 實作 →
接線登記 ＋ 機器檢查）。oracle 可用，DOS 素材盤點完成。目前有的東西：
`CLAUDE.md`（方法論）、`LICENSE`（PolyForm Noncommercial 1.0.0）、`.gitignore`、`README.md`。

### 已盤點的素材

| 素材 | 狀態 |
|---|---|
| DOS 版 1.10（69 檔）| 已列檔案清單。**是破解版**，見 `CLAUDE.md` §2.1 三項後果 |
| DUX X11 版（SGI／SunOS／Solaris）| 已列內容：30 個 Tcl、154 個 XPM、46 個 au、23 個 `.cty`，無 C 原始碼 |
| 軟體世界珍藏版 29 說明書 | 56 張跨頁掃描已解到 `workplace/`，尚未轉錄 |
| Micropolis 原始碼 | **尚未取得** |

### 驗證入口

```bash
tools/go.sh test ./...            # 全部測試（docker，含接線檢查）
tools/oracle/build.sh             # 建 Micropolis oracle
tools/oracle/drive.sh <tcl> <json>  # 用 pty 驅動 oracle 取狀態
tools/oracle/run.sh 10 shot.png   # Xvfb 下跑起來並截圖
```

### 已確認的事實（可引用）

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

| 問題 | 現況 | 要怎麼定案 |
|---|---|---|
| 規則層扣款用 Micropolis 的 `CostOf[]`，顯示用原版訊息檔 | 兩者只差體育館（3000／5000）與海港（5000／3000）。顯示已改用訊息檔；扣款還沒改，因為 `CostOf[]` 是逐次元對拍的基準，動它要重跑對拍。 | 改扣款並重跑 `docs/re/12` 的三層對拍 |
| `.PGF` 每張圖那四個位元組的表頭 | 多半是繪製偏移或裁切框 | 比對同一張圖在不同風格的值，或反組譯繪圖常式 |
| MCGA／CEGA 圖形檔尾端約 9 KB 的資料 | 看起來仍是像素值 | 同上 |
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
    驗收：單一分區的微實驗 692.5 刻逐次元完全一致；分段對拍 23 段中
    9 段完全一致（含含完整城市評估的段落）。見 `docs/re/12-tick-parity.md`。
13. ~~精靈系統~~ **完成**：`docs/re/13-sprites.md`，
    `internal/sim/sprite.go`、`sprite_move.go`、`sprite_effects.go`。
    ⚠ 對拍實驗沒有觸發精靈（沒有機場、港口、鐵路，災難也關著），
    所以精靈本身還沒有逐次元證據。
14. ~~訊息系統~~ **完成**：`docs/re/14-messages.md`，`internal/sim/message.go`。
    含分區上限旗標、人口里程碑與八個劇本的勝敗條件。
15. ~~玩家工具~~ **完成**：`docs/re/15-tools.md`，
    `internal/sim/tool.go`、`internal/sim/connect.go`。
    自動接線用八座劇本城市驗證，15 447 格線路裡 99.83% 形狀一致。
16. 逐次元對拍：微實驗完全一致；分段對拍 23 段中 9 段完全一致。
    見 `docs/re/12-tick-parity.md`。剩下的差距多半是重建不出來的
    內部狀態，不一定是實作錯誤——**繼續縮小要靠新的微實驗**。
17. ~~`.PGF` 圖形版面~~ **完成**：`docs/formats/03-pgf-graphics.md`，
    `internal/assets/pgf.go`。24 個風格圖形檔（4 種顯示模式 × 6 種風格）
    全部解開，第 0 庫一律 **960 張地圖圖塊**——與 Micropolis 的 `TILE_COUNT`
    對得上，是圖塊編號的獨立佐證。
    下一步：呈現層（`internal/ui`，Ebiten）。
18. ~~軟體世界說明書轉錄~~ **操作手冊部分完成**：`docs/manual-cht/`。
    譯名表長出 118 條有一手依據的詞（工具、災難、選單、地圖圖層、預算、
    評估）。安裝步驟、密碼表與參考手冊的策略討論還沒轉錄。
19. ~~遊戲文字翻譯~~ **完成**：`internal/i18n/messages/`。
    基本檔 226 條 ＋ 六個風格包的覆寫（含各自改寫的圖片訊息與劇本簡介），
    合計 695 條。譯文來源在 `tools/i18n/`，`tools/i18n.sh` 合併。
20. ~~呈現層~~ **完成**：`internal/ui`（Ebiten）、`internal/game`（組裝層）。
    圖塊渲染、工具列、四個視窗、存讀檔、八個劇本、六種風格。
21. 正常玩家路徑試玩驗收、發行包、README 更新。

## 7. 現行驗證入口

見第 1 節「驗證入口」。
