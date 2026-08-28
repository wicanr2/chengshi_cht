# 14 — 訊息系統：不只是提示文字

**推論等級：已確認**（一手出處 `s_msg.c`、`s_sim.c`、`s_eval.c`）。
日期 2026-08-29。接線：`internal/sim/message.go`。

## 一、為什麼不能等到有 UI 再做

「訊息是顯示層的事，先跳過」——這個判斷是錯的。`SendMessages()`
（十六相位主迴圈的相位 10）除了送提示，還做三件會改變模擬結果的事：

1. **設定 `ResCap` / `ComCap` / `IndCap`**。這三個旗標分別壓住住宅、商業、
   工業的成長：`SetValves` 會把對應的閥門夾成 0（`s_sim.c:516`），
   `GetScore` 會把城市評分乘 0.85（`s_eval.c:294`）。
   少了它，「人口五百人以上沒有體育場」這條規則整條不存在。
2. **`CheckGrowth`** 判定人口里程碑，維護 `LastCityPop` 與 `LastCategory`。
3. **`DoScenarioScore`** 判定八個劇本的勝敗。

## 二、訊息埠：兩套不同的規則

只有**一個**訊息埠（`MessagePort`），一次放得下一則。編號的正負決定行為：

| 編號 | 意義 | 送出規則 |
|---|---|---|
| 正數 | 純文字 | 埠上有東西就丟掉——**先到先得** |
| 負數 | 有圖 | 覆蓋埠上的內容，但**同一張圖不會重送**（靠 `LastPicNum` 去重）|

`SendMesAt` 只有在 `SendMes` 真的送出去時才記座標。沒送出去卻記了座標，
「前往」按鈕會跳到不相干的地方。

## 三、十七個檢查輪流跑

`SendMessages` 用 `CityTime & 63` 當 switch，一個 64 刻的週期裡分散檢查
十七個狀況。這個設計有兩個後果：

- **同一個問題最快 64 刻才會再提醒一次**（約 1.33 遊戲年）。
- **檢查的是當下那一刻的狀態**，不是區間內的平均。人口在門檻附近震盪時，
  提不提醒取決於運氣。

三個設上限的 case（26 體育場、28 港口、30 機場）**在條件不成立時會清掉
旗標**。漏掉那個 else 分支的話，蓋好體育場之後住宅區永遠長不起來——
而且症狀是「城市莫名其妙停止成長」，很難聯想到訊息系統。

## 四、`CheckGrowth` 用的是第三種人口

```c
ThisCityPop = (ResPop + ComPop*8 + IndPop*8) * 20;
```

這和 `TakeCensus` 算的 `TotalPop`、以及畫面上顯示的 `CityPop` **都不一樣**。
同一款遊戲裡並存好幾個「人口」，數字對不上是正常的。
拿其中一個去校正另一個會得到自洽但錯的結論。

兩個容易漏的細節：

- 每四刻才檢查一次（`CityTime & 3`）。
- 五個門檻是**五個獨立的 `if`，後面覆蓋前面**。人口一次跨過好幾個門檻時
  只會發出最高的那一則，中間的等級悄悄跳過。

## 五、劇本勝敗：只判一次

`DoScenarioScore` 在 `ScoreWait` 倒數歸零的那一刻判定一次，之後不再判。
中途達標沒有用，中途失守也沒有關係。

| 劇本 | 過關條件 |
|---|---|
| 1 無聊鎮 Dullsville | `CityClass >= 4` |
| 2 舊金山 San Francisco | `CityClass >= 4` |
| 3 漢堡 Hamburg | `CityClass >= 4` |
| 4 伯恩 Bern | `TrafficAverage < 80` |
| 5 東京 Tokyo | `CityScore > 500` |
| 6 底特律 Detroit | `CrimeAverage < 60` |
| 7 波士頓 Boston | `CityScore > 500` |
| 8 里約熱內盧 Rio de Janeiro | `CityScore > 500` |

`ScoreWait` 的初值來自 `s_sim.c:384` 的 `ScoreWaitTab`，見
[`08-disasters.md`](08-disasters.md)。

**過關只送訊息，不呼叫 `DoWinGame`**——原版只有輸的時候叫
`DoLoseGame()`。贏的處理在 Tcl 那一層看到訊息編號才做。

## 六、汙染門檻的官方修改

```c
case 35:
  if (PolluteAverage > /* 80 */ 60)
```

註解裡留著舊值 80，實際用的是 60。**以程式碼為準**——這種
「註解記錄舊值」的寫法在這份原始碼裡不只一處，把註解當規格會錯。

## 七、亂數

`SendMessages` 這條路徑**不消耗亂數**。`s_msg.c` 裡唯一的三個 `Rand(5)`
在 `doMessage()` 的音效分支（訊息 12「交通壅塞」要挑三種喇叭聲），
那屬於呈現層。所以訊息系統上線不會改變
[`12-tick-parity.md`](12-tick-parity.md) 的對拍結果——實測也確認沒變。

## 八、中文化的接點

訊息文本不寫在程式裡。`message.go` 只有編號常數，文本在
`translations/messages/*.toml`，依索引對應。原版每個圖形組
（`asia`／`medi`／`west`／`moon`／`feur`／`fusa`）各有一份，
因為城市名與地標會隨風格改。
