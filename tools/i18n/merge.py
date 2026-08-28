#!/usr/bin/env python3
"""把譯文合併進 internal/i18n/messages/*.toml。

骨架由 `simtool messages` 產生（只有鍵與原文長度，不含原文）。
譯文寫在同目錄的 `base_zh.py` 與 `styles_zh.py`：

  base_zh.ZH      基本檔（MESSAGE.PTF）的譯文
  styles_zh.STYLES  六個風格包**與基本檔不同**的那些鍵

風格包的鍵沒有覆寫時就沿用基本檔——原版本來就只換掉一部分。

用法（在 docker 裡跑，見 tools/i18n.sh）：
    tools/i18n/merge.py internal/i18n/messages
"""
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from base_zh import ZH as BASE  # noqa: E402

try:
    from styles_zh import STYLES  # noqa: E402
except ImportError:
    STYLES = {}

# 第 2 段（圖片訊息與劇本簡介）的風格改寫版。每個風格一個檔，
# 因為那些是整段文字，混在 styles_zh 裡會讓那個檔難以閱讀。
for _style, _mod in (("asia", "pic_asia"), ("medi", "pic_medi"),
                     ("west", "pic_west"), ("fusa", "pic_fusa"),
                     ("feur", "pic_feur"), ("moon", "pic_moon")):
    try:
        _m = __import__(_mod)
    except ImportError:
        continue
    STYLES.setdefault(_style, {}).update(_m.PIC)

# 檔名 → 風格代號
FILES = {
    "message": None,
    "asia_msg": "asia",
    "medi_msg": "medi",
    "west_msg": "west",
    "fusa_msg": "fusa",
    "feur_msg": "feur",
    "moon_msg": "moon",
}


def toml_escape(s):
    return s.replace("\\", "\\\\").replace('"', '\\"').replace("\n", "\\n")


def merge(path, table):
    with open(path, encoding="utf-8") as f:
        text = f.read()
    filled = 0
    total = 0

    def repl(m):
        nonlocal filled, total
        key, body = m.group(1), m.group(2)
        total += 1
        zh = table.get(key)
        if not zh:
            return m.group(0)
        filled += 1
        # ⚠ 替換字串一定要用 lambda。re.sub 會對**替換字串**再解析一次
        # 反斜線跳脫，所以 "\\n" 會變成真正的換行，多行的圖片訊息就
        # 被寫成好幾行、載入時只讀到第一行——而且 TOML 看起來還是合法的。
        new = 'zh = "%s"' % toml_escape(zh)
        body = re.sub(r'zh = ".*?"(?=\n)', lambda _m: new, body, count=1)
        return '["%s"]\n%s' % (key, body)

    text = re.sub(r'\["([\d.]+)"\]\n((?:[a-z]+ = .*\n)+)', repl, text)
    with open(path, "w", encoding="utf-8") as f:
        f.write(text)
    return filled, total


def main():
    d = sys.argv[1] if len(sys.argv) > 1 else "internal/i18n/messages"
    for name, style in sorted(FILES.items()):
        p = os.path.join(d, name + ".toml")
        if not os.path.exists(p):
            continue
        table = dict(BASE)
        if style:
            table.update(STYLES.get(style, {}))
        filled, total = merge(p, table)
        print("%-12s %3d/%3d 已翻譯（%.0f%%）" % (name, filled, total, 100.0 * filled / total))


if __name__ == "__main__":
    main()
