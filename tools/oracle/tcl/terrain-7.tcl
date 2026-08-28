# 固定種子 7 的地形黃金樣本。s_gen.c:129 SeedRand(r)
sim Speed 0
sim Pause
sim GenerateSomeCity 7
sim Pause
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "MAPDUMP [llength $_m] [join $_m ,]"
