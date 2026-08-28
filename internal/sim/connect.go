package sim

// 道路／鐵路／電線的自動接線。證據：docs/re/15-tools.md／一手出處：w_con.c
//
// 玩家點一格路，遊戲要自己判斷該畫成直的、彎的、丁字還是十字，而且要
// **回頭修四個鄰居**。這一整套就是 ConnecTile。
//
// ⚠ 原版全程用指標算術：`TileAdrPtr[WORLD_Y]` 是右邊那一格
// （地圖配置成一整塊，每欄 WORLD_Y 個 short），`TileAdrPtr[1]` 是下面。
// 看到 `[1]` 會直覺以為是「下一格」，其實方向是 **+y**；`[WORLD_Y]` 才是 +x。
// 讀錯這一點，接線表的索引位元會整組轉九十度，而且畫面看起來「幾乎對」。

// 三張接線表：四個鄰居的連通位元組成 0..15 的索引，查出該用哪個圖塊。
// 位元順序是 上=1、右=2、下=4、左=8。w_con.c:65
var (
	roadTable = [16]int{
		66, 67, 66, 68,
		67, 67, 69, 73,
		66, 71, 66, 72,
		70, 75, 74, 76,
	}
	railTable = [16]int{
		226, 227, 226, 228,
		227, 227, 229, 233,
		226, 231, 226, 232,
		230, 235, 234, 236,
	}
	wireTable = [16]int{
		210, 211, 210, 212,
		211, 211, 213, 217,
		210, 215, 210, 216,
		214, 219, 218, 220,
	}
)

// neutralizeRoad 把「有車流的路」還原成「空路」再比較。w_con.c:87
//
// 道路圖塊 64..207 依車流量分成三段（無車、輕度、重度），低四位是形狀。
// 判斷連通性時只看形狀，所以先把車流資訊抹掉。**忘了做這一步的話，
// 塞車的路段會被判成不能接**——而且只在車多的時候才出錯。
func neutralizeRoad(t int) int {
	t &= LOMASK
	if t >= 64 && t <= 207 {
		return (t & 0x000F) + 64
	}
	return t
}

// 接線指令。w_con.c:96 ConnecTile 的第四個參數。
const (
	ConnFixZone = 0 // 只修形狀，不放東西
	ConnDoze    = 1
	ConnRoad    = 2
	ConnRail    = 3
	ConnWire    = 4
)

// ConnecTile 是四種放置的共同入口。w_con.c:96
//
// 回傳 1 成功、0 這裡不能放、-1 出界、-2 錢不夠。
func (w *World) ConnecTile(x, y, cmd int) int {
	if !InBounds(x, y) {
		return -1
	}

	// 自動推土只對放置類指令生效（推土機自己不需要）。
	if cmd >= ConnRoad && cmd <= ConnWire {
		if w.AutoBulldoze && w.TotalFunds > 0 && w.Map[x][y]&BULLBIT != 0 {
			t := neutralizeRoad(int(w.Map[x][y]))
			if (t >= TINYEXP && t <= LASTTINYEXP) || (t < 64 && t != 0) {
				w.spend(1)
				w.Map[x][y] = 0
			}
		}
	}

	result := 1
	switch cmd {
	case ConnFixZone:
		w.fixZone(x, y)
		return 1
	case ConnDoze:
		result = w.layDoze(x, y)
	case ConnRoad:
		result = w.layRoad(x, y)
	case ConnRail:
		result = w.layRail(x, y)
	case ConnWire:
		result = w.layWire(x, y)
	default:
		return 1
	}
	w.fixZone(x, y)
	return result
}

// spend 扣錢。w_stubs.c:87 Spend
func (w *World) spend(n int) { w.TotalFunds -= n }

