#!/usr/bin/env bash
# Build the Annie OS ISO.
#
# Runs on Arch Linux as root, with archiso, go and git installed. On any other
# system, use ./build-in-docker.sh instead. The ISO lands in annie-os/out/.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
repo=$(dirname "$here")
out=${OUT_DIR:-$here/out}
work=${WORK_DIR:-$here/work}
profile=$work/profile

[ "$(id -u)" = 0 ] || { echo "build.sh must run as root (mkarchiso needs it)" >&2; exit 1; }
command -v mkarchiso >/dev/null || { echo "mkarchiso not found: pacman -S archiso" >&2; exit 1; }

# Work on a copy, so the Oliver binary built below never lands in the source tree.
rm -rf -- "$work"
mkdir -p -- "$profile" "$out"
cp -a -- "$here"/{airootfs,efiboot,grub,syslinux,packages.x86_64,pacman.conf,profiledef.sh} "$profile/"

echo "==> Building Oliver"
version=$(git -C "$repo" describe --tags --always --dirty 2>/dev/null || echo dev)
(cd "$repo" && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$version" \
	-o "$profile/airootfs/usr/local/bin/oliver" ./cmd/oliver)
install -Dm 0644 "$repo/packaging/oliver.desktop" "$profile/airootfs/usr/share/applications/oliver.desktop"
install -Dm 0644 "$repo/packaging/oliver.svg" "$profile/airootfs/usr/share/icons/hicolor/scalable/apps/oliver.svg"

echo "==> Building the ISO"
mkarchiso -v -r -w "$work/build" -o "$out" "$profile"
rm -rf -- "$work"
ls -lh "$out"/*.iso
