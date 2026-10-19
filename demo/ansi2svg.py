#!/usr/bin/env python3
"""Render ANSI terminal output (stdin) as an SVG "screenshot" (stdout).
Usage: ansi2svg.py [title [cols rows]] < out.txt   (cols/rows fix the window size, like a real terminal)"""
import re, sys
from xml.sax.saxutils import escape

COL = {30: "#6e7681", 90: "#8b949e", 91: "#ff7b72", 92: "#56d364", 93: "#e3b341", 94: "#79c0ff", 95: "#d2a8ff", 96: "#76e3ea"}
FG, BG = "#c9d1d9", "#0d1117"
CW, LH, FS, PAD, BAR = 8.4, 18, 14, 18, 34

raw = sys.stdin.read()
raw = re.sub(r"\x1b\]8;;[^\x1b\x07]*(?:\x1b\\|\x07)", "", raw)  # drop OSC 8 hyperlinks
title = sys.argv[1] if len(sys.argv) > 1 else "pipeline"

rows = []
for line in raw.rstrip("\n").split("\n"):
    runs, color, bold, pos = [], FG, False, 0
    for m in re.finditer(r"\x1b\[([0-9;]*)m|([^\x1b]+)", line):
        if m.group(2) is not None:
            runs.append((m.group(2), color, bold)); continue
        for c in (m.group(1) or "0").split(";"):
            c = int(c or 0)
            if c == 0: color, bold = FG, False
            elif c == 1: bold = True
            elif c in COL: color = COL[c]
    rows.append(runs)

cols = max(sum(len(t) for t, _, _ in r) for r in rows)
if len(sys.argv) > 3:
    cols = max(cols, int(sys.argv[2]))
    rows += [[]] * (int(sys.argv[3]) - len(rows))
W = round(cols * CW + 2 * PAD)
H = round(len(rows) * LH + 2 * PAD + BAR)
out = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" role="img" aria-label="{escape(title)}">',
       f'<rect width="{W}" height="{H}" rx="10" fill="{BG}"/>',
       f'<rect width="{W}" height="{BAR}" rx="10" fill="#161b22"/><rect y="{BAR-10}" width="{W}" height="10" fill="#161b22"/>',
       '<circle cx="20" cy="17" r="6" fill="#ff5f56"/><circle cx="40" cy="17" r="6" fill="#ffbd2e"/><circle cx="60" cy="17" r="6" fill="#27c93f"/>',
       f'<text x="{W/2}" y="22" fill="#8b949e" font-family="ui-monospace,Menlo,monospace" font-size="12" text-anchor="middle">{escape(title)}</text>',
       f'<g font-family="ui-monospace,SFMono-Regular,Menlo,Consolas,monospace" font-size="{FS}" xml:space="preserve">']
for i, runs in enumerate(rows):
    y = BAR + PAD + i * LH + FS
    x = PAD
    for text, color, bold in runs:
        n = len(text)
        if text.strip():
            w = 'font-weight="bold" ' if bold else ""
            xs = " ".join(f"{x + k * CW:.1f}" for k in range(n))  # one x per character: exact grid in any font
            out.append(f'<text x="{xs}" y="{y}" fill="{color}" {w}xml:space="preserve">{escape(text)}</text>')
        x += n * CW
out.append("</g></svg>")
print("\n".join(out))
