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
import time

PROMPT = re.compile(rb"^sim:\s*$", re.M)


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

    with open(out_path, "w", encoding="utf-8") as fh:
        json.dump({"ok": ok,
                   "banner": banner.decode("utf-8", "replace").splitlines(),
                   "results": results}, fh, ensure_ascii=False, indent=1)
    print(f"寫入 {out_path}（{len(results)} 條指令，ok={ok}）")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
