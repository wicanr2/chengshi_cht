#!/usr/bin/env python3
"""給 oracle 的原始碼加觀測用的指令。冪等，重跑不會疊。

用法：python3 tools/oracle/patches/apply.py <micropolis-activity 目錄>

**只加觀測手段，不動任何規則。** 每一處插入都用 `/* chengshi:` 開頭的
註解標記，方便日後 grep 出「我們動過哪裡」。

### `sim Frame N`

原版沒有單步指令，只有 `sim Speed 3` ／ `sim Speed 0`，中間跑掉幾個 frame
由 Tcl 的事件迴圈決定——所以分段對拍每一段的長度是不確定的（實測 85 到
2039 次抽樣都有）。有了 `sim Frame N`，「哪一個 frame 開始分岔」就從
搜尋變成查表。

`SimFrame` 開頭會在 `SimSpeed` 為 0 時直接返回，所以指令裡臨時把它設成 3，
跑完還原。`MoveObjects` 照原樣跟著呼叫（`sim_loop` 就是這個順序）。

### `sim Scycle` 與 `sim Fcycle`

兩個迴圈計數器從外面都觀察不到：`Scycle` 決定五個週期性掃描落在哪一刻
（`Scycle % 17／18／19／20／5`），`Fcycle & 15` 是十六相位主迴圈的相位。
加上唯讀存取子之後，對拍就不必再搜它們——那本來是搜尋空間的主要來源。

### `sim Valves`

三個需求閥門（`RValve`／`CValve`／`IValve`）決定分區長不長。逐 frame 對拍
分岔時，它們是第一個要比對的量——抽樣次數不同通常是因為某個分區的
`zscore` 不同，而 `zscore = RValve + EvalRes(...)`。
"""

import re
import sys

MARK = "/* chengshi:"

