package sim

// 地形產生。證據：docs/re/04-terrain-generation.md／規格：docs/spec/terrain.md
// 一手出處：s_gen.c
//
// 這一整份的重點是**亂數消耗順序**：每一個分支、每一個提前 return 都會改變
// 之後所有格子的內容。所以下面照抄控制流，包含看起來可以合併的判斷。

// 地形產生用到的圖塊範圍。s_gen.c:68-71
const (
	waterLow  = RIVER         // 2
	waterHigh = LASTRIVEDGE   // 20
	woodsLow  = TREEBASE      // 21
	woodsHigh = UNUSED_TRASH2 // 39 —— 是的，上界是一個被標成「未用」的編號
)

// TerrainParams 對應 s_gen.c:76-79 的四個全域旋鈕。
// 負值代表「用預設的隨機行為」，這與「0」不同：0 是「完全不做」。
type TerrainParams struct {
	TreeLevel    int // -1 => 隨機量
	LakeLevel    int // -1 => 隨機量；0 => 不造湖
	CurveLevel   int // -1 => 預設彎曲度；0 => 不造河
	CreateIsland int // -1 => 一成機率；0 => 不造島；1 => 一定造島
}

// DefaultTerrainParams 是原版的初值。s_gen.c:76-79
func DefaultTerrainParams() TerrainParams {
	return TerrainParams{TreeLevel: -1, LakeLevel: -1, CurveLevel: -1, CreateIsland: -1}
}

// terrainGen 持有 s_gen.c 的那幾個全域變數（MapX/MapY/XStart/YStart/Dir/LastDir）。
type terrainGen struct {
	w              *World
	p              TerrainParams
	mapX, mapY     int
	xStart, yStart int
	dir, lastDir   int
}

// GenerateMap 依種子產生地形。s_gen.c:127 GenerateMap(int r)
//
// 原版在最後呼叫 RandomlySeedRand()（s_gen.c:153），把亂數重新亂數播種。
// 這裡刻意不做——不決定性的東西不進規則層（docs/spec/rng.md 的「刻意不照做」）。
// 呼叫端要繼續模擬時自己決定種子。
func (w *World) GenerateMap(seed uint32, p TerrainParams) {
	w.Rand.Seed(seed) // s_gen.c:129 SeedRand(r)
	g := &terrainGen{w: w, p: p}

	if p.CreateIsland < 0 {
		if w.Rand.Rand(100) < 10 { // 一成機率造島。s_gen.c:132
			g.makeIsland()
			return
		}
	}
	if p.CreateIsland == 1 {
		g.makeNakedIsland()
	} else {
		w.clearMap()
	}
	g.getRandStart()
	if p.CurveLevel != 0 {
		g.doRivers()
	}
	if p.LakeLevel != 0 {
		g.makeLakes()
	}
	g.smoothRiver()
	if p.TreeLevel != 0 {
		g.doTrees()
	}
}

// SmoothTerrain 把手工填出來的粗胚地形換成正確的邊界圖塊：
// 水面填 REDGE、樹林填 WOODS，呼叫這個之後就會長出原版的岸線與林緣。
//
// 為什麼要匯出：地圖工具（tools/citymap）要畫出與原版同一套邊緣規則的地形，
// 而規則就在 smoothRiver／smoothTrees 裡。與其在工具裡重寫一份會漂移的複製品，
// 不如把原版這兩支直接開出來用。
//
// 這裡**不擲亂數**，所以同一份粗胚每次得到同一張地圖。
// 樹林跑兩次與 doTrees 一致（s_gen.c:299-300）。
func (w *World) SmoothTerrain() {
	g := &terrainGen{w: w, p: DefaultTerrainParams()}
	g.smoothRiver()
	g.smoothTrees()
	g.smoothTrees()
}

// clearMap 把整張圖填成空地。s_gen.c:156
func (w *World) clearMap() {
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			w.Map[x][y] = DIRT
		}
	}
}

// s_gen.c:168 #define RADIUS 18
const islandRadius = 18

