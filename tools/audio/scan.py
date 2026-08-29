# oto 的輸出流是 float32 立體聲（Ebiten 內部混音就是 float32），
# 不是 int16——照 int16 讀會得到剛好兩倍的長度與假的峰值 32768。
import struct
f = open('/cap/out.raw', 'rb')
total = nz = 0
peak = 0.0
first = last = -1
while True:
    b = f.read(1 << 20)
    if not b:
        break
    n = len(b) // 4
    for i, v in enumerate(struct.unpack("<%df" % n, b[:n * 4])):
        if v != 0.0:
            nz += 1
            if first < 0:
                first = total + i
            last = total + i
            if abs(v) > peak:
                peak = abs(v)
    total += n
print("總 float32", total, "非零", nz, "峰值 %.3f" % peak)
if nz:
    frames = (last - first + 1) / 2
    print("非零跨度 %.3f 秒（48000 Hz 立體聲）" % (frames / 48000))
