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
/* chengshi: 跑 N 個模擬 frame（預設 1），不經過事件迴圈。只加觀測手段。
   順便把抽樣拆成「SimFrame（規則）」與「MoveObjects（精靈）」兩段——
   逐 frame 對拍少抽一次時，第一件事就是問它在哪一邊。 */
extern unsigned int chengshi_rand_calls;
unsigned int chengshi_sf_draws, chengshi_mo_draws;

int SimCmdFrameStats(ARGS)
{
  if (argc != 2) {
    return (TCL_ERROR);
  }
  sprintf(interp->result, "%u %u", chengshi_sf_draws, chengshi_mo_draws);
  return (TCL_OK);
}

/* chengshi: 倒出場上所有精靈的完整狀態，一隻一行的欄位串在一起。
   載入城市之後精靈是**不可重建的狀態**：DoSimInit 的 MapScan 會依當下的
   亂數決定要不要生飛機／直昇機，而載入時 RandomlySeedRand 重設過種子，
   所以外面重建不出同一組。要逐 frame 對拍精靈，只能把它倒出來。
   欄位順序：type frame x y orig_x orig_y dest_x dest_y count sound_count
             dir new_dir step flag control turn accel speed */
int SimCmdSprites(ARGS)
{
  SimSprite *sp;
  char buf[4096];
  int n = 0;

  if (argc != 2) {
    return (TCL_ERROR);
  }
  for (sp = sim->sprite; sp != NULL; sp = sp->next) {
    if (!sp->frame) continue;
    if (n > 3500) break;
    n += sprintf(buf + n, "%s%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
		 n ? " ; " : "",
		 sp->type, sp->frame, sp->x, sp->y, sp->orig_x, sp->orig_y,
		 sp->dest_x, sp->dest_y, sp->count, sp->sound_count,
		 sp->dir, sp->new_dir, sp->step, sp->flag, sp->control,
		 sp->turn, sp->accel, sp->speed);
  }
  buf[n] = 0;
  /* ⚠ 不能寫進 interp->result：那是 199 位元組的固定緩衝
     （tcl.h 的 TCL_RESULT_SIZE），精靈一多就寫爆，症狀是整個直譯器
     開始亂跑、指令變得極慢。長字串一律走 Tcl_SetResult。 */
  Tcl_SetResult(interp, buf, TCL_VOLATILE);
  return (TCL_OK);
}


/* chengshi: w_sprite.c 的四個檔案層級全域，**載入城市時都不會重設**：
   Cycle    動畫計數器，DoCopter／DoPlane 用 `% 3`／`% 5` 決定要不要重算
            方向——也就是要不要抽亂數。
   absDist  GetDir 的副作用輸出。飛機用「上一次算出來的距離」判斷到了沒，
            而那一次可能是別隻精靈算的、甚至是上一座城市算的。
   CrashX/Y 墜機位置。 */
extern short Cycle;
extern int absDist;
extern short CrashX, CrashY;

extern unsigned int chengshi_sprite_draws[];

int SimCmdSpriteDraws(ARGS)
{
  char buf[128];
  int i, n = 0;

  if (argc != 2) {
    return (TCL_ERROR);
  }
  for (i = 0; i < 9; i++) n += sprintf(buf + n, "%s%u", i ? " " : "",
				       chengshi_sprite_draws[i]);
  buf[n] = 0;
  Tcl_SetResult(interp, buf, TCL_VOLATILE);
  return (TCL_OK);
}