// makeNakedIsland 造一座沒有樹的島。s_gen.c:170
func (g *terrainGen) makeNakedIsland() {
	w := g.w
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			w.Map[x][y] = RIVER
		}
	}
	for x := 5; x < WorldX-5; x++ {
		for y := 5; y < WorldY-5; y++ {
			w.Map[x][y] = DIRT
		}
	}
	for x := 0; x < WorldX-5; x += 2 {
		g.mapX = x
		g.mapY = w.Rand.ERand(islandRadius)
		g.bRivPlop()
		g.mapY = (WorldY - 10) - w.Rand.ERand(islandRadius)
		g.bRivPlop()
		g.mapY = 0
		g.sRivPlop()
		g.mapY = WorldY - 6
		g.sRivPlop()
	}
	for y := 0; y < WorldY-5; y += 2 {
		g.mapY = y
		g.mapX = w.Rand.ERand(islandRadius)
		g.bRivPlop()
		g.mapX = (WorldX - 10) - w.Rand.ERand(islandRadius)
		g.bRivPlop()
		g.mapX = 0
		g.sRivPlop()
		g.mapX = WorldX - 6
		g.sRivPlop()
	}
}

// makeIsland。s_gen.c:216
func (g *terrainGen) makeIsland() {
	g.makeNakedIsland()
	g.smoothRiver()
	g.doTrees()
}

// makeLakes。s_gen.c:224
func (g *terrainGen) makeLakes() {
	w := g.w
	var lim1 int
	if g.p.LakeLevel < 0 {
		lim1 = w.Rand.Rand(10)
	} else {
		lim1 = g.p.LakeLevel / 2
	}
	for t := 0; t < lim1; t++ {
		x := w.Rand.Rand(WorldX-21) + 10
		y := w.Rand.Rand(WorldY-20) + 10
		lim2 := w.Rand.Rand(12) + 2
		for z := 0; z < lim2; z++ {
			g.mapX = x - 6 + w.Rand.Rand(12)
			g.mapY = y - 6 + w.Rand.Rand(12)
			if w.Rand.Rand(4) != 0 {
				g.sRivPlop()
			} else {
				g.bRivPlop()
			}
		}
	}
}

// getRandStart 決定河流的起點。s_gen.c:251
func (g *terrainGen) getRandStart() {
	g.xStart = 40 + g.w.Rand.Rand(WorldX-80)
	g.yStart = 33 + g.w.Rand.Rand(WorldY-67)
	g.mapX = g.xStart
	g.mapY = g.yStart
}

// dirTab 是八方向位移。s_gen.c:260
var dirTabX = [8]int{0, 1, 1, 1, 0, -1, -1, -1}
var dirTabY = [8]int{-1, -1, 0, 1, 1, 1, 0, -1}

// moveMap。s_gen.c:259
//
// 原版是 `dir = dir & 7`，而 Dir 會被 Dir-- 減成負數；C 的 short 是二補數，
// 所以 -1 & 7 == 7。Go 的 int 同樣是二補數，行為一致。
func (g *terrainGen) moveMap(dir int) {
	dir &= 7
	g.mapX += dirTabX[dir]
	g.mapY += dirTabY[dir]
}

// treeSplash 從一點灑出一條樹的隨機遊走。s_gen.c:268
func (g *terrainGen) treeSplash(xloc, yloc int) {
	w := g.w
	var dis int
	if g.p.TreeLevel < 0 {
		dis = w.Rand.Rand(150) + 50
	} else {
		dis = w.Rand.Rand(100+(g.p.TreeLevel*2)) + 50
	}
	g.mapX = xloc
	g.mapY = yloc
	for z := 0; z < dis; z++ {
		dir := w.Rand.Rand(7)
		g.moveMap(dir)
		if !InBounds(g.mapX, g.mapY) {
			return
		}
		if int(w.Map[g.mapX][g.mapY]&LOMASK) == DIRT {
			w.Map[g.mapX][g.mapY] = WOODS + BLBNBIT
		}
	}
}

// doTrees。s_gen.c:287
func (g *terrainGen) doTrees() {
	w := g.w
	var amount int
	if g.p.TreeLevel < 0 {
		amount = w.Rand.Rand(100) + 50
	} else {
		amount = g.p.TreeLevel + 3
	}
	for x := 0; x < amount; x++ {
		xloc := w.Rand.Rand(WorldX - 1)
		yloc := w.Rand.Rand(WorldY - 1)
		g.treeSplash(xloc, yloc)
	}
	g.smoothTrees()
	g.smoothTrees()
}

