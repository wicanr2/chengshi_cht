# Remake 設定 — CONFORMED

**日期：2026-08-31。範圍：語系設定。** 這是 remake 的玩家便利功能，原版 DOS
1.10 沒有多語設定，不能宣稱為原版行為。使用者已確認把入口放在 SYSTEM 選單底部。

## 玩家路徑

1. 玩家開啟 SYSTEM 下拉選單。
2. 原版十二列及其分隔線保持原順序；最下方另加分隔線與「設定」。
3. 選擇「設定」後開啟語言視窗，可選繁體中文、简体中文、日本語或 English。
4. 選擇後立即更新目前畫面的所有動態文字，並寫入使用者設定檔。
5. 下次啟動若沒有明確的 `-lang`，讀回上次語系；`-lang` 只覆蓋本次啟動，
   不在未經玩家選單操作時改寫設定。

## 儲存契約

- 路徑：`os.UserConfigDir()/chengshi/settings.json`；無法取得使用者設定目錄時，
  本次仍以繁體中文啟動，但不假裝已持久化。
- 格式：JSON 物件，`version` 固定為 1，`language` 只接受
  `zh_hant`、`zh_hans`、`ja`、`en`。
- 未知版本、未知語言、截斷或不合法 JSON 採失敗即關閉（fail-closed）：忽略該檔並
  回到繁體中文，不把壞值帶入 UI。
- 寫入先建立同目錄暫存檔，再以 rename 原子替換；檔案權限 `0600`、目錄 `0700`。
- 設定不寫進 `.cty`，不改變原版城市存檔格式，也不隨城市切換。

## 語言來源與權利邊界

- 繁體、簡體與日文來自版控內 TSV；日文未翻譯的敘事文字依既有退路顯示繁體。
- English 的原版訊息執行時取自玩家自備 `.PTF`；remake 自有介面文字來自 `ui.tsv`。
- 設定檔只存語言代號，不複製或散布原版文字。

## 接受條件

- 設定檔合法／缺失／損壞／未知語言都有單元測試。
- SYSTEM 原版各列的索引與動作不變，擴充列另有 UI 測試。
- 四種語言均可由正常 SYSTEM→設定路徑選取，截圖確認標題、選單與文字沒有裁切。
- 重啟測試證明未帶 `-lang` 時讀回設定，帶 `-lang` 時採命令列值。
- 原版畫面報告把 SYSTEM 標為「原版項目區同版面＋remake 擴充」，不再暗示整張選單
  與原版內容完全相同。

## 驗收收據

- 四語視窗：`workplace/shots/settings-verified-{zh-Hant,zh-Hans,ja,en}.png`。
- SYSTEM 設定列正常畫面：`workplace/shots/settings-after-fix-row.png`。
- UI 選取簡體後：`workplace/shots/settings-selected-zh-Hans.png`；產生的隔離設定檔為
  `workplace/settings-smoke-final/chengshi/settings.json`，權限 `0600`。
- 不帶 `-lang` 的第二個容器重啟：`workplace/shots/settings-restart-zh-Hans.png`，
  主畫面顯示簡體中文。
- 自動化限制：無頭 xdotool 從 SYSTEM 最後一次 Enter 沒有留下語言視窗畫面；因此
  入口以 SYSTEM 實際反白截圖＋完整狀態轉移測試驗證，持久化另用上述兩行程收據。
  不把 direct-entry 單獨當成正常玩家路徑證據。