CMDS = r'''
/* chengshi: 跑 N 個模擬 frame（預設 1），不經過事件迴圈。只加觀測手段。 */
int SimCmdFrame(ARGS)
{
  int n = 1, i, saved;

  if ((argc != 2) && (argc != 3)) {
    return (TCL_ERROR);
  }
  if (argc == 3) {
    if ((Tcl_GetInt(interp, argv[2], &n) != TCL_OK) || (n < 0)) {
      return (TCL_ERROR);
    }
  }

  saved = SimSpeed;
  SimSpeed = 3;			/* SimFrame 在 SimSpeed 為 0 時會直接返回 */
  for (i = 0; i < n; i++) {
    SimFrame();
    MoveObjects();
  }
  SimSpeed = saved;

  sprintf(interp->result, "%d", n);
  return (TCL_OK);
}


/* chengshi: 唯讀的迴圈計數器存取子。Scycle 決定五個週期性掃描落在哪一刻
   （Scycle % 17／18／19／20／5），Fcycle & 15 是十六相位主迴圈的相位。
   兩個都是原版沒有開放的內部狀態，對拍本來只能用搜的。
   它們定義在 s_sim.c，這份原始碼是 K&R 風格、標頭裡沒有宣告。 */
extern short Scycle;
extern short Fcycle;

int SimCmdScycle(ARGS)
{
  if (argc != 2) {
    return (TCL_ERROR);
  }
  sprintf(interp->result, "%d", Scycle);
  return (TCL_OK);
}


/* chengshi */
int SimCmdFcycle(ARGS)
{
  if (argc != 2) {
    return (TCL_ERROR);
  }
  sprintf(interp->result, "%d", Fcycle);
  return (TCL_OK);
}


/* chengshi: 三個需求閥門。它們決定分區長不長，是逐 frame 對拍分岔時
   第一個要比對的量。原版只在圖表視窗裡間接顯示。 */
extern short RValve, CValve, IValve;

int SimCmdValves(ARGS)
{
  if (argc != 2) {
    return (TCL_ERROR);
  }
  sprintf(interp->result, "%d %d %d", RValve, CValve, IValve);
  return (TCL_OK);
}


/* chengshi: 讀衍生陣列的某一格。用法 `sim Mem <名稱> <x> <y>`。
   座標是**該陣列自己的座標系**（半解析度的傳 x>>1、y>>1，
   四分之一解析度的傳 x>>2、y>>2），與原始碼裡的索引方式一致。
   這些陣列決定分區的 zscore，是逐 frame 對拍分岔時要比對的下一層。 */
extern Byte *PopDensity[], *TrfDensity[], *PollutionMem[];
extern Byte *LandValueMem[], *CrimeMem[], *TerrainMem[];
extern short RateOGMem[][SmY];
extern short ComRate[][SmY];

int SimCmdMem(ARGS)
{
  int x, y;
  char *n;

  if (argc != 5) {
    return (TCL_ERROR);
  }
  n = argv[2];
  if ((Tcl_GetInt(interp, argv[3], &x) != TCL_OK) ||
      (Tcl_GetInt(interp, argv[4], &y) != TCL_OK)) {
    return (TCL_ERROR);
  }

#define MEM_BYTE(NAME, ARR, W, H)					\
  if (!strcmp(n, NAME)) {						\
    if ((x < 0) || (x >= (W)) || (y < 0) || (y >= (H))) return (TCL_ERROR); \
    sprintf(interp->result, "%d", (int)ARR[x][y]);			\
    return (TCL_OK);							\
  }
#define MEM_SHORT(NAME, ARR, W, H)					\
  if (!strcmp(n, NAME)) {						\
    if ((x < 0) || (x >= (W)) || (y < 0) || (y >= (H))) return (TCL_ERROR); \
    sprintf(interp->result, "%d", (int)ARR[x][y]);			\
    return (TCL_OK);							\
  }

  MEM_BYTE("PopDensity", PopDensity, HWLDX, HWLDY)
  MEM_BYTE("TrfDensity", TrfDensity, HWLDX, HWLDY)
  MEM_BYTE("PollutionMem", PollutionMem, HWLDX, HWLDY)
  MEM_BYTE("LandValueMem", LandValueMem, HWLDX, HWLDY)
  MEM_BYTE("CrimeMem", CrimeMem, HWLDX, HWLDY)
  MEM_BYTE("TerrainMem", TerrainMem, QWX, QWY)
  MEM_SHORT("RateOGMem", RateOGMem, SmX, SmY)
  MEM_SHORT("ComRate", ComRate, SmX, SmY)

#undef MEM_BYTE
#undef MEM_SHORT

  return (TCL_ERROR);
}


'''

REG = ("  /* chengshi */ SIM_CMD(Frame);\n"
       "  /* chengshi */ SIM_CMD(Scycle);\n"
       "  /* chengshi */ SIM_CMD(Fcycle);\n"
       "  /* chengshi */ SIM_CMD(Valves);\n"
       "  /* chengshi */ SIM_CMD(Mem);\n")


def main():
    root = sys.argv[1]
    p = f"{root}/src/sim/w_sim.c"
    # 原始碼是 ASCII，但保險起見用 surrogateescape，非法位元組原樣帶過去。
    s = open(p, encoding="utf-8", errors="surrogateescape").read()
    if MARK in s:
        print("已經加過了，跳過")
        return
    anchor = "int SimCmdSkips(ARGS)\n"
    if anchor not in s:
        sys.exit(f"找不到插入點 {anchor!r} —— 原始碼版本可能不同")
    s = s.replace(anchor, CMDS.lstrip("\n") + anchor, 1)
    reg_anchor = "  SIM_CMD(Speed);\n"
    if reg_anchor not in s:
        sys.exit("找不到指令註冊區")
    s = s.replace(reg_anchor, reg_anchor + REG, 1)
    open(p, "w", encoding="utf-8", errors="surrogateescape").write(s)
    print(f"已加上 sim Frame 與 sim Scycle：{p}")


main()
