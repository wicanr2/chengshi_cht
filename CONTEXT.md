# CONTEXT — 模擬城市重製與繁中化（chengshi_cht）

全專案的單一現況入口。接手時先讀本檔，再讀 [`CLAUDE.md`](CLAUDE.md)
與目標規格。

最後核對：2026-08-29。

## 1. 目前狀態

**第一個垂直切片已經走完四道閘門**（亂數：讀原始碼 → READY 規格 → Go 實作 →
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

## 5. 已被推翻的斷言

（目前沒有。這張表只放「已經知道錯、還不知道對」的；正確答案定案後刪掉該列。）

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
8. **`.cty` 存檔格式**（`s_fileio.c`，27120 bytes）→ `docs/formats/01`。
   解出來就能直接吃 X11 版的 24 個城市檔當測試資料，也才能載入八個劇本。
9. **電力傳導**（`s_power.c`）——它是分區成長的前提，而且是純函式，好對拍。
10. **每格掃描**（`s_scan.c`：人口密度、汙染、地價、犯罪）。
11. **交通生成**（`s_traf.c`）、**分區成長**（`s_zone.c`）、**稅收與預算**、
    **城市評分**（`s_eval.c`）、**災難**（`s_disast.c`）、**精靈**（`w_sprite.c`）、
    **工具**（`w_tool.c`）、**訊息**（`s_msg.c`）。
12. **軟體世界說明書逐頁轉錄** → `docs/manual-cht/`，同時長出 `translations/glossary.md`。
13. 呈現層（`internal/ui`）與 DOS 資料解碼（`internal/assets`）。

## 7. 現行驗證入口

見第 1 節「驗證入口」。
