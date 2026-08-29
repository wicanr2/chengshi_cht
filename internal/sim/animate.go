package sim

// 圖塊動畫。證據：docs/re/17-tile-animation.md／一手出處：g_ani.c:67 animateTiles()。

// AnimateTiles 把每一格帶 `ANIMBIT` 的圖塊換成 `aniTile[圖塊編號]`。
// 走完一圈就是一個動畫循環——火在燒、煙在冒、車在跑、雷達在轉、
// 噴泉在噴、體育館裡的人在動。
//
// ⚠ **這是呈現層驅動的，但它會改地圖。** 原版從 `doEditWindow`
// （`w_editor.c:874`）呼叫，條件是 `DoAnimation && SimSpeed && !heat_steps
// && !TilesAnimated`——也就是**每個畫格一次、暫停時不動、而且一輪只做一次**
// （多個視窗開著也只做一次，`TilesAnimated` 是那個閘）。
//
// 所以：
//
//   - 它**不在** `SimFrame` 裡，`internal/sim` 自己不會呼叫它；
//   - 逐 frame 對拍不受影響——oracle 跑的時候 `sim Speed 0`，
//     原版那一側也不會動；
//   - 呈現層要自己顧「暫停時不呼叫」與「一個畫格只呼叫一次」。
//
// 保留的旗標語意：`DoAnimation` 對應功能選單的「加快市區景物活動」
// （訊息檔第 18 段第 6 筆）。
func (w *World) AnimateTiles() {
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			t := w.Map[x][y]
			if t&ANIMBIT == 0 {
				continue
			}
			w.Map[x][y] = (t & ALLBITS) | aniTile[t&LOMASK]
		}
	}
}
