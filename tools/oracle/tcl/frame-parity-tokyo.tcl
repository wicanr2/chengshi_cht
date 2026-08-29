# 逐 frame 對拍（劇本版・東京）：載入 Tokyo，然後單步 8000 個 frame。
#
# 選東京是為了**精靈**：它的劇本災難就是怪獸，而怪獸是精靈系統唯一
# 在對拍實驗裡穩定出現得了的東西（Dullsville 那份實測整段 0 個精靈）。
#
# 比 frame-parity.tcl 那座「三個空住宅區」的城市豐富得多——有人口、
# 有交通、有稅收、有精靈（火車、直昇機），所以掃到的規則也多得多。
sim SoundOff
sim Speed 0
sim Pause
puts stdout "PRE [sim RandState]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem TrfDensity $x $y] } } ; puts stdout "PRETRF [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem RateOGMem $x $y] } } ; puts stdout "PREROG [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem ComRate $x $y] } } ; puts stdout "PRECOM [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem PopDensity $x $y] } } ; puts stdout "PREPOP [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem LandValueMem $x $y] } } ; puts stdout "PRELV [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem PollutionMem $x $y] } } ; puts stdout "PREPOL [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem CrimeMem $x $y] } } ; puts stdout "PRECRI [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem FireStMap $x $y] } } ; puts stdout "PREFST [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem PoliceMap $x $y] } } ; puts stdout "PREPLC [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem PoliceMapEffect $x $y] } } ; puts stdout "PREPLE [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem FireRate $x $y] } } ; puts stdout "PREFRT [llength $_a] [join $_a ,]"
sim LoadScenario 5
sim Pause
sim Speed 0
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem TrfDensity $x $y] } } ; puts stdout "POSTTRF [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem RateOGMem $x $y] } } ; puts stdout "POSTROG [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem ComRate $x $y] } } ; puts stdout "POSTCOM [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem PopDensity $x $y] } } ; puts stdout "POSTPOP [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem LandValueMem $x $y] } } ; puts stdout "POSTLV [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem PollutionMem $x $y] } } ; puts stdout "POSTPOL [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 50} {incr y} { for {set x 0} {$x < 60} {incr x} { lappend _a [sim Mem CrimeMem $x $y] } } ; puts stdout "POSTCRI [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem FireStMap $x $y] } } ; puts stdout "POSTFST [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem PoliceMap $x $y] } } ; puts stdout "POSTPLC [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem PoliceMapEffect $x $y] } } ; puts stdout "POSTPLE [llength $_a] [join $_a ,]"
set _a {} ; for {set y 0} {$y < 13} {incr y} { for {set x 0} {$x < 15} {incr x} { lappend _a [sim Mem FireRate $x $y] } } ; puts stdout "POSTFRT [llength $_a] [join $_a ,]"
puts stdout "CHK [sim Mem LandValueMem 30 25] [sim Mem PopDensity 30 25] [sim Mem LandValueMem 10 40] [sim Mem PopDensity 10 40] [sim Mem ComRate 3 7]"
puts stdout "INIT [sim Fcycle] [sim Scycle] [sim Funds]"
puts stdout "R0 [sim Rand] [sim Rand] [sim Rand] [sim Rand]"
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "CP0 [llength $_m] [join $_m ,]"
for {set i 0} {$i < 8000} {incr i} { sim Frame 1 ; puts stdout "F $i [sim Fcycle] [sim Scycle] [sim Valves] [sim Rand] [sim Rand] [sim Rand] [sim Rand] [sim Problems] [sim VoteStats]" }
puts stdout "END [sim Fcycle] [sim Scycle] [sim Funds]"
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "CPEND [llength $_m] [join $_m ,]"
