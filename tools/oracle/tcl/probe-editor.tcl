sim SoundOff
sim Disasters 0
sim Speed 0
sim GenerateSomeCity 12345
UINewGame
sim Speed 0
set _e {}
proc scanw {w} { global _e ; if {[winfo class $w] == "Editorview"} { lappend _e $w } ; foreach c [winfo children $w] { scanw $c } }
foreach t [winfo children .] { scanw $t }
puts stdout "EDITORS $_e"
