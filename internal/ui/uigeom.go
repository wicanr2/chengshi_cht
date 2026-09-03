package ui

// 六個顯示模式的介面美術格線。
//
// **為什麼要一張表**：remake 只有一套 640×350 的版面（`docs/spec/ui-layout.md`），
// 但六個模式的介面美術是各自為自己的螢幕畫的——原版 CEGA 是 640×350、
// sega 與 CGA 是 640×200、mcga 與 tdy 是 320×200。所以同一個工具盤，
// CEGA 是 57×182、tdy 是 56×123、CGA 是 55×120，**格子大小與列距全都不同**。
// 拿 CEGA 的格線去點 tdy 的美術會**點到隔壁那一格**，而畫面看起來只是
// 「圖小了一點」——這種錯玩家不會回報成 bug。
//
// **值怎麼來的**：每一格外面有一圈同色的框，所以「一整列（行）裡最長的
// 同色連續段長到接近整張圖寬（高）」的位置就是格線。六個模式的庫 2、4、5
// 全部這樣掃出來，再對照 CEGA 已知的量測值當正對照——CEGA 掃出來的
// 分隔列是 3、28、53、78、103、128、153、178（間距 25、第一格上緣 5），
// 與 2026-08 從 `workplace/gfx/bank02-00.png` 逐像素量到的完全相同。
// 推論等級：已確認（工具盤、統計圖、圖層圖示）／假說（需求長條的最大長度）。
//
// ⚠ mcga 的工具盤掃不出分隔列（它的框是 256 色漸層，沒有整列同色），
// 那一組是放大八倍逐像素看出來的：119 除以 7 剛好是 17，格子上緣貼齊。
type uiGeom struct {
	// 工具盤（庫 2）：2 欄 × 7 列。
	palXOff, palPitchX, palCellW int
	palYOff, palPitchY, palCellH int
	// 統計圖按鈕（庫 4）：2 欄 × 4 列。
	grfXOff, grfPitchX, grfCellW int
	grfYOff, grfPitchY, grfCellH int
	// 圖層圖示（庫 5）：九格，**排法每個模式不一樣**——CEGA 與 MONO 是
	// 直的一欄九列，sega／tdy／mcga 是兩欄五列（最後一格空著），
	// CGA 是**橫的九欄一列**。remake 的圖示欄只有 30 像素寬，
	// 塞不下兩欄，所以一律切成九張再排進自己那一欄。
	icnCols                      int
	icnXOff, icnPitchX, icnCellW int
	icnYOff, icnPitchY, icnCellH int
	// 需求指標（庫 3）：三根長條的左緣、往上長的底邊、往下長的頂邊，
	// 全部相對於庫 3 那張圖的左上角。
	demBarX            [3]int
	demUpBot, demDnTop int
	demUpMax, demDnMax int
}

