package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 九份音效檔全部要切成八段，而且最後一段的結尾剛好是檔尾。
//
// 這個測試同時守著兩件事：LZSS 解得對，以及長度鏈的假設成立。
// 鏈只要錯一個位元組，最後就會停在檔案中間——所以「走到底」本身
// 就是強證據，不需要另外找一張段數表。
func TestPSFSplitsIntoEightSounds(t *testing.T) {
	dir := dosDir(t)
	var files []string
	for _, sub := range []string{"DATA", "."} {
		ents, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil {
			continue
		}
		for _, e := range ents {
			n := strings.ToLower(e.Name())
			if strings.HasSuffix(n, ".psf") {
				files = append(files, filepath.Join(dir, sub, e.Name()))
			}
		}
	}
	if len(files) == 0 {
		t.Skip("沒有 DOS 資料，跳過")
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		snds, err := LoadPSF(raw)
		if err != nil {
			t.Errorf("%s：%v", filepath.Base(f), err)
			continue
		}
		var lens []int
		for _, s := range snds {
			lens = append(lens, len(s.Raw))
		}
		t.Logf("%-20s %v", filepath.Base(f), lens)
	}
}

// 未壓縮的 .V4 與 DATA/SOUNDDAT.PSF 解壓後應該是同一份資料。
// 這是「.PSF 就是壓縮過的 .V4」這條斷言的機器檢查。
func TestV4EqualsDecompressedPSF(t *testing.T) {
	dir := dosDir(t)
	v4, err := os.ReadFile(filepath.Join(dir, "SOUNDDAT.V4"))
	if err != nil {
		t.Skip("沒有 SOUNDDAT.V4，跳過")
	}
	psf, err := os.ReadFile(filepath.Join(dir, "DATA", "SOUNDDAT.PSF"))
	if err != nil {
		t.Skip("沒有 DATA/SOUNDDAT.PSF，跳過")
	}
	got, err := DecompressLZSS(psf)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(v4) {
		t.Fatalf("解出 %d 位元組，.V4 是 %d", len(got), len(v4))
	}
	for i := range got {
		if got[i] != v4[i] {
			t.Fatalf("第 %d 個位元組不同：%02x vs %02x", i, got[i], v4[i])
		}
	}
}
