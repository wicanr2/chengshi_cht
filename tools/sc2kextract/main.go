// sc2kextract 從玩家自備的 SimCity 2000 DOS SC2000.DAT 擷取指定資源。
//
// 這支工具不包含、下載或散布任何原版內容；輸出必須留在 gitignore 的
// workplace/ 或合法的本機完整版目錄。
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const entrySize = 16

type entry struct {
	name   string
	offset uint32
}

func main() {
	in := flag.String("in", "", "SC2000.DAT 路徑")
	out := flag.String("out", "", "輸出目錄")
	ext := flag.String("ext", ".XMI", "只擷取這個副檔名；空字串表示全部")
	flag.Parse()
	if *in == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}

	raw, err := os.ReadFile(*in)
	if err != nil {
		fatal(err)
	}
	entries, err := parseDirectory(raw)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}

	written := 0
	for i, e := range entries {
		if *ext != "" && !strings.EqualFold(filepath.Ext(e.name), *ext) {
			continue
		}
		end := uint32(len(raw))
		if i+1 < len(entries) {
			end = entries[i+1].offset
		}
		if end < e.offset || end > uint32(len(raw)) {
			fatal(fmt.Errorf("%s 的範圍無效：%d..%d", e.name, e.offset, end))
		}
		if err := os.WriteFile(filepath.Join(*out, e.name), raw[e.offset:end], 0o644); err != nil {
			fatal(err)
		}
		fmt.Printf("%-12s offset=%7d size=%7d\n", e.name, e.offset, end-e.offset)
		written++
	}
	if written == 0 {
		fatal(fmt.Errorf("找不到副檔名 %q 的資源", *ext))
	}
	fmt.Printf("共擷取 %d 個資源（目錄共 %d 筆）\n", written, len(entries))
}

func parseDirectory(raw []byte) ([]entry, error) {
	if len(raw) < entrySize {
		return nil, fmt.Errorf("檔案太短：%d", len(raw))
	}
	firstOffset := binary.LittleEndian.Uint32(raw[12:16])
	if firstOffset == 0 || firstOffset%entrySize != 0 || firstOffset > uint32(len(raw)) {
		return nil, fmt.Errorf("第一筆 offset 不是有效目錄長度：%d", firstOffset)
	}
	count := int(firstOffset / entrySize)
	entries := make([]entry, 0, count)
	var prev uint32
	for i := 0; i < count; i++ {
		p := i * entrySize
		name := strings.TrimRight(string(raw[p:p+12]), "\x00 ")
		off := binary.LittleEndian.Uint32(raw[p+12 : p+16])
		if name == "" || strings.ContainsAny(name, "/\\") {
			return nil, fmt.Errorf("第 %d 筆資源名無效：%q", i, name)
		}
		if off < firstOffset || off > uint32(len(raw)) || (i > 0 && off < prev) {
			return nil, fmt.Errorf("第 %d 筆 %s offset 無效：%d", i, name, off)
		}
		entries = append(entries, entry{name: name, offset: off})
		prev = off
	}
	return entries, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sc2kextract：", err)
	os.Exit(1)
}
