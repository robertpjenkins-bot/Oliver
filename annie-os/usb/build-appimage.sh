#!/bin/bash
# Build Annie OS USB Maker as AppImages in annie-os/out/:
#   Annie-OS-USB-Maker-x86_64.AppImage      takes the ISO, its two parts, or a zip
#   Annie-OS-USB-Maker-Zip-x86_64.AppImage  the Zip edition: takes only a zip
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

build() { # build NAME-SUFFIX DISPLAY-NAME ENV-LINE
    local appdir="$out/AnnieUSB$1.AppDir"
    rm -rf "$appdir"
    install -d "$appdir"
    cat >"$appdir/AppRun" <<APPRUN
#!/bin/sh
# AppImage entry point: start the USB maker, passing along a dropped file.
HERE="\$(dirname "\$(readlink -f "\$0")")"
$3
exec "\$HERE/usr/bin/annie-usb" "\$@"
APPRUN
    chmod 0755 "$appdir/AppRun"
    install -D -m 0755 "$here/annie-usb" "$appdir/usr/bin/annie-usb"
    sed "s/^Name=.*/Name=$2/" "$here/annie-usb.desktop" >"$appdir/annie-usb.desktop"
    install -D -m 0644 "$os/airootfs/usr/share/icons/hicolor/scalable/apps/annie-os.svg" "$appdir/annie-os.svg"
    # --appimage-extract-and-run: works without FUSE, as in containers and CI.
    ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Annie-OS-USB-Maker$1-x86_64.AppImage"
    rm -rf "$appdir"
}

build "" "Annie OS USB Maker" ""
build "-Zip" "Annie OS USB Maker (Zip)" "export ANNIE_USB_ZIP_ONLY=1"
