#!/bin/bash
# Build Ollie OS: a small Debian arm64 desktop, packed as one squashfs.
#
#   build-ollie.sh ART_DIR OUT.squashfs
#
# ART_DIR is the output of make-art.py (for the Bobby wallpaper). Needs root
# and mmdebstrap; on an x86 machine also qemu-user-static with binfmt set up.
#
# Ollie OS has no kernel of its own. It boots on the Batocera kernel of
# whichever device it is on, and finds that kernel's modules and firmware in
# the Batocera image, which the Etros initramfs mounts at /.batocera.
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
ETROS=$(dirname "$HERE")
ART=$(realpath "$1")
OUT=$(realpath -m "$2")
SUITE=${OLLIE_SUITE:-trixie}

PACKAGES=(
    # base system, sysvinit rather than systemd: the Batocera kernels are
    # not built with everything systemd insists on
    sysvinit-core sysv-rc initscripts orphan-sysvinit-scripts
    udev dbus kmod procps psmisc sudo ca-certificates locales
    e2fsprogs dosfstools util-linux
    # network
    network-manager network-manager-gnome wpasupplicant iw polkitd
    # desktop
    xserver-xorg-core xserver-xorg-legacy xserver-xorg-input-libinput
    xinit x11-xserver-utils x11-utils
    openbox tint2 pcmanfm lxterminal adwaita-icon-theme
    fonts-dejavu-core
    # the reason it exists
    chromium python3 python3-evdev unzip p7zip-full
)
# Not every Debian release has every on-screen keyboard; take the first one
# that installs.
KEYBOARDS=(onboard matchbox-keyboard)

ROOT=$(mktemp -d)
trap 'rm -rf "$ROOT"' EXIT

echo "=== bootstrapping Debian ${SUITE} arm64"
# shellcheck disable=SC2016  # $1 is for mmdebstrap's hook, not this shell
mmdebstrap --arch=arm64 --variant=apt --mode=root \
    --aptopt='APT::Install-Recommends "false"' \
    --setup-hook='mkdir -p "$1/etc/apt/preferences.d" && printf "Package: systemd-sysv\nPin: release *\nPin-Priority: -1\n" > "$1/etc/apt/preferences.d/no-systemd"' \
    --include="$(IFS=,; echo "${PACKAGES[*]}")" \
    "$SUITE" "$ROOT" "http://deb.debian.org/debian"

echo "=== configuring"
for kb in "${KEYBOARDS[@]}"; do
    if chroot "$ROOT" apt-get install -y --no-install-recommends "$kb"; then break; fi
done

chroot "$ROOT" useradd -m -s /bin/bash -G video,input,audio,netdev,plugdev ollie || \
    chroot "$ROOT" useradd -m -s /bin/bash -G video,input,audio ollie
chroot "$ROOT" passwd -d ollie
echo "root:ollie" | chroot "$ROOT" chpasswd

cp -a "$HERE/rootfs/." "$ROOT/"
install -m 0755 "$ETROS/common/etros-sort-roms" "$ROOT/usr/local/bin/etros-sort-roms"
install -m 0644 "$ART/ollie/bobby.png" "$ROOT/usr/share/ollie/bobby.png"
chmod 0440 "$ROOT/etc/sudoers.d/ollie"

# Openbox reads one rc.xml, so start from Debian's and add to it: the
# keyboard toggle the gamepad's X button sends, and every window maximised
# since the R36S screen is 640x480.
mkdir -p "$ROOT/home/ollie/.config/openbox"
sed -e 's|</keyboard>|  <keybind key="W-k"><action name="Execute"><command>/usr/local/bin/ollie-keyboard-toggle</command></action></keybind>\n</keyboard>|' \
    -e 's|</applications>|  <application class="*" type="normal"><maximized>yes</maximized></application>\n</applications>|' \
    "$ROOT/etc/xdg/openbox/rc.xml" > "$ROOT/home/ollie/.config/openbox/rc.xml"
grep -q ollie-keyboard-toggle "$ROOT/home/ollie/.config/openbox/rc.xml"
chroot "$ROOT" chown -R ollie:ollie /home/ollie

# Drivers come from the Batocera image of the device Ollie OS is running on.
rm -rf "$ROOT/usr/lib/modules" "$ROOT/usr/lib/firmware"
ln -s /.batocera/lib/modules "$ROOT/usr/lib/modules"
ln -s /.batocera/lib/firmware "$ROOT/usr/lib/firmware"
mkdir -p "$ROOT/.batocera" "$ROOT/boot" "$ROOT/userdata"

echo "en_US.UTF-8 UTF-8" > "$ROOT/etc/locale.gen"
chroot "$ROOT" locale-gen
chroot "$ROOT" apt-get clean
rm -rf "$ROOT"/var/lib/apt/lists/* "$ROOT"/var/cache/apt/*.bin "$ROOT"/usr/share/doc/* "$ROOT"/usr/share/man/*

echo "=== packing"
rm -f "$OUT"
mksquashfs "$ROOT" "$OUT" -comp zstd -Xcompression-level 19 -noappend -quiet -no-progress
ls -lh "$OUT"
