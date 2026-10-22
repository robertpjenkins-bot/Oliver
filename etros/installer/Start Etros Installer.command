#!/bin/sh
# Starts Etros Installer from this window (double-click this file), for when
# the app won't open by itself: it clears macOS's "downloaded from the
# internet" mark from the app next to it, then runs the installer here, so
# that if anything goes wrong it's written in this window.
cd "$(dirname "$0")" || exit 1
app="$PWD/Etros Installer.app"
if [ ! -d "$app" ]; then
    echo "Etros Installer.app isn't next to this file: keep them in the same folder."
    read -r _
    exit 1
fi
xattr -cr "$app" 2>/dev/null
chmod +x "$app/Contents/MacOS/Etros Installer"
echo "Starting Etros Installer. It opens in Safari; leave this window open while you use it."
echo
"$app/Contents/MacOS/Etros Installer"
echo
echo "Etros Installer has stopped. If something went wrong, take a photo of this window."
echo "Press Enter to close it."
read -r _
