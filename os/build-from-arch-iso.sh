#!/bin/bash
# Ollie OS from the Arch Linux ISO: downloads the official Arch Linux ISO
# from the Arch Linux servers (checked against their checksums), unpacks the
# system inside it, and in that system turns it into Ollie OS (os/build.sh:
# Arch's own ISO recipe with Ollie OS's packages, settings and apps). No
# GitHub, no sign-in. The new ISO lands in os/out/, as with os/build.sh.
#
# Runs as root in any Arch Linux with internet: the Ollie OS Builder's
# Arch Linux in WSL (Windows), or an Arch Linux container (Mac: Docker).
#
#   build-from-arch-iso.sh [WORKDIR]
#
# WORKDIR (default /var/tmp/ollie-from-iso) keeps the ISO and the package
# downloads, so running it again after a failure is quicker. With
# OLLIE_ROOTLESS=1 (a container without real root) it is passed on to
# os/build.sh.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
work="${1:-/var/tmp/ollie-from-iso}"
iso=archlinux-x86_64.iso
root="$work/arch-iso-system"
step() { printf '\n\e[1;36m==> %s\e[0m\n' "$*"; }
mkdir -p "$work" "$work/pkg-cache" "$work/tmp"
chmod 1777 "$work/tmp"

step "Getting the tools to open the Arch Linux ISO"
pacman-key --init >/dev/null 2>&1 || true
pacman-key --populate archlinux >/dev/null 2>&1 || true
sed -i '/^## Worldwide/,/^$/ s/^#Server/Server/' /etc/pacman.d/mirrorlist
# shellcheck disable=SC2016 # pacman's own $repo and $arch
grep -q '^Server' /etc/pacman.d/mirrorlist || echo 'Server = https://geo.mirror.pkgbuild.com/$repo/os/$arch' >>/etc/pacman.d/mirrorlist
pacman -Syu --needed --noconfirm curl libarchive squashfs-tools

step "Downloading the Arch Linux ISO from the Arch Linux servers (about 1.4 GB)"
got=""
for base in ${OLLIE_BUILD_ARCH_MIRRORS:-https://geo.mirror.pkgbuild.com https://fastly.mirror.pkgbuild.com https://mirror.rackspace.com/archlinux https://mirrors.kernel.org/archlinux}; do
    sum="$(curl -fsSL --retry 3 "$base/iso/latest/sha256sums.txt" 2>/dev/null | awk -v f="$iso" '$2 == f {print $1}')" || continue
    [[ -n $sum ]] || continue
    if [[ -f "$work/$iso" && "$(sha256sum "$work/$iso" | cut -d ' ' -f 1)" == "$sum" ]]; then
        got=1
        break
    fi
    if curl -fL --retry 5 -C - -o "$work/$iso.part" "$base/iso/latest/$iso" &&
        [[ "$(sha256sum "$work/$iso.part" | cut -d ' ' -f 1)" == "$sum" ]]; then
        mv -f "$work/$iso.part" "$work/$iso"
        got=1
        break
    fi
    rm -f "$work/$iso.part"
done
[[ -n $got ]] || { echo "Could not download the Arch Linux ISO. Is the internet connected?" >&2; exit 1; }

step "Opening the Arch Linux ISO"
rm -rf --one-file-system "${root:?}" "$work/arch"
bsdtar -xf "$work/$iso" -C "$work" arch/x86_64/airootfs.sfs
unsquashfs -no-progress -no-xattrs -d "$root" "$work/arch/x86_64/airootfs.sfs" >/dev/null
rm -rf "${work:?}/arch"

step "Turning it into Ollie OS"
# The ISO's system, as a running one: the same as the Builder on Linux.
mkdir -p "$root/src" "$root/var/cache/pacman/pkg"
rm -f "$root/etc/resolv.conf"
cp -L /etc/resolv.conf "$root/etc/resolv.conf" 2>/dev/null || true
sed -i '/^## Worldwide/,/^$/ s/^#Server/Server/' "$root/etc/pacman.d/mirrorlist"
# shellcheck disable=SC2016
grep -q '^Server' "$root/etc/pacman.d/mirrorlist" || echo 'Server = https://geo.mirror.pkgbuild.com/$repo/os/$arch' >>"$root/etc/pacman.d/mirrorlist"
unshare --mount --fork bash -euo pipefail -c '
root=$1 repo=$2 cache=$3 tmp=$4
mount --make-rprivate /
mount --bind "$root" "$root"
mount -t proc proc "$root/proc"
mount --rbind /sys "$root/sys"
mount --rbind /dev "$root/dev"
mount -t tmpfs -o mode=0755 tmpfs "$root/run"
mount --bind "$tmp" "$root/tmp"
mount --bind "$repo" "$root/src"
mount --bind "$cache" "$root/var/cache/pacman/pkg"
exec chroot "$root" /usr/bin/env -i HOME=/root TERM="${TERM:-dumb}" LANG=C.UTF-8 \
    PATH=/usr/local/sbin:/usr/local/bin:/usr/bin ${OLLIE_ROOTLESS:+OLLIE_ROOTLESS=1} \
    bash -c "pacman-key --init && pacman-key --populate archlinux && cd /src && exec bash os/build.sh"
' bash "$root" "$repo" "$work/pkg-cache" "$work/tmp"

# The unpacked Arch Linux and the build's scratch files are not needed any
# more (the ISO and the package downloads are kept).
rm -rf --one-file-system "${root:?}" "${work:?}/tmp"
step "Done: $(ls -t "$repo"/os/out/ollie-os-*.iso | head -n 1)"
