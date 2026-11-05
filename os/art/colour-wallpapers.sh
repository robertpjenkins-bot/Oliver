#!/bin/bash
# Ollie OS's background in the other colours Ollie OS Settings offers
# (ollie-colours): wallpaper.svg with its teal swapped, drawn as Plasma
# wallpaper packages OllieOS-Purple, OllieOS-Red and OllieOS-Gold (a
# secret: the Ollie Gold theme, found in Ollie Finder); and the seasonal
# ones ollie-seasons puts on (OllieOS-Halloween, -Winter, -Birthday), with
# their decorations (seasons/*.svg) under the cube. Needs rsvg-convert.
#   colour-wallpapers.sh WALLPAPERS-DIR
set -euo pipefail
art="$(cd "$(dirname "$0")" && pwd)"
dest=${1:?}
#            darkest  edge     dark     main     side     glow     light
teal=(       083c38   0b4f4a   0f766e   14b8a6   2dd4bf   5eead4   9eeee4)
purple=(     2e1065   3b0764   7e22ce   a855f7   c084fc   d8b4fe   e9d5ff)
red=(        4c0519   4c0519   be123c   f43f5e   fb7185   fda4af   ffe4e6)
gold=(       451a03   78350f   b45309   f59e0b   fbbf24   fcd34d   fef3c7)
halloween=(  1a0b2e   2e1065   4c1d95   7c3aed   f97316   fb923c   fed7aa)
winter=(     0c2340   0f3a63   1e5a8a   3b82c4   7dd3fc   bae6fd   e0f2fe)
birthday=(   4a044e   701a75   a21caf   d946ef   f472b6   f9a8d4   fce7f3)
make() { # make NAME COLOURS... (DECOR=FILE: decorations under the cube)
    local name=$1 dir="$dest/OllieOS-$1" sed=() i size
    shift
    local -a to=("$@")
    for i in "${!teal[@]}"; do sed+=(-e "s/#${teal[$i]}/#${to[$i]}/gI"); done
    mkdir -p "$dir/contents/images"
    sed "${sed[@]}" "$art/wallpaper.svg" |
        awk -v decor="${DECOR:-}" 'decor != "" && /<g transform="translate\(960 450\)/ { while ((getline l < decor) > 0) print l } { print }' >"$dir/wallpaper.svg"
    for size in 1280x720 1920x1080 2560x1440 3840x2160; do
        rsvg-convert -w "${size%x*}" -h "${size#*x}" -o "$dir/contents/images/$size.png" "$dir/wallpaper.svg"
    done
    rm -f "$dir/wallpaper.svg"
    printf '{\n    "KPlugin": {\n        "Id": "OllieOS-%s",\n        "Name": "Ollie OS (%s)",\n        "Authors": [{ "Name": "Ollie OS" }],\n        "License": "MIT"\n    }\n}\n' "$name" "$name" >"$dir/metadata.json"
}
make Purple "${purple[@]}"
make Red "${red[@]}"
make Gold "${gold[@]}"
DECOR="$art/seasons/halloween.svg" make Halloween "${halloween[@]}"
DECOR="$art/seasons/winter.svg" make Winter "${winter[@]}"
DECOR="$art/seasons/birthday.svg" make Birthday "${birthday[@]}"
