package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// cmdPPF 把一幅 `.PPF` 整頁畫面解成 PNG。
//
// `.PPF` 是 LZSS 壓過的整頁畫面（開場招牌與劇本選單），六種顯示模式各一套。
// **寬度與每列位元組是固定的，高度要由長度反推**，而 `mono` 與 `cga` 兩種
// 每列都是 80 個位元組，光看長度分不出來——所以 `-mode` 是必填，不猜。
//
// 256 色的 `mcga` 另外要同一個圖形集的調色盤（`-pal` 指一個 `.PGF`）；
// 其餘模式的色盤在版面裡，不必給。
func cmdPPF(args []string) {
	fs := flag.NewFlagSet("ppf", flag.ExitOnError)
	in := fs.String("file", "", "要解的 .PPF")
	mode := fs.String("mode", "", "顯示模式：cega／sega／tdy／mcga／mono／cga")
	pal := fs.String("pal", "", "同一個圖形集的 .PGF（只有 mcga 需要）")
	out := fs.String("out", "", "輸出的 PNG")
	fs.Parse(args)
	if *in == "" || *mode == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "用法：simtool ppf -file X.PPF -mode cega -out X.png [-pal Y.PGF]")
		os.Exit(2)
	}

	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d, err := assets.DecompressLZSS(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s 解壓失敗：%v\n", *in, err)
		os.Exit(1)
	}

	var colors []assets.PGFColor
	if *pal != "" {
		praw, err := os.ReadFile(*pal)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		g, err := assets.ParsePGF(praw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s：%v\n", *pal, err)
			os.Exit(1)
		}
		colors = g.Palette
	}

	im, err := assets.ParsePPFAs(d, colors, strings.ToLower(*mode))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s（%s）：%v\n", *in, *mode, err)
		os.Exit(1)
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, im); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b := im.Bounds()
	fmt.Printf("%s（%s）→ %s　%d×%d\n", *in, *mode, *out, b.Dx(), b.Dy())
}
