# 用固定種子產生地形，然後把整張 120x100 地圖倒出來。
# GenerateMap(r) 一進來就 SeedRand(r)（s_gen.c:129），所以同一個 r 一定同一張圖。
sim Speed 0
sim Pause
sim GenerateSomeCity 12345
sim Pause
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "MAPDUMP [llength $_m] [join $_m ,]"
