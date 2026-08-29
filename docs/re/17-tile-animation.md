# 17 — 圖塊動畫

**推論等級：已確認**（一手出處 `g_ani.c`、`headers/animtab.h`、`w_editor.c`）。
日期 2026-08-30。接線：`internal/sim/animate.go`、`internal/sim/anitab.go`、
`internal/ui/game.go`。

## 一、機制只有一張表加一個迴圈

```c
/* g_ani.c:67 */
animateTiles(void)
{
  tMapPtr = (unsigned short *)&(Map[0][0]);
  for (i = WORLD_X * WORLD_Y; i > 0; i--) {
    tilevalue = (*tMapPtr);
    if (tilevalue & ANIMBIT) {
      tileflags = tilevalue & ALLBITS;
      tilevalue &= LOMASK;
      tilevalue = aniTile[tilevalue];
      tilevalue |= tileflags;
      (*tMapPtr) = tilevalue;
    }
    tMapPtr++;
  }
}
```

`aniTile[1024]`（`headers/animtab.h`）是「下一格」表：每一格帶 `ANIMBIT`
的圖塊換成 `aniTile[圖塊編號]`，走完一圈就是一個循環。火是 56–63 八格、
雷達 832–839、噴泉 840–843、冒煙的煙囪四格一組、體育館的球場 932–947、
核電廠的漩渦 952–955。

## 二、三個會產生「自洽但錯」結論的地方

### 2.1 這支會改地圖，但**不是**模擬的一部分

呼叫點在 `w_editor.c:874`，也就是**畫編輯視窗的時候**：

```c
if (DoAnimation && SimSpeed && !heat_steps && !TilesAnimated) {
  TilesAnimated = 1;
  animateTiles();
}
```

四個條件都要成立：使用者開了動畫、**沒有暫停**、不是熱力學模式、
而且**這一輪還沒動過**（`TilesAnimated` 是那個閘——多個視窗開著時只做一次）。

所以它是**呈現層驅動的地圖寫入**。Go 版放在 `internal/sim`（因為它寫 `Map`），
但 `SimFrame` 不呼叫它，由 `internal/ui` 每個畫格呼叫一次。

**逐 frame 對拍不受影響**：oracle 一律 `sim Speed 0`，原版那一側也不會動。
反過來說，如果哪天把它搬進 `SimFrame`，四份對拍會立刻整片紅——那是好事，
表示護欄有效。

### 2.2 `animtab.h` 宣告 `[1024]` 卻只列 956 筆

其餘 68 筆**由 C 補 0**。這是語言規則，不是資料遺失——照補 0，
不要「維持原圖塊」。實際上也用不到：`TILE_COUNT` 是 960。

### 2.3 路面車流那一段有兩個版本

`animtab.h` 裡「Light Traffic」與「Heavy Traffic」各夾在 `#if 0`／`#else`
之間。`#if 0` 那一半**沒有編進去**，取 `#else` 之後的。
產生器 `tools/gen_anitab.py` 會處理，並在取出的筆數不合理時直接失敗。

## 三、與功能選單的對應

`DoAnimation` 就是功能選單的「加快市區景物活動」（訊息檔第 18 段第 6 筆）。
`World` 的預設值是 true（`sim.c:92`）。目前 remake 沒有把這個開關做成選單項，
記在 [`docs/spec/controls.md`](../spec/controls.md) 的差距表。

## 四、驗收

- `TestAniTabMatchesGenerator`：重跑產生器比對 `anitab.go`，防手改與封存脫節。
- `TestAnimateTiles`：不帶 `ANIMBIT` 的格子不動、旗標位元原樣保留、
  火的八格走完一圈回到原點。
- 逐 frame 對拍四份仍然全綠（證明它沒有混進規則層）。
