#!/bin/bash
# Build Ollie OS USB Maker as an AppImage: os/out/Ollie-OS-USB-Maker-x86_64.AppImage
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
os="$(dirname "$here")"
out="${OUTDIR:-$os/out}"
appdir="$out/OllieUSB.AppDir"

rm -rf "$appdir"
install -D -m 0755 "$here/AppRun" "$appdir/AppRun"
install -D -m 0755 "$here/ollie-usb" "$appdir/usr/bin/ollie-usb"
install -D -m 0644 "$here/ollie-usb.desktop" "$appdir/ollie-usb.desktop"
install -D -m 0644 "$os/art/ollie-os.svg" "$appdir/ollie-os.svg"

tool="${APPIMAGETOOL:-$out/appimagetool-x86_64.AppImage}"
if [[ ! -x "$tool" ]]; then
    curl -fsSL -o "$tool" https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
    chmod +x "$tool"
fi
# --appimage-extract-and-run: works without FUSE, as in containers and CI.
ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Ollie-OS-USB-Maker-x86_64.AppImage"
rm -rf "$appdir"
