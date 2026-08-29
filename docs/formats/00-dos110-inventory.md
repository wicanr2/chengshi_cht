# 00 — DOS 版 1.10 素材盤點

**推論等級：已確認**（逐檔讀 zip 目錄、算 SHA-256、對官方手冊附錄的磁片清單）。
核對日期 2026-08-29。來源檔 `SimCity_DOS_EN_v110.zip`（玩家自備，不入版控）。

## 一、檔案是三個年代疊起來的

zip 裡的 69 個檔案照時間戳分成四群，界線非常乾淨：

| 群 | 檔數 | 內容 | 判讀 |
|---|---:|---|---|
| **1991-05-01／05-04** | 29 | `SIMCITY.EXE`、`SETTINGS.EXE`、8 個劇本、`CEGA/` 與 `MONO/` 的基本圖形 ＋ `FEUR`／`FUSA`／`MOON` 三個圖形集、對應的 `DATA/*_MSG.PTF` 與 `*_SND.PSF`、`SOUNDDAT.V4`、`DATA/SOUNDDAT.PSF` | **原廠 1.10** |
| **1996-12-24** | 19 | `mcga/` 與 `sega/` 兩整套（含各自的 `*DAT`／`*NTRO`／`*SCEN`）＋ `read.me` | **"Knight Rider" 那批**：`read.me` 就在這一群，署名者自稱移除了防拷 |
| **2012-05-26** | 14 | `ASIA`／`MEDI`／`WEST` 三個圖形集的 `CEGA`／`MONO`／`DATA` 檔 ＋ `SIMCITY.CFG` ＋ 根目錄的 `SOUNDDAT.PSF` | **2012 年補齊的古城風情資料片** |
| 2015-01-13 | 0 | 只有目錄項目（打包當下建立）| 重新壓縮的時間 |

⚠ **時間戳有兩個欄位，不要只看一個。** `unzip -l` 顯示 1991-05-05，Python `zipfile`
顯示 1991-05-04——前者讀的是 zip 的擴充 UTC 時間戳欄位並套用時區，後者讀的是 DOS
本地時間欄位。**同一個檔案在兩個工具下會差一天**，判讀時要說明用的是哪一個
（本表用 DOS 欄位）。

### 由此得到的三條結論

1. **`SIMCITY.EXE` 與原廠其他檔同一批（1991-05-04），不在 1996 那一群。**
   `read.me` 說防拷已移除，但執行檔的時間戳沒有被動過。兩種可能：移除防拷時保留了
   時間戳，或者 `read.me` 講的是「這是已安裝好的版本」而不是「我改了 EXE」。
   **可驗證**：SimCity DOS 的防拷是翻手冊查詞型的，在 DOSBox 跑一次就知道會不會問。
   在驗證之前，這件事的等級是**未解**，不是「已被破解」。
2. **時間戳不等於內容被改。** `sega/`（EGA 低解析）整套是 1996 的時間戳，但官方手冊
   附錄的磁片清單**列了 `SEGADAT.PGF`**——它是原廠檔，只是重新打包時被 touch 過。
   反過來，2012 那批的 ASIA／MEDI／WEST 是原廠沒有的資料片。
   **判「這個檔被動過了嗎」要比內容，不能比時間戳。**
3. **`SOUNDDAT.PSF` 有兩份而且不同**：`DATA/SOUNDDAT.PSF`（1991，`10572c0b3de9…`）
   與根目錄的 `SOUNDDAT.PSF`（2012，`77cf45a12d96…`）。哪一份是遊戲實際讀的還沒查。

## 二、副檔名與檔名規則（官方手冊附錄，已確認）

IBM PC 版手冊的「LIST OF FILES ON THE DISKS」逐項列出：

| 副檔名 | 意義 |
|---|---|
| `.PGF` | Graphics File（圖形）|
| `.PPF` | Intro Screen 或 Scenario Menu Screen（整幅畫面）|
| `.PSN` | Scenario File（劇本）|
| `.V4` | Sound Data File |
| `.PTF` | 訊息文字（**手冊未列**，見下）|
| `.PSF` | 音效（**手冊未列**，見下）|

檔名規則是 `<圖形集><模式>`，`SIMCITY.CFG` 的 `Graphics Set: WESTCEGA` 直接印證。
基本組沒有圖形集前綴，寫成 `<模式>DAT`／`<模式>NTRO`／`<模式>SCEN`。

### 顯示模式代號（手冊逐字，已確認）

