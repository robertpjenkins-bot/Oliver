#!/usr/bin/env python3
"""Ollie OS's start-up animation and sound, drawn here once and kept in the repo.

   python3 -I os/art/make-startup-art.py

Writes into os/profile/airootfs:
  usr/share/plymouth/themes/ollie/cube-0.png ... cube-23.png
      the Ollie OS cube turning a quarter turn (a cube looks the same after
      one, so the 24 pictures loop for ever without a jump)
  usr/share/plymouth/themes/ollie/glow.png   the soft light behind it
  usr/share/plymouth/themes/ollie/name.png   "Ollie OS" (Noto Sans Bold, as on the wallpaper)
  usr/share/sounds/ollie/startup.ogg          the start-up sound: a little rising chime

Needs Pillow, the Noto Sans font, and ffmpeg (for the .ogg).
"""
import math
import os
import struct
import subprocess
import tempfile
import wave

from PIL import Image, ImageDraw, ImageFilter, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.join(HERE, "..", "profile", "airootfs")
THEME = os.path.join(ROOT, "usr/share/plymouth/themes/ollie")
SOUNDS = os.path.join(ROOT, "usr/share/sounds/ollie")

FRAMES = 24
SIZE = 256  # each cube picture, in pixels
SS = 4  # drawn this many times bigger, then made smaller: smooth edges

WHITE = (255, 255, 255)
LIGHT = (0x9E, 0xEE, 0xE4)  # the cube's left side on the logo
TEAL = (0x2D, 0xD4, 0xBF)  # its right side
EDGE = (0x0B, 0x4F, 0x4A)


def rotate_y(p, a):
    x, y, z = p
    return (x * math.cos(a) + z * math.sin(a), y, -x * math.sin(a) + z * math.cos(a))


def rotate_x(p, a):
    x, y, z = p
    return (x, y * math.cos(a) - z * math.sin(a), y * math.sin(a) + z * math.cos(a))


# The cube's six sides: four corners each, and which way it faces.
SIDES = [
    ([(-1, 1, -1), (1, 1, -1), (1, 1, 1), (-1, 1, 1)], (0, 1, 0)),  # top
    ([(-1, -1, -1), (-1, -1, 1), (1, -1, 1), (1, -1, -1)], (0, -1, 0)),  # bottom
    ([(-1, -1, 1), (-1, 1, 1), (1, 1, 1), (1, -1, 1)], (0, 0, 1)),
    ([(1, -1, -1), (1, 1, -1), (-1, 1, -1), (-1, -1, -1)], (0, 0, -1)),
    ([(1, -1, 1), (1, 1, 1), (1, 1, -1), (1, -1, -1)], (1, 0, 0)),
    ([(-1, -1, -1), (-1, 1, -1), (-1, 1, 1), (-1, -1, 1)], (-1, 0, 0)),
]
TILT = math.asin(math.tan(math.radians(30)))  # the logo's view: a true isometric cube


def view(p, turn):
    return rotate_x(rotate_y(p, turn), TILT)


def mix(a, b, t):
    return tuple(round(a[i] + (b[i] - a[i]) * t) for i in range(3))


def cube_frame(turn, scale):
    big = SIZE * SS
    img = Image.new("RGBA", (big, big), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    sides = []
    for corners, normal in SIDES:
        n = view(normal, turn)
        if n[2] <= 1e-6:  # facing away
            continue
        pts = [view(c, turn) for c in corners]
        depth = sum(p[2] for p in pts) / 4
        if normal[1] > 0:
            colour = WHITE
        else:  # lit like the logo: sides facing left lighter, right ones teal
            t = min(1.0, max(0.0, (n[0] / math.sin(math.radians(45)) + 1) / 2))
            colour = mix(LIGHT, TEAL, t)
        xy = [(big / 2 + p[0] * scale * SS, big / 2 - p[1] * scale * SS) for p in pts]
        sides.append((depth, xy, colour))
    width = round(scale * 0.1 * SS)
    for _, xy, colour in sorted(sides):
        draw.polygon(xy, fill=colour)
        draw.line(xy + [xy[0]], fill=EDGE, width=width, joint="curve")
        for x, y in xy:  # round corners
            r = width / 2
            draw.ellipse((x - r, y - r, x + r, y + r), fill=EDGE)
    return img.resize((SIZE, SIZE), Image.LANCZOS)


def glow():
    size = 640
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    px = img.load()
    for y in range(size):
        for x in range(size):
            d = math.hypot(x - size / 2, y - size / 2) / (size / 2)
            a = max(0.0, 1 - d) ** 2
            px[x, y] = (0x5E, 0xEA, 0xD4, round(110 * a))
    return img


def name():
    font = ImageFont.truetype("/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf", 72 * SS)
    box = font.getbbox("Ollie OS")
    w, h = box[2] - box[0], box[3] - box[1]
    pad = 8 * SS
    img = Image.new("RGBA", (w + 2 * pad, h + 2 * pad), (0, 0, 0, 0))
    ImageDraw.Draw(img).text((pad - box[0], pad - box[1]), "Ollie OS", font=font, fill=WHITE + (255,))
    return img.resize((img.width // SS, img.height // SS), Image.LANCZOS)


def chime(path):
    """A rising chime, C - E - G - high C, each note a soft bell."""
    rate = 44100
    notes = [(0.00, 523.25), (0.16, 659.25), (0.32, 783.99), (0.48, 1046.50)]
    length = 2.6
    samples = [0.0] * int(rate * length)
    for start, freq in notes:
        first = int(start * rate)
        for i in range(first, len(samples)):
            t = (i - first) / rate
            env = min(1.0, t / 0.008) * math.exp(-t * 2.6)
            bell = (math.sin(2 * math.pi * freq * t)
                    + 0.35 * math.sin(2 * math.pi * freq * 2 * t) * math.exp(-t * 3)
                    + 0.12 * math.sin(2 * math.pi * freq * 3.01 * t) * math.exp(-t * 6))
            samples[i] += 0.22 * env * bell
    fade = int(0.3 * rate)
    for i in range(fade):
        samples[-1 - i] *= i / fade
    peak = max(abs(s) for s in samples)
    with tempfile.TemporaryDirectory() as tmp:
        wav = os.path.join(tmp, "startup.wav")
        with wave.open(wav, "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(2)
            w.setframerate(rate)
            w.writeframes(b"".join(struct.pack("<h", round(s / peak * 0.8 * 32767)) for s in samples))
        subprocess.run(["ffmpeg", "-loglevel", "error", "-y", "-i", wav, "-c:a", "libvorbis", "-q:a", "4", path], check=True)


def main():
    os.makedirs(THEME, exist_ok=True)
    os.makedirs(SOUNDS, exist_ok=True)
    # Big enough that the cube at its widest (the logo's pose) just fits.
    widest = max(abs(view(c, math.radians(45))[i]) for corners, _ in SIDES for c in corners for i in (0, 1))
    scale = SIZE * 0.44 / widest
    for f in range(FRAMES):
        turn = math.radians(45 + 90 * f / FRAMES)
        cube_frame(turn, scale).save(os.path.join(THEME, f"cube-{f}.png"), optimize=True)
    glow().save(os.path.join(THEME, "glow.png"), optimize=True)
    name().save(os.path.join(THEME, "name.png"), optimize=True)
    chime(os.path.join(SOUNDS, "startup.ogg"))


if __name__ == "__main__":
    main()
