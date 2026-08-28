# 翻譯資料

| 檔案 | 內容 |
|---|---|
| [`glossary.md`](glossary.md) | 譯名表，唯一真相 |
| `../internal/i18n/messages/*.toml` | 遊戲文字的翻譯，**以「段落.索引」為鍵** |

⚠ **訊息翻譯檔放在 `internal/i18n/messages/`，不在這裡。**
`go:embed` 不能跨上層目錄，而遊戲要把文字編進執行檔（發行包不必帶
外部檔案）。放兩份會漂移，所以只留一份，正本在那邊。

## 為什麼以「段落.索引」為鍵

1. **原文不進版控。** 原版的訊息文字屬於原權利人（`CLAUDE.md` §8）。
   骨架只帶鍵與原文長度。
2. **原版 `.PTF` 本來就是分段的**：一段一個用途（狀態訊息、工具造價、
   月份、選單…）。用段落與索引當鍵，就直接對得上原版的結構，
   不必自己發明一套編號。見 [`docs/formats/04-ptf-messages.md`](../docs/formats/04-ptf-messages.md)。
3. **六個圖形集的同一個鍵講同一件事**，只是用詞隨主題不同
   （古代亞洲的「水井」就是基本檔的「發電廠」）。用鍵才對得起來。

## 譯文寫在哪

譯文的來源是 `tools/i18n/base_zh.py`（基本檔）與 `tools/i18n/styles_zh.py`
（六個風格包**與基本檔不同**的鍵）。`tools/i18n.sh` 把它們合併進 TOML。

為什麼不直接編輯 TOML：風格包只換掉一部分文字，其餘沿用基本檔。
直接編 TOML 的話同一句話要維護七份，改一個詞就會漏掉六處。

骨架由 `simtool` 產生：

```bash
tools/go.sh run ./cmd/simtool messages -dos "<DOS 1.10 目錄>" -out translations/messages
```

`len` 欄位是原文的位元組長度，給排版用：中文字寬是英數的兩倍，
譯文超過原文長度就要確認畫面裝得下（`CLAUDE.md` §3.3）。
