#!/bin/bash
# Build the installer for every platform, each zipped with its READ ME, into
# dist/.
set -euo pipefail
cd "$(dirname "$0")"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}
rm -rf dist && mkdir -p dist/stage
package() {  # package GOOS GOARCH ZIPNAME
    local exe=etros-installer
    [ "$1" = windows ] && exe="Etros Installer.exe"
    local dir="dist/stage/$3"
    mkdir -p "$dir"
    CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$dir/$exe" .
    sed 's/$/\r/' "READ ME.txt" > "$dir/READ ME.txt"
    chmod +x "$dir/$exe"
    if [ "$1" = linux ]; then
        install -m 0755 "Etros Installer.desktop" "$dir/Etros Installer.desktop"
    fi
    (cd dist/stage && zip -q -9 -r "../$3.zip" "$3")
}
package windows amd64 "Etros Installer - Windows"
package windows arm64 "Etros Installer - Windows (ARM)"
package darwin  arm64 "Etros Installer - Mac (Apple chip)"
package darwin  amd64 "Etros Installer - Mac (Intel)"
package linux   amd64 "Etros Installer - Linux"
package linux   arm64 "Etros Installer - Linux (ARM)"
rm -rf dist/stage
ls -lh dist
