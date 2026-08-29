sim SoundOff
sim Disasters 0
sim Speed 0
sim GenerateSomeCity 777
sim Speed 0
sim Fill 0
sim Speed 3
#sleep 3
sim Speed 0
sim Tile 49 49 29288
sim Tile 50 49 29288
sim Tile 51 49 29288
sim Tile 49 50 29288
sim Tile 50 50 30312
sim Tile 51 50 29288
sim Tile 49 51 29288
sim Tile 50 51 29288
sim Tile 51 51 29288
sim Year
sim Rand
sim Rand
sim Rand
sim Rand
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "ZA [llength $_m] [join $_m ,]"
sim Speed 3
#sleep 2
sim Speed 0
sim Year
sim Rand
sim Rand
sim Rand
sim Rand
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "ZB [llength $_m] [join $_m ,]"