int SimCmdSpriteCycle(ARGS)
{
  if (argc != 2) {
    return (TCL_ERROR);
  }
  sprintf(interp->result, "%d %d %d %d", Cycle, absDist, CrashX, CrashY);
  return (TCL_OK);
}


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
  chengshi_sf_draws = chengshi_mo_draws = 0;
  for (i = 0; i < 16; i++) chengshi_sprite_draws[i] = 0;
  for (i = 0; i < n; i++) {
    unsigned int a = chengshi_rand_calls;
    SimFrame();
    chengshi_sf_draws += chengshi_rand_calls - a;
    a = chengshi_rand_calls;
    MoveObjects();
    chengshi_mo_draws += chengshi_rand_calls - a;
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
extern short FireStMap[][SmY], PoliceMap[][SmY];
extern short PoliceMapEffect[][SmY], FireRate[][SmY];

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
  MEM_SHORT("FireStMap", FireStMap, SmX, SmY)
  MEM_SHORT("PoliceMap", PoliceMap, SmX, SmY)
  MEM_SHORT("PoliceMapEffect", PoliceMapEffect, SmX, SmY)
  MEM_SHORT("FireRate", FireRate, SmX, SmY)

#undef MEM_BYTE
#undef MEM_SHORT

  return (TCL_ERROR);
}


/* chengshi: 直接讀亂數產生器的內部狀態（低 24 位元）。
   載入劇本時 DoSimInit 自己會抽亂數，所以「載入前的狀態」是重建同一份
   起始地圖的必要條件——沒有它就只能事後把地圖蓋回去，而衍生陣列
   （地價、汙染…）已經是在不同的地圖上算出來的了。
   next 定義在 rand.c 且是 static，所以那邊也加了一個取值函式。 */
extern unsigned int chengshi_rand_state(void);

/* chengshi: 城市評估的問題表。逐 frame 對拍在評估那個 frame 分岔時，
   要先分清楚是「投票迴圈跑的次數不同」還是「別的地方多抽了」。 */
extern short ProblemTable[], ProblemVotes[], ProblemOrder[], ProblemTaken[];
extern short CityScore, CityYes, CityNo;

/* chengshi: 投票迴圈的抽樣計數（見 s_eval.c 的插入）。 */
extern unsigned int chengshi_vp0, chengshi_vp1, chengshi_dv0, chengshi_dv1;
extern int chengshi_vp_iters, chengshi_vp_z;

int SimCmdVoteStats(ARGS)
{
  if (argc != 2) {
    return (TCL_ERROR);
  }
  sprintf(interp->result, "%u %u %d %d",
	  chengshi_vp1 - chengshi_vp0, chengshi_dv1 - chengshi_dv0,
	  chengshi_vp_iters, chengshi_vp_z);
  return (TCL_OK);
}


int SimCmdProblems(ARGS)
{
  char buf[512];
  int i, n = 0;

  if (argc != 2) {
    return (TCL_ERROR);
  }
  n += sprintf(buf + n, "%d %d %d |", CityScore, CityYes, CityNo);
  for (i = 0; i < 10; i++) n += sprintf(buf + n, " %d", ProblemTable[i]);
  n += sprintf(buf + n, " |");
  for (i = 0; i < 10; i++) n += sprintf(buf + n, " %d", ProblemVotes[i]);
  n += sprintf(buf + n, " |");
  for (i = 0; i < 10; i++) n += sprintf(buf + n, " %d", ProblemTaken[i]);
  buf[n] = 0;
  Tcl_SetResult(interp, buf, TCL_VOLATILE);	/* 同上，可能超過 199 */
  return (TCL_OK);
}


int SimCmdRandState(ARGS)
{
  if (argc != 2) {
    return (TCL_ERROR);
  }
  sprintf(interp->result, "%u", chengshi_rand_state());
  return (TCL_OK);
}


