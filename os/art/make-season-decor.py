#!/usr/bin/env python3
"""The decorations on Ollie OS's seasonal backgrounds (seasons/*.svg, made
once): pumpkins, bats and stars for Halloween; snowflakes and snowy hills for
winter; balloons and confetti for birthdays. colour-wallpapers.sh puts each
under the cube of its background. Kept clear of the middle, where the cube
and "Ollie OS" are.

  python3 -I os/art/make-season-decor.py
"""
import math
import os
import random

W, H = 1920, 1080
here = os.path.join(os.path.dirname(os.path.abspath(__file__)), "seasons")


def spots(n, seed, margin=60):
    """n places around the edges, not on the cube or the name."""
    r = random.Random(seed)
    out = []
    while len(out) < n:
        x, y = r.uniform(margin, W - margin), r.uniform(margin, H - margin)
        if abs(x - 960) < 420 and 80 < y < 830:
            continue
        if all(math.hypot(x - a, y - b) > 110 for a, b, _ in out):
            out.append((x, y, r.random()))
    return out


def pumpkin(x, y, s):
    g = [f'<g transform="translate({x:.0f} {y:.0f}) scale({s:.2f})">',
         '<ellipse cx="0" cy="0" rx="62" ry="50" fill="#ea580c"/>',
         '<ellipse cx="-30" cy="0" rx="30" ry="48" fill="#f97316"/>',
         '<ellipse cx="30" cy="0" rx="30" ry="48" fill="#f97316"/>',
         '<ellipse cx="0" cy="0" rx="22" ry="50" fill="#fb923c"/>',
         '<path d="M-4,-46 q2,-22 16,-26" stroke="#3f6212" stroke-width="9" fill="none" stroke-linecap="round"/>',
         '<polygon points="-30,-10 -14,-10 -22,-24" fill="#1c1917"/><polygon points="14,-10 30,-10 22,-24" fill="#1c1917"/>',
         '<path d="M-32,12 L-20,22 L-8,12 L4,24 L16,12 L28,22 L36,10 Q2,46 -32,12 Z" fill="#1c1917"/>',
         '</g>']
    return "".join(g)


def bat(x, y, s):
    return (f'<g transform="translate({x:.0f} {y:.0f}) scale({s:.2f})" fill="#1c1917" opacity="0.85">'
            '<path d="M0,0 C-10,-12 -30,-14 -60,-4 C-48,-2 -44,6 -46,14 C-36,6 -24,8 -18,16 C-14,8 -8,6 0,10 '
            'C8,6 14,8 18,16 C24,8 36,6 46,14 C44,6 48,-2 60,-4 C30,-14 10,-12 0,0 Z"/>'
            '<circle cx="0" cy="-2" r="8"/></g>')


def star(x, y, s, colour):
    pts = " ".join(f"{x + math.cos(math.pi / 2 + i * math.pi / 5) * (s if i % 2 == 0 else s * 0.45):.1f},"
                   f"{y - math.sin(math.pi / 2 + i * math.pi / 5) * (s if i % 2 == 0 else s * 0.45):.1f}" for i in range(10))
    return f'<polygon points="{pts}" fill="{colour}"/>'


def snowflake(x, y, s, o):
    arms = []
    for i in range(6):
        a = i * math.pi / 3
        ex, ey = x + math.cos(a) * s, y + math.sin(a) * s
        arms.append(f'M{x:.1f},{y:.1f} L{ex:.1f},{ey:.1f}')
        for f in (0.45, 0.7):
            bx, by = x + math.cos(a) * s * f, y + math.sin(a) * s * f
            for d in (-0.6, 0.6):
                arms.append(f'M{bx:.1f},{by:.1f} L{bx + math.cos(a + d) * s * 0.28:.1f},{by + math.sin(a + d) * s * 0.28:.1f}')
    return f'<path d="{" ".join(arms)}" stroke="#ffffff" stroke-width="{max(2, s / 12):.1f}" stroke-linecap="round" fill="none" opacity="{o:.2f}"/>'


def balloon(x, y, s, colour):
    return (f'<g transform="translate({x:.0f} {y:.0f}) scale({s:.2f})">'
            f'<path d="M0,58 q-14,40 6,80 q14,30 -4,60" stroke="#fdf4ff" stroke-width="3" fill="none" opacity="0.8"/>'
            f'<ellipse cx="0" cy="0" rx="46" ry="58" fill="{colour}"/>'
            f'<polygon points="-8,56 8,56 0,66" fill="{colour}"/>'
            f'<ellipse cx="-16" cy="-22" rx="10" ry="16" fill="#ffffff" opacity="0.45"/></g>')


def write(name, parts):
    os.makedirs(here, exist_ok=True)
    with open(os.path.join(here, name + ".svg"), "w") as f:
        f.write("<!-- Decorations for the " + name + " background (make-season-decor.py) -->\n<g>\n" + "\n".join(parts) + "\n</g>\n")


r = random.Random(7)
# Halloween
parts = [star(x, y, 6 + 10 * k, "#fde68a") for x, y, k in spots(26, 1, 30)]
for i, (x, y, k) in enumerate(spots(16, 2)):
    parts.append(pumpkin(x, y, 0.7 + 0.8 * k) if i % 2 == 0 else bat(x, y, 0.7 + 0.9 * k))
write("halloween", parts)
# Winter
parts = ['<ellipse cx="300" cy="1150" rx="700" ry="200" fill="#f0f9ff" opacity="0.9"/>',
         '<ellipse cx="1600" cy="1170" rx="800" ry="220" fill="#e0f2fe" opacity="0.9"/>',
         '<ellipse cx="960" cy="1210" rx="900" ry="200" fill="#ffffff"/>']
parts += [snowflake(x, y, 14 + 34 * k, 0.45 + 0.5 * k) for x, y, k in spots(40, 3, 30)]
parts += [f'<circle cx="{r.uniform(0, W):.0f}" cy="{r.uniform(0, H):.0f}" r="{r.uniform(2, 5):.1f}" fill="#ffffff" opacity="{r.uniform(0.4, 0.9):.2f}"/>' for _ in range(160)]
write("winter", parts)
# Birthday
colours = ["#f43f5e", "#facc15", "#22c55e", "#3b82f6", "#a855f7", "#f97316", "#06b6d4", "#ffffff"]
parts = []
for _ in range(170):
    x, y = r.uniform(0, W), r.uniform(0, H)
    if abs(x - 960) < 380 and 100 < y < 800:
        continue
    parts.append(f'<rect x="{x:.0f}" y="{y:.0f}" width="{r.uniform(8, 18):.0f}" height="{r.uniform(14, 30):.0f}" '
                 f'fill="{r.choice(colours)}" transform="rotate({r.uniform(0, 180):.0f} {x:.0f} {y:.0f})" opacity="0.9"/>')
for i, (x, y, k) in enumerate(spots(9, 5, 120)):
    parts.append(balloon(x, y - 40, 0.8 + 0.6 * k, colours[i % 7]))
parts.append('<text x="960" y="880" text-anchor="middle" font-family="Noto Sans, DejaVu Sans, sans-serif" '
             'font-size="64" font-weight="700" fill="#fde047">Happy Birthday!</text>')
write("birthday", parts)
print("ok")
