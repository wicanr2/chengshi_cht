#!/usr/bin/env python3
"""在 pty 底下驅動 Micropolis oracle，逐行送 Tcl 指令並收結果。

為什麼要 pty：`sim -t` 只有在 `isatty(0)` 為真時才會註冊 stdin 的 Tcl 讀取器
（sim.c:629、w_tk.c:808）。stdin 是管線的話這個 REPL 根本不會開，
而且**不會有任何錯誤訊息**——看起來就只是「指令沒反應」。

協定：tty 模式下每執行完一條指令會印一行 `sim:`（w_tk.c:541）。
以它當提示字元切分結果。

用法（容器內）：
    drive.py <指令檔> <輸出 json>
指令檔一行一條 Tcl；`#` 開頭與空行略過。
"""
import json
import os
import pty
import re
import select
import subprocess
import sys
import termios
import time
import tty

PROMPT = re.compile(rb"^sim:\s*$", re.M)

# 指令行的長度上限（見 main() 裡那段註解）。留一點餘裕。
MAX_CMD = 180


def read_until_prompt(fd, timeout=30.0, want=1):
    """讀到看見 want 個提示字元為止。回傳原始 bytes。"""
    buf = b""
    deadline = time.time() + timeout
    while time.time() < deadline:
        r, _, _ = select.select([fd], [], [], 0.2)
        if r:
            try:
                chunk = os.read(fd, 65536)
            except OSError:
                break
            if not chunk:
                break
            buf += chunk
            if len(PROMPT.findall(buf)) >= want:
                return buf
    return buf


def main():
    if len(sys.argv) != 3:
        print(__doc__)
        return 2
    script_path, out_path = sys.argv[1], sys.argv[2]

    with open(script_path, encoding="utf-8") as fh:
        raw_lines = [ln.rstrip("\n") for ln in fh]
    # `#sleep N` 是給驅動器看的指令：讓 sim 的事件迴圈自己跑 N 秒。
    # 需要它是因為模擬要靠 Tk 的計時器推進，而我們每送一條指令就等提示字元回來，
    # 中間幾乎不留時間給它跑。
    cmds = []
    for ln in raw_lines:
        st = ln.strip()
        if not st:
            continue
        if st.startswith("#sleep"):
            cmds.append(st)
            continue
        if st.startswith("#"):
            continue
        cmds.append(ln)

    master, slave = pty.openpty()
    # ⚠ **把 pty 切成 raw**（關掉 ICANON 與 ECHO）。
    # 需要 pty 是因為 `sim -t` 只在 `isatty(0)` 為真時才開 REPL；
    # 但**不需要行規則**。留著 canonical 模式的話，長的指令行會在某些
    # 組合下整條卡住：sim 的 CPU 掉到 0（在等輸入），而下一次寫入一送出，
    # 前一條就連同輸出一起冒出來。切成 raw 之後沒再看過這個現象。
    tty.setraw(slave, termios.TCSANOW)
    proc = subprocess.Popen(
        ["res/sim", "-t"],
        stdin=slave, stdout=slave, stderr=subprocess.STDOUT,
        close_fds=True,
    )
    os.close(slave)

    banner = read_until_prompt(master, timeout=60.0)
    results = []
    ok = True
    for cmd in cmds:
        if cmd.startswith("#sleep"):
            secs = float(cmd.split()[1])
            # 一邊睡一邊把 sim 的輸出讀掉，否則 pty 緩衝區滿了它會卡住。
            end = time.time() + secs
            while time.time() < end:
                r, _, _ = select.select([master], [], [], 0.2)
                if r:
                    try:
                        os.read(master, 65536)
                    except OSError:
                        break
            results.append({"cmd": cmd, "out": []})
            continue
        # ⚠ **一行不能超過 199 個字元。** oracle 的 stdin 讀取器是
        # `w_tk.c:508 StdinProc` 的 `char line[200]` ＋ `fgets(line, 200, stdin)`：
        # 更長的一行會被切成兩段，第一段組不成完整指令（`gotPartial = 1`）就
        # return，而**剩下的位元組已經在 stdio 的緩衝裡、不在 fd 上**——
        # `Tk_CreateFileHandler` 的 select 因此永遠不再說「可讀」，整個 REPL
        # 就停在那裡（CPU 掉到 0），直到下一次寫入才把前一條連同輸出一起吐出來。
        # 對策：太長的指令寫成檔案再 `source`，那一行只有二十幾個字元。
        if len(cmd) > MAX_CMD:
            side = os.path.join(os.path.dirname(out_path) or ".", "_cmd.tcl")
            with open(side, "w", encoding="utf-8") as fh:
                fh.write(cmd + "\n")
            os.write(master, f"source {side}\n".encode())
        else:
            os.write(master, (cmd + "\n").encode())
        # 單步 800 個 frame 的那一行會跑很久，逾時放寬（可用 ORACLE_TIMEOUT 調）。
        raw = read_until_prompt(master, timeout=float(os.environ.get("ORACLE_TIMEOUT", "600")))
        text = raw.decode("utf-8", "replace")
        # 回顯的指令與提示字元都去掉，剩下的才是結果
        lines = [ln for ln in text.splitlines()
                 if ln.strip() and ln.strip() != "sim:" and ln.strip() != cmd.strip()]
        results.append({"cmd": cmd, "out": lines})
        if not raw:
            ok = False
            break

    try:
        os.write(master, b"sim ReallyQuit\n")
        time.sleep(0.5)
    except OSError:
        pass
    proc.terminate()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()

    # 側錄檔：Tcl 腳本可以 `set fh [open /out/lines.txt w]` 把逐 frame 的資料
    # 寫成檔案，不走 pty。**大量逐 frame 輸出一定要走這條**——同一個迴圈
    # （400 個 frame、含 MapHash）走 pty 會卡住不吐任何一行，走檔案 3 秒跑完。
    # 卡住的確切機制沒查出來，但兩邊的對照很乾淨，記在 docs/re/12 §六之九。
    side = os.path.join(os.path.dirname(out_path) or ".", "lines.txt")
    if os.path.exists(side):
        with open(side, encoding="utf-8", errors="replace") as fh:
            results.append({"cmd": "<lines.txt>",
                            "out": [ln.rstrip("\n") for ln in fh if ln.strip()]})
        os.remove(side)

    with open(out_path, "w", encoding="utf-8") as fh:
        json.dump({"ok": ok,
                   "banner": banner.decode("utf-8", "replace").splitlines(),
                   "results": results}, fh, ensure_ascii=False, indent=1)
    print(f"寫入 {out_path}（{len(results)} 條指令，ok={ok}）")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
