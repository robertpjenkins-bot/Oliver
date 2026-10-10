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

# Whoever asked for the build gets the ISO. Without a password, OS Maker
# runs this inside a user namespace where "root" already is that person.
if [ "${OSMAKER_ROOTLESS:-0}" = 1 ]; then
	owner=0
	group=0
else
	owner="${PKEXEC_UID:-${SUDO_UID:-0}}"
	group="$(getent passwd "${owner}" | cut -d: -f4)"
fi

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

if [ "${OSMAKER_ROOTLESS:-0}" = 1 ]; then
	# Inside a user namespace the kernel refuses the devtmpfs and sysfs mounts
	# pacstrap makes. This stand-in bind-mounts them instead, as pacstrap's
	# own unshare mode does, and is found first on PATH.
	mkdir -p "${root}/usr/local/bin"
	cat > "${root}/usr/local/bin/pacstrap" <<'PACSTRAP'
#!/bin/bash
# OS Maker's pacstrap for building without root. Accepts mkarchiso's options.
set -euo pipefail
conf=/etc/pacman.conf
hostcache=0
confirm=--noconfirm
while getopts ':C:cGiKMNP' opt; do
	case "${opt}" in
		C) conf="${OPTARG}" ;;
		c) hostcache=1 ;;
		i) confirm= ;;
		*) ;;
	esac
done
shift $((OPTIND - 1))
newroot="$1"
shift
mkdir -m 0755 -p "${newroot}"/var/{cache/pacman/pkg,lib/pacman,log} "${newroot}"/{dev,run,etc/pacman.d,proc,sys}
mkdir -m 1777 -p "${newroot}/tmp"
cache=(--cachedir "${newroot}/var/cache/pacman/pkg")
[ "${hostcache}" = 1 ] && cache=()
exec unshare --mount --propagation private bash -euo pipefail -c '
newroot="$1"; shift
mount --bind "${newroot}" "${newroot}"
mount -t proc proc "${newroot}/proc" -o nosuid,noexec,nodev
mount --rbind /sys "${newroot}/sys"
mount --rbind /dev "${newroot}/dev"
mount -t tmpfs run "${newroot}/run" -o nosuid,nodev,mode=0755
mount -t tmpfs tmp "${newroot}/tmp" -o mode=1777,strictatime,nodev,nosuid
pacman -r "${newroot}" "$@"
' pacstrap "${newroot}" --config "${conf}" ${confirm} "${cache[@]}" -Sy "$@"
PACSTRAP
	chmod 0755 "${root}/usr/local/bin/pacstrap"
else
	rm -f "${root}/usr/local/bin/pacstrap"
fi

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
