package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// 用指定的顯示模式解一幅 `.PPF`。
// 用法：ppfas <檔.ppf> <模式> <輸出.png>
func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "用法：ppfas <檔.ppf> <cega|sega|tdy|mcga|mono|cga> <輸出.png>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, err := assets.DecompressLZSS(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	im, err := assets.ParsePPFAs(body, nil, os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.Create(os.Args[3])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	fmt.Printf("%s 用 %s 解出 %dx%d\n", os.Args[1], os.Args[2],
		im.Bounds().Dx(), im.Bounds().Dy())
	if err := png.Encode(f, im); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