| 代號 | 手冊原文 | `SIMCITY.CFG` 的模式字母 |
|---|---|---|
| `CEGA` | EGA High-Res Color | `E` |
| `SEGA` | EGA **Low-Res** Color | `e` |
| `CGA` | CGA | `C` |
| `MONO` | Monochrome (for Hercules and EGA Mono) | `H` / `M` |
| `TDY` | Tandy Color | `T` |
| `MCGA` | **手冊未列** | `V` / `2` |

> `SEGA` 不是「Super EGA」也不是那家遊戲公司——是 **EGA 低解析彩色**。
> 只靠目錄名與檔案大小推得出這個結論，但手冊把它變成已確認。

### 圖形集前綴

| 前綴 | 圖形集 | 資料片 |
|---|---|---|
| `FUSA` | Future USA | 回到未來系列 |
| `FEUR` | Future Europe | 回到未來系列 |
| `MOON` | Moon Colony | 回到未來系列 |
| `ASIA` | Ancient Asia | 古城風情系列 |
| `MEDI` | Medieval | 古城風情系列 |
| `WEST` | Wild West | 古城風情系列 |

（前綴語意為**強證據**：由檔名 ＋ 骨灰集散地列出的兩套資料片名稱推得，
尚未從資料檔內容或執行檔字串直接證實。）

## 三、這份副本缺什麼

手冊清單列了、這份 zip **沒有**的檔案：

| 缺少 | 手冊寫的用途 |
|---|---|
| `CGADAT.PGF`／`CGANTRO.PPF`／`CGASCEN.PPF` | CGA 的圖形、開場、劇本選單 |
| `TDYDAT.PGF`／`TDYNTRO.PPF`／`TDYSCEN.PPF` | Tandy Color 的三個檔 |
| `INSTALL.EXE` | 安裝程式（`read.me` 說「已包含」，但實際不在 zip 裡）|
| `SIMCHEAT.EXE` | `read.me` 說附了，實際不在 zip 裡 |
| `MONONTRO.PPF`／`MONOSCEN.PPF` | 單色的開場與劇本選單 |
| `SEGANTRO`／`SEGASCEN` 的大寫版 | 只有 1996 的小寫 `sega/` 版本 |

**這是「查詢回空」的正對照**：不是沒找到，是真的沒有。`SIMCITY.CFG` 的模式清單
有 CGA 與 Tandy，但對應的資料檔不在——選了那兩個模式遊戲會找不到檔案。
補齊要另外找一份完整的原版磁片映像。

## 四、逐檔清單

見 `workplace/dos110/`（解開後）與本檔第一節的分群。
每個檔案的 SHA-256 前 12 碼記在 `workplace/research/` 的盤點輸出；
**要引用某個檔案的內容時，一律附完整 SHA-256**。

## 五、未解

| 項目 | 怎麼解 |
|---|---|
| `SIMCITY.EXE` 到底有沒有被改 | DOSBox 跑一次看會不會問手冊；另找一份未破解 1.10 比對 SHA-256 |
| ~~`.PTF`／`.PSF` 的內部格式~~ | **已解**：`04-ptf-messages.md`、`05-psf-sound.md` |
| 兩份 `SOUNDDAT.PSF` 哪一份被讀 | 反組譯 `SIMCITY.EXE` 的檔名字串，或 DOSBox 追檔案開啟。兩份只差第 2 段（`05-psf-sound.md` §4）|
| 圖形集前綴的語意 | 從 `.PGF` 解出圖塊後看內容，或找執行檔裡的字串 |

## 補充：圖形集前綴的語意（已解）

`ASIA`／`MEDI`／`WEST`／`FEUR`／`FUSA`／`MOON` 六個前綴是**城市風格**，
名稱寫在 `.PGF` 自己的檔頭裡，不必反組譯也不必猜：

| 前綴 | 風格名 | 風格編號 |
|---|---|---:|
| `ASIA` | Ancient Asia | 1234 |
| `MEDI` | Medieval Times | 1491 |
| `WEST` | Wild West | 1849 |
| `FUSA` | Future USA | 2055 |
| `FEUR` | Future Europe | 2155 |
| `MOON` | Moon Colony | 2195 |

每個 `.PGF` 的檔頭同時列出配套的 `*_MSG.PTF`、`*_SND.PSF` 與單色版圖形檔，
所以「哪些檔案屬於同一組」是檔案自己講的。細節見
[`03-pgf-graphics.md`](03-pgf-graphics.md)。
