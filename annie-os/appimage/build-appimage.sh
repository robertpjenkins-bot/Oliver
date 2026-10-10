#!/usr/bin/env bash
# Builds AnnieDiskWriter-x86_64.AppImage.
#
# Nothing is downloaded: it needs an appimagetool you already have, either
# on PATH or given as APPIMAGETOOL=/path/to/appimagetool. Without one it
# stops after making the AppDir, which runs as-is with ./AppRun.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
out="${OUTDIR:-${here}/../out}"
appdir="${out}/AnnieDiskWriter.AppDir"

rm -rf "$appdir"
mkdir -p "$appdir/usr/share/applications" "$appdir/usr/share/icons/hicolor/scalable/apps"
src="$here/AnnieDiskWriter"
install -m755 "$src/AppRun" "$src/annie-disk-writer" "$src/annie-disk-helper" "$appdir/"
install -m644 "$src/annie-disk-writer.desktop" "$src/annie-disk-writer.svg" "$appdir/"
install -m644 "$src/annie-disk-writer.desktop" "$appdir/usr/share/applications/"
install -m644 "$src/annie-disk-writer.svg" "$appdir/usr/share/icons/hicolor/scalable/apps/"
echo "AppDir ready: $appdir"

tool="${APPIMAGETOOL:-$(command -v appimagetool || true)}"
if [ -z "$tool" ]; then
	echo "appimagetool not found; skipping the single-file AppImage (the AppDir above works on its own)."
	exit 0
fi
"$tool" --version >/dev/null 2>&1 || tool_args=--appimage-extract-and-run
ARCH=x86_64 "$tool" ${tool_args:-} "$appdir" "$out/AnnieDiskWriter-x86_64.AppImage"
echo "built $out/AnnieDiskWriter-x86_64.AppImage"
