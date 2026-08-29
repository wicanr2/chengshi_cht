# 逐 frame 對拍在評估那個 frame 分岔：問清楚原版的抽樣是誰用掉的。
sim SoundOff
sim Speed 0
sim Pause
sim LoadScenario 1
sim Pause
sim Speed 0
sim Frame 1512
puts stdout "V1512 [sim VoteStats] | [sim Problems]"
sim Frame 48
puts stdout "V1560 [sim VoteStats] | [sim Problems]"
