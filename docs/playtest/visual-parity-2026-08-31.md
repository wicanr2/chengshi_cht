# DOS 原版畫面對拍報告（2026-08-31）

## 結論

DOS 1.10 的主要玩家畫面已完成同狀態或版面對拍。能建立相同遊戲狀態的畫面採
逐像素／逐格判準；因語言、年份訂正或防拷流程不同而不能建立相同內容的畫面，
只判定幾何與可讀性，不把低相同比例誤稱為原版一致。

這份報告是目前唯一的畫面對拍狀態表。歷史截圖或舊收據若與本表衝突，以本表、
當日腳本重跑結果及 `CONTEXT.md` 為準。

## 輸入與正規化契約

- 原版：DOS 1.10 `SIMCITY.EXE`，SHA-256
  `66457cc456570b356bd44fa5c8617f27d3dffddd997e1591e97230f85a34189f`。
- 原版畫布：從 DOSBox 截圖裁切 `(192,184)` 起的 `640×350`。
- remake 畫布：以最近鄰由三倍視窗縮回 `640×350`，再正規化到最近的 EGA 色階。
- 一般介面差分：`tools/shot_diff_ui.py`；編輯區圖塊：
  `tools/shot_diff_cells.py`；完整重跑入口：`tools/screen_parity.sh`。
- 原版截圖與差分 PNG 含玩家自備原版素材，只留在已忽略的 `workplace/`，不進
  Git 或公開發行包。JSON 收據記錄來源路徑與 SHA-256。

## 涵蓋矩陣

| 玩家畫面 | 分類 | 目前結果 | 收據／判準 |
|---|---|---:|---|
| 招牌 | 同狀態 | 223849／224000（99.933%） | `workplace/visual-parity/ppf-title/report.json` |
| 劇本選單 | 同狀態 | 223867／224000（99.941%） | `workplace/visual-parity/ppf-scenario/report.json` |
| 編輯視窗（City Form 關閉） | 同狀態 | 502／512 格逐位元相同 | `tools/screen_parity.sh`；10 格全是兩側游標覆蓋 |
| City Form 在前 | 同狀態 | 最新重跑：視窗 130581／131600（99.226%）；地圖本體 107834／108300（99.570%）；游標取樣會小幅變動 | `workplace/screen-parity/city-form-report/report.json` |
| 四個下拉選單 | 僅版面 | OPTIONS／DISASTERS／WINDOWS 維持原版內容；SYSTEM 原版 0–12 列維持原序，底部另加 remake「設定」區 | `workplace/visual-parity/menu-*/report.json`；`workplace/shots/settings-after-fix-row.png` |
| 四語設定視窗 | remake 擴充 | 繁體／簡體／日文／英文均無裁切，開窗時反白目前語言 | `workplace/shots/settings-verified-*.png`；`docs/spec/settings.md` |
| 統計圖 | 僅版面 | 外框位置／大小與繪圖區已覆核 | `workplace/visual-parity/graphs-scen1/report.json` |
| 預算 | 僅版面 | 強制回應視窗位置／大小已覆核 | `workplace/visual-parity/budget-scen1/report.json` |
| 評估 | 僅版面 | 原錯誤高度 210 已修回 196 | `workplace/visual-parity/eval-layout/report.json` |
| 按住查詢面板 | 僅版面 | `(8,208)`、`168×114` 與原版相同 | `workplace/visual-parity/query-held/report.json` |
| 招牌→新城市 | 僅版面 | 對話框 88.301%；框外灰桌面 99.810% | `workplace/visual-parity/newcity-title-path/report.json` |
| 劇本簡介 | 僅版面 | 原版矩形、色彩與置中規格已實作，繁中七行無溢出 | `workplace/visual-parity/scenario-brief/report.json` |

「僅版面」的像素比例不是忠實度分數：英文與繁中每個字都會形成差分，統計資料
不同也會改變曲線與數字。這些畫面只以原版量測矩形、正常操作可達、文字不裁切及
相關幾何回歸測試作接受條件。

## 本輪由對拍發現並修正

- `WINDOWS→Maps`、`Ctrl-M` 與 `OpenWindow("maps")` 改為把既有 City Form 拉到
  最前面，不再開第二個全畫面地圖。
- 評估視窗高度由 210 修回 READY 規格的 196 原版像素。
- 招牌進入新城市時先畫原版灰桌面；遊戲內重新建城仍保留城市背景。
- 劇本簡介改用原版白底、藍框、紅字與置中按鈕，不再沿用一般深色訊息框。
- 穩定截圖改以連續兩張解碼像素完全相同才接受，避免把撕裂幀當證據。
- 舊對拍只比到 City Form 後方露出的 176 格，已改成關閉 City Form 後檢查完整
  32×16＝512 格，最低門檻提高到 490 格。

## 已確認的刻意差異

- remake 預設繁體中文；原版 oracle 是英文，因此文字區不要求逐像素相同。
- DOS 顯示年份的基準是 1849；remake 採資料與說明書一致的 1900。這是已記錄的
  玩家可見訂正，不把 DOS 顯示錯誤複製回來。
- 原版劇本簡介後有已破解且一律通過的防拷人口問題與通過訊息；remake 省略這兩幕。
- 縮小、多語與背景音樂是 remake 新增功能。音樂來自玩家自備的《SimCity 2000》
  DOS 封存，輸出 OGG 只留本機，不宣稱為 1989 原版配樂，也不進公開發行包。
- SYSTEM 選單底部的「設定」與四語視窗是 remake 擴充；對拍只主張其上方原版列的
  順序、文字來源與動作仍保持一致，不主張整張 SYSTEM 選單內容逐項相同。
- 火力／核能共用工具格的原版核能操作仍未知；remake 的副選單是明示的可用性補充。

## Oracle 未知與非阻塞限制

以下不是「remake 已證實等同原版」，也不以猜測填補：各浮動視窗邊框／捲軸的每一
像素、`Position`／`Resize` 的完整互動、OPTIONS 六項的全部原版語意、數值圖層第五級
門檻，以及原版如何從共用工具格選核能。統計圖年份標籤位置仍有已知小差異，列為
可選 polish；它不影響曲線資料、操作可達性或存檔。

此限制不否定本表已覆蓋的主玩家畫面；若未來要實作上述項目，必須先取得新原版證據，
依「證據 → DRAFT → READY → implementation → CONFORMED」流程另開窄任務。
