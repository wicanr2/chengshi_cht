package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// cmdRaw 把一個 LZSS 壓縮的 DOS 資料檔解開，倒出原始位元組。
//
// 解格式的時候要看的是**解壓之後**的內容，而 `.PSF`／`.PPF` 這些還沒有
// 專用解析器。有這支就不必每次為了看幾個位元組另外寫程式。
func cmdRaw(args []string) {
	fs := flag.NewFlagSet("raw", flag.ExitOnError)
	in := fs.String("file", "", "要解開的檔案")
	out := fs.String("out", "", "輸出檔（留空就印十六進位）")
	skip := fs.Int("skip", 0, "壓縮流開始前要跳過的位元組數")
	n := fs.Int("n", 256, "印幾個位元組")
	off := fs.Int("off", 0, "從第幾個位元組開始印")
	fs.Parse(args)
	if *in == "" {
		fs.Usage()
		os.Exit(2)
	}
	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d, err := assets.DecompressLZSS(raw[*skip:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "解壓失敗：", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "解出 %d 位元組\n", len(d))
	if *out != "" {
		if err := os.WriteFile(*out, d, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	for i := *off; i < *off+*n && i < len(d); i += 16 {
		fmt.Printf("%06x  ", i)
		for j := 0; j < 16 && i+j < len(d); j++ {
			fmt.Printf("%02x ", d[i+j])
		}
		fmt.Print(" ")
		for j := 0; j < 16 && i+j < len(d); j++ {
			c := d[i+j]
			if c < 0x20 || c > 0x7e {
				c = '.'
			}
			fmt.Printf("%c", c)
		}
		fmt.Println()
	}
}