'''

REG = ("  /* chengshi */ SIM_CMD(Frame);\n"
       "  /* chengshi */ SIM_CMD(Scycle);\n"
       "  /* chengshi */ SIM_CMD(Fcycle);\n"
       "  /* chengshi */ SIM_CMD(Valves);\n"
       "  /* chengshi */ SIM_CMD(Mem);\n"
       "  /* chengshi */ SIM_CMD(RandState);\n"
       "  /* chengshi */ SIM_CMD(Problems);\n"
       "  /* chengshi */ SIM_CMD(VoteStats);\n"
       "  /* chengshi */ SIM_CMD(FrameStats);\n"
       "  /* chengshi */ SIM_CMD(Sprites);\n"
       "  /* chengshi */ SIM_CMD(SpriteCycle);\n"
       "  /* chengshi */ SIM_CMD(SpriteDraws);\n")


EVALC_OLD = """  x = 0;
  z = 0;
  count = 0;
  while ((z < 100) && (count < 600)) {"""

EVALC_NEW = """  x = 0;
  z = 0;
  count = 0;
  chengshi_vp0 = chengshi_rand_calls;	/* chengshi */
  while ((z < 100) && (count < 600)) {"""

EVALC_OLD2 = """    count++;
  }
}"""

EVALC_NEW2 = """    count++;
  }
  /* chengshi */
  chengshi_vp1 = chengshi_rand_calls;
  chengshi_vp_iters = count;
  chengshi_vp_z = z;
}"""

EVALC_OLD3 = """  CityYes = 0;
  CityNo = 0;
  for (z = 0; z < 100; z++) {"""

EVALC_NEW3 = """  CityYes = 0;
  CityNo = 0;
  chengshi_dv0 = chengshi_rand_calls;	/* chengshi */
  for (z = 0; z < 100; z++) {"""

EVALC_OLD4 = """      CityNo++;
  }
}"""

EVALC_NEW4 = """      CityNo++;
  }
  chengshi_dv1 = chengshi_rand_calls;	/* chengshi */
}"""

EVALC_DECL = """
/* chengshi: 投票迴圈的抽樣計數。逐 frame 對拍在評估那個 frame 分岔時，
   要分清楚是投票迴圈跑的次數不同、拒絕取樣的次數不同，還是別處多抽了。 */
extern unsigned int chengshi_rand_calls;
unsigned int chengshi_vp0, chengshi_vp1, chengshi_dv0, chengshi_dv1;
int chengshi_vp_iters, chengshi_vp_z;
"""


def patch_eval(root):
    p = f"{root}/src/sim/s_eval.c"
    s = open(p, encoding="utf-8", errors="surrogateescape").read()
    if MARK in s:
        return
    for old, new in ((EVALC_OLD, EVALC_NEW), (EVALC_OLD2, EVALC_NEW2),
                     (EVALC_OLD3, EVALC_NEW3), (EVALC_OLD4, EVALC_NEW4)):
        if old not in s:
            sys.exit(f"s_eval.c 找不到插入點：{old[:40]!r}")
        s = s.replace(old, new, 1)
    s = s.replace("#include", EVALC_DECL + "\n#include", 1)
    open(p, "w", encoding="utf-8", errors="surrogateescape").write(s)
    print(f"已加上投票計數器：{p}")


RANDC = """

/* chengshi: 把 next 的低 24 位元開放出來給對拍用。
   遞迴式對 2^24 取模是封閉的，所以低 24 位元就是完整的狀態。 */
unsigned int
chengshi_rand_state()
{
	return ((unsigned int)(next & 0xFFFFFF));
}
"""

RANDC_COUNT_OLD = """int
sim_rand()
{
	next = next * 1103515245 + 12345;"""

RANDC_COUNT_NEW = """unsigned int chengshi_rand_calls = 0;	/* chengshi */

int
sim_rand()
{
	chengshi_rand_calls++;		/* chengshi */
	next = next * 1103515245 + 12345;"""


def patch_rand(root):
    p = f"{root}/src/sim/rand.c"
    s = open(p, encoding="utf-8", errors="surrogateescape").read()
    if MARK in s:
        return
    if RANDC_COUNT_OLD not in s:
        sys.exit("rand.c 找不到 sim_rand 的插入點")
    s = s.replace(RANDC_COUNT_OLD, RANDC_COUNT_NEW, 1) + RANDC
    open(p, "w", encoding="utf-8", errors="surrogateescape").write(s)
    print(f"已加上 chengshi_rand_state 與抽樣計數器：{p}")


# X11 移植版的**呈現層**有幾處和模擬共用同一個亂數產生器。逐 frame 對拍
# 會看到「原版莫名多抽一次」——而那一次是 UI 抽的，不是規則抽的。
# 把它們拿掉，oracle 才只反映規則層。（我們的 remake 呈現層不共用亂數。）
UI_RAND = [
    ("src/sim/s_msg.c",
     """    case  12:
      if (Rand(5) == 1) {
	MakeSound("city", "HonkHonk-Med");
      } else if (Rand(5) == 1) {
	MakeSound("city", "HonkHonk-Low");
      } else if (Rand(5) == 1) {
	MakeSound("city", "HonkHonk-High");
      }
      break;""",
     """    case  12:
      /* chengshi: 原版在這裡抽最多三次亂數挑喇叭聲。那是呈現層的決定，
         卻和模擬共用同一個產生器——逐 frame 對拍會看到原版多抽。 */
      break;"""),
    ("src/sim/w_map.c",
     """  for (dx = dy = i = 0; i < ShakeNow; i++) {
    dx += Rand(16) - 8;
    dy += Rand(16) - 8;
  }""",
     """  /* chengshi: 地震震動的位移原本從模擬用的亂數抽，改成固定 0。 */
  dx = dy = 0;"""),
    ("src/sim/w_editor.c",
     """  for (dx = dy = i = 0; i < ShakeNow; i++) {
    dx += Rand(16) - 8;
    dy += Rand(16) - 8;
  }""",
     """  /* chengshi: 同 w_map.c，震動位移不再消耗模擬的亂數。 */
  dx = dy = 0;"""),
]


SPRITE_OLD = """    if (sprite->frame) {
      switch (sprite->type) {"""

SPRITE_NEW = """    if (sprite->frame) {
      unsigned int chengshi_a = chengshi_rand_calls;	/* chengshi */
      switch (sprite->type) {"""

SPRITE_OLD2 = """      }
      sprite = sprite->next;
    } else {"""

SPRITE_NEW2 = """      }
      /* chengshi: 記下這一隻抽了幾次。逐 frame 對拍在精靈那一側分岔時，
         要先知道是哪一隻多抽的。 */
      if ((sprite->type >= 0) && (sprite->type < 16))
	chengshi_sprite_draws[sprite->type] += chengshi_rand_calls - chengshi_a;
      sprite = sprite->next;
    } else {"""

SPRITE_DECL = """
/* chengshi: 每一型精靈在這一次 sim Frame 裡各抽了幾次。 */
extern unsigned int chengshi_rand_calls;
unsigned int chengshi_sprite_draws[16];
"""


def patch_sprite(root):
    p = f"{root}/src/sim/w_sprite.c"
    s = open(p, encoding="utf-8", errors="surrogateescape").read()
    if MARK in s:
        return
    for old, new in ((SPRITE_OLD, SPRITE_NEW), (SPRITE_OLD2, SPRITE_NEW2)):
        if old not in s:
            sys.exit(f"w_sprite.c 找不到插入點：{old.splitlines()[0]!r}")
        s = s.replace(old, new, 1)
    s = s.replace("#include", SPRITE_DECL + "\n#include", 1)
    open(p, "w", encoding="utf-8", errors="surrogateescape").write(s)
    print(f"已加上逐隻精靈的抽樣計數：{p}")


def patch_ui(root):
    for rel, old, new in UI_RAND:
        p = f"{root}/{rel}"
        s = open(p, encoding="utf-8", errors="surrogateescape").read()
        if MARK in s:
            continue
        if old not in s:
            sys.exit(f"{rel} 找不到插入點：{old.splitlines()[0]!r}")
        s = s.replace(old, new, 1)
        open(p, "w", encoding="utf-8", errors="surrogateescape").write(s)
        print(f"已移除呈現層的亂數消耗：{rel}")


def main():
    root = sys.argv[1]
    patch_rand(root)
    patch_eval(root)
    patch_sprite(root)
    patch_ui(root)
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
