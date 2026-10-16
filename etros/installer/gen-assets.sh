#!/bin/bash
# Regenerate the files the installer carries inside it (assets/). Run after
# changing anything in ../layer, ../ollie, ../initrd or the artwork. Needs
# python3 with Pillow, ffmpeg, squashfs-tools and e2fsprogs.
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
ETROS=$(dirname "$HERE")
A=$HERE/assets
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

VERSION=${VERSION:-$(git -C "$ETROS" describe --tags --always --dirty 2>/dev/null || echo dev)}

python3 "$ETROS/scripts/make-art.py" "$WORK/art"
mkdir -p "$A"
"$ETROS/scripts/make-layer.sh" "$WORK/art" "$A/etros.squashfs" "$VERSION"

cp "$ETROS/initrd/etros-hook.sh" "$A/etros-hook.sh"
cp "$ETROS/my-roms-README.txt" "$A/my-roms-README.txt"

rm -rf "$A/ollie"
cp -r "$ETROS/ollie/files" "$A/ollie"
install -m 0755 "$ETROS/common/etros-sort-roms" "$A/ollie/usr/local/bin/etros-sort-roms"
cp "$WORK/art/ollie/bobby.png" "$A/ollie/usr/share/ollie/bobby.png"

mkdir -p "$A/ui"
cp "$WORK/art/theme/splash.png" "$A/ui/splash.png"
cp "$ETROS/layer/usr/share/emulationstation/themes/etros-metroid/fonts/PressStart2P-Regular.ttf" "$A/ui/"

# An empty 512 MiB ext4 "SHARE" partition. Batocera formats and grows it on
# the first start; it only has to be recognisable as ext4 until then.
mkfs.ext4 -q -F -L SHARE -m 0 -O ^metadata_csum_seed,^orphan_file "$WORK/share.ext4" 512M 2>/dev/null ||
    mkfs.ext4 -q -F -L SHARE -m 0 "$WORK/share.ext4" 512M
gzip -9 -n -c "$WORK/share.ext4" > "$A/share.ext4.gz"

ls -lh "$A"
