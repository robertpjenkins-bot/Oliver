#!/usr/bin/env bash
# Builds the small JS Academy AppImage: the app without the Electron engine,
# which the AppImage downloads on its first launch. See scripts/AppRun-slim.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "${here}/.." && pwd)"
cd "${root}"

node scripts/fetch-tools.js
version="$(node -p 'require("./package.json").version')"
electron="$(node -p 'require("electron/package.json").version')"
out="${root}/dist/JS-Academy-${version}-small-x86_64.AppImage"
appdir="$(mktemp -d)/JS-Academy.AppDir"
trap 'rm -rf "$(dirname "${appdir}")"' EXIT

app="${appdir}/app"
mkdir -p "${app}/node_modules/codemirror" "${app}/build-tools"
cp -r src package.json "${app}/"
cm=node_modules/codemirror
mkdir -p "${app}/${cm}/lib" "${app}/${cm}/mode" "${app}/${cm}/addon"
cp "${cm}/package.json" "${cm}/LICENSE" "${app}/${cm}/"
cp -r "${cm}/lib/." "${app}/${cm}/lib/"
for m in javascript xml css htmlmixed; do cp -r "${cm}/mode/${m}" "${app}/${cm}/mode/"; done
for a in edit comment selection runmode hint; do cp -r "${cm}/addon/${a}" "${app}/${cm}/addon/"; done
cp build-tools/appimagetool-x86_64.AppImage build-tools/runtime-x86_64 "${app}/build-tools/"

echo "${electron}" > "${appdir}/electron-version"
install -m 0755 scripts/AppRun-slim "${appdir}/AppRun"
install -m 0644 build/icon.png "${appdir}/js-academy.png"
ln -s js-academy.png "${appdir}/.DirIcon"
cat > "${appdir}/js-academy.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=JS Academy
Comment=Learn JavaScript with 500 lessons and a playground
Exec=js-academy
Icon=js-academy
Categories=Education;Development;
Terminal=false
DESKTOP

mkdir -p dist
ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1 build-tools/appimagetool-x86_64.AppImage \
	--no-appstream --runtime-file build-tools/runtime-x86_64 "${appdir}" "${out}"
chmod +x "${out}"
ls -lh "${out}"
