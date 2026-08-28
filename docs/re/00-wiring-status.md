# 00 — 接線狀態

**每寫完一份機制筆記或格式文件，同一輪就要在這張表登記。**
`TestWiringStatus` 雙向都會紅：沒登記會紅、說「已接」卻沒有 Go 程式碼引用會紅、
說「未接」或「不適用」卻被引用了也會紅。

理由見 `CLAUDE.md` §0：前三道閘門各自都會綠，而結論仍然可以躺在筆記裡沒人用。
荒野遊俠踩過——公式解出來寫成筆記了，remake 三處還傳著寫死的常數，
而**編得過、測得過、玩得動**。

## 表格規則

- 第一欄是**儲存庫根目錄起算的路徑**，寫在反引號裡；連結可以是相對路徑。
- `已接`：至少一個 `internal/` 或 `cmd/` 底下的 `.go` 檔在註解裡引用那個路徑。
  引用要寫在**用到那個結論的地方**，不是集中在檔頭。
- `未接`：目前沒有任何 Go 程式碼引用它。理由欄要寫「在等什麼」。
- `不適用`：該文件講的是工具鏈或原版環境，本來就不會進 remake。

| 文件 | 主題 | 狀態 | 引用點／理由 |
|---|---|---|---|
| [`docs/re/00-source-map.md`](00-source-map.md) | Micropolis 原始碼地圖 | 不適用 | 索引文件，不含可實作的規則 |
| [`docs/re/01-oracle-harness.md`](01-oracle-harness.md) | oracle 建置與驅動 ＋ 第一批已確認常數 | **已接** | 工具鏈在 `tools/oracle/`；§4 的起始常數接在 `internal/sim/world.go` 的 `NewWorld` |
| [`docs/re/02-rng.md`](02-rng.md) | 亂數產生器 | **已接** | `internal/sim/rand.go` |
| [`docs/re/03-map-and-tiles.md`](03-map-and-tiles.md) | 地圖陣列與圖塊編碼 | **已接** | `internal/sim/world.go`、`tiles.go`（由 `tools/gen_tiles.py` 重產）|
| [`docs/re/04-terrain-generation.md`](04-terrain-generation.md) | 地形產生 | **已接** | `internal/sim/terrain.go`；四顆種子逐格對拍 |
| [`docs/re/05-power-scan.md`](05-power-scan.md) | 電力傳導 | **已接** | `internal/sim/power.go`；受控實驗 12000 格逐格對拍 ＋ 劇本 1 端到端 |
| [`docs/re/06-scans.md`](06-scans.md) | 四個逐格掃描 | **已接** | `internal/sim/scan.go`；收斂後三個平均值對拍 |
| [`docs/re/07-traffic-and-zones.md`](07-traffic-and-zones.md) | 交通生成與分區成長 | **已接** | `internal/sim/traffic.go`、`zone.go`、`mapscan.go`。⚠ 驗收只到強證據，見 12 |
| [`docs/re/08-disasters.md`](08-disasters.md) | 災難 | **已接** | `internal/sim/disaster.go`。⚠ 三種要靠精靈的災難是空實作 |
| [`docs/re/09-census-valves-budget.md`](09-census-valves-budget.md) | 普查、需求閥、稅收與預算 | **已接** | `internal/sim/census.go` |
| [`docs/re/10-evaluation.md`](10-evaluation.md) | 城市評分與投票 | **已接** | `internal/sim/eval.go` |
| [`docs/re/11-simulate-loop.md`](11-simulate-loop.md) | 十六相位主迴圈 | **已接** | `internal/sim/simulate.go` |
| [`docs/re/12-tick-parity.md`](12-tick-parity.md) | 整刻對拍工具與現況 | **已接** | `internal/sim/tickparity_test.go` |
| [`docs/formats/00-dos110-inventory.md`](../formats/00-dos110-inventory.md) | DOS 1.10 素材盤點 | 不適用 | 素材盤點，規則在別處 |
| [`docs/formats/01-city-file.md`](../formats/01-city-file.md) | 城市檔格式 | **已接** | `internal/sim/cityfile.go`、`scenario.go` |
| [`docs/formats/02-dos-lzss.md`](../formats/02-dos-lzss.md) | DOS 共用壓縮與四種資料檔 | **已接** | `internal/assets/lzss.go`、`ptf.go`、`psn.go` |
