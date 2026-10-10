#!/usr/bin/env bash
# Builds an OS Maker ISO on any Linux distribution, inside Arch Linux's
# official bootstrap image, so neither an Arch install nor Docker is needed.
#
# Run as root:  osmaker-arch-build.sh BOOTSTRAP.tar CACHE_DIR BUILD_DIR
#
# OS Maker downloads and checks BOOTSTRAP.tar first; this unpacks it into
# CACHE_DIR/root.x86_64 (once), then runs BUILD_DIR/build.sh inside it.
set -euo pipefail

say() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
die() { printf '\nerror: %s\n' "$*" >&2; exit 1; }

[ "$#" -eq 3 ] || die "usage: $0 BOOTSTRAP.tar CACHE_DIR BUILD_DIR"
tarball="$1"
cache="$2"
build="$3"
root="${cache}/root.x86_64"

[ "$(id -u)" -eq 0 ] || die "this needs to run as root"
[ -f "${tarball}" ] || die "${tarball} is missing"
[ -f "${build}/build.sh" ] || die "${build} has no build.sh"
for tool in tar unshare chroot mount; do
	command -v "${tool}" >/dev/null || die "${tool} is missing on this computer"
done

# Whoever asked for the build gets the ISO.
owner="${PKEXEC_UID:-${SUDO_UID:-0}}"
group="$(getent passwd "${owner}" | cut -d: -f4)"

if [ ! -x "${root}/usr/bin/pacman" ]; then
	say "unpacking Arch Linux"
	rm -rf "${root}"
	tar -xpf "${tarball}" -C "${cache}" --numeric-owner
fi
[ -x "${root}/usr/bin/pacman" ] || die "the Arch Linux image did not unpack properly"

printf 'Server = %s\n' \
	'https://geo.mirror.pkgbuild.com/$repo/os/$arch' \
	'https://mirrors.kernel.org/archlinux/$repo/os/$arch' \
	'https://mirror.rackspace.com/archlinux/$repo/os/$arch' \
	> "${root}/etc/pacman.d/mirrorlist"
mkdir -p "${root}/osmaker"

say "entering Arch Linux"
# A private mount namespace: every mount below disappears when the build
# ends, however it ends, so nothing is left mounted on this computer.
exec unshare --mount --propagation private \
	env R="${root}" B="${build}" OWNER="${owner}" GROUP="${group:-${owner}}" \
	bash -euo pipefail -c '
mount --bind "$R" "$R"
mount -t proc proc "$R/proc"
mount --rbind /sys "$R/sys"
mount --rbind /dev "$R/dev"
mount -t tmpfs tmpfs "$R/run"
mount -t tmpfs tmpfs "$R/tmp"
mount --bind "$B" "$R/osmaker"
rm -f "$R/etc/resolv.conf"
cp -L /etc/resolv.conf "$R/etc/resolv.conf"

chroot "$R" /usr/bin/env -i \
	PATH=/usr/local/sbin:/usr/local/bin:/usr/bin \
	HOME=/root TERM=dumb LANG=C.UTF-8 \
	OSMAKER_CONTAINER=1 WORK=/var/tmp/osmaker-work \
	HOST_UID="$OWNER" HOST_GID="$GROUP" \
	/bin/bash -euo pipefail -c "
if [ ! -s /etc/pacman.d/gnupg/trustdb.gpg ]; then
	pacman-key --init
	pacman-key --populate archlinux
fi
bash /osmaker/build.sh
"
'
