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
    elif [ "$kind" = updater ]; then
        exe=etros-updater readme="READ ME - Etros Updater.txt" desktop="Etros Updater.desktop" title="Etros Updater"
    fi
    [ "$1" = windows ] && exe="$title.exe"
    # The folder inside carries the version, so unzipping a newer one never
    # lands on top of an older one.
    local folder="$3 - version $VERSION"
    local dir="dist/stage/$folder"
    mkdir -p "$dir"
    CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -trimpath \
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
package windows amd64 "Etros Installer - Windows"
package windows arm64 "Etros Installer - Windows (ARM)"
package darwin  arm64 "Etros Installer - Mac (Apple chip)"
package darwin  amd64 "Etros Installer - Mac (Intel)"
package linux   amd64 "Etros Installer - Linux"
package linux   arm64 "Etros Installer - Linux (ARM)"
rm -rf dist/stage
ls -lh dist
