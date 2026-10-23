#!/usr/bin/env bash
# Copy the Annie OS archiso profile to DEST, with Oliver built and added.
# Used by build.sh and by maker/package.sh. Needs go.
#
#   prepare-profile.sh DEST
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
repo=$(dirname "$here")
dest=$1

rm -rf -- "$dest"
mkdir -p -- "$dest"
cp -a -- "$here"/{airootfs,efiboot,grub,syslinux,packages.x86_64,pacman.conf,profiledef.sh} "$dest/"

echo "==> Building Oliver"
version=$(git -C "$repo" describe --tags --always --dirty 2>/dev/null || echo dev)
(cd "$repo" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$version" \
	-o "$dest/airootfs/usr/local/bin/oliver" ./cmd/oliver)
install -Dm 0644 "$repo/packaging/oliver.desktop" "$dest/airootfs/usr/share/applications/oliver.desktop"
install -Dm 0644 "$repo/packaging/oliver.svg" "$dest/airootfs/usr/share/icons/hicolor/scalable/apps/oliver.svg"