// layDoze 推平一格。w_con.c:155
func (w *World) layDoze(x, y int) int {
	if w.TotalFunds == 0 {
		return -2
	}
	if w.Map[x][y]&BULLBIT == 0 {
		return 0 // 這格不能推
	}
	t := neutralizeRoad(int(w.Map[x][y]))

	// ⚠ 推掉水上的東西要還原成水，不是空地。少了這張清單，
	// 拆橋之後會多出一塊憑空出現的陸地。
	switch t {
	case HBRIDGE, VBRIDGE, BRWV, BRWH,
		HBRDG0, HBRDG1, HBRDG2, HBRDG3,
		VBRDG0, VBRDG1, VBRDG2, VBRDG3,
		HPOWER, VPOWER, HRAIL, VRAIL:
		w.Map[x][y] = RIVER
	default:
		w.Map[x][y] = DIRT
	}
	w.spend(1)
	return 1
}

// tileAt 讀鄰居；出界回 0。取代原版的指標算術。
func (w *World) tileAt(x, y int) int {
	if !InBounds(x, y) {
		return 0
	}
	return int(w.Map[x][y])
}

// layRoad 鋪路。w_con.c:203
func (w *World) layRoad(x, y int) int {
	if w.TotalFunds < 10 {
		return -2
	}
	cost := 10
	t := int(w.Map[x][y]) & LOMASK

	switch t {
	case DIRT:
		w.Map[x][y] = ROADS | BULLBIT | BURNBIT

	case RIVER, REDGE, CHANNEL:
		// 水面上要蓋橋，而且只有在**已經有東西可以接**的方向才蓋。
		if w.TotalFunds < 50 {
			return -2
		}
		cost = 50
		laid := false
		if x < WorldX-1 {
			n := neutralizeRoad(w.tileAt(x+1, y))
			if n == VRAILROAD || n == HBRIDGE || (n >= ROADS && n <= HROADPOWER) {
				w.Map[x][y] = HBRIDGE | BULLBIT
				laid = true
			}
		}
		if !laid && x > 0 {
			n := neutralizeRoad(w.tileAt(x-1, y))
			if n == VRAILROAD || n == HBRIDGE || (n >= ROADS && n <= INTERSECTION) {
				w.Map[x][y] = HBRIDGE | BULLBIT
				laid = true
			}
		}
		if !laid && y < WorldY-1 {
			n := neutralizeRoad(w.tileAt(x, y+1))
			if n == HRAILROAD || n == VROADPOWER || (n >= VBRIDGE && n <= INTERSECTION) {
				w.Map[x][y] = VBRIDGE | BULLBIT
				laid = true
			}
		}
		if !laid && y > 0 {
			n := neutralizeRoad(w.tileAt(x, y-1))
			if n == HRAILROAD || n == VROADPOWER || (n >= VBRIDGE && n <= INTERSECTION) {
				w.Map[x][y] = VBRIDGE | BULLBIT
				laid = true
			}
		}
		if !laid {
			return 0
		}

	case LHPOWER:
		w.Map[x][y] = VROADPOWER | CONDBIT | BURNBIT | BULLBIT
	case LVPOWER:
		w.Map[x][y] = HROADPOWER | CONDBIT | BURNBIT | BULLBIT
	case LHRAIL:
		w.Map[x][y] = HRAILROAD | BURNBIT | BULLBIT
	case LVRAIL:
		w.Map[x][y] = VRAILROAD | BURNBIT | BULLBIT
	default:
		return 0
	}
	w.spend(cost)
	return 1
}

