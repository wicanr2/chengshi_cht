package ui

import "github.com/wicanr2/chengshi_cht/internal/sim"

// messageText 把模擬層的訊息編號換成中文。
//
// 用詞全部依 translations/glossary.md，那裡每一條都標了說明書頁碼。
// 說明書沒有收句子本身（它只列選單與工具），所以句子是本專案新寫的，
// 但**名詞一律用說明書的譯法**。
//
// 完整的 166 條訊息在 translations/messages/，那是原版 .PTF 解出來的
// 索引與英文原文；這裡先接上會實際觸發的那些。
func messageText(n int) string {
	switch n {
	case sim.MsgNeedRes:
		return "市民需要更多住宅區"
	case sim.MsgNeedCom:
		return "市民需要更多商業區"
	case sim.MsgNeedInd:
		return "市民需要更多工業用地"
	case sim.MsgNeedRoads:
		return "道路不足，市區交通不便"
	case sim.MsgNeedRail:
		return "該蓋鐵軌了"
	case sim.MsgNeedPower:
		return "全市缺電，請興建發電廠"
	case sim.MsgNeedStadium:
		return "市民要求興建體育館"
	case sim.MsgNeedSeaport:
		return "工業需要海港才能繼續成長"
	case sim.MsgNeedAirport:
		return "商業需要機場才能繼續成長"
	case sim.MsgPollutionHigh:
		return "污染太嚴重了"
	case sim.MsgCrimeHigh:
		return "犯罪率居高不下"
	case sim.MsgTrafficHigh:
		return "交通嚴重壅塞"
	case sim.MsgNeedFireStation:
		return "市區需要消防隊"
	case sim.MsgNeedPolice:
		return "市區需要警察局"
	case sim.MsgBlackout:
		return "部分地區停電"
	case sim.MsgTaxTooHigh:
		return "稅率過高，市民怨聲載道"
	case sim.MsgRoadsDeteriorat:
		return "道路年久失修"
	case sim.MsgFireDeptNeeds:
		return "消防隊經費不足"
	case sim.MsgPoliceNeeds:
		return "警察局經費不足"
	case sim.MsgPopTown:
		return "恭喜！本市人口突破兩千，升格為城鎮"
	case sim.MsgPopCity:
		return "恭喜！本市人口突破一萬，升格為城市"
	case sim.MsgPopCapital:
		return "恭喜！本市人口突破五萬，升格為首府"
	case sim.MsgPopMetro:
		return "恭喜！本市人口突破十萬，升格為大都會"
	case sim.MsgPopMegalop:
		return "恭喜！本市人口突破五十萬，成為超級都會"
	case sim.MsgScenarioWin:
		return "任務達成——市鑰是你的了"
	case sim.MsgScenarioLose:
		return "任務失敗"
	}
	return ""
}
