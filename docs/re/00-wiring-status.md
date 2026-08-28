# 00 — 接線狀態

**每寫完一份 `docs/re/NN`，同一輪就要在這張表登記。**
`TestWiringStatus` 雙向都會紅：沒登記會紅、說「已接」卻沒有 Go 程式碼引用會紅、
說「未接」卻被引用了也會紅。

理由見 `CLAUDE.md` §0：前三道閘門各自都會綠，而結論仍然可以躺在筆記裡沒人用。
荒野遊俠踩過——公式解出來寫成筆記了，remake 三處還傳著寫死的常數，
而**編得過、測得過、玩得動**。

## 表格規則

- `已接`：至少一個 `internal/` 或 `cmd/` 底下的 `.go` 檔案在註解裡引用該筆記的路徑。
  引用要寫在**用到那個結論的地方**，不是集中在檔頭。
- `未接`：目前沒有任何 Go 程式碼引用它。理由欄要寫「在等什麼」。
- `不適用`：該筆記講的是工具鏈或原版環境，本來就不會進 remake。

| 筆記 | 主題 | 狀態 | 引用點／理由 |
|---|---|---|---|
| [`00-source-map.md`](00-source-map.md) | Micropolis 原始碼地圖 | 不適用 | 索引文件，不含可實作的規則 |
| [`01-oracle-harness.md`](01-oracle-harness.md) | oracle 建置與驅動 | 不適用 | 工具鏈；產物在 `tools/oracle/` |
| [`02-rng.md`](02-rng.md) | 亂數產生器 | **已接** | `internal/sim/rand.go` |
| [`03-map-and-tiles.md`](03-map-and-tiles.md) | 地圖陣列與圖塊編碼 | **已接** | `internal/sim/world.go`、`internal/sim/tiles.go`（由 `tools/gen_tiles.py` 重產）|
