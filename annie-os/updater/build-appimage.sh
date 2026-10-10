#!/bin/bash
# Build Annie-OS-Updater-x86_64.AppImage in annie-os/out/. It carries the
# newest Annie apps and cursor, taken from this repository.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
os="$(dirname "$here")"
out="${OUTDIR:-$os/out}"
mkdir -p "$out"

tool="${APPIMAGETOOL:-$out/appimagetool-x86_64.AppImage}"
if [[ ! -x "$tool" ]]; then
    curl -fsSL -o "$tool" https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
    chmod +x "$tool"
fi

appdir="$out/AnnieUpdater.AppDir"
rm -rf "$appdir"
files="$appdir/usr/bin/payload/files"
mkdir -p "$files"
# The Annie parts of the system: apps, helpers, cursor, icons, wallpaper,
# menu entries and login setup. Live-USB-only setup stays out.
r="$os/airootfs"
for d in usr/local/bin usr/lib/annie usr/share/icons/AnnieRainbow usr/share/icons/hicolor/scalable/apps \
         usr/share/backgrounds/annie usr/share/applications etc/xdg/autostart etc/skel/.config; do
    mkdir -p "$files/$d"
    cp -a "$r/$d/." "$files/$d/"
done
# Live-USB-only: the live user setup, and the installer.
rm -f "$files/usr/lib/annie/annie-live-setup" "$files/usr/local/bin/annie-install" "$files/usr/share/applications/annie-install.desktop"
install -m755 "$os/appimage/AnnieDiskWriter/annie-disk-writer" "$files/usr/local/bin/annie-disk-writer"
install -m755 "$os/appimage/AnnieDiskWriter/annie-disk-helper" "$files/usr/lib/annie/annie-disk-helper"
install -m644 "$os/appimage/AnnieDiskWriter/annie-disk-writer.desktop" "$files/usr/share/applications/"
install -m644 "$os/appimage/AnnieDiskWriter/annie-disk-writer.svg" "$files/usr/share/icons/hicolor/scalable/apps/"
install -Dm644 "$os/packages.txt" "$appdir/usr/bin/payload/packages.txt"
install -m755 "$here/root-update.sh" "$appdir/usr/bin/payload/root-update.sh"
install -m755 "$here/annie-update" "$appdir/usr/bin/annie-update"

cat >"$appdir/AppRun" <<'APPRUN'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/usr/bin/annie-update" "$@"
APPRUN
chmod 755 "$appdir/AppRun"
install -m644 "$here/annie-update.desktop" "$appdir/annie-update.desktop"
install -m644 "$os/airootfs/usr/share/icons/hicolor/scalable/apps/annie-os.svg" "$appdir/annie-os.svg"
ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Annie-OS-Updater-x86_64.AppImage"
rm -rf "$appdir"
