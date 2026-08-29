# 逐 frame 對拍：用 sim Frame 單步，每一步記下 Fcycle、Scycle 與亂數狀態。
#
# 這份取代 tick-parity-seg.tcl 的「Speed 3 然後馬上 Speed 0」——那種切法
# 每段跑幾個 frame 由事件迴圈決定，長度不可控（實測 85 到 2039 次抽樣）。
# 單步之後「哪一個 frame 開始分岔」是查表，不是搜尋。
#
# 需要 tools/oracle/patches/apply.py 加的三個觀測指令。
# 整刻對拍實驗 C：產生地形 → 暫停 → 種一座電廠、一條路、幾個空住宅區
# → 讀亂數狀態 → 倒地圖 → 跑一段 → 暫停 → 再讀狀態與地圖。
sim SoundOff
sim Disasters 0
sim Speed 0
sim GenerateSomeCity 12345
sim Speed 0
sim Tile 10 10 25321
sim Tile 11 10 25322
sim Tile 12 10 25323
sim Tile 13 10 25324
sim Tile 10 11 25325
sim Tile 11 11 26350
sim Tile 12 11 25327
sim Tile 13 11 25328
sim Tile 10 12 25329
sim Tile 11 12 27378
sim Tile 12 12 25331
sim Tile 13 12 25332
sim Tile 10 13 25333
sim Tile 11 13 25334
sim Tile 12 13 25335
sim Tile 13 13 25336
sim Tile 14 11 28880
sim Tile 15 11 28880
sim Tile 16 11 28880
sim Tile 17 11 28880
sim Tile 18 11 28880
sim Tile 19 11 28880
sim Tile 20 11 28880
sim Tile 21 11 28880
sim Tile 22 11 28880
sim Tile 23 11 28880
sim Tile 24 11 28880
sim Tile 25 11 28880
sim Tile 26 11 28880
sim Tile 27 11 28880
sim Tile 28 11 28880
sim Tile 29 11 28880
sim Tile 30 11 28880
sim Tile 10 20 28738
sim Tile 11 20 28738
sim Tile 12 20 28738
sim Tile 13 20 28738
sim Tile 14 20 28738
sim Tile 15 20 28738
sim Tile 16 20 28738
sim Tile 17 20 28738
sim Tile 18 20 28738
sim Tile 19 20 28738
sim Tile 20 20 28738
sim Tile 21 20 28738
sim Tile 22 20 28738
sim Tile 23 20 28738
sim Tile 24 20 28738
sim Tile 25 20 28738
sim Tile 26 20 28738
sim Tile 27 20 28738
sim Tile 28 20 28738
sim Tile 29 20 28738
sim Tile 30 20 28738
sim Tile 31 20 28738
sim Tile 32 20 28738
sim Tile 33 20 28738
sim Tile 34 20 28738
sim Tile 35 20 28738
sim Tile 36 20 28738
sim Tile 37 20 28738
sim Tile 38 20 28738
sim Tile 39 20 28738
sim Tile 40 20 28738
sim Tile 14 17 28916
sim Tile 15 17 28916
sim Tile 16 17 28916
sim Tile 14 18 28916
sim Tile 15 18 29940
sim Tile 16 18 28916
sim Tile 14 19 28916
sim Tile 15 19 28916
sim Tile 16 19 28916
sim Tile 19 17 28916
sim Tile 20 17 28916
sim Tile 21 17 28916
sim Tile 19 18 28916
sim Tile 20 18 29940
sim Tile 21 18 28916
sim Tile 19 19 28916
sim Tile 20 19 28916
sim Tile 21 19 28916
sim Tile 24 17 28916
sim Tile 25 17 28916
sim Tile 26 17 28916
sim Tile 24 18 28916
sim Tile 25 18 29940
sim Tile 26 18 28916
sim Tile 24 19 28916
sim Tile 25 19 28916
sim Tile 26 19 28916
sim Tile 14 16 28880
sim Tile 15 16 28880
sim Tile 16 16 28880
sim Tile 17 16 28880
sim Tile 18 16 28880
sim Tile 19 16 28880
sim Tile 20 16 28880
sim Tile 21 16 28880
sim Tile 22 16 28880
sim Tile 23 16 28880
sim Tile 24 16 28880
sim Tile 25 16 28880
sim Tile 26 16 28880
sim Tile 14 12 28880
sim Tile 14 13 28880
sim Tile 14 14 28880
sim Tile 14 15 28880
puts stdout "INIT [sim Fcycle] [sim Scycle] [sim Funds]"
puts stdout "R0 [sim Rand] [sim Rand] [sim Rand] [sim Rand]"
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "CP0 [llength $_m] [join $_m ,]"
for {set i 0} {$i < 8000} {incr i} { sim Frame 1 ; puts stdout "F $i [sim Fcycle] [sim Scycle] [sim Valves] [sim Rand] [sim Rand] [sim Rand] [sim Rand]" }
puts stdout "END [sim Fcycle] [sim Scycle] [sim Funds]"
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "CPEND [llength $_m] [join $_m ,]"
