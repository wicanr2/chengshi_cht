package autoplay

// 需求封頂的接線測試。
//
// 這一支測的不是「策略夠不夠好」，是**策略有沒有回應模擬層送出的封頂訊號**。
// 兩件事要分開量，所以這裡把錢的因素拿掉——每年把資金補滿，讓失敗只可能
// 來自策略本身。
//
// 沒有這一支的時候，`ResCap`／`ComCap`／`IndCap` 被忽略了很久而完全沒被發現：
// 劇本測試只看通關與否，而封頂的症狀是「城市長到某個大小就不動了」——
// 在資金、評分、通關數上都看不出來，看起來只像「這個劇本比較難」。

import (
	"os"
	"testing"

	"github.com/wicanr2/chengshi_cht/internal/game"
	"github.com/wicanr2/chengshi_cht/internal/sim"
)

// TestDemandCapsLifted：把錢管夠，達斯維利六十年要長過第 3 級。
//
// 對照值：忽略封頂的那一版，給無限資金、跑滿一百二十年、蓋到 385 個分區，
// 人口一樣停在 25 000–35 000 之間震盪（第 2 級），因為 `RValve` 長期被
// `ResCap` 壓在 0。所以這個門檻分得開「有接」與「沒接」。
func TestDemandCapsLifted(t *testing.T) {
	dir := os.Getenv("SIMCITY_DATA")
	if dir == "" {
		dir = "../../workplace/dos110/SIMCITY 1.10"
	}
	w, err := game.LoadScenarioSeed(dir, 1, 1)
	if err != nil {
		t.Skipf("沒有原版資料，跳過：%v", err)
	}
	w.AutoBudget = true
	p := New(w, ScenarioGoal[1])
	ticks := 0
	for i := 0; i < 60*48*16; i++ {
		w.Frame()
		if w.CityTime > ticks && w.CityTime%48 == 0 {
			ticks = w.CityTime
			w.TotalFunds = 500000 // 只留策略這一個變因
			p.Year()
		}
	}
	if w.CityClass < 3 {
		t.Errorf("六十年後城市等級 = %d，人口 = %d；錢管夠還長不過第 3 級，"+
			"多半是三個需求封頂沒解（demandcap.go）", w.CityClass, w.Eval.CityPop)
	}
	// 三樣解封頂的建物至少要蓋出一座，否則上面那條可能是碰巧過的。
	if n := p.countTile(sim.STADIUM) + p.countTile(sim.FULLSTADIUM) +
		p.countTile(sim.PORT) + p.countTile(sim.AIRPORT); n == 0 {
		t.Error("六十年下來體育場、海港、機場一座都沒蓋")
	}
}
