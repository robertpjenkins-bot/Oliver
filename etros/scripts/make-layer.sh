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
# folder, see etros-launch-windows), for the Pi 5 and R36S's ARM processor
# and for PCs (Intel Macs); /usr/bin/oliver runs the one for this machine.
for arch in arm64:aarch64 amd64:x86_64; do
    (cd "$(dirname "$ETROS")" && CGO_ENABLED=0 GOOS=linux GOARCH=${arch%%:*} go build -trimpath -buildvcs=false \
        -ldflags "-s -w" -o "$STAGE/usr/bin/oliver.${arch##*:}" ./cmd/oliver)
done
cat > "$STAGE/usr/bin/oliver" <<'EOF'
#!/bin/sh
exec "/usr/bin/oliver.$(uname -m)" "$@"
EOF
chmod 0755 "$STAGE/usr/bin/oliver"
# Starting Ollie OS's own kernel on PCs (see etros-boot-ollie).
(cd "$(dirname "$ETROS")" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false \
    -ldflags "-s -w" -o "$STAGE/usr/bin/etros-kexec" ./etros/kexec)
echo "Etros $VERSION" > "$STAGE/usr/share/etros/version"
# Making Ollie OS on an Intel Mac whose Etros came from macOS
# (etros-setup-ollie-pc): the build script and Etros's files for Ollie OS.
install -m 0644 "$ETROS/installer/archbuild.sh" "$STAGE/usr/share/etros/archbuild.sh"
OF="$STAGE/usr/share/etros/ollie-files"
mkdir -p "$OF"
cp -r "$ETROS/ollie/files/." "$OF/"
install -m 0755 "$ETROS/common/etros-sort-roms" "$OF/usr/local/bin/etros-sort-roms"
mkdir -p "$OF/usr/share/ollie"
cp "$ART/ollie/bobby.png" "$OF/usr/share/ollie/bobby.png"
chmod 0755 "$OF"/usr/local/bin/* "$OF"/usr/local/sbin/* "$OF"/etc/systemd/system-generators/*

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
