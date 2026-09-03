package sim

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// loadGoldenMap 讀 oracle 倒出來的黃金地圖（y 外層、x 內層）。
func loadGoldenMap(t *testing.T, path string) [WorldX][WorldY]uint16 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var m [WorldX][WorldY]uint16
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	y := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) != WorldX {
			t.Fatalf("第 %d 列有 %d 格，應為 %d", y, len(parts), WorldX)
		}
		for x, p := range parts {
			v, err := strconv.Atoi(p)
			if err != nil {
				t.Fatalf("第 %d 列第 %d 格解析失敗：%v", y, x, err)
			}
			m[x][y] = uint16(v)
		}
		y++
	}
	if y != WorldY {
		t.Fatalf("讀到 %d 列，應為 %d", y, WorldY)
	}
	return m
}

// 地形產生的逐格對拍：同一個種子，Go 版產生的 12000 格必須與原版完全相同。
//
// 黃金資料取自活的 Micropolis oracle（tools/oracle/tcl/terrain-*.tcl），
// 並已驗證同一個種子跑兩次結果相同。
//
// 種子 5 走的是造島分支（GenerateMap 開頭 Rand(100)<10 的一成機率，
// s_gen.c:132），與其他三顆走的路徑完全不同——沒有它，MakeNakedIsland、
// ERand 與提前 return 那一整條路都不會被測到。
func TestGenerateMapMatchesOracle(t *testing.T) {
	for _, seed := range []uint32{5, 7, 12345, 4242} {
		seed := seed
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			want := loadGoldenMap(t, fmt.Sprintf("testdata/terrain-seed%d.csv", seed))

			w := NewWorld(0)
			w.GenerateMap(seed, DefaultTerrainParams())

			diff, firstX, firstY := 0, -1, -1
			for y := 0; y < WorldY; y++ {
				for x := 0; x < WorldX; x++ {
					if w.Map[x][y] != want[x][y] {
						if diff == 0 {
							firstX, firstY = x, y
						}
						diff++
					}
				}
			}
			if diff != 0 {
				t.Fatalf("有 %d 格不同（共 %d 格）。第一處 (%d,%d)：得到 %d，原版 %d",
					diff, WorldX*WorldY, firstX, firstY,
					w.Map[firstX][firstY], want[firstX][firstY])
			}
		})
	}
}

// 造島那條分支真的被走到了：種子 5 的水面應該遠多於一般地圖。
func TestIslandSeedTakesIslandBranch(t *testing.T) {
	count := func(seed uint32) int {
		w := NewWorld(0)
		w.GenerateMap(seed, DefaultTerrainParams())
		n := 0
		for y := 0; y < WorldY; y++ {
			for x := 0; x < WorldX; x++ {
				if v := w.TileNum(x, y); v >= waterLow && v <= waterHigh {
					n++
				}
			}
		}
		return n
	}
	island, normal := count(5), count(7)
	if island <= normal*2 {
		t.Errorf("種子 5 的水面 %d 格，種子 7 是 %d 格 —— 造島分支看起來沒走到", island, normal)
	}
}

// 同一個種子跑兩次要完全相同——規則層不得有任何隱藏的不決定性來源。
func TestGenerateMapIsDeterministic(t *testing.T) {
	a := NewWorld(0)
	a.GenerateMap(999, DefaultTerrainParams())
	b := NewWorld(0)
	b.GenerateMap(999, DefaultTerrainParams())
	if a.Map != b.Map {
		t.Fatal("同一個種子產生了兩張不同的地圖")
	}
}

// 不同種子要產生不同地圖（否則種子沒接上）。
func TestGenerateMapUsesSeed(t *testing.T) {
	a := NewWorld(0)
	a.GenerateMap(1, DefaultTerrainParams())
	b := NewWorld(0)
	b.GenerateMap(2, DefaultTerrainParams())
	if a.Map == b.Map {
		t.Fatal("不同種子產生了同一張地圖 —— 種子沒有接上")
	}
}

// 產生出來的每一格都必須是合法圖塊編號。
func TestGenerateMapProducesValidTiles(t *testing.T) {
	w := NewWorld(0)
	w.GenerateMap(12345, DefaultTerrainParams())
	for y := 0; y < WorldY; y++ {
		for x := 0; x < WorldX; x++ {
			if n := w.TileNum(x, y); n >= TILE_COUNT {
				t.Fatalf("(%d,%d) 的圖塊編號 %d 超過 TILE_COUNT %d", x, y, n, TILE_COUNT)
			}
		}
	}
}

// DOS 地形編輯器那一式。差別只有樹叢數量：TERRAIN.EXE 的 sub_11ED8 算
// `3 × 百分比`，s_gen.c:301 算 `TreeLevel + 3`。100% 是 300 叢對 103 叢，
// 差三倍——所以同一個種子下 DOS 那一式的樹一定明顯多。
// 證據：docs/re/20-terrain-editor.md §七。
func TestEditorDOSPlantsMoreTrees(t *testing.T) {
	count := func(dos bool) int {
		w := NewWorld(12345)
		p := TerrainParams{TreeLevel: 100, LakeLevel: 0, CurveLevel: 0,
			CreateIsland: 0, EditorDOS: dos}
		w.GenerateMap(12345, p)
		n := 0
		for x := 0; x < WorldX; x++ {
			for y := 0; y < WorldY; y++ {
				if isTree(w.Map[x][y]) {
					n++
				}
			}
		}
		return n
	}
	micro, dos := count(false), count(true)
	if micro == 0 {
		t.Fatal("s_gen.c 那一式一棵樹都沒種，測試前提壞了")
	}
	if dos <= micro {
		t.Fatalf("DOS 那一式的樹沒有比較多：%d vs %d", dos, micro)
	}
}

// 同樣的參數與種子要得到同樣的地圖——逐 tick 對拍靠這個。
func TestEditorDOSIsDeterministic(t *testing.T) {
	p := TerrainParams{TreeLevel: 37, LakeLevel: 62, CurveLevel: 15,
		CreateIsland: 0, EditorDOS: true}
	a, b := NewWorld(7), NewWorld(7)
	a.GenerateMap(7, p)
	b.GenerateMap(7, p)
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			if a.Map[x][y] != b.Map[x][y] {
				t.Fatalf("(%d,%d) 不一樣：%d vs %d", x, y, a.Map[x][y], b.Map[x][y])
			}
		}
	}
}

// 樹木百分比為零時原版仍然跑兩次 SmoothTrees（sub_10A0A＋0x010C9C 在
// 判斷外面）。看不出畫面差別，但接線不能少——這裡確認它不會炸也不改地圖。
func TestEditorDOSZeroTreesStillSmooths(t *testing.T) {
	p := TerrainParams{TreeLevel: 0, LakeLevel: 50, CurveLevel: 50,
		CreateIsland: 0, EditorDOS: true}
	w := NewWorld(3)
	w.GenerateMap(3, p)
	for x := 0; x < WorldX; x++ {
		for y := 0; y < WorldY; y++ {
			if isTree(w.Map[x][y]) {
				t.Fatalf("樹木 0%% 卻在 (%d,%d) 長出樹", x, y)
			}
		}
	}
}
