#!/bin/bash
# Ollie OS's background in the other colours Ollie OS Settings offers
# (ollie-colours): wallpaper.svg with its teal swapped, drawn as Plasma
# wallpaper packages OllieOS-Purple and OllieOS-Red. Needs rsvg-convert.
#   colour-wallpapers.sh WALLPAPERS-DIR
set -euo pipefail
art="$(cd "$(dirname "$0")" && pwd)"
dest=${1:?}
#            darkest  edge     dark     main     side     glow     light
teal=(       083c38   0b4f4a   0f766e   14b8a6   2dd4bf   5eead4   9eeee4)
purple=(     2e1065   3b0764   7e22ce   a855f7   c084fc   d8b4fe   e9d5ff)
red=(        4c0519   4c0519   be123c   f43f5e   fb7185   fda4af   ffe4e6)
make() { # make NAME COLOURS...
    local name=$1 dir="$dest/OllieOS-$1" sed=() i size
    shift
    local -a to=("$@")
    for i in "${!teal[@]}"; do sed+=(-e "s/#${teal[$i]}/#${to[$i]}/gI"); done
    mkdir -p "$dir/contents/images"
    sed "${sed[@]}" "$art/wallpaper.svg" >"$dir/wallpaper.svg"
    for size in 1280x720 1920x1080 2560x1440 3840x2160; do
        rsvg-convert -w "${size%x*}" -h "${size#*x}" -o "$dir/contents/images/$size.png" "$dir/wallpaper.svg"
    done
    rm -f "$dir/wallpaper.svg"
    printf '{\n    "KPlugin": {\n        "Id": "OllieOS-%s",\n        "Name": "Ollie OS (%s)",\n        "Authors": [{ "Name": "Ollie OS" }],\n        "License": "MIT"\n    }\n}\n' "$name" "$name" >"$dir/metadata.json"
}
make Purple "${purple[@]}"
make Red "${red[@]}"
