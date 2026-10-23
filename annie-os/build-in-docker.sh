#!/usr/bin/env bash
# Build the Annie OS ISO from any Linux machine with Docker, inside an Arch
# Linux container. The ISO lands in annie-os/out/.
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)

# --privileged: mkarchiso mounts filesystems while it assembles the image.
docker run --rm --privileged -v "$repo:/src" -w /src archlinux:latest bash -c '
	set -euo pipefail
	pacman -Syu --noconfirm --needed archiso go git
	git config --global --add safe.directory /src
	annie-os/build.sh
	chown -R '"$(id -u):$(id -g)"' annie-os/out
'
