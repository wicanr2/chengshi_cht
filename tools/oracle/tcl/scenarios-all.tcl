sim SoundOff
sim Disasters 0
sim Speed 0
sim LoadScenario 1
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "SC1 [llength $_m] [join $_m ,]"
sim LoadScenario 2
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "SC2 [llength $_m] [join $_m ,]"
sim LoadScenario 3
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "SC3 [llength $_m] [join $_m ,]"
sim LoadScenario 4
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "SC4 [llength $_m] [join $_m ,]"
sim LoadScenario 5
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "SC5 [llength $_m] [join $_m ,]"
sim LoadScenario 6
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "SC6 [llength $_m] [join $_m ,]"
sim LoadScenario 7
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "SC7 [llength $_m] [join $_m ,]"
sim LoadScenario 8
sim Speed 0
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "SC8 [llength $_m] [join $_m ,]"
