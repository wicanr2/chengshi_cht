package sim

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// 表要和原版資料檔一致。七個訊息檔逐則相同，所以七個都核。
//
// 這支測試擋的是「手改表」：類別改錯不會讓任何東西壞掉，
// 只會讓災難時安靜、或者催蓋公園時響警笛。
func TestMsgClassMatchesOriginal(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..",
		"workplace", "dos110", "SIMCITY 1.10", "DATA")
	names, err := filepath.Glob(filepath.Join(dir, "*.PTF"))
	if err != nil || len(names) == 0 {
		t.Skip("沒有解開的 DOS 1.10 資料，跳過（玩家自備）")
	}
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		secs, err := assets.LoadPTF(raw)
		if err != nil {
			t.Fatalf("%s：%v", name, err)
		}
		for i := 0; i < len(msgClass); i++ {
			got := assets.MessageClass(secs[0], i)
			if got != msgClass[i] {
				t.Errorf("%s 訊息 %d：檔案是 %d，表是 %d",
					filepath.Base(name), i+1, got, msgClass[i])
			}
		}
	}
}

// 警笛的觸發集合。判準逐則列出來，不是「呼叫函式看它回什麼」——
// 那樣測不到表本身。
func TestWantsSiren(t *testing.T) {
	want := map[int]bool{
		20: true, 21: true, 22: true, 23: true, 24: true,
		25: true, 26: true, 27: true, 42: true, 43: true,
		32: false, // 爆炸：精靈自己播過段 1
		1:  false, 12: false, 29: false, 33: false, 41: false,
	}
	for n, w := range want {
		if got := WantsSiren(n); got != w {
			t.Errorf("訊息 %d：WantsSiren %v，要 %v", n, got, w)
		}
	}
}
