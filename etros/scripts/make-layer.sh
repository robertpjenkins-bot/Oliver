#!/bin/bash
# Build etros.squashfs, the layer stacked on top of both Batocera images.
#
#   make-layer.sh ART_DIR OUT.squashfs VERSION
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
ETROS=$(dirname "$HERE")
ART=$(realpath "$1")
OUT=$(realpath -m "$2")
VERSION=$3

STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

cp -a "$ETROS/layer/." "$STAGE/"
install -m 0755 "$ETROS/common/etros-sort-roms" "$STAGE/usr/bin/etros-sort-roms"
install -m 0755 "$HERE/add-my-roms.py" "$STAGE/usr/bin/etros-add-my-roms"
# Oliver, which picks the best way to run a Windows program (the Windows
# folder, see etros-launch-windows), built for the Pi 5's ARM processor.
(cd "$(dirname "$ETROS")" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false \
    -ldflags "-s -w" -o "$STAGE/usr/bin/oliver" ./cmd/oliver)
echo "Etros $VERSION" > "$STAGE/usr/share/etros/version"

THEME="$STAGE/usr/share/emulationstation/themes/etros-metroid"
mkdir -p "$THEME/art"
cp "$ART"/theme/*.png "$THEME/art/"

SPLASH="$STAGE/usr/share/batocera/splash"
mkdir -p "$SPLASH"
cp "$ART/boot-logo.png" "$SPLASH/boot-logo.png"
encode() {
    ffmpeg -loglevel error -y -framerate 30 -i "$1/%04d.png" \
        -c:v libx264 -profile:v main -pix_fmt yuv420p -crf 20 -movflags +faststart "$2"
}
encode "$ART/splash-16x9" "$SPLASH/splash.mp4"
encode "$ART/splash-4x3" "$STAGE/usr/share/etros/splash-4x3.mp4"

rm -f "$OUT"
mksquashfs "$STAGE" "$OUT" -comp zstd -all-root -noappend -quiet -no-progress
ls -lh "$OUT"