// uiGeoms 以 `.PGF` 的模式碼為鍵。
var uiGeoms = map[byte]uiGeom{
	// CEGA 640×350：工具盤 57×182、統計圖 51×102、圖示 26×226、需求 46×39。
	// 這一組是 2026-08 就量好的，其餘五組以它當正對照。
	'E': {
		palXOff: 2, palPitchX: 29, palCellW: 26,
		palYOff: 5, palPitchY: 25, palCellH: 23,
		grfXOff: 0, grfPitchX: 25, grfCellW: 24,
		grfYOff: 2, grfPitchY: 25, grfCellH: 23,
		icnCols: 1,
		icnXOff: 0, icnPitchX: 0, icnCellW: 26,
		icnYOff: 0, icnPitchY: 25, icnCellH: 25,
		demBarX: [3]int{8, 21, 34}, demUpBot: 14, demDnTop: 24,
		demUpMax: 12, demDnMax: 11,
	},
	// sega 640×200：工具盤 56×123、統計圖 56×76、圖示 56×100（2×5）、需求 48×32。
	'e': {
		palXOff: 2, palPitchX: 26, palCellW: 25,
		palYOff: 3, palPitchY: 17, palCellH: 16,
		grfXOff: 2, grfPitchX: 26, grfCellW: 25,
		grfYOff: 3, grfPitchY: 18, grfCellH: 16,
		icnCols: 2,
		icnXOff: 3, icnPitchX: 26, icnCellW: 24,
		icnYOff: 4, icnPitchY: 19, icnCellH: 16,
		demBarX: [3]int{9, 22, 35}, demUpBot: 10, demDnTop: 20,
		demUpMax: 8, demDnMax: 8,
	},
	// Tandy 320×200：與 sega 同一份版面（兩者的介面庫尺寸逐項相同）。
	'T': {
		palXOff: 2, palPitchX: 26, palCellW: 25,
		palYOff: 3, palPitchY: 17, palCellH: 16,
		grfXOff: 2, grfPitchX: 26, grfCellW: 25,
		grfYOff: 3, grfPitchY: 18, grfCellH: 16,
		icnCols: 2,
		icnXOff: 3, icnPitchX: 26, icnCellW: 24,
		icnYOff: 4, icnPitchY: 19, icnCellH: 16,
		demBarX: [3]int{9, 22, 35}, demUpBot: 10, demDnTop: 20,
		demUpMax: 8, demDnMax: 8,
	},
	// mcga 320×200：工具盤 56×119、統計圖 56×76、圖示 56×97（2×5）、需求 48×32。
	'2': {
		palXOff: 2, palPitchX: 26, palCellW: 25,
		palYOff: 1, palPitchY: 17, palCellH: 16,
		grfXOff: 2, grfPitchX: 26, grfCellW: 25,
		grfYOff: 3, grfPitchY: 18, grfCellH: 16,
		icnCols: 2,
		icnXOff: 3, icnPitchX: 26, icnCellW: 24,
		icnYOff: 2, icnPitchY: 19, icnCellH: 17,
		demBarX: [3]int{9, 22, 35}, demUpBot: 10, demDnTop: 20,
		demUpMax: 8, demDnMax: 8,
	},
	// MONO 640×350：工具盤 56×182、統計圖 48×99、圖示 23×223（1×9）、需求 46×39。
	// 尺寸與 CEGA 同一級，只有寬度各少一兩個像素。
	'V': {
		palXOff: 2, palPitchX: 26, palCellW: 24,
		palYOff: 4, palPitchY: 25, palCellH: 23,
		grfXOff: 0, grfPitchX: 24, grfCellW: 23,
		grfYOff: 0, grfPitchY: 25, grfCellH: 23,
		icnCols: 1,
		icnXOff: 0, icnPitchX: 0, icnCellW: 23,
		icnYOff: 0, icnPitchY: 25, icnCellH: 23,
		demBarX: [3]int{8, 21, 34}, demUpBot: 15, demDnTop: 23,
		demUpMax: 13, demDnMax: 12,
	},
	// CGA Mono 640×200：工具盤 55×120、統計圖 60×77、
	// **圖示 242×22（橫的九欄一列）**、需求 48×25。
	'C': {
		palXOff: 2, palPitchX: 26, palCellW: 25,
		palYOff: 0, palPitchY: 17, palCellH: 16,
		grfXOff: 5, grfPitchX: 27, grfCellW: 25,
		grfYOff: 3, grfPitchY: 18, grfCellH: 16,
		icnCols: 9,
		icnXOff: 6, icnPitchX: 26, icnCellW: 25,
		icnYOff: 2, icnPitchY: 0, icnCellH: 18,
		demBarX: [3]int{10, 23, 36}, demUpBot: 8, demDnTop: 16,
		demUpMax: 6, demDnMax: 5,
	},
}

// geomFor 回傳一個模式碼的格線；沒登記的退回 CEGA 那一組。
// 退回不是「安全的預設值」——格線不對就會點錯格，所以呼叫端要確保
// 模式碼真的來自 `.PGF` 的檔頭。
func geomFor(mode byte) uiGeom {
	if g, ok := uiGeoms[mode]; ok {
		return g
	}
	return uiGeoms['E']
}

// icnPos 回傳第 i 張圖層圖示在庫 5 那張圖裡的左上角。
// 九張一律**由左到右、由上到下**數，與 CEGA 那一欄由上往下的順序相同。
func (u uiGeom) icnPos(i int) (int, int) {
	c, r := i%u.icnCols, i/u.icnCols
	return u.icnXOff + c*u.icnPitchX, u.icnYOff + r*u.icnPitchY
}
