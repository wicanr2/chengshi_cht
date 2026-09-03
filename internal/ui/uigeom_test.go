package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func modesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	p := filepath.Join(filepath.Dir(thisFile), "..", "..", "workplace", "allmodes")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Skip("沒有六種顯示模式齊全的資料目錄（workplace/allmodes），跳過（玩家自備）")
	}
	return p
}

// 六個顯示模式的介面美術都要切得出九張圖層圖示，而且工具盤與統計圖的
// 格子要落在美術裡面、彼此不重疊。
//
// ⚠ 少了這一支的後果**沒有症狀**：格線算錯不會報錯，畫面看起來只是
// 「圖小了一點」，而按鈕會點到隔壁那一格。
func TestUIGeomFitsEveryMode(t *testing.T) {
	dir := modesDir(t)
	// ⚠ **六個圖形集都要測，不能只測一個。** 同一個顯示模式的介面美術
	// 尺寸在不同圖形集之間會差一兩個像素（`ASIACEGA` 的庫 5 是 26×226、
	// `WESTCEGA` 是 25×225），只測 asia 的話 west 整排圖示消失也不會變紅。
	for _, style := range []string{"asia", "medi", "west", "fusa", "feur", "moon"} {
		for _, m := range DisplayModes {
			checkGeom(t, dir, style, m.Key)
		}
	}
}

func checkGeom(t *testing.T, dir, style, mode string) {
	t.Helper()
	{
		m := struct{ Key string }{mode + "／" + style}
		ts, err := LoadTileSetMode(dir, style, mode)
		if err != nil {
			t.Errorf("%s：%v", m.Key, err)
			return
		}
		u := ts.Geom
		if len(ts.MapIcons) != mapIconCount {
			t.Errorf("%s 切出 %d 張圖層圖示，要 %d 張", m.Key, len(ts.MapIcons), mapIconCount)
		}
		for i, im := range ts.MapIcons {
			if im == nil {
				t.Errorf("%s 第 %d 張圖層圖示是 nil", m.Key, i)
				continue
			}
			if im.Bounds().Dx() < 8 || im.Bounds().Dy() < 8 {
				t.Errorf("%s 第 %d 張圖層圖示只有 %dx%d，切壞了",
					m.Key, i, im.Bounds().Dx(), im.Bounds().Dy())
			}
			// 圖示欄只有 26 像素寬、一格 25 高，切出來的不能比它大。
			if w, h := im.Bounds().Dx(), im.Bounds().Dy(); w > 26 || h > mapIconH {
				t.Errorf("%s 第 %d 張圖層圖示 %dx%d，欄位只有 26x%d",
					m.Key, i, w, h, mapIconH)
			}
		}
		pal := ts.UIImage(BankToolPalette, 0)
		if pal == nil {
			t.Errorf("%s 沒有工具盤美術", m.Key)
			return
		}
		// 判準是**最後一格的中心**落在美術裡面，不是右下角——
		// 同一個模式的六個圖形集，介面美術的寬高會差一兩個像素
		// （CEGA 的工具盤 asia 是 57 寬、west 是 56），格子的右緣
		// 因此常常超出一兩格。真正會出事的是**間距**算錯，
		// 那會讓最後一格差好幾十像素，用中心就抓得到。
		pw, ph := pal.Bounds().Dx(), pal.Bounds().Dy()
		if x := u.palXOff + u.palPitchX + u.palCellW/2; x >= pw {
			t.Errorf("%s 工具盤右欄的中心在 x=%d，美術只有 %d 寬", m.Key, x, pw)
		}
		if y := u.palYOff + 6*u.palPitchY + u.palCellH/2; y >= ph {
			t.Errorf("%s 工具盤第七列的中心在 y=%d，美術只有 %d 高", m.Key, y, ph)
		}
		if u.palPitchX < u.palCellW || u.palPitchY < u.palCellH {
			t.Errorf("%s 工具盤的格子比間距大，會互相重疊", m.Key)
		}
		grf := ts.UIImage(BankGraphBtns, 0)
		if grf == nil {
			t.Errorf("%s 沒有統計圖按鈕美術", m.Key)
			return
		}
		gw, gh := grf.Bounds().Dx(), grf.Bounds().Dy()
		if x := u.grfXOff + u.grfPitchX + u.grfCellW/2; x >= gw {
			t.Errorf("%s 統計圖右欄的中心在 x=%d，美術只有 %d 寬", m.Key, x, gw)
		}
		if y := u.grfYOff + 3*u.grfPitchY + u.grfCellH/2; y >= gh {
			t.Errorf("%s 統計圖第四列的中心在 y=%d，美術只有 %d 高", m.Key, y, gh)
		}
		dem := ts.UIImage(BankDemand, 0)
		if dem == nil {
			t.Errorf("%s 沒有需求指標美術", m.Key)
			return
		}
		dw, dh := dem.Bounds().Dx(), dem.Bounds().Dy()
		if x := u.demBarX[2] + demandBarW; x > dw {
			t.Errorf("%s 第三根需求長條到 x=%d，美術只有 %d 寬", m.Key, x, dw)
		}
		if u.demUpBot-u.demUpMax < 0 || u.demDnTop+u.demDnMax > dh {
			t.Errorf("%s 需求長條長到美術外面（上 %d−%d、下 %d+%d，高 %d）",
				m.Key, u.demUpBot, u.demUpMax, u.demDnTop, u.demDnMax, dh)
		}
	}
}
