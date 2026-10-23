#!/usr/bin/env bash
# Build the Annie OS ISO.
#
# Runs on Arch Linux as root, with archiso, go and git installed. On any other
# system, use ./build-in-docker.sh instead. The ISO lands in annie-os/out/.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
out=${OUT_DIR:-$here/out}
work=${WORK_DIR:-$here/work}
profile=$work/profile

[ "$(id -u)" = 0 ] || { echo "build.sh must run as root (mkarchiso needs it)" >&2; exit 1; }
command -v mkarchiso >/dev/null || { echo "mkarchiso not found: pacman -S archiso" >&2; exit 1; }

# Work on a copy, so the Oliver binary built for it never lands in the source tree.
rm -rf -- "$work"
mkdir -p -- "$out"
"$here/prepare-profile.sh" "$profile"

echo "==> Building the ISO"
mkarchiso -v -r -w "$work/build" -o "$out" "$profile"
rm -rf -- "$work"
ls -lh "$out"/*.iso
