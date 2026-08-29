package sim

// 訊息類別。原版**不是**把「哪些訊息要響警笛」寫死在程式裡，而是每一則
// 訊息在資料檔裡帶一個類別位元組，發聲時查它：
//
//	if ((類別 == 6 || 類別 == 7) && 第一次顯示 && 訊息 != 32) PlaySound(3);
//
// 證據：`.PTF` 第 0 段每一則後面的 `FE xx`（docs/formats/04-ptf-messages.md §二）
// 與解壓映像 `0x14F8F` 的訊息派送常式（docs/re/16-dos-oracle.md §五之四）。
//
// 表放在這裡而不是執行期讀 `.PTF`：七個訊息檔（基本組 ＋ 六個風格包）的類別
// **逐則相同**，所以它是規則不是素材。`msgclass_test.go` 會拿原版檔核對，
// 手改會被擋下來。

// msgClass[n] 是訊息 n+1 的類別。索引 0 ＝ 訊息 1。
var msgClass = [49]int{
	2, 2, 2, 2, 2, 4, 2, 2,
	2, 3, 3, 3, 3, 3, 5, 3,
	4, 3, 3, 6, 6, 6, 7, 6,
	6, 6, 6, 3, 5, 3, 2, 6,
	9, 9, 5, 5, 5, 5, 5, 5,
	8, 6, 6, 9, 9, 8, 8, 8,
	8,
}

// 類別的語意。
const (
	MsgClassAdvice   = 2 // 建議蓋東西
	MsgClassProblem  = 3 // 城市問題
	MsgClassUrgent   = 4 // 要蓋發電廠、道路失修
	MsgClassNotice   = 5 // 停電、破產、人口里程碑
	MsgClassDisaster = 6 // 災難
	MsgClassQuake    = 7 // 大地震
	MsgClassTraffic  = 8 // 交通壅塞、不能推平
	MsgClassToolFail = 9 // 工具錯誤
)

// MessageClass 回傳訊息 n 的類別；查不到回 0。n 是 1 起算的訊息編號。
func MessageClass(n int) int {
	if n < 1 || n > len(msgClass) {
		return 0
	}
	return msgClass[n-1]
}

// WantsSiren 回報顯示訊息 n 時要不要響警笛（音效段 3）。
//
// ⚠ 爆炸（32）是特例：它自己的精靈已經播過段 1，再疊警笛就變成兩聲。
// 這個特例同時是「類別屬於前一則」那個對齊的證據
// （docs/re/16-dos-oracle.md §五之四）。
func WantsSiren(n int) bool {
	if n == MsgExplosion {
		return false
	}
	c := MessageClass(n)
	return c == MsgClassDisaster || c == MsgClassQuake
}
