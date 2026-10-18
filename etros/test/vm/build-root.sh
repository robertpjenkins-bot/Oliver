#!/bin/bash
# Stand-in for Raspberry Pi OS: Ubuntu arm64 with LightDM, PCManFM's desktop
# and Openbox (Pi OS uses LightDM + PCManFM too).
set -euo pipefail
R=/var/tmp/ovm/root
rm -rf "$R"
unset http_proxy
debootstrap --arch=arm64 --components=main,universe --include=systemd-sysv,udev,python3,sudo,dbus,kmod,iproute2,procps \
    noble "$R" http://ports.ubuntu.com/ubuntu-ports
cat > "$R/etc/apt/sources.list" <<S
deb http://ports.ubuntu.com/ubuntu-ports noble main universe
S
mount -t proc proc "$R/proc"; mount --bind /dev "$R/dev"; mount --bind /sys "$R/sys"
trap 'umount -l "$R/proc" "$R/dev" "$R/sys" 2>/dev/null || true' EXIT
cp /etc/resolv.conf "$R/etc/resolv.conf" 2>/dev/null || true
chroot "$R" env DEBIAN_FRONTEND=noninteractive apt-get update
chroot "$R" env DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    linux-image-generic lightdm lightdm-gtk-greeter xserver-xorg-core xserver-xorg-input-libinput \
    openbox pcmanfm lxsession-logout lxterminal lxsession xinit x11-xserver-utils fonts-dejavu-core \
    adwaita-icon-theme-full librsvg2-common
echo done-root
