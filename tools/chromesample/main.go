package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"sort"
)

// 數一張原版截圖的遊戲畫面區裡各出現幾種顏色。
// 用途：確認單色模式的介面真的只有兩色，以及彩色模式用了哪幾個 EGA 色。
// 用法：chromesample <png>...            整張圖的顏色統計
//
//	chromesample <png> x y w h        只數那一塊
func main() {
	args := os.Args[1:]
	var rect []int
	if len(args) >= 5 {
		for _, v := range args[len(args)-4:] {
			n := 0
			fmt.Sscanf(v, "%d", &n)
			rect = append(rect, n)
		}
		args = args[:len(args)-4]
	}
	for _, p := range args {
		f, err := os.Open(p)
		if err != nil {
			fmt.Println(err)
			continue
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			fmt.Println(p, err)
			continue
		}
		b := img.Bounds()
		if len(rect) == 4 {
			b = image.Rect(rect[0], rect[1], rect[0]+rect[2], rect[1]+rect[3]).Intersect(b)
		}
		hist := map[[3]uint8]int{}
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bb, _ := img.At(x, y).RGBA()
				hist[[3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(bb >> 8)}]++
			}
		}
		type kv struct {
			c [3]uint8
			n int
		}
		var all []kv
		for c, n := range hist {
			all = append(all, kv{c, n})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
		fmt.Printf("== %s：%d 種顏色\n", p, len(all))
		for i, e := range all {
			if i >= 6 {
				break
			}
			fmt.Printf("   #%02x%02x%02x  %d\n", e.c[0], e.c[1], e.c[2], e.n)
		}
	}
}
