# WORKLOG

逐輪的日期、命令、失敗與 checkpoint。**正文的現況寫在 `CONTEXT.md` 與
`docs/`，這裡只放流水帳**——被推翻的斷言、走過的死路、跑過的命令。

## 2026-08-30 — DOS 原版抽樣對拍

**目標**：把「可玩完成度」的驗收從 remake 自己的試玩腳本，換成與 DOS 原版對拍。

**產出**：`docs/re/18-dos-parity.md`、`cmd/simtool/dosparity.go`、
`tools/dos_parity.sh`、`tools/dosbox/act-scen-{load,run}.txt`。
commit `2f8b186`（未推送）。

**命令**：

```bash
tools/dos_parity.sh                 # ① 載入即存，八個劇本
MODE=run tools/dos_parity.sh        # ② 跑一段再存
tools/go.sh run ./cmd/simtool dosparity-scen 1 workplace/dosbox/save/run1.cty sweep=5
```

**踩到的坑（依發生順序）**：

1. **同一份存檔跑三次得到三個答案**（10224／10215／10223）。原因不是
   `Frame()` 不決定性，是載入路徑用時鐘播種，而 `DoSimInit` 的那次
   `MapScan` 會擲亂數動地圖——**載入完成後才重設種子已經來不及**。
2. **人口全部報「差 100%」**。讀 `w.ResPop` 讀到半份 census。改成唯讀重數。
3. **地價差 1212%**。剛載入的那份是拿重心 (0,0) 算的；補掃一次也不對，
   `PTLScan`／`CrimeScan` 不是冪等的（犯罪從 71 變 159）。三個平均值退出判準。
4. **第一版的取樣點只跑了一到四刻**，而數字很漂亮（96%–99%）。
   起因是一個誤判：把 `OVERWRITE` 確認框看成「Ctrl-S 沒反應」，
   於是整條管道被設計成「載入後立刻存、中間什麼都不能碰」。
5. **先催速度再開 Auto-Budget 會卡住**：第一個遊戲年兩三秒就到，預算對話框
   蓋住選單列，後面的 press 全落在對話框上。等 90 秒只前進一年。
6. **跑 120 秒回到標題畫面**：劇本時限到了，判定輸，`DoLoseGame` 直接踢出來。
7. **災難訊息對話框會擋掉 Ctrl-S**（「A Giant Tumbleweed has been sighted!!」），
   而且畫面上沒有錯誤，症狀只是「存檔對話框沒出現」。

**量到的結果**：載入層 96.4%–99.4% 逐格相同；真的跑起來之後地物格數仍逐項相同
或差不到 0.2%，但**商業人口與資金系統性偏低 25%–30%**（五個種子都成立）。
候選與下一步在 `docs/re/18-dos-parity.md` §六。

**還沒查**：原版狀態列顯示的年份與存檔裡的 `CityTime` 對不上
（東京存檔 2739＝1957，狀態列顯示 Feb 1906）。
