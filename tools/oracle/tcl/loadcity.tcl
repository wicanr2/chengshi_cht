sim SoundOff
sim Disasters 0
sim Speed 0
sim LoadCity /out/roundtrip.cty
sim Speed 0
sim Funds
sim Year
set _m {} ; for {set y 0} {$y < 100} {incr y} { for {set x 0} {$x < 120} {incr x} { lappend _m [sim Tile $x $y] } } ; puts stdout "M [llength $_m] [join $_m ,]"
