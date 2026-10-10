# OS Maker 2.0

Design your own operating system the way you'd build a level in Mario Maker,
then download it as a real, bootable ISO.

1. **Design the desktop.** Drag a search bar, clock, calendar, system monitor,
   sticky notes, text and app icons onto the screen. Move and resize them,
   then pick their colours, see-through amount, corners and text size.
2. **Choose the look.** Set a dark or light theme, an accent colour, and a
   wallpaper (a colour, a gradient or your own picture). You can also move the
   taskbar to the top or bottom, change its colour and height, rename the Start
   button and pin apps to it.
3. **Pick the apps.** Tick the apps to pre-install: Firefox, Chromium, Discord,
   Steam, Minecraft (Prism Launcher), VLC, Spotify, GIMP, Krita, Blender,
   LibreOffice, Code - OSS and plenty more.
4. **Set it up.** Choose the OS name, the computer name, your username and
   password, automatic login, time zone and keyboard layout.
5. **Build ISO.** OS Maker turns the design into an Arch Linux ISO that boots
   straight into your desktop, laid out exactly as you placed it.

## Running it

```console
$ chmod +x OS_Maker-*-x86_64.AppImage
$ ./OS_Maker-*-x86_64.AppImage
```

The designer opens in its own browser window (Chromium-based browsers get a
window without tabs; anything else opens a tab). Your work saves itself as you
go. **Save** downloads a `.osmaker` file you can **Open** again later or
share.

## Building the ISO

ISOs are built with [archiso](https://wiki.archlinux.org/title/Archiso), the
same tool that builds the official Arch Linux ISO. OS Maker uses whichever of
these it finds:

| On this computer | What happens |
|---|---|
| Arch Linux | runs archiso directly (installing it if needed) and asks for your password |
| Docker or Podman | runs archiso inside an `archlinux` container |
| neither | **Only make the build folder** still works: copy it to an Arch machine and run `sudo bash build.sh` |

A build downloads every package (often 1–3 GB), needs about 10 GB of free
space and takes 10–40 minutes. The finished ISO lands in
`~/OS Maker/<computer-name>/out/`. Boot it in VirtualBox, GNOME Boxes or QEMU,
or write it to a USB stick with balenaEtcher or Fedora Media Writer.

From a terminal:

```console
$ osmaker build my-os.osmaker            # build the ISO
$ osmaker folder my-os.osmaker -o ~/out  # only write the build folder
```

## What's in the finished OS

- Arch Linux with the Xfce desktop, LightDM login and NetworkManager (Wi-Fi
  icon in the tray), plus PipeWire sound and the Papirus icons.
- Your desktop, drawn by `osmaker-desktop`, a small GTK program that reads
  `/etc/osmaker/layout.json` at login and places every widget where you put it.
  The design canvas is 1280×720, so on other screen sizes everything is scaled
  to the same proportions. The search bar launches apps as you type and sends
  anything else to a web search. Sticky notes save what you type in them.
- Your taskbar, which is Xfce's panel set up with your colours and pinned apps.
- Your account in the `wheel` group, so `sudo` works.

## Installing it on a computer

The ISO starts as a *live* system: it runs from the USB stick or DVD and starts
fresh on every boot. To keep it, click the **💿 Install** button on its
desktop (you place it in the designer like any app icon; it only shows up on
the live USB). It asks for:

1. **The disk to install to.** It must be at least 12 GB. The USB stick it's
   running from is never offered. **The whole disk is erased.**
2. **An optional password** for your account on the installed system.

It then copies the OS exactly as designed, with the same desktop, apps and
settings, onto the disk. It works on both newer (UEFI) and older (BIOS)
computers and installs a GRUB boot menu named after your OS. Restart, remove
the USB stick, and the computer boots into your OS. Things you save are kept
from then on.

On the installed system, everything that only belongs on a live USB is taken
out again: the Install button, the passwordless permission it used, and the
live-boot setup.

## Developing

```console
$ make run        # build and open the designer
$ make check      # gofmt, vet, tests
$ make appimage   # build dist/OS_Maker-*.AppImage
```

| Path | What it is |
|---|---|
| `internal/design` | the `.osmaker` format, validation, the app catalog |
| `internal/archiso` | turns a design into an archiso build folder; `assets/` holds `build.sh` and the desktop shell |
| `internal/builder` | runs `build.sh` with archiso, Docker or Podman |
| `internal/editor` | the local web server and the designer page (`web/`) |
