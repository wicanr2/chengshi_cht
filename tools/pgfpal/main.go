package main

import (
	"fmt"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var pal []assets.PGFColor
	if g, err := assets.ParsePGF(raw); err == nil {
		pal = g.Palette
	} else if g, err2 := assets.LoadPGFBase(raw, 8, 8, 0); err2 == nil {
		pal = g.Palette
	} else {
		panic(err)
	}
	for i, c := range pal {
		fmt.Printf("%d %02x%02x%02x\n", i, c.R, c.G, c.B)
	}
}
