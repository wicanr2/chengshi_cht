# 驗證加進去的兩個觀測指令：sim Frame N 與 sim Scycle。
sim SoundOff
sim Disasters 0
sim Speed 0
sim GenerateSomeCity 12345
sim Speed 0
puts stdout "SCY0 [sim Scycle]"
sim Rand
sim Frame 1
puts stdout "SCY1 [sim Scycle]"
sim Rand
sim Frame 16
puts stdout "SCY16 [sim Scycle]"
sim Rand
sim Frame 160
puts stdout "SCY160 [sim Scycle]"
sim Rand
