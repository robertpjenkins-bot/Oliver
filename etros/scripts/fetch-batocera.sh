#!/bin/bash
# Download a Batocera release image.
#
#   fetch-batocera.sh BOARD PATTERN OUT.img.gz
#
# Looks in $BATOCERA_URL/BOARD/$BATOCERA_CHANNEL/last/ (the same place
# Batocera's own updater looks) for an image whose name matches PATTERN.
set -euo pipefail

BOARD=$1
PATTERN=$2
OUT=$3
BASE=${BATOCERA_URL:-https://updates.batocera.org}
CHANNEL=${BATOCERA_CHANNEL:-stable}
DIR="$BASE/$BOARD/$CHANNEL/last/"

listing=$(curl -fsSL "$DIR")
name=$(printf '%s' "$listing" | grep -oE 'href="[^"]+\.img\.gz"' | sed -e 's/^href="//' -e 's/"$//' |
       sed 's|.*/||' | grep -E "$PATTERN" | head -n1 || true)
if [ -z "$name" ]; then
    echo "no image matching '$PATTERN' in $DIR. It has:" >&2
    printf '%s' "$listing" | grep -oE 'href="[^"]+"' >&2 || true
    exit 1
fi

echo "downloading $DIR$name"
curl -fL --retry 3 -o "$OUT" "$DIR$name"
# Batocera publishes a checksum next to each image; use it when it is there.
if sum=$(curl -fsSL "$DIR$name.md5" 2>/dev/null); then
    echo "${sum%% *}  $OUT" | md5sum -c -
fi