// smoothRiver 把河岸換成正確的邊界圖塊。s_gen.c:305
//
// ⚠ 原版在這裡用 `register short temp, MapX, MapY` **遮蔽了全域的 MapX/MapY**，
// 所以這個函式不會動到河流繪製留下的游標位置。Go 版用區域變數自然一致，
// 但這一點要寫下來——照著全域改會讓之後的亂數位置錯開。
func (g *terrainGen) smoothRiver() {
	w := g.w
	dx := [4]int{-1, 0, 1, 0}
	dy := [4]int{0, 1, 0, -1}
	rEdTab := [16]uint16{
		13 + BULLBIT, 13 + BULLBIT, 17 + BULLBIT, 15 + BULLBIT,
		5 + BULLBIT, 2, 19 + BULLBIT, 17 + BULLBIT,
		9 + BULLBIT, 11 + BULLBIT, 2, 13 + BULLBIT,
		7 + BULLBIT, 9 + BULLBIT, 5 + BULLBIT, 2,
	}
	for mapX := 0; mapX < WorldX; mapX++ {
		for mapY := 0; mapY < WorldY; mapY++ {
			if w.Map[mapX][mapY] != REDGE {
				continue
			}
			bitindex := 0
			for z := 0; z < 4; z++ {
				bitindex <<= 1
				xt, yt := mapX+dx[z], mapY+dy[z]
				if InBounds(xt, yt) &&
					int(w.Map[xt][yt]&LOMASK) != DIRT &&
					(int(w.Map[xt][yt]&LOMASK) < woodsLow ||
						int(w.Map[xt][yt]&LOMASK) > woodsHigh) {
					bitindex++
				}
			}
			temp := rEdTab[bitindex&15]
			if temp != RIVER && g.w.Rand.Rand(1) != 0 {
				temp++
			}
			w.Map[mapX][mapY] = temp
		}
	}
}

// isTree。s_gen.c:349
func isTree(cell uint16) bool {
	v := int(cell & LOMASK)
	return v >= woodsLow && v <= woodsHigh
}

// smoothTrees 把樹叢換成正確的邊界圖塊。s_gen.c:358
func (g *terrainGen) smoothTrees() {
	w := g.w
	dx := [4]int{-1, 0, 1, 0}
	dy := [4]int{0, 1, 0, -1}
	tEdTab := [16]uint16{
		0, 0, 0, 34,
		0, 0, 36, 35,
		0, 32, 0, 33,
		30, 31, 29, 37,
	}
	for mapX := 0; mapX < WorldX; mapX++ {
		for mapY := 0; mapY < WorldY; mapY++ {
			if !isTree(w.Map[mapX][mapY]) {
				continue
			}
			bitindex := 0
			for z := 0; z < 4; z++ {
				bitindex <<= 1
				xt, yt := mapX+dx[z], mapY+dy[z]
				if InBounds(xt, yt) && isTree(w.Map[xt][yt]) {
					bitindex++
				}
			}
			temp := tEdTab[bitindex&15]
			if temp != 0 {
				if temp != WOODS && (mapX+mapY)&1 != 0 {
					temp -= 8
				}
				w.Map[mapX][mapY] = temp + BLBNBIT
			} else {
				w.Map[mapX][mapY] = temp
			}
		}
	}
}

// doRivers 畫兩條大河與一條小河。s_gen.c:395
//
// ⚠ 第三段（小河）只重設 LastDir，**沒有重設 Dir**——Dir 沿用第二條大河結束時的值。
// 這不是筆誤，照抄。
func (g *terrainGen) doRivers() {
	g.lastDir = g.w.Rand.Rand(3)
	g.dir = g.lastDir
	g.doBRiv()
	g.mapX, g.mapY = g.xStart, g.yStart
	g.lastDir ^= 4
	g.dir = g.lastDir
	g.doBRiv()
	g.mapX, g.mapY = g.xStart, g.yStart
	g.lastDir = g.w.Rand.Rand(3)
	g.doSRiv()
}

