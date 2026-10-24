#!/usr/bin/env bash
# Make the Annie OS Maker download: annie-os/out/Annie-OS-Maker.tar.gz, a
# folder with the Maker, its READ ME and the Annie OS recipe (with Oliver
# already built). Needs go.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
out=${OUT_DIR:-$here/../out}
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
dir="$stage/Annie OS Maker"

mkdir -p -- "$dir" "$out"
install -m 0755 "$here/annie-os-maker" "$here/mksquashfs-owners" "$here/annie-make-disk" "$dir/"
install -m 0755 "$here/Annie OS Maker.desktop" "$dir/"
install -m 0644 "$here/READ ME.txt" "$dir/"
"$here/../prepare-profile.sh" "$dir/profile"

tar -C "$stage" -czf "$out/Annie-OS-Maker.tar.gz" "Annie OS Maker"
ls -lh "$out/Annie-OS-Maker.tar.gz"
