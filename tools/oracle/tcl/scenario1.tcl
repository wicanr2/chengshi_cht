# 載入劇本 1（Dullsville），倒出地圖與幾個純量。
# s_fileio.c:396 LoadScenario(1) → snro.111，CityTime=(1900-1900)*48+2，Funds=5000
sim Speed 0
sim Pause
sim LoadScenario 1
sim Pause
sim CityName
sim Funds
sim TaxRate
sim Year
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "MAPDUMP [llength $_m] [join $_m ,]"
