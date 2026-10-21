#!/bin/bash
# Build the installer for every platform, each zipped with its READ ME, into
# dist/.
set -euo pipefail
cd "$(dirname "$0")"
VERSION=${VERSION:-$(date -u +%Y-%m-%d.%H%M)}  # shown on the title screen
rm -rf dist && mkdir -p dist/stage
package() {  # package GOOS GOARCH ZIPNAME [maker]
    local kind=${4:-installer}
    local exe=etros-installer readme="READ ME.txt" desktop="Etros Installer.desktop"
    local title="Etros Installer"
    if [ "$kind" = maker ]; then
        exe=etros-maker readme="READ ME - Etros Maker.txt" desktop="Etros Maker.desktop" title="Etros Maker"
    elif [ "$kind" = mac ]; then
        exe=etros-mac-installer readme="READ ME - Etros Mac Installer.txt" desktop="Etros Mac Installer.desktop" title="Etros Mac Installer"
        [ "$1" = darwin ] && readme="READ ME - Etros Mac Installer (on the Mac).txt"
    elif [ "$kind" = updater ]; then
        exe=etros-updater readme="READ ME - Etros Updater.txt" desktop="Etros Updater.desktop" title="Etros Updater"
    fi
    [ "$1" = windows ] && exe="$title.exe"
    # The folder inside carries the version, so unzipping a newer one never
    # lands on top of an older one.
    local folder="$3 - version $VERSION"
    local dir="dist/stage/$folder"
    mkdir -p "$dir"
    # Go 1.25 and newer make Mac programs that need macOS 12; Go 1.24 still
    # makes them for macOS 11 (Big Sur), which Intel Macs like the 12-inch
    # MacBook stay on.
    local toolchain=${GOTOOLCHAIN:-}
    [ "$1" = darwin ] && toolchain=go1.24.11
    GOTOOLCHAIN=$toolchain CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -trimpath \
        -ldflags "-s -w -X main.version=$VERSION -X main.mode=$kind" -o "$dir/$exe" .
    sed 's/$/\r/' "$readme" > "$dir/READ ME.txt"
    chmod +x "$dir/$exe"
    if [ "$1" = linux ]; then
        sed "s/etros-installer/$exe/g; s/Name=Etros Installer/Name=$title/" \
            "Etros Installer.desktop" > "$dir/$desktop"
        chmod 0755 "$dir/$desktop"
    fi
    (cd dist/stage && zip -q -9 -r "../$3.zip" "$folder")
}
package windows amd64 "Etros Maker - Windows" maker
package windows arm64 "Etros Maker - Windows (ARM)" maker
package darwin  arm64 "Etros Maker - Mac (Apple chip)" maker
package darwin  amd64 "Etros Maker - Mac (Intel)" maker
package linux   amd64 "Etros Maker - Linux" maker
package linux   arm64 "Etros Maker - Linux (ARM)" maker
package windows amd64 "Etros Updater - Windows" updater
package darwin  arm64 "Etros Updater - Mac (Apple chip)" updater
package darwin  amd64 "Etros Updater - Mac (Intel)" updater
package linux   amd64 "Etros Updater - Linux" updater
package linux   arm64 "Etros Updater - Linux (ARM)" updater
package linux   amd64 "Etros Mac Installer - Linux" mac
package darwin  amd64 "Etros Mac Installer - Mac (Intel)" mac
package windows amd64 "Etros Installer - Windows"
package windows arm64 "Etros Installer - Windows (ARM)"
package darwin  arm64 "Etros Installer - Mac (Apple chip)"
package darwin  amd64 "Etros Installer - Mac (Intel)"
package linux   amd64 "Etros Installer - Linux"
package linux   arm64 "Etros Installer - Linux (ARM)"

# For the Mac: a disk image with Etros Installer.app in it, which opens the
# installer as a page in Safari (macweb.go). Made as an ISO 9660 image with
# Rock Ridge, which macOS opens as a .dmg and which keeps the program
# runnable.
dmg() {  # dmg FOLDER-OF-A-DARWIN-PACKAGE DMG-NAME
    local src="dist/stage/$1" root=dist/stage/dmg app
    rm -rf "$root" && mkdir -p "$root"
    app="$root/Etros Installer.app/Contents"
    mkdir -p "$app/MacOS"
    cp "$src/etros-mac-installer" "$app/MacOS/Etros Installer"
    cat > "$app/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>Etros Installer</string>
	<key>CFBundleDisplayName</key><string>Etros Installer</string>
	<key>CFBundleIdentifier</key><string>com.github.robertpjenkins-bot.etros-installer</string>
	<key>CFBundleExecutable</key><string>Etros Installer</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleVersion</key><string>$VERSION</string>
	<key>CFBundleShortVersionString</key><string>$VERSION</string>
	<key>LSMinimumSystemVersion</key><string>11.0</string>
	<key>LSUIElement</key><true/>
</dict>
</plist>
EOF
    chmod 0755 "$app/MacOS/Etros Installer"
    cp "$src/READ ME.txt" "$root/READ ME.txt"
    genisoimage -quiet -V "Etros Installer" -R -J -joliet-long -o "dist/$2" "$root"
}
dmg "Etros Mac Installer - Mac (Intel) - version $VERSION" "Etros Mac Installer.dmg"
rm -rf dist/stage
ls -lh dist
