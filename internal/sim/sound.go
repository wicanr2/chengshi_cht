package sim

// 規則層發出的音效事件。規格：docs/spec/sound.md
//
// 為什麼放在規則層：原版的四段精靈音效（交通壅塞、爆炸、怪獸、船笛）
// 就是在精靈的移動常式裡播的，而那些常式的條件（`SoundCount` 歸零、
// `Rand16()&3 == 1`）本來就在這裡。把觸發點搬到呈現層等於要在兩邊各維護
// 一份相同的條件。
//
// 警笛與工具音**不在**這裡：警笛的判準是訊息檔裡的類別（呈現層才有訊息檔），
// 工具音的判準是玩家的點擊結果。兩者都由 internal/ui 送。
//
// ⚠ 這個佇列**不進存檔、不參與對拍**，而且一個亂數都不抽。
// 規則層的決定性是逐 tick 對拍的前提（CLAUDE.md §4）。

// 音效段編號。與 `.PSF` 的段序號一致，見 docs/re/16-dos-oracle.md §五之四。
const (
	SoundHeavyTraffic = 0
	SoundExplosion    = 1
	SoundMonster      = 2
	SoundSiren        = 3
	SoundShipHorn     = 4
	SoundToolSmall    = 5
	SoundToolBig      = 6
	SoundToolFail     = 7
)

// maxQueuedSounds 是佇列上限。
//
// 沒有上限的話，無頭長跑（對拍、自動玩家）會一路累積到記憶體用完——
// 那些跑法從來不取用佇列。溢出時丟**新的**，因為舊的才是先發生的。
const maxQueuedSounds = 16

// playSound 把一個音效事件排進佇列。
func (w *World) playSound(n int) {
	if len(w.soundQueue) >= maxQueuedSounds {
		return
	}
	w.soundQueue = append(w.soundQueue, n)
}

// TakeSounds 取走並清空佇列。呈現層每個畫格呼叫一次。
func (w *World) TakeSounds() []int {
	if len(w.soundQueue) == 0 {
		return nil
	}
	out := w.soundQueue
	w.soundQueue = nil
	return out
}
