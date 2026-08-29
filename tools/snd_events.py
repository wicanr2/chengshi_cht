"""從 DOSBox 錄音裡切出「真的有內容」的聲音事件，對到動作時間表上。

用法：python3 tools/snd_events.py <錄音.raw> <時間表.marks>

⚠ 判準用的是**每 5 毫秒的標準差**，不是音量。DOSBox 0.74 的 PC 喇叭在
遊戲放完聲音之後會停在一個固定電位，音量看起來很大但完全沒有內容；
用音量當門檻會把那一段長長的直流也算成聲音，量出來每一段都剛好等於
兩個動作之間的間隔——一個看起來很合理但完全是假的數字。
"""

import array
import statistics
import sys

RATE, FRAME = 22050, 110  # 5 毫秒


def main():
    a = array.array("h")
    with open(sys.argv[1], "rb") as f:
        a.frombytes(f.read())
    if sys.byteorder == "big":
        a.byteswap()
    n = len(a) // 2
    left = [a[2 * i] for i in range(n)]

    ac = [statistics.pstdev(left[i : i + FRAME])
          for i in range(0, n - FRAME + 1, FRAME)]
    peak = max(ac) or 1
    floor = peak * 0.05

    marks = []
    for line in open(sys.argv[2], encoding="utf-8"):
        line = line.strip()
        if line:
            t, rest = line.split(None, 1)
            marks.append((float(t), rest.strip()))

    regs, i = [], 0
    while i < len(ac):
        if ac[i] > floor:
            j = i
            # 允許 100 毫秒的空檔，同一個音效裡的停頓不要拆成兩段
            quiet = 0
            while j < len(ac):
                if ac[j] > floor:
                    quiet = 0
                else:
                    quiet += 1
                    if quiet > 20:
                        break
                j += 1
            end = j - quiet
            if end - i >= 2:
                regs.append((i * FRAME / RATE, (end - i) * FRAME / RATE,
                             max(ac[i:end])))
            i = j
        else:
            i += 1

    print(f"錄音 {n / RATE:.1f} 秒，{len(regs)} 個有內容的事件")
    for t0, d, p in regs:
        before = [m for m in marks if m[0] <= t0 + 0.5]
        label = before[-1][1] if before else ""
        print(f"  {t0:7.2f} 秒  長 {d:6.3f} 秒  強度 {p:5.0f}  ← {label}")


if __name__ == "__main__":
    main()
