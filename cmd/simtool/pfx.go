package main

import (
	"fmt"
	"os"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

// cmdPrefix 列出第 2 段每一筆圖片訊息的編號，用來對照訊息事件。
func cmdPrefix(args []string) {
	raw, _ := os.ReadFile(args[0])
	secs, err := assets.LoadPTF(raw)
	if err != nil {
		fmt.Println(err)
		return
	}
	sec := secs[2]
	for i := range sec.Strings {
		body := assets.TrimPrefix(sec.Strings[i])
		if len(body) > 34 {
			body = body[:34]
		}
		fmt.Printf("2.%-2d 編號 %5d  %s\n", i, assets.PictureID(sec, i), body)
	}
}
