# 兩片資料片（Graphics Set 1／2）— 盤點

Maxis 的兩套官方資料片：**Graphics Set 1 — Ancient Cities**（古城風情）與
**Graphics Set 2 — Future Cities**（回到未來）。玩家自備，不入版控、不進發行包
（[`CLAUDE.md`](../../CLAUDE.md) §8）。解開到 `workplace/addon/`。

六個圖形集本身**早就接進 remake 了**：`-style asia／medi／west／fusa／feur／moon`
（`cmd/chengshi/main.go`），而 1.10 那份副本已經帶了 CEGA／sega／mcga／MONO
四種模式的 `.PGF`，`DATA/` 底下六套 `_MSG.PTF` 與 `_SND.PSF` 也都在。
所以這兩片磁片對本專案的價值不在「多了六種風格」，在下面三件事。

## 一、補上 CGA 與 Tandy 的十二個圖形檔

1.10 那份副本**完全沒有** CGA 與 Tandy 的資料（`CLAUDE.md` §2.1 記著這個缺口，
基本集是靠 DOS 1.03 補的）。資料片把六套風格的這兩種模式都帶齊了：

`ASIACGA`／`ASIATDY`／`MEDICGA`／`MEDITDY`／`WESTCGA`／`WESTTDY`／
`FUSACGA`／`FUSATDY`／`FEURCGA`／`FEURTDY`／`MOONCGA`／`MOONTDY`（`.PGF`）

加上 1.03 補的基本集，**六種顯示模式 × 七套圖形集**現在只差
基本集以外沒有 `mcga` 的那幾個組合（1.10 有 mcga，1.03 沒有）。

## 二、`UPDATE.DAT` 是主程式的替換版（強證據）

兩片各帶一個 `UPDATE.DAT`，都是 **MS-DOS MZ 執行檔**，125 703 與 125 823 位元組
——與 1.10 的 `SIMCITY.EXE`（126 542）同一個量級。三者都含有同一條字串
`Initializing SimCity, please wait`，所以那是 SimCity 主程式，不是資料檔。

| 檔案 | 位元組 | 可讀字串 |
|---|---:|---:|
| 1.10 `SIMCITY.EXE`（破解重打包）| 126542 | 266 |
| 古城風情 `UPDATE.DAT` | 125703 | 237 |
| 回到未來 `UPDATE.DAT` | 125823 | 228 |
| 1.03 `SIMCITY.EXE`（未打包）| 192795 | 859 |

三顆 125–126 KB 的都被壓過，可讀字串只剩解壓器殘留的那十來條，
**不適合直接反組譯**；1.03 那顆沒打包，仍是目前最乾淨的目標。

但 `UPDATE.DAT` 有一個 1.10 沒有的性質：**它來自原廠資料片，不是破解版**。
`CLAUDE.md` §2.4 把「手上的 1.10 被動過」列為反組譯的前提問題，
這兩顆可以當對照組——**還沒比對過，等級：待驗**。

## 三、1990 年的原廠檔外面包了兩層後來的東西

時間戳分三層，判「這個檔被動過了嗎」要比內容不能只看日期：

| 層 | 檔案 | 時間戳 |
|---|---|---|
| 原廠資料 | 全部 `.PGF`／`.PTF`／`.PSF`／`.PPF`、`README` | 1989-10 – 1990-11 |
| 後來的安裝程式 | `INSTALL.EXE`、`SETTINGS.EXE` | 1992-05 / 1992-08 |
| 散布時塞進來的 | `SETUP.COM`、`SIM1–4.TXT`、`file_id.diz` | 1992-10 – 2011-08 |

⚠ **`SIM1–4.TXT` 不是原廠檔，也與 SimCity 無關**：內容是一組 ANSI 字元畫，
標題寫 `SIM CITY by ITALSOFT GAMES`，列的是世界都市人口排名，1993 年的檔。
是當年 BBS 散布時附上去的東西。

## 逐檔盤點

### 古城風情（Graphics Set 1 — Ancient Cities）

