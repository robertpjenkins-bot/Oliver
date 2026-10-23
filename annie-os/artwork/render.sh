#!/bin/sh
# Render the SVG artwork to the PNGs the ISO ships, using headless Chromium.
# Usage: ./render.sh   (set CHROMIUM=/path/to/chromium if it is not on PATH)
set -eu
cd "$(dirname "$0")"
chromium=${CHROMIUM:-$(command -v chromium || command -v chromium-browser || command -v google-chrome)}

render() { # svg png width height
	"$chromium" --headless --disable-gpu --no-sandbox --hide-scrollbars \
		--default-background-color=00000000 \
		--window-size="$3,$4" --screenshot="$PWD/$2" "file://$PWD/$1" 2>/dev/null
	echo "rendered $2"
}

render wallpaper.svg wallpaper.png 1920 1080
render splash.svg splash.png 640 480
render logo.svg logo.png 256 256
