#!/bin/bash
# Build the installer for every platform into dist/.
set -euo pipefail
cd "$(dirname "$0")"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}
rm -rf dist && mkdir dist
build() {  # build GOOS GOARCH OUTPUT
    CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "dist/$3" .
}
build windows amd64 "Etros Installer.exe"
build darwin  arm64 etros-installer-mac-apple-silicon
build darwin  amd64 etros-installer-mac-intel
build linux   amd64 etros-installer-linux-x86_64
build linux   arm64 etros-installer-linux-arm64
# Zips keep the Mac and Linux programs executable after downloading.
(cd dist && for f in etros-installer-*; do chmod +x "$f"; zip -q -9 "$f.zip" "$f" && rm "$f"; done)
ls -lh dist