// curveParams 回傳河流轉向的兩個亂數上界。s_gen.c:415 / :443
func (g *terrainGen) curveParams() (r1, r2 int) {
	if g.p.CurveLevel < 0 {
		return 100, 200
	}
	return g.p.CurveLevel + 10, g.p.CurveLevel + 100
}

// doBRiv 畫一條大河。s_gen.c:412
func (g *terrainGen) doBRiv() {
	r1, r2 := g.curveParams()
	for InBounds(g.mapX+4, g.mapY+4) {
		g.bRivPlop()
		if g.w.Rand.Rand(r1) < 10 {
			g.dir = g.lastDir
		} else {
			// 兩次 Rand(r2) 一定都會取，即使第一次就轉了向。
			if g.w.Rand.Rand(r2) > 90 {
				g.dir++
			}
			if g.w.Rand.Rand(r2) > 90 {
				g.dir--
			}
		}
		g.moveMap(g.dir)
	}
}

// doSRiv 畫一條小河。s_gen.c:440
func (g *terrainGen) doSRiv() {
	r1, r2 := g.curveParams()
	for InBounds(g.mapX+3, g.mapY+3) {
		g.sRivPlop()
		if g.w.Rand.Rand(r1) < 10 {
			g.dir = g.lastDir
		} else {
			if g.w.Rand.Rand(r2) > 90 {
				g.dir++
			}
			if g.w.Rand.Rand(r2) > 90 {
				g.dir--
			}
		}
		g.moveMap(g.dir)
	}
}

// putOnMap 蓋一格，但水面有讓路規則。s_gen.c:468
//
// 原版的 `if (temp = Map[Xloc][Yloc])` 是賦值兼判斷：只有原本非零的格子才會
// 進入讓路判斷；原本是 DIRT(0) 的格子直接覆蓋。
func (g *terrainGen) putOnMap(mchar uint16, xoff, yoff int) {
	if mchar == 0 {
		return
	}
	xloc, yloc := g.mapX+xoff, g.mapY+yoff
	if !InBounds(xloc, yloc) {
		return
	}
	if temp := g.w.Map[xloc][yloc]; temp != 0 {
		t := int(temp & LOMASK)
		if t == RIVER && mchar != CHANNEL {
			return
		}
		if t == CHANNEL {
			return
		}
	}
	g.w.Map[xloc][yloc] = mchar
}

// bRivPlop 蓋一個 9×9 的大河斷面。s_gen.c:487
//
// 注意索引：矩陣是 BRMatrix[y][x]，而位移傳的是 (x, y)。
var bRMatrix = [9][9]uint16{
	{0, 0, 0, 3, 3, 3, 0, 0, 0},
	{0, 0, 3, 2, 2, 2, 3, 0, 0},
	{0, 3, 2, 2, 2, 2, 2, 3, 0},
	{3, 2, 2, 2, 2, 2, 2, 2, 3},
	{3, 2, 2, 2, 4, 2, 2, 2, 3},
	{3, 2, 2, 2, 2, 2, 2, 2, 3},
	{0, 3, 2, 2, 2, 2, 2, 3, 0},
	{0, 0, 3, 2, 2, 2, 3, 0, 0},
	{0, 0, 0, 3, 3, 3, 0, 0, 0},
}

func (g *terrainGen) bRivPlop() {
	for x := 0; x < 9; x++ {
		for y := 0; y < 9; y++ {
			g.putOnMap(bRMatrix[y][x], x, y)
		}
	}
}

// sRivPlop 蓋一個 6×6 的小河斷面。s_gen.c:506
var sRMatrix = [6][6]uint16{
	{0, 0, 3, 3, 0, 0},
	{0, 3, 2, 2, 3, 0},
	{3, 2, 2, 2, 2, 3},
	{3, 2, 2, 2, 2, 3},
	{0, 3, 2, 2, 3, 0},
	{0, 0, 3, 3, 0, 0},
}

func (g *terrainGen) sRivPlop() {
	for x := 0; x < 6; x++ {
		for y := 0; y < 6; y++ {
			g.putOnMap(sRMatrix[y][x], x, y)
		}
	}
}
