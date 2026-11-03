#!/bin/bash
# Build Ollie OS Demo for the three kinds of computer, into OUT (os/out):
#   Ollie-OS-Demo-Windows.exe          Windows
#   Ollie-OS-Demo-Mac.dmg              Mac (Intel and Apple Silicon), an app in a disk image
#   Ollie-OS-Demo-x86_64.AppImage      Linux and SteamOS
# Needs Go, xorriso, and appimagetool (APPIMAGETOOL, else downloaded).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
os="$(dirname "$here")"
out="${OUT:-$os/out}"
mkdir -p "$out"
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp:?}"' EXIT
build() { (cd "$here" && GOOS=$1 GOARCH=$2 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$3" .); }

build windows amd64 "$out/Ollie-OS-Demo-Windows.exe"

# The Mac app: one program for Intel and Apple Silicon, opening the demo in the browser.
app="$tmp/dmg/Ollie OS Demo.app"
mkdir -p "$app/Contents/MacOS"
build darwin amd64 "$tmp/mac-amd64"
build darwin arm64 "$tmp/mac-arm64"
python3 "$here/make-fat.py" "$app/Contents/MacOS/ollie-os-demo" "$tmp/mac-amd64" "$tmp/mac-arm64"
chmod 0755 "$app/Contents/MacOS/ollie-os-demo"
cat >"$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Ollie OS Demo</string>
  <key>CFBundleDisplayName</key><string>Ollie OS Demo</string>
  <key>CFBundleIdentifier</key><string>io.github.ollie-os.demo</string>
  <key>CFBundleExecutable</key><string>ollie-os-demo</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>1.0</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>LSUIElement</key><true/>
</dict>
</plist>
PLIST
printf 'Ollie OS Demo for Mac\r\n\r\nRight-click "Ollie OS Demo" and choose Open (the first time, macOS asks\r\nbecause it is not from the App Store: click Open). The demo opens in your\r\nweb browser and stops by itself when you close its tab.\r\n' >"$tmp/dmg/Read me.txt"
xorriso -as mkisofs -quiet -V "Ollie OS Demo" -R -J -hfsplus -o "$out/Ollie-OS-Demo-Mac.dmg" "$tmp/dmg"

# The AppImage.
appdir="$tmp/OllieDemo.AppDir"
mkdir -p "$appdir/usr/bin"
build linux amd64 "$appdir/usr/bin/ollie-os-demo"
printf '#!/bin/sh\n# AppImage entry point: open the Ollie OS demo.\nHERE="$(dirname "$(readlink -f "$0")")"\nexec "$HERE/usr/bin/ollie-os-demo" "$@"\n' >"$appdir/AppRun"
chmod 0755 "$appdir/AppRun"
printf '[Desktop Entry]\nType=Application\nName=Ollie OS Demo\nComment=Try Ollie OS in your web browser\nExec=ollie-os-demo\nIcon=ollie-os\nTerminal=false\nCategories=Utility;\n' >"$appdir/ollie-os-demo.desktop"
install -m 0644 "$os/art/ollie-os.svg" "$appdir/ollie-os.svg"
tool="${APPIMAGETOOL:-$tmp/appimagetool}"
if [[ ! -x $tool ]]; then
    curl -fsSL -o "$tool" https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
    chmod +x "$tool"
fi
ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out/Ollie-OS-Demo-x86_64.AppImage" >/dev/null
ls -la "$out"/Ollie-OS-Demo*
