package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// 卡死偵測。
//
// 為什麼要有這個：試玩回報「蓋一段時間後畫面變雪花亂碼卡死，但音樂持續」
// （issue #1 第三項）。那個形狀——畫面停住而聲音還在——說明**別的
// goroutine 還活著，只有主迴圈不動了**。這種狀況跑掉之後什麼線索都不留，
// 六分鐘的壓力測試也沒重現，所以與其一直猜，不如讓遊戲自己在卡住的當下
// 把現場寫下來。
//
// 作法是最土的一種：`Update` 每一格敲一次心跳，另一條 goroutine 每秒看一次，
// 超過門檻沒動就把**所有 goroutine 的堆疊**與最後一次的狀態快照寫成檔案。
// 主迴圈卡在哪一行，堆疊裡就會有那一行。
//
// ⚠ **狀態快照不是在偵測時才讀的。** 從別的 goroutine 讀主迴圈正在改的欄位
// 是資料競爭，而且可能讀到寫到一半的值——「為了診斷當機而自己造出當機」。
// 所以由 `beat()` 在主迴圈裡把要記的東西複製到一個上鎖的小結構，
// 偵測那一側只讀那份副本。

// watchState 是主迴圈每一格留下的現場副本。只放純量，不放指標。
type watchState struct {
	mu     sync.Mutex
	frame  uint64
	at     time.Time
	fields []string
}

func (w *watchState) set(frame uint64, fields []string) {
	w.mu.Lock()
	w.frame, w.at, w.fields = frame, time.Now(), fields
	w.mu.Unlock()
}

func (w *watchState) get() (uint64, time.Time, []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.frame, w.at, w.fields
}

// beat 由 Update 在每一格開頭呼叫。除了敲心跳，也順手把現場記下來。
func (g *Game) beat() {
	if g.watch == nil {
		return
	}
	g.frameNo++
	// 每十格記一次就夠——卡死是以秒計的，而字串組裝每格都做太浪費。
	if g.frameNo%10 != 0 {
		g.watch.mu.Lock()
		g.watch.frame = g.frameNo
		g.watch.at = time.Now()
		g.watch.mu.Unlock()
		return
	}
	w := g.world
	fields := []string{
		fmt.Sprintf("城市：%s", w.CityName),
		fmt.Sprintf("年月：%d 年第 %d 月　資金：%d",
			1900+w.CityTime/48, (w.CityTime%48)/4+1, w.TotalFunds),
		fmt.Sprintf("住商工人口：%d/%d/%d　模擬速度：%d　城市時間：%d",
			w.ResPop, w.ComPop, w.IndPop, w.SimSpeed, w.CityTime),
		fmt.Sprintf("鏡頭：%d,%d　縮小：%d　工具：%d", g.camX, g.camY, g.zoom, int(g.tool)),
		fmt.Sprintf("視窗：%d　地圖關閉：%v　編輯在前：%v　選單：%d",
			int(g.win), g.mapClosed, g.editFront, g.openMenu),
		fmt.Sprintf("圖片訊息：%q　語言：%v", g.picture, g.lang),
	}
	g.watch.set(g.frameNo, fields)
}

// StartWatchdog 開始盯著主迴圈。after 是判定卡死的門檻，0 代表不啟用。
// dir 是寫報告的目錄（通常就是存檔目錄）。
func (g *Game) StartWatchdog(dir string, after time.Duration) {
	if after <= 0 {
		return
	}
	g.watch = &watchState{}
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		var lastFrame uint64
		var dumped bool
		for range tick.C {
			frame, at, fields := g.watch.get()
			if frame != lastFrame {
				lastFrame, dumped = frame, false
				continue
			}
			stall := time.Since(at)
			if stall < after || dumped {
				continue
			}
			dumped = true // 同一次卡死只寫一份，不要每秒生一個檔
			writeFreezeReport(dir, stall, frame, fields)
		}
	}()
}

// writeFreezeReport 把現場寫成檔案，同時印一行到 stderr。
//
// 檔案寫不出來也要印——玩家可能是在唯讀的目錄底下跑（AppImage 掛載點），
// 那時 stderr 是唯一的出路。
func writeFreezeReport(dir string, stall time.Duration, frame uint64, fields []string) {
	var b []byte
	add := func(format string, a ...any) { b = append(b, fmt.Sprintf(format, a...)...) }

	add("# 城市 —— 主迴圈停住了\n\n")
	add("時間：%s\n", time.Now().Format(time.RFC3339))
	add("停住多久：%.1f 秒（第 %d 格之後就沒有再進位）\n", stall.Seconds(), frame)
	add("Go：%s　平台：%s/%s　CPU：%d\n\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())

	add("## 最後一次的狀態\n\n")
	for _, f := range fields {
		add("  %s\n", f)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	add("\n## 記憶體\n\n")
	add("  堆疊使用 %d MiB　系統要到 %d MiB　GC %d 次　goroutine %d 條\n",
		m.HeapAlloc>>20, m.Sys>>20, m.NumGC, runtime.NumGoroutine())

	add("\n## 所有 goroutine 的堆疊\n\n")
	add("卡在哪一行就在下面。主迴圈那條通常叫 `ebiten` 或 `ui.(*Game).Update`。\n\n```\n")
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	b = append(b, buf...)
	add("\n```\n")

	name := fmt.Sprintf("chengshi-freeze-%s.log", time.Now().Format("20060102-150405"))
	path := name
	if dir != "" {
		path = filepath.Join(dir, name)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ 主迴圈停住 %.1f 秒，報告寫不出去（%v）。以下是全文：\n%s",
			stall.Seconds(), err, b)
		return
	}
	fmt.Fprintf(os.Stderr, "⚠ 主迴圈停住 %.1f 秒，現場已寫到 %s\n", stall.Seconds(), path)
}

// SetFreezeTest 讓主迴圈在跑滿 d 之後故意卡住，用來驗收卡死偵測本身。
//
// **內部用。** 沒有這個開關的話，「偵測器會不會動」只能等下一次真的卡死
// 才知道——而那正是最不該碰運氣的時候。故意製造一次可重現的卡死，
// 才驗得了報告寫不寫得出來、堆疊裡看不看得到主迴圈。
func (g *Game) SetFreezeTest(d time.Duration) {
	if d <= 0 {
		return
	}
	g.freezeAt = time.Now().Add(d)
}

// maybeFreeze 由 Update 呼叫。時間到就永遠不返回。
func (g *Game) maybeFreeze() {
	if g.freezeAt.IsZero() || time.Now().Before(g.freezeAt) {
		return
	}
	fmt.Fprintln(os.Stderr, "（-freeze-test：主迴圈從這裡開始故意不返回）")
	select {}
}
