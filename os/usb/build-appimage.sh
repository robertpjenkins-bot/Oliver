#!/bin/bash
# Build Ollie OS USB Maker as AppImages in os/out/:
#   Ollie-OS-USB-Maker-x86_64.AppImage      takes the ISO, its parts, or a zip
#   Ollie-OS-USB-Maker-Zip-x86_64.AppImage  the Zip edition: takes only a zip
#   Ollie-OS-Drive-Doctor-x86_64.AppImage   checks an Ollie OS drive and fixes it
#   Ollie-OS-Get-ISO-x86_64.AppImage        downloads the ISO next to USB Maker
#   Ollie-OS-Installer-x86_64.AppImage      downloads the ISO, then USB Maker
#   Ollie-OS-Updater-x86_64.AppImage        brings an ISO up to the newest Ollie OS
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

build_get_iso() {
    local appdir="$out/OllieGetISO.AppDir"
    rm -rf "$appdir"
    install -D -m 0755 "$here/ollie-get-iso" "$appdir/usr/bin/ollie-get-iso"
    cat >"$appdir/AppRun" <<'APPRUN'
#!/bin/sh
# AppImage entry point: start Get ISO.
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/usr/bin/ollie-get-iso" "$@"
APPRUN
    chmod 0755 "$appdir/AppRun"
    install -m 0644 "$here/ollie-get-iso.desktop" "$appdir/ollie-get-iso.desktop"
    install -D -m 0644 "$os/art/ollie-os.svg" "$appdir/ollie-os.svg"
    ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Ollie-OS-Get-ISO-x86_64.AppImage"
    rm -rf "$appdir"
}
build_get_iso

# Get ISO and USB Maker in one: download, then straight on to choosing the drive.
build_installer() {
    local appdir="$out/OllieInstaller.AppDir"
    rm -rf "$appdir"
    install -D -m 0755 "$here/ollie-get-iso" "$appdir/usr/bin/ollie-get-iso"
    install -D -m 0755 "$here/ollie-usb" "$appdir/usr/bin/ollie-usb"
    cat >"$appdir/AppRun" <<'APPRUN'
#!/bin/sh
# AppImage entry point: download Ollie OS, then hand over to USB Maker.
HERE="$(dirname "$(readlink -f "$0")")"
export OLLIE_USB_MAKER="$HERE/usr/bin/ollie-usb"
exec "$HERE/usr/bin/ollie-get-iso" "$@"
APPRUN
    chmod 0755 "$appdir/AppRun"
    install -m 0644 "$here/ollie-installer.desktop" "$appdir/ollie-installer.desktop"
    install -D -m 0644 "$os/art/ollie-os.svg" "$appdir/ollie-os.svg"
    ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Ollie-OS-Installer-x86_64.AppImage"
    rm -rf "$appdir"
}
build_installer

# The Updater carries the Ollie OS files changed since the ISO on the release
# was built (UPDATE_BASE), and xorriso and squashfs-tools with the libraries
# they need beyond the C library, taken from this (build) computer.
build_updater() {
    local appdir="$out/OllieUpdater.AppDir" kit rel b lib base="${UPDATE_BASE:-dc16a2a}"
    rm -rf "$appdir"
    kit="$appdir/usr/lib/ollie-update"
    install -d "$kit/bin" "$kit/libexec" "$kit/lib"
    for b in xorriso unsquashfs mksquashfs; do
        install -m 0755 "$(command -v "$b")" "$kit/libexec/$b"
        ldd "$(command -v "$b")" | awk '$3 ~ /^\// {print $3}' | while read -r lib; do
            case "$(basename "$lib")" in libc.so* | libm.so* | ld-linux* | libpthread* | libdl.so* | librt.so*) continue ;; esac
            cp -nL "$lib" "$kit/lib/"
        done
        printf '#!/bin/sh\nd="$(dirname "$(readlink -f "$0")")"\nLD_LIBRARY_PATH="$d/../lib${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}" exec "$d/../libexec/%s" "$@"\n' "$b" >"$kit/bin/$b"
        chmod 0755 "$kit/bin/$b"
    done
    (cd "$os/.." && git diff --name-only "$base" HEAD -- os/profile/airootfs) | while read -r rel; do
        rel="${rel#os/profile/airootfs/}"
        [[ -e "$os/profile/airootfs/$rel" || -L "$os/profile/airootfs/$rel" ]] || continue
        mkdir -p "$(dirname "$appdir/usr/share/ollie-update/airootfs/$rel")"
        cp -a "$os/profile/airootfs/$rel" "$appdir/usr/share/ollie-update/airootfs/$rel"
    done
    install -D -m 0755 "$here/ollie-update" "$appdir/usr/bin/ollie-update"
    install -D -m 0755 "$here/ollie-usb" "$appdir/usr/bin/ollie-usb"
    cat >"$appdir/AppRun" <<'APPRUN'
#!/bin/sh
# AppImage entry point: update the ISO, then hand over to USB Maker.
HERE="$(dirname "$(readlink -f "$0")")"
export OLLIE_USB_MAKER="$HERE/usr/bin/ollie-usb"
exec "$HERE/usr/bin/ollie-update" "$@"
APPRUN
    chmod 0755 "$appdir/AppRun"
    install -m 0644 "$here/ollie-update.desktop" "$appdir/ollie-update.desktop"
    install -D -m 0644 "$os/art/ollie-os.svg" "$appdir/ollie-os.svg"
    ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Ollie-OS-Updater-x86_64.AppImage"
    rm -rf "$appdir"
}
build_updater
