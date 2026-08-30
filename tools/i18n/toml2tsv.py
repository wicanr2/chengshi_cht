#!/usr/bin/env python3
"""一次性轉檔：把舊的 `internal/i18n/messages/*.toml` 轉成 `*.tsv`。

留著是為了說明 TSV 那批檔案怎麼來的，不是流程的一部分——轉完之後
**TSV 就是唯一真相**，不再從任何 python 表重新產生。

用法：tools/i18n/toml2tsv.py internal/i18n/messages
"""
import os
import sys

LANGS = ["zh_hant", "zh_hans", "ja"]


def esc(s):
    return s.replace("\\", "\\\\").replace("\n", "\\n").replace("\t", "\\t")


def parse(path):
    rows = []
    key = None
    ln = 0
    zh = ""
    for line in open(path, encoding="utf-8"):
        line = line.rstrip("\n").strip()
        if line.startswith('["') and line.endswith('"]'):
            if key is not None:
                rows.append((key, ln, zh))
            key, ln, zh = line[2:-2], 0, ""
        elif line.startswith("len = ") and key is not None:
            ln = int(line[6:])
        elif line.startswith('zh = "') and key is not None:
            v = line[6:-1]
            out = []
            i = 0
            while i < len(v):
                if v[i] == "\\" and i + 1 < len(v):
                    out.append({"n": "\n", '"': '"', "\\": "\\"}.get(v[i + 1], v[i + 1]))
                    i += 2
                else:
                    out.append(v[i])
                    i += 1
            zh = "".join(out)
    if key is not None:
        rows.append((key, ln, zh))
    return rows


def main():
    d = sys.argv[1] if len(sys.argv) > 1 else "internal/i18n/messages"
    for name in sorted(os.listdir(d)):
        if not name.endswith(".toml"):
            continue
        rows = parse(os.path.join(d, name))
        out = os.path.join(d, name[:-5] + ".tsv")
        with open(out, "w", encoding="utf-8") as f:
            f.write("key\tlen\t" + "\t".join(LANGS) + "\n")
            for key, ln, zh in rows:
                f.write("%s\t%d\t%s\t\t\n" % (key, ln, esc(zh)))
        print("%-14s %d 列 → %s" % (name, len(rows), os.path.basename(out)))


if __name__ == "__main__":
    main()
