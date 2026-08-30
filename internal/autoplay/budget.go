package autoplay

import "github.com/wicanr2/chengshi_cht/internal/sim"

// 年度預算：把一年能動用的錢**先分配好**，再讓各項去花。
//
// ⚠ 為什麼需要這個。原本 `year()` 是一串各自貪心的步驟：接線、蓋警消、
// 種公園、劃分區，每一項自己判斷「錢夠不夠」然後花到自己滿意為止。
// 排在前面的把錢拿光，排在後面的就一毛都沒有——而**誰排前面是寫死的**，
// 不隨局面變。三個劇本的卡點都是這個形狀：
//
//   - 從零開的新城市：警消排在成長前面，一年一座，十四年把錢吃光，
//     城市停在 33 個分區（見 bootstrap.go）。
//   - 舊金山：電廠排在接線前面，按暗區比例一年蓋四座 $12 000，
//     第三年破產，而真正需要的是 $5 一格的電線（見 wire.go）。
//   - 達斯維利：分區排在最後，前面幾項花完就沒錢劃分區了。
//
// 分配的原則是**每一塊錢帶來多少成長**：
//
//  1. **供電容量**不受額度限制。撞上限的話 `DoPowerScan` 整個中止，
//     城市會死——這不是投資，是保命。
//  2. **接回暗區**：一格暗區就是一份停產的資產，而電線一格只要 $5。
//     單位成本最低，額度給四分之一。
//  3. **成長**（分區與路）：稅收的來源，額度給一半。
//  4. **服務**（警消）：壓犯罪與火災，但**維護費每年都要付**，
//     額度給四分之一，而且座數另有上限（見 services）。
//  5. **公園**：剩下的零錢。一格 $10，拉地價很划算，但沒有它不會死。

// purse 是某一項這一年的支出額度。
//
// 它不預先扣錢，而是記下起始餘額，用「花掉多少」反推還剩多少額度——
// 因為每一次 `ApplyTool` 的實際花費（整地、橋、海底電纜）事前算不準。
type purse struct {
	w     *sim.World
	start int
	limit int
}

func (p *Player) purse(limit int) *purse {
	if limit < 0 {
		limit = 0
	}
	return &purse{w: p.w, start: p.w.TotalFunds, limit: limit}
}

// spent 是這一項到目前為止花掉的錢。
func (s *purse) spent() int { return s.start - s.w.TotalFunds }

// left 是還剩多少額度。
func (s *purse) left() int { return s.limit - s.spent() }

// ok 判斷還有沒有額度，順便擋住「存款已經低於準備金」的情況。
func (s *purse) ok(reserve int) bool {
	return s.left() > 0 && s.w.TotalFunds > reserve
}
