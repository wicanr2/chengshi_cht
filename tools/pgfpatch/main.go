// pgfpatch：把 .PGF 解壓、（可選）改掉某一庫的像素、再用「全字面」LZSS 重壓回去。
//
// 全字面重壓是刻意的：不必實作壓縮搜尋，輸出一定能被原版解碼器讀回來，
// 代價是檔案大 12.5%。**不改內容的重壓是這個實驗的正對照**——
// 先證明重壓本身不會讓畫面變樣，改動造成的差異才算數。
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
)

func header(raw []byte) int {
	eof := bytes.IndexByte(raw, 0x1a)
	if eof < 0 || eof > 300 {
		return 0
	}
	p := eof + 1
	z := bytes.IndexByte(raw[p:], 0)
	if z < 0 || z > 64 {
		return 0
	}
	p += z + 1
	p++ // mode
	for i := 0; i < 3; i++ {
		z := bytes.IndexByte(raw[p:], 0)
		if z < 0 || z > 64 {
			break
		}
		p += z + 1
	}
	return p
}

func decomp(src []byte) []byte {
	var win [4096]byte
	for i := range win {
		win[i] = 0x20
	}
	out := []byte{}
	r, i := 4078, 0
	for i < len(src) {
		fl := src[i]
		i++
		for b := 0; b < 8 && i < len(src); b++ {
			if fl&(1<<uint(b)) != 0 {
				c := src[i]
				i++
				out = append(out, c)
				win[r] = c
				r = (r + 1) % 4096
				continue
			}
			if i+1 >= len(src) {
				return out
			}
			b1, b2 := int(src[i]), int(src[i+1])
			i += 2
			off, ln := b1|((b2&0xF0)<<4), (b2&0x0F)+3
			for k := 0; k < ln; k++ {
				c := win[(off+k)%4096]
				out = append(out, c)
				win[r] = c
				r = (r + 1) % 4096
			}
		}
	}
	return out
}

// comp 全部當字面：每八個位元組前面加一個 0xFF 旗標。
func comp(d []byte) []byte {
	out := make([]byte, 0, len(d)*9/8+8)
	for i := 0; i < len(d); i += 8 {
		n := len(d) - i
		if n > 8 {
			n = 8
		}
		out = append(out, byte((1<<uint(n))-1))
		out = append(out, d[i:i+n]...)
	}
	return out
}

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	start := header(raw)
	d := decomp(raw[start:])
	fmt.Println("解出", len(d), "位元組")

	if len(os.Args) > 3 { // 有指定要改的庫
		want, _ := strconv.Atoi(os.Args[3])
		fill := byte(0xFF)
		if len(os.Args) > 4 {
			v, _ := strconv.ParseUint(os.Args[4], 0, 8)
			fill = byte(v)
		}
		n := int(binary.LittleEndian.Uint16(d[0:]))
		bpp := int(d[2])
		p := 5 + (1<<uint(bpp))*3
		for i := 0; i < n; i++ {
			w := int(binary.LittleEndian.Uint16(d[p:]))
			h := int(binary.LittleEndian.Uint16(d[p+2:]))
			cnt := int(binary.LittleEndian.Uint16(d[p+4:]))
			fl := binary.LittleEndian.Uint16(d[p+6:])
			size := int(binary.LittleEndian.Uint32(d[p+8:]))
			p += 12
			if i == want {
				planes := bpp
				if fl&0x0100 != 0 {
					planes = 1
				}
				per := w * h * planes / 8
				q := p
				for j := 0; j < cnt; j++ {
					if fl&0x0001 != 0 {
						q += 4
					}
					for k := 0; k < per; k++ {
						d[q+k] = fill
					}
					q += per
				}
				fmt.Printf("改了第 %d 庫：%dx%d ×%d，每張 %d 位元組，共 %d，填 %#02x\n",
					i, w, h, cnt, per, q-p, fill)
			}
			p += size
		}
	}
	out := append(append([]byte{}, raw[:start]...), comp(d)...)
	if err := os.WriteFile(os.Args[2], out, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("寫出", os.Args[2], len(out), "位元組（原本", len(raw), "）")
}