| 檔案 | 位元組 | 時間戳 | SHA-256（前 16 碼）| 1.10 有嗎 |
|---|---:|---|---|:-:|
| `ASIA_MSG.PTF` | 4608 | 1990-10-15 | `266490f9096ce3de…` | 有 |
| `ASIA_SND.PSF` | 14206 | 1990-09-24 | `5f149056c949ca94…` | 有 |
| `ASIACEGA.PGF` | 102144 | 1990-10-12 | `5a1366d3641ea9be…` | 有 |
| `ASIACGA.PGF` | 21119 | 1990-10-01 | `ffe8b23a1a610a01…` | **新** |
| `ASIAMCGA.PGF` | 44612 | 1990-10-10 | `dacef21d09867de3…` | 有 |
| `ASIAMONO.PGF` | 37534 | 1990-10-10 | `d540be8a9eb5f000…` | 有 |
| `ASIASEGA.PGF` | 35829 | 1990-09-27 | `507f39d456140502…` | 有 |
| `ASIATDY.PGF` | 31148 | 1990-10-08 | `5e8c921c34e7ac4e…` | **新** |
| `file_id.diz` | 236 | 2011-01-24 | `a6f9ea33e953012e…` | **新** |
| `INSTALL.EXE` | 27144 | 1992-05-19 | `bb003baae80a6fab…` | **新** |
| `MCGADAT.PGF` | 41671 | 1990-03-16 | `884e24fd716f7a82…` | 有 |
| `MCGANTRO.PPF` | 21955 | 1989-11-06 | `53ac1f069490339a…` | 有 |
| `MCGASCEN.PPF` | 10772 | 1989-10-29 | `799e3e2aa51d1f5c…` | 有 |
| `MEDI_MSG.PTF` | 4885 | 1990-10-15 | `a76ecc76567c1c50…` | 有 |
| `MEDI_SND.PSF` | 15776 | 1990-09-24 | `8cab42e6ba88284a…` | 有 |
| `MEDICEGA.PGF` | 110468 | 1990-10-15 | `3de9c7f31b34dc2e…` | 有 |
| `MEDICGA.PGF` | 21690 | 1990-09-27 | `731639dd33e03a06…` | **新** |
| `MEDIMCGA.PGF` | 52235 | 1990-09-27 | `a2c398e9652e6ee1…` | 有 |
| `MEDIMONO.PGF` | 44359 | 1990-10-10 | `1f8b52abc2cb56cb…` | 有 |
| `MEDISEGA.PGF` | 45383 | 1990-09-17 | `e0c1e453c13c9603…` | 有 |
| `MEDITDY.PGF` | 39556 | 1990-10-08 | `e4f25addc18e78c2…` | **新** |
| `MESSAGE.PTF` | 4885 | 1990-10-15 | `cfc1a72e72e86cd5…` | 有 |
| `README` | 657 | 1990-11-12 | `214f3c83e55bff41…` | **新** |
| `SETTINGS.EXE` | 23458 | 1992-05-19 | `380a996b6a98cc4a…` | 有 |
| `SOUNDDAT.PSF` | 17951 | 1990-09-19 | `10572c0b3de9199b…` | 有 |
| `UPDATE.DAT` | 125703 | 1991-01-03 | `6cfb9faf2be44951…` | **新** |
| `WEST_MSG.PTF` | 5091 | 1990-10-15 | `d95e6c1faf26732b…` | 有 |
| `WEST_SND.PSF` | 16160 | 1990-09-24 | `5fe13f32fa3acff4…` | 有 |
| `WESTCEGA.PGF` | 92882 | 1990-09-27 | `2ae214e9c55180af…` | 有 |
| `WESTCGA.PGF` | 17193 | 1990-10-01 | `91e49782fa72a856…` | **新** |
| `WESTMCGA.PGF` | 45020 | 1990-09-27 | `a0b3cbd7544cfd56…` | 有 |
| `WESTMONO.PGF` | 33010 | 1990-10-10 | `17d05f1d1b882dca…` | 有 |
| `WESTSEGA.PGF` | 36893 | 1990-09-27 | `e9510035a981b266…` | 有 |
| `WESTTDY.PGF` | 32849 | 1990-10-08 | `6467c262e706c30f…` | **新** |

### 回到未來（Graphics Set 2 — Future Cities）

