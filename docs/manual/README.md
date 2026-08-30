# 官方英文手冊（IBM PC 版）— 繁體中文整理

《SimCity: The City Simulator — User Documentation, DOS Version for IBM/Tandy & Compatibles》
（Maxis，1989／1991，ISBN 0-929750-01-2）。全文 OCR 在
`workplace/research/simcity-ibm-manual-ocr.txt`（不入版控），
原掃描件：<https://archive.org/details/simcity_ibm_manual>。

## 這個目錄放什麼、不放什麼

`CLAUDE.md` §1.1 把這本手冊列為**第 6 順位的規則輔助 oracle**——
它的用途是佐證機制，不是當規格。所以這裡放的是**機制章節的繁中整理**，
規則性敘述逐條翻出來並**與已讀出的原始碼對照**；不做全書逐頁複製。

理由有兩層：

1. **用途上**：手冊排在 Micropolis 原始碼與 DOS 資料檔之後。逐頁複製一本
   1989 年的操作手冊對規則層沒有增益，而**「手冊說 X，程式做 Y」的對照表**
   才是這一層真正要留下的東西。
2. **授權上**：這本手冊的版權頁明寫禁止複製與翻譯
   （`THIS MANUAL IS COPYRIGHTED. NO PORTION OF THIS MANUAL MAY BE COPIED,
   REPRODUCED, TRANSLATED…`）。軟體世界那本的逐頁轉錄有「保存台灣代理商
   一手史料」這個明確理由撐著（`CLAUDE.md` §3.1），而且掃描件本身不入版控；
   這本沒有同樣的理由。所以英文原文只在**當證據引用時**短句引用並標頁碼。

> 這是本專案著作權人的取捨，不是法律意見——與 `CLAUDE.md` §7 對
> Micropolis 授權張力的態度一致：講清楚立場，不粉飾。
> 要改成更完整的轉錄，是使用者的決定。

⚠ **OCR 有錯字。** 引用前要對原掃描件核字（`CLAUDE.md` §3.1）。
本目錄引用的每個英文句子都標了行號，可回 OCR 全文核對；
數字與表格另外標了是否已對過程式碼。

## 目錄

| 檔案 | 對應章節 | 狀態 |
|---|---|---|
| [`05-inside-simcity.md`](05-inside-simcity.md) | V. Inside SimCity — 模擬器怎麼運作 | 完成（規則層最有用的一章）|
| 其餘章節（I 導論、II 開始遊戲、III 教學、IV 使用參考、VI 城市史、VII 疑難、VIII 書目）| — | 未整理。IV 的操作說明與軟體世界說明書重疊，VI 是 Cliff Ellis 寫的城市規劃史散文（與規則無關）|

軟體世界的中文說明書轉錄在 [`../manual-cht/`](../manual-cht/)——
那一份是**逐頁完整轉錄**，性質與這裡不同。
