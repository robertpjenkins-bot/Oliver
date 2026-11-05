#!/usr/bin/env python3
"""Ollie OS's app icons (os/profile/airootfs/usr/share/icons/hicolor/*/apps/
ollie-*.png), drawn once with icons/AppIcon.qml (needs PySide6 and the Noto
Color Emoji font; the emoji pictures are Google's Noto Emoji, Apache 2.0):

  QT_QPA_PLATFORM=offscreen QT_QUICK_BACKEND=software python3 os/art/make-app-icons.py

Each is drawn at 256 px, then made smaller (128, 64, 48) with ImageMagick.
"""
import os
import subprocess
import sys

from PySide6.QtCore import QUrl
from PySide6.QtGui import QGuiApplication
from PySide6.QtQml import QQmlComponent, QQmlEngine
from PySide6.QtQuick import QQuickWindow
import shiboken6

# icon name: (emoji, top colour, bottom colour)
ICONS = {
    "ollie-games": ("🕹️", "#a855f7", "#5b21b6"),
    "ollie-pet": ("pet", "#7dd3fc", "#0284c7"),
    "ollie-finder": ("🔍", "#60a5fa", "#1d4ed8"),
    "ollie-typing": ("⌨️", "#2dd4bf", "#0f766e"),
    "ollie-coop": ("🎮", "#fb923c", "#c2410c"),
    "ollie-sticker": ("🌟", "#f472b6", "#be185d"),
    "ollie-comic": ("💬", "#fde047", "#ca8a04"),
    "ollie-animator": ("🎞️", "#f87171", "#b91c1c"),
    "ollie-soundboard": ("🔊", "#818cf8", "#3730a3"),
    "ollie-codequest": ("🤖", "#4ade80", "#15803d"),
    "ollie-space": ("🪐", "#334155", "#020617"),
    "ollie-spelling": ("🐝", "#fcd34d", "#d97706"),
    "ollie-world": ("🌍", "#38bdf8", "#0369a1"),
    "ollie-recovery": ("🛟", "#94a3b8", "#334155"),
    "ollie-settings": ("⚙️", "#5eead4", "#0f766e"),
    "ollie-os-update": ("⬆️", "#34d399", "#047857"),
    "ollie-update-checker": ("🔔", "#fbbf24", "#b45309"),
    "ollie-scanner-notes": ("📝", "#cbd5e1", "#475569"),
    "ollie-wallpaper-maker": ("🖼️", "#c084fc", "#7e22ce"),
    "ollie-fixer": ("🔧", "#fdba74", "#c2410c"),
    "ollie-storage": ("💾", "#93c5fd", "#1e40af"),
    "ollie-stuff-saver": ("📦", "#fcd34d", "#a16207"),
    "ollie-download-tracker": ("📥", "#67e8f9", "#0e7490"),
    "ollie-write": ("💿", "#a5b4fc", "#4338ca"),
    "ollie-restart-windows": ("🪟", "#7dd3fc", "#0369a1"),
    "ollie-appcenter": ("🛍️", "#f472b6", "#9d174d"),
    "ollie-mathquest": ("🧮", "#67e8f9", "#0e7490"),
    "ollie-trophies": ("🏆", "#fde047", "#b45309"),
}

here = os.path.dirname(os.path.abspath(__file__))
dest = os.path.join(here, "..", "profile", "airootfs", "usr", "share", "icons", "hicolor")
app = QGuiApplication(sys.argv[:1])
engine = QQmlEngine()
comp = QQmlComponent(engine, QUrl.fromLocalFile(os.path.join(here, "icons", "AppIcon.qml")))
for name, (emoji, top, bottom) in ICONS.items():
    win = comp.createWithInitialProperties({"emoji": emoji, "top": top, "bottom": bottom, "pet": emoji == "pet"})
    if win is None:
        sys.exit("AppIcon.qml: " + comp.errorString())
    for _ in range(20):
        app.processEvents()
    big = os.path.join(dest, "256x256", "apps", name + ".png")
    os.makedirs(os.path.dirname(big), exist_ok=True)
    shiboken6.wrapInstance(shiboken6.getCppPointer(win)[0], QQuickWindow).grabWindow().save(big)
    win.close()
    for size in (128, 64, 48):
        out = os.path.join(dest, f"{size}x{size}", "apps", name + ".png")
        os.makedirs(os.path.dirname(out), exist_ok=True)
        subprocess.run(["convert", big, "-resize", f"{size}x{size}", "-strip", out], check=True)
print(len(ICONS), "icons")
