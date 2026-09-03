# 隨附的城市地圖

五張地圖，兩個來源。都是 120×100 格的原版 `.cty` 格式，用
`-load` 直接讀。

```bash
chengshi -data "/path/to/SIMCITY 1.10" -load cities/TAIPEI.CTY
```

城市名讀自檔案的 128 位元組檔頭，所以標題列會顯示地名而不是預設名。

## 一、軟體世界的兩張（1990）

`TAIWAN.CTY` 與 `KAOHSIUN.CTY` 出自軟體世界研究中心（高雄）1990 年重新打包
發行的《SimCity Terrain Editor》磁片，是台灣代理商當年自己畫的台灣島與高雄
地形。與珍藏版 29 中文說明書同一個代理商。

同一片磁片上的 Maxis 程式、美術與示範城市**不在**本儲存庫內。
磁片逐檔盤點見 [`../docs/formats/00-e220-terrain-editor.md`](../docs/formats/00-e220-terrain-editor.md)。

### 權利狀態：誠實揭露

檔案內容是 27 248 個位元組的地形資料（128 位元組檔頭 ＋ 120×100 格的地圖），
不含任何 Maxis 的程式碼或美術；格式規格見
[`../docs/formats/01-city-file.md`](../docs/formats/01-city-file.md)。

台灣在 1992 年著作權法修正之前採註冊主義，原則上不保護無互惠關係的外國人著作，
因此軟體世界當年的重新打包發行在當時的台灣法下很可能是合法的。**但「當年在台灣
合法」與「今日可自由散布」是兩件事**：現行法下相關權利的歸屬與存續並未經法院認定。

收錄這兩個檔案是著作權人的判斷，不是法律意見。權利人如有異議請寄
<wicanr2@gmail.com>，本專案會立即移除。

The two files come from the 1990 Soft-World Research Center (Kaohsiung,
Taiwan) repackaging of the Maxis "SimCity Terrain Editor" disks. They are
27 248-byte terrain files drawn by the Taiwanese distributor; they contain
no Maxis code or artwork. Taiwan's pre-1992 copyright statute did not
generally protect works by foreign nationals without a reciprocity treaty,
so that repackaging was probably lawful under the law of the time — but
"lawful in Taiwan then" is not the same as "freely distributable now", and
no court has ruled on the present status. Including them is the copyright
holder's judgement, not legal advice. Rights holders with an objection may
write to wicanr2@gmail.com and the files will be removed.

這兩個檔案在 [`../LICENSE`](../LICENSE) 第 2 條 (c) 逐項點名為「含有原版成分」，
不在本專案的授權範圍內。

| 檔案 | 位元組 | SHA-256（前 16 碼）|
|---|---:|---|
| `TAIWAN.CTY` | 27248 | `f7106751e566fe7d…` |
| `KAOHSIUN.CTY` | 27248 | `b23948faf401b288…` |

## 二、本專案畫的三張

`TAIPEI.CTY`、`TAICHUNG.CTY`、`TAINAN.CTY` 是本專案自己畫的，
**可以自由散布**，不受上一節那個保留影響。

| 地圖 | 地形 |
|---|---|
| 台北 | 盆地被大屯、內湖南港、木柵新店與林口台地圍住；基隆河自東、新店溪自南、大漢溪自西南匯流成淡水河，往西北出海 |
| 台中 | 西為台灣海峽與台中港，海岸與盆地之間隔著大肚台地；大甲溪在北、大肚溪在南，東側是中央山脈山腳 |
| 台南 | 西岸是沙洲與潟湖（台江內海殘跡）與安平港；曾文溪、鹽水溪、二仁溪由東往西入海；東側是新化丘陵 |

**這是風格化的地形，不是測繪資料。** 120×100 格要放下一座城市的水系與地勢，
比例必然取捨過：河道加寬到看得見、丘陵化成連續的林地、海岸線簡化。
目標是玩起來認得出這是哪裡。

### 怎麼重產

```bash
python3 tools/gen_city_masks.py                       # 畫粗胚 → tools/maps/*.txt
go run ./tools/citymap tools/maps/taipei.txt cities/TAIPEI.CTY TAIPEI
```

粗胚只有三種字元（`.` 陸、`~` 水、`T` 林），岸線與林緣的圖塊交給引擎自己的
`smoothRiver`／`smoothTrees` 去長——那是原版 `s_gen.c` 的邊界規則，
所以產出的地形與原版的地形產生器同一套外觀。

形狀全部由固定種子的值雜訊推出來，不用亂數：同一份腳本每次畫出同一張圖。