// layRail 鋪鐵路。w_con.c:309
//
// ⚠ 原版這一段全部寫死數字（226、224、221…），沒有用具名常數。
// 照抄數字，另外在旁邊註明語意。
func (w *World) layRail(x, y int) int {
	if w.TotalFunds < 20 {
		return -2
	}
	cost := 20
	t := neutralizeRoad(int(w.Map[x][y]))

	switch t {
	case 0: // 空地
		w.Map[x][y] = 226 | BULLBIT | BURNBIT

	case 2, 3, 4: // 水面：海底隧道
		if w.TotalFunds < 100 {
			return -2
		}
		cost = 100
		laid := false
		if x < WorldX-1 {
			n := neutralizeRoad(w.tileAt(x+1, y))
			if n == 221 || n == 224 || (n >= 226 && n <= 237) {
				w.Map[x][y] = 224 | BULLBIT
				laid = true
			}
		}
		if !laid && x > 0 {
			n := neutralizeRoad(w.tileAt(x-1, y))
			// ⚠ 這一條的範圍寫成 `> 225 && < 238`，和上面的
			// `>= 226 && <= 237` 其實一樣，但原版就是寫成兩種形式。
			if n == 221 || n == 224 || (n > 225 && n < 238) {
				w.Map[x][y] = 224 | BULLBIT
				laid = true
			}
		}
		if !laid && y < WorldY-1 {
			n := neutralizeRoad(w.tileAt(x, y+1))
			if n == 222 || n == 238 || (n > 224 && n < 237) {
				w.Map[x][y] = 225 | BULLBIT
				laid = true
			}
		}
		if !laid && y > 0 {
			n := neutralizeRoad(w.tileAt(x, y-1))
			if n == 222 || n == 238 || (n > 224 && n < 237) {
				w.Map[x][y] = 225 | BULLBIT
				laid = true
			}
		}
		if !laid {
			return 0
		}

	case 210: // 電線上鋪鐵路
		w.Map[x][y] = 222 | CONDBIT | BURNBIT | BULLBIT
	case 211:
		w.Map[x][y] = 221 | CONDBIT | BURNBIT | BULLBIT
	case 66: // 道路上鋪鐵路（平交道）
		w.Map[x][y] = 238 | BURNBIT | BULLBIT
	case 67:
		w.Map[x][y] = 237 | BURNBIT | BULLBIT
	default:
		return 0
	}
	w.spend(cost)
	return 1
}

// layWire 拉電線。w_con.c:400
//
// ⚠ 水下電纜的判斷和道路／鐵路不一樣：先看鄰居有沒有 CONDBIT
// （導電），**才**做 neutralizeRoad。順序反過來會把旗標抹掉。
func (w *World) layWire(x, y int) int {
	if w.TotalFunds < 5 {
		return -2
	}
	cost := 5
	t := neutralizeRoad(int(w.Map[x][y]))

	switch t {
	case 0:
		w.Map[x][y] = 210 | CONDBIT | BURNBIT | BULLBIT

	case 2, 3, 4: // 水面：海底電纜
		if w.TotalFunds < 25 {
			return -2
		}
		cost = 25
		laid := false
		try := func(nx, ny, tile, a, b, c int) bool {
			raw := w.tileAt(nx, ny)
			if raw&CONDBIT == 0 {
				return false
			}
			n := neutralizeRoad(raw)
			if n != a && n != b && n != c {
				w.Map[x][y] = uint16(tile | CONDBIT | BULLBIT)
				return true
			}
			return false
		}
		if x < WorldX-1 {
			laid = try(x+1, y, 209, 77, 221, 208)
		}
		if !laid && x > 0 {
			laid = try(x-1, y, 209, 77, 221, 208)
		}
		if !laid && y < WorldY-1 {
			laid = try(x, y+1, 208, 78, 222, 209)
		}
		if !laid && y > 0 {
			laid = try(x, y-1, 208, 78, 222, 209)
		}
		if !laid {
			return 0
		}

	case 66: // 道路上拉電線
		w.Map[x][y] = 77 | CONDBIT | BURNBIT | BULLBIT
	case 67:
		w.Map[x][y] = 78 | CONDBIT | BURNBIT | BULLBIT
	case 226: // 鐵路上拉電線
		w.Map[x][y] = 221 | CONDBIT | BURNBIT | BULLBIT
	case 227:
		w.Map[x][y] = 222 | CONDBIT | BURNBIT | BULLBIT
	default:
		return 0
	}
	w.spend(cost)
	return 1
}

