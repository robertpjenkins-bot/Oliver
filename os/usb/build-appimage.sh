#!/bin/bash
# Build Ollie OS USB Maker as AppImages in os/out/:
#   Ollie-OS-USB-Maker-x86_64.AppImage      takes the ISO, its parts, or a zip
#   Ollie-OS-USB-Maker-Zip-x86_64.AppImage  the Zip edition: takes only a zip
#   Ollie-OS-Drive-Doctor-x86_64.AppImage   checks an Ollie OS drive and fixes it
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
    local appdir="$out/OllieUSB$1.AppDir"
    rm -rf "$appdir"
    install -d "$appdir"
    cat >"$appdir/AppRun" <<APPRUN
#!/bin/sh
# AppImage entry point: start the USB maker, passing along a dropped file.
HERE="\$(dirname "\$(readlink -f "\$0")")"
$3
exec "\$HERE/usr/bin/ollie-usb" "\$@"
APPRUN
    chmod 0755 "$appdir/AppRun"
    install -D -m 0755 "$here/ollie-usb" "$appdir/usr/bin/ollie-usb"
    sed "s/^Name=.*/Name=$2/" "$here/ollie-usb.desktop" >"$appdir/ollie-usb.desktop"
    install -D -m 0644 "$os/art/ollie-os.svg" "$appdir/ollie-os.svg"
    # --appimage-extract-and-run: works without FUSE, as in containers and CI.
    ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Ollie-OS-USB-Maker$1-x86_64.AppImage"
    rm -rf "$appdir"
}

build "" "Ollie OS USB Maker" ""
build "-Zip" "Ollie OS USB Maker (Zip)" "export OLLIE_USB_ZIP_ONLY=1"

# Drive Doctor carries SYSLINUX (and the mtools it uses) to add the start-up
# code for older BIOS computers. They come from Ubuntu's packages, which only
# need the C library.
build_doctor() {
    local appdir="$out/OllieDoctor.AppDir" debs="$out/doctor-debs" sl
    rm -rf "$appdir" "$debs"
    mkdir -p "$debs/x"
    (cd "$debs" && apt-get download syslinux syslinux-common mtools >/dev/null)
    for d in "$debs"/*.deb; do dpkg-deb -x "$d" "$debs/x"; done
    sl="$appdir/usr/lib/ollie-doctor/syslinux"
    install -d "$sl/modules"
    install -m 0755 "$debs/x/usr/bin/syslinux" "$debs/x/usr/bin/mtools" "$sl/"
    for l in mattrib mcopy mdel mdir mmove mformat minfo mlabel mmd mtype; do ln -s mtools "$sl/$l"; done
    install -m 0644 "$debs/x/usr/lib/syslinux/mbr/gptmbr.bin" "$debs/x/usr/lib/syslinux/mbr/mbr.bin" "$sl/"
    install -m 0644 "$debs/x/usr/lib/syslinux/modules/bios/"*.c32 "$sl/modules/"
    install -D -m 0755 "$here/ollie-doctor" "$appdir/usr/bin/ollie-doctor"
    cat >"$appdir/AppRun" <<'APPRUN'
#!/bin/sh
# AppImage entry point: start Drive Doctor.
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/usr/bin/ollie-doctor" "$@"
APPRUN
    chmod 0755 "$appdir/AppRun"
    install -m 0644 "$here/ollie-doctor.desktop" "$appdir/ollie-doctor.desktop"
    install -D -m 0644 "$os/art/ollie-os.svg" "$appdir/ollie-os.svg"
    ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Ollie-OS-Drive-Doctor-x86_64.AppImage"
    rm -rf "$appdir" "$debs"
}
build_doctor
