# 逐 frame 對拍（劇本版・東京）：載入 Tokyo，然後單步 8000 個 frame。
#
# 選東京是為了**精靈**：它的劇本災難就是怪獸，而怪獸是精靈系統唯一
# 在對拍實驗裡穩定出現得了的東西（Dullsville 那份實測整段 0 個精靈）。
#
# 東京大得多（8000 個 frame 近九十萬次抽樣），所以每個 frame 只問
# FrameStats（規則與精靈各抽幾次），不問 Problems／VoteStats——
# 那兩個各要三十幾個 token，八千個 frame 會讓 pty 塞到跑不完。
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
puts stdout "SPR [sim SpriteCycle] ; [sim Sprites]"
puts stdout "CHK [sim Mem LandValueMem 30 25] [sim Mem PopDensity 30 25] [sim Mem LandValueMem 10 40] [sim Mem PopDensity 10 40] [sim Mem ComRate 3 7]"
puts stdout "INIT [sim Fcycle] [sim Scycle] [sim Funds]"
puts stdout "R0 [sim Rand] [sim Rand] [sim Rand] [sim Rand]"
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "CP0 [llength $_m] [join $_m ,]"
# ⚠ 逐 frame 的資料**寫檔案，不走 pty**（drive.py 收尾時把 /out/lines.txt
# 併回結果）。走 pty 會在某些輸出組合下卡到一行都不吐；走檔案沒有這問題，
# 而且快得多。原因見 docs/re/12 §六之九。
set fh [open /out/lines.txt w]
for {set i 0} {$i < 8000} {incr i} { sim Frame 1 ; puts $fh "F $i [sim Fcycle] [sim Scycle] [sim Valves] [sim Rand] [sim Rand] [sim Rand] [sim Rand] [sim FrameStats]" ; puts $fh "MH $i [sim MapHash]" ; if {$i < 120} { puts $fh "S $i ; [sim Sprites]" } }
close $fh
puts stdout "LOOPDONE"
puts stdout "END [sim Fcycle] [sim Scycle] [sim Funds]"
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "CPEND [llength $_m] [join $_m ,]"
