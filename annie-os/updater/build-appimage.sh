#!/bin/bash
# Build Annie-OS-Updater-x86_64.AppImage in annie-os/out/. It carries the
# newest Annie apps, settings and cursor, taken from annie-os/profile.
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
pl="$appdir/usr/bin/payload"
files="$pl/files"
mkdir -p "$files"
r="$os/profile/airootfs"
# What the updater puts in place: Annie's programs, menu entries, icons,
# cursor, wallpaper, desktop layout and default settings. Start-up services,
# the live USB's account setup and the login settings stay as they are.
for d in usr/local/bin usr/share/applications usr/share/icons usr/share/plasma usr/share/kio \
         usr/share/wallpapers etc/xdg etc/environment.d; do
    [[ -d "$r/$d" ]] || continue
    mkdir -p "$files/$d"
    cp -a "$r/$d/." "$files/$d/"
done
(cd "$files/usr/local/bin" && rm -f annie-live-setup annie-selftest-* annie-install annie-first-setup annie-set-account)
rm -f "$files/usr/share/applications/annie-install.desktop" "$files/etc/xdg/autostart/annie-first-setup.desktop"
rmdir "$files/etc/xdg/autostart" 2>/dev/null || true
for svg in "$os"/art/*.svg; do
    install -D -m644 "$svg" "$files/usr/share/icons/hicolor/scalable/apps/$(basename "$svg")"
done
# The wallpaper image itself is drawn when the ISO is built; carry one size.
if python3 -c "import PIL" 2>/dev/null; then
    mkdir -p "$files/usr/share/wallpapers/AnnieOS/contents/images"
    python3 "$os/tools/make-wallpaper.py" "$files/usr/share/wallpapers/AnnieOS/contents/images/1920x1080.png" 1920 1080
fi
# The parts that also work on the first Annie OS (Xfce).
cat >"$pl/xfce-files.txt" <<'LIST'
usr/local/bin/annie-paint
usr/local/bin/annie-disk-writer
usr/local/bin/annie-disk-helper
usr/local/bin/annie-run-appimage
usr/share/applications/annie-paint.desktop
usr/share/applications/annie-disk-writer.desktop
usr/share/applications/annie-run-appimage.desktop
usr/share/icons/AnnieRainbow
usr/share/icons/hicolor/scalable/apps/annie-os.svg
LIST
cat >"$pl/packages-essential.txt" <<'LIST'
fuse2
fuse3
python
python-pillow
tk
LIST
{
    sed -e 's/#.*//' -e 's/[[:space:]]//g' -e '/^$/d' "$os/profile/packages.x86_64"
    sed -e 's/#.*//' -e 's/[[:space:]]//g' -e '/^$/d' "$os/profile/packages-optional.x86_64"
} >"$pl/packages.txt"
install -m755 "$here/root-update.sh" "$pl/root-update.sh"
install -m755 "$here/annie-update" "$appdir/usr/bin/annie-update"

cat >"$appdir/AppRun" <<'APPRUN'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/usr/bin/annie-update" "$@"
APPRUN
chmod 755 "$appdir/AppRun"
install -m644 "$here/annie-update.desktop" "$appdir/annie-update.desktop"
install -m644 "$os/art/annie-os.svg" "$appdir/annie-os.svg"
ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Annie-OS-Updater-x86_64.AppImage"
rm -rf "$appdir"
