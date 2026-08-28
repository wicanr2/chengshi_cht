sim SoundOff
sim Disasters 0
sim Speed 0
sim Fill 0
sim Rand
sim Rand
sim Rand
sim Rand
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "EMPTYA [llength $_m] [join $_m ,]"
sim Speed 3
#sleep 2
sim Speed 0
sim Rand
sim Rand
sim Rand
sim Rand
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "EMPTYB [llength $_m] [join $_m ,]"
