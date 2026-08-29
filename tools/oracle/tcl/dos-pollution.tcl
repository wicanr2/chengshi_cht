# 拿 DOS 原版的存檔問 Micropolis：同一張地圖，它算出來的汙染／地價／犯罪
# 平均是多少？
#
# 這是三方對質：DOS 自己記在 MiscHis 的值、我們的 Go 版算的、Micropolis
# 算的。前兩者差很多（docs/re/18-dos-parity.md §六），這一份決定是
# 「我們與 Micropolis 不同」還是「DOS 與 Micropolis 不同」。
sim SoundOff
sim Disasters 0
sim Speed 0
set fh [open /out/dospoll.txt w]
foreach f {scen1 scen2 scen3 scen4 scen5 scen6 scen7 scen8 run1 run2 run3 run4 run5 run6 run7 run8} { sim LoadCity /out/$f.cty ; sim Speed 0 ; puts $fh "$f pollution [sim Pollution] landvalue [sim LandValue] crime [sim Crime] funds [sim Funds] year [sim Year]" }
close $fh
puts stdout "OK"
