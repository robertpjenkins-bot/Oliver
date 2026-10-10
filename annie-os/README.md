# Annie OS 🦄

A friendly Arch Linux desktop with a unicorn wallpaper, ready for games.

![Annie OS desktop preview](desktop-preview.png)

*A drawn preview of the desktop, made from its settings (not a screenshot).*

## What's inside

| | |
| --- | --- |
| **Base** | Arch Linux (rolling), Xfce desktop, boots straight into the desktop as the user `annie` |
| **Look** | Unicorn and rainbow wallpaper, an **app dock down the left side** and a top bar with the **search box on the right** (type to search the web or run a program; the Apps button next to it, or the Super/Windows key, searches your apps) |
| **Games** | Steam, with **Proton** switched on for every game (Steam Play), plus 32-bit Vulkan drivers and GameMode |
| **Annie Paint** | A drawing app: brush, rainbow brush, eraser, lines, boxes, circles, paint bucket, undo/redo, open and save PNG/JPEG |
| **Annie Fixer** | One-click repairs: unstick the package manager, repair signing keys, update, reinstall broken packages, free disk space, restart Wi-Fi or sound, reset the taskbar, repair Steam |
| **Annie Drivers** | Detects your graphics card and Wi-Fi and installs the right drivers (NVIDIA, AMD, Intel, Broadcom, firmware, Bluetooth, printers) |
| **Annie Disk Writer** | Formats USB drives as **exFAT** or **FAT32** and copies files onto them, or writes an `.iso` / `.img` to make a bootable stick. Also builds as a portable **AppImage** |
| **Oliver** | Runs Windows `.exe` programs with the best Wine or Proton on the system (from the folder above) |
| **Install Annie OS** | Installs everything to the computer's disk |

The finished ISO is about 3 GB, and the build refuses to finish if it ever goes over **25 GB**.

## Building the ISO

You need a computer (or virtual machine) running Arch Linux.

```console
$ sudo pacman -S archiso python-pillow
$ sudo ./build.sh
```

The ISO lands in `iso/annie-os-YYYY.MM.DD-x86_64.iso`. Everything is downloaded
from the official Arch package mirrors. Nothing comes from GitHub or the AUR.

Put it on a USB stick with Annie Disk Writer, or from any Linux computer:

```console
$ sudo dd if=iso/annie-os-*.iso of=/dev/sdX bs=4M conv=fsync status=progress
```

## Annie OS USB Maker

The easiest way onto a USB drive. Download `Annie-OS-USB-Maker-x86_64.AppImage`
from the release, along with the ISO (whole, in its two parts, or in a zip),
and run it on Linux or SteamOS. It finds Annie OS in Downloads, joins the
parts, checks them, asks which drive, and sets it up:

| Part | Size | |
| --- | --- | --- |
| ANNIE_BOOT | 5 GB, FAT32 | Annie OS itself |
| ANNIE_DATA | up to 20 GB, ext4 | everything you save and install, kept between starts |
| ANNIE_SHARE | the rest, exFAT or FAT32 | opens on Windows, macOS and SteamOS too |

Drives from 8 GB up work; under 25 GB, ANNIE_DATA shrinks to fit. On a
drive that's already FAT32 it copies Annie OS on without erasing it or
asking for a password (nothing is kept between starts then).
`Annie-OS-USB-Maker-Zip-x86_64.AppImage` is the same, but only takes a zip.
Build them with `./usb/build-appimage.sh`.

## The Disk Writer AppImage

```console
$ ./appimage/build-appimage.sh
```

This makes `out/AnnieDiskWriter.AppDir`, which you can run directly with
`./AppRun`. If `appimagetool` is installed (or you set `APPIMAGETOOL=/path/to/it`)
it also packs it into the single file `out/AnnieDiskWriter-x86_64.AppImage`.
The script never downloads anything. Running it needs Python 3 with Tk on the
computer (`sudo pacman -S python tk`).

The disk writer won't touch the disk your running system is on, and it warns
you before erasing a drive that looks internal rather than a USB stick. FAT32
can't hold files over 4 GB, so it tells you to use exFAT for those.

## Layout

```
build.sh                ISO build (extends Arch's releng profile)
packages.txt            packages added to the ISO and to installed systems
airootfs/               files placed into the Annie OS system
  usr/local/bin/        annie-paint, annie-fixer, annie-drivers, annie-install
  usr/lib/annie/        root helpers, live-user and login setup
  etc/skel/.config/     the Xfce panel layout (dock left, search right)
  usr/share/backgrounds/annie/unicorn.png
appimage/               Annie Disk Writer and its AppImage build
tools/make-wallpaper.py redraws the wallpaper: make-wallpaper.py out.png 2560 1440
```

## Notes

- The live user `annie` has no password and can do admin tasks without a
  prompt. After installing, you use the user and password you chose in the
  installer.
- Proton is downloaded by Steam itself the first time a Windows game runs.
- NVIDIA cards older than the GTX 16xx series need a legacy driver that Arch
  no longer ships in its official repositories, so Annie Drivers can't install
  it.
