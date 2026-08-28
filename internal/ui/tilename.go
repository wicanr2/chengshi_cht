package ui

import "github.com/wicanr2/chengshi_cht/internal/sim"

// tileNameIndex 把圖塊編號換成訊息檔第 14 段的索引。
//
// 原版的查詢工具就是這樣做的：三十七個名稱涵蓋所有圖塊，靠一串範圍比較
// 分類。這裡照同一組範圍分。
//
// ⚠ 範圍要**由小到大**依序比，因為圖塊編號是連續配置的，
// 用 switch-case 逐一列舉會漏掉中間的變體（例如有車流的道路是 64–207，
// 只認 66 會讓塞車的路段變成「？？？」）。
func tileNameIndex(t int) int {
	switch {
	case t == 0:
		return 1 // 空地
	case t < sim.TREEBASE:
		return 2 // 水域
	case t <= sim.WOODS5:
		return 3 // 樹林
	case t <= sim.LASTRUBBLE:
		return 5 // 瓦礫
	case t <= sim.LASTFLOOD:
		return 6 // 水災
	case t == sim.RADTILE:
		return 7 // 輻射污染
	case t < sim.ROADBASE:
		return 8 // 火災
	case t < sim.POWERBASE:
		return 9 // 道路
	case t < sim.RAILBASE:
		return 10 // 電力線
	case t < sim.RESBASE:
		return 11 // 鐵軌
	case t < sim.HOSPITAL:
		return 12 // 住宅區
	case t < sim.CHURCH:
		return 13 // 醫院
	case t < sim.COMBASE:
		return 14 // 教堂
	case t < sim.INDBASE:
		return 15 // 商業區
	case t < sim.PORTBASE:
		return 18 // 工業用地
	case t < sim.AIRPORTBASE:
		return 19 // 海港
	case t < sim.COALBASE:
		return 20 // 機場
	case t < sim.FIRESTBASE:
		return 21 // 火力發電廠
	case t < sim.POLICESTBASE:
		return 22 // 消防隊
	case t < sim.STADIUMBASE:
		return 23 // 警察局
	case t < sim.NUCLEARBASE:
		return 24 // 體育館
	case t <= sim.LASTZONE:
		return 25 // 核能發電廠
	}
	return 36 // ？？？
}
