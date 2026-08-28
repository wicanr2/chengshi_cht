// simtool 是開發用的命令列工具：解 DOS 資料檔、產翻譯骨架、倒狀態。
//
// 它不是遊戲。玩家用不到這支。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/chengshi_cht/internal/assets"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "messages":
		cmdMessages(os.Args[2:])
	case "prefix":
		cmdPrefix(os.Args[2:])
	case "save":
		cmdSave(os.Args[2:])
	case "scenario":
		cmdScenario(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `用法：
  simtool messages -dos <DOS 1.10 目錄> [-out <輸出目錄>]
        解開七個 .PTF 訊息檔，印出每一筆的序號、標記與長度。
        加 -out 就產生以序號為鍵的翻譯骨架。

  simtool scenario -file <劇本.PSN>
        解開一個 DOS 劇本，印出城市名與城市資料大小。`)
}

func cmdMessages(args []string) {
	fs := flag.NewFlagSet("messages", flag.ExitOnError)
	dos := fs.String("dos", "", "DOS 1.10 的目錄（裡面要有 DATA/）")
	out := fs.String("out", "", "翻譯骨架的輸出目錄（留空就只印摘要）")
	fs.Parse(args)
	if *dos == "" {
		fs.Usage()
		os.Exit(2)
	}

	dataDir := filepath.Join(*dos, "DATA")
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "讀不到 %s：%v\n", dataDir, err)
		os.Exit(1)
	}
	var names []string
	for _, e := range entries {
		if strings.EqualFold(filepath.Ext(e.Name()), ".ptf") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, n := range names {
		raw, err := os.ReadFile(filepath.Join(dataDir, n))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s：%v\n", n, err)
			continue
		}
		secs, err := assets.LoadPTF(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s：%v\n", n, err)
			continue
		}
		msgs := assets.TextMessages(secs)
		fmt.Printf("%-16s %2d 段、%3d 筆文字\n", n, len(secs), len(msgs))
		if *out == "" {
			continue
		}
		if err := writeSkeleton(*out, n, msgs); err != nil {
			fmt.Fprintf(os.Stderr, "%s：寫翻譯骨架失敗 %v\n", n, err)
		}
	}
}

// writeSkeleton 產生以 段落.索引 為鍵的翻譯骨架。
//
// **不寫入原文。** 原文屬於原權利人，不進本專案的版控（CLAUDE.md §8）；
// 骨架只帶鍵與原文長度，譯者對照著遊戲畫面或自己的原版副本填。
// 長度欄位是給排版用的：中文字寬是英數的兩倍，超過原文長度就要注意會不會爆版。
func writeSkeleton(dir, srcName string, msgs []assets.Message) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	set := strings.TrimSuffix(strings.ToLower(srcName), ".ptf")
	var b strings.Builder
	fmt.Fprintf(&b, "# %s 的訊息翻譯\n#\n", set)
	fmt.Fprintf(&b, "# 鍵是「段落.索引」，不是原文：原文屬於原權利人不進版控，\n")
	fmt.Fprintf(&b, "# 而且六個圖形集的同一個鍵講同一件事、用詞不同。\n")
	fmt.Fprintf(&b, "# 譯名以軟體世界珍藏版 29 中文說明書為準（translations/glossary.md）。\n#\n")
	fmt.Fprintf(&b, "# len 是原文的位元組長度，給排版參考：中文字寬是英數的兩倍。\n\n")
	sec := -1
	for _, m := range msgs {
		if m.Section != sec {
			sec = m.Section
			fmt.Fprintf(&b, "# ── 第 %d 段 ──\n", sec)
		}
		fmt.Fprintf(&b, "[\"%d.%d\"]\n", m.Section, m.Index)
		fmt.Fprintf(&b, "len = %d\n", len(m.Text))
		fmt.Fprintf(&b, "zh = \"\"\n\n")
	}
	return os.WriteFile(filepath.Join(dir, set+".toml"), []byte(b.String()), 0o644)
}

func cmdScenario(args []string) {
	fs := flag.NewFlagSet("scenario", flag.ExitOnError)
	file := fs.String("file", "", "劇本檔（.PSN）")
	fs.Parse(args)
	if *file == "" {
		fs.Usage()
		os.Exit(2)
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sc, err := assets.LoadPSN(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("城市名 %q，檔頭 %d 位元組，城市資料 %d 位元組\n",
		sc.Name, len(sc.Header), len(sc.Body))
}
