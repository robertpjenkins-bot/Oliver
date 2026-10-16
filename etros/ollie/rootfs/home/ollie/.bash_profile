# Start the desktop on the auto-logged-in console. If X stops, so does the
# login, and the console logs straight back in and starts it again.
if [ -z "$DISPLAY" ] && [ "$(tty)" = /dev/tty1 ]; then
    exec startx 2>/tmp/xorg-session.log
fi
