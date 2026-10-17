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
    if [ "$kind" = maker ]; then
        exe=etros-maker readme="READ ME - Etros Maker.txt" desktop="Etros Maker.desktop"
    fi
    [ "$1" = windows ] && exe=$( [ "$kind" = maker ] && echo "Etros Maker.exe" || echo "Etros Installer.exe")
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
        sed "s/etros-installer/$exe/g; s/Name=Etros Installer/Name=$( [ "$kind" = maker ] && echo 'Etros Maker' || echo 'Etros Installer')/" \
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
package windows amd64 "Etros Installer - Windows"
package windows arm64 "Etros Installer - Windows (ARM)"
package darwin  arm64 "Etros Installer - Mac (Apple chip)"
package darwin  amd64 "Etros Installer - Mac (Intel)"
package linux   amd64 "Etros Installer - Linux"
package linux   arm64 "Etros Installer - Linux (ARM)"
rm -rf dist/stage
ls -lh dist