// fixZone 修這一格與四個鄰居的形狀。w_con.c:497
func (w *World) fixZone(x, y int) {
	w.fixSingle(x, y)
	if y > 0 {
		w.fixSingle(x, y-1)
	}
	if x < WorldX-1 {
		w.fixSingle(x+1, y)
	}
	if y < WorldY-1 {
		w.fixSingle(x, y+1)
	}
	if x > 0 {
		w.fixSingle(x-1, y)
	}
}

// fixSingle 依四個鄰居重新決定一格的形狀。w_con.c:521
//
// 三種線路各有各的「哪些鄰居算連通」規則，而且排除清單不一樣——
// 例如道路不接 77（路上的電線橫向）與 238（橫向平交道），因為那兩個
// 在這個方向上是**穿過去**而不是接上來。
func (w *World) fixSingle(x, y int) {
	t := neutralizeRoad(int(w.Map[x][y]))
	adj := 0

	switch {
	case t >= 66 && t <= 76: // 道路
		if y > 0 {
			n := neutralizeRoad(w.tileAt(x, y-1))
			if (n == 237 || (n >= 64 && n <= 78)) && n != 77 && n != 238 && n != 64 {
				adj |= 1
			}
		}
		if x < WorldX-1 {
			n := neutralizeRoad(w.tileAt(x+1, y))
			if (n == 238 || (n >= 64 && n <= 78)) && n != 78 && n != 237 && n != 65 {
				adj |= 2
			}
		}
		if y < WorldY-1 {
			n := neutralizeRoad(w.tileAt(x, y+1))
			if (n == 237 || (n >= 64 && n <= 78)) && n != 77 && n != 238 && n != 64 {
				adj |= 4
			}
		}
		if x > 0 {
			n := neutralizeRoad(w.tileAt(x-1, y))
			if (n == 238 || (n >= 64 && n <= 78)) && n != 78 && n != 237 && n != 65 {
				adj |= 8
			}
		}
		w.Map[x][y] = uint16(roadTable[adj] | BULLBIT | BURNBIT)

	case t >= 226 && t <= 236: // 鐵路
		if y > 0 {
			n := neutralizeRoad(w.tileAt(x, y-1))
			if n >= 221 && n <= 238 && n != 221 && n != 237 && n != 224 {
				adj |= 1
			}
		}
		if x < WorldX-1 {
			n := neutralizeRoad(w.tileAt(x+1, y))
			if n >= 221 && n <= 238 && n != 222 && n != 238 && n != 225 {
				adj |= 2
			}
		}
		if y < WorldY-1 {
			n := neutralizeRoad(w.tileAt(x, y+1))
			if n >= 221 && n <= 238 && n != 221 && n != 237 && n != 224 {
				adj |= 4
			}
		}
		if x > 0 {
			n := neutralizeRoad(w.tileAt(x-1, y))
			if n >= 221 && n <= 238 && n != 222 && n != 238 && n != 225 {
				adj |= 8
			}
		}
		w.Map[x][y] = uint16(railTable[adj] | BULLBIT | BURNBIT)

	case t >= 210 && t <= 220: // 電線
		cond := func(nx, ny, a, b, c int) bool {
			raw := w.tileAt(nx, ny)
			if raw&CONDBIT == 0 {
				return false
			}
			n := neutralizeRoad(raw)
			return n != a && n != b && n != c
		}
		if y > 0 && cond(x, y-1, 209, 78, 222) {
			adj |= 1
		}
		if x < WorldX-1 && cond(x+1, y, 208, 77, 221) {
			adj |= 2
		}
		if y < WorldY-1 && cond(x, y+1, 209, 78, 222) {
			adj |= 4
		}
		if x > 0 && cond(x-1, y, 208, 77, 221) {
			adj |= 8
		}
		w.Map[x][y] = uint16(wireTable[adj] | BULLBIT | BURNBIT | CONDBIT)
	}
}
