# 跑到逐 frame 對拍分岔的那個 frame，把評估的問題表倒出來。
sim SoundOff
sim Speed 0
sim Pause
sim LoadScenario 1
sim Pause
sim Speed 0
puts stdout "R0 [sim Rand] [sim Rand] [sim Rand] [sim Rand]"
sim Frame 1512
puts stdout "BEFORE [sim Problems]"
puts stdout "BR [sim Rand] [sim Rand] [sim Rand] [sim Rand]"
sim Frame 1
puts stdout "AFTER [sim Problems]"
puts stdout "AR [sim Rand] [sim Rand] [sim Rand] [sim Rand]"