| 檔案 | 位元組 | 時間戳 | SHA-256（前 16 碼）| 1.10 有嗎 |
|---|---:|---|---|:-:|
| `FEUR_MSG.PTF` | 5099 | 1990-11-12 | `9e1579a92b87fb38…` | 有 |
| `FEUR_SND.PSF` | 15537 | 1990-09-24 | `ea38a08347b45793…` | 有 |
| `FEURCEGA.PGF` | 110866 | 1990-09-20 | `ff8fdce630f3c1cc…` | 有 |
| `FEURCGA.PGF` | 20399 | 1990-09-14 | `15fd899f64ee11cc…` | **新** |
| `FEURMCGA.PGF` | 50125 | 1990-10-09 | `0cec1826a330eeb4…` | 有 |
| `FEURMONO.PGF` | 40199 | 1990-10-10 | `aeb48da8bf21004f…` | 有 |
| `FEURSEGA.PGF` | 43681 | 1990-09-13 | `118f10cd9670ab40…` | 有 |
| `FEURTDY.PGF` | 38928 | 1990-10-08 | `c1df52eec045aed1…` | **新** |
| `file_id.diz` | 252 | 2011-08-09 | `c959ebb9d3cc5c16…` | **新** |
| `FUSA_MSG.PTF` | 5346 | 1990-10-15 | `db6c5b289f144460…` | 有 |
| `FUSA_SND.PSF` | 14538 | 1990-09-24 | `fe138e9a1e237a9f…` | 有 |
| `FUSACEGA.PGF` | 86119 | 1990-10-01 | `f8655b4ec3daaa4a…` | 有 |
| `FUSACGA.PGF` | 17793 | 1990-09-05 | `84fbe36755b17ba5…` | **新** |
| `FUSAMCGA.PGF` | 44705 | 1990-09-12 | `8716b6ae432fb766…` | 有 |
| `FUSAMONO.PGF` | 35501 | 1990-10-10 | `8b64f17739d96cdf…` | 有 |
| `FUSASEGA.PGF` | 34965 | 1990-09-12 | `4043f264a10d488e…` | 有 |
| `FUSATDY.PGF` | 31834 | 1990-10-08 | `300689f0d68683de…` | **新** |
| `INSTALL.EXE` | 27144 | 1992-08-04 | `b94f2f73797abbca…` | **新** |
| `MCGADAT.PGF` | 41671 | 1990-03-16 | `884e24fd716f7a82…` | 有 |
| `MCGANTRO.PPF` | 21955 | 1989-11-06 | `53ac1f069490339a…` | 有 |
| `MCGASCEN.PPF` | 10772 | 1989-10-29 | `799e3e2aa51d1f5c…` | 有 |
| `MESSAGE.PTF` | 5354 | 1990-08-19 | `deb1a33a2b4a811e…` | 有 |
| `MOON_MSG.PTF` | 5372 | 1990-11-12 | `870dfe9f37398dfd…` | 有 |
| `MOON_SND.PSF` | 17783 | 1990-09-24 | `3ae88a132127f852…` | 有 |
| `MOONCEGA.PGF` | 86744 | 1990-10-11 | `1e8667527bb0b406…` | 有 |
| `MOONCGA.PGF` | 19060 | 1990-09-27 | `bd320140ab0f711e…` | **新** |
| `MOONMCGA.PGF` | 42566 | 1990-10-11 | `f1f5bb5e582a8d4b…` | 有 |
| `MOONMONO.PGF` | 37420 | 1990-10-15 | `8bc9a5f92537782c…` | 有 |
| `MOONSEGA.PGF` | 35476 | 1990-10-11 | `6b989597b08a58bb…` | 有 |
| `MOONTDY.PGF` | 32580 | 1990-10-11 | `8f875cd2f6c63df3…` | **新** |
| `README.___` | 657 | 1990-11-12 | `214f3c83e55bff41…` | **新** |
| `SETTINGS.EXE` | 23911 | 1992-08-04 | `7bf667152315d19d…` | 有 |
| `SETUP.COM` | 1469 | 1992-10-20 | `58091c019a73c0ad…` | **新** |
| `SIM1.TXT` | 5528 | 1993-04-07 | `cac2f431c37c61ee…` | **新** |
| `SIM2.TXT` | 5512 | 1993-04-07 | `28588dc9d368e5d6…` | **新** |
| `SIM3.TXT` | 5452 | 1993-04-07 | `56cda24fef8dbacb…` | **新** |
| `SIM4.TXT` | 5463 | 1993-04-07 | `77611bfae9f4f49f…` | **新** |
| `SOUNDDAT.PSF` | 17951 | 1989-10-16 | `10572c0b3de9199b…` | 有 |
| `UPDATE.DAT` | 125823 | 1990-11-11 | `480eca218bfb5d44…` | **新** |