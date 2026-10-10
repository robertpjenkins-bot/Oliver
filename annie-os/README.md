# Annie OS 🦄

A friendly Arch Linux desktop with a unicorn wallpaper, ready for games and
for running apps you download. Annie OS 2 is built on the same tested
foundation as Ollie OS (KDE Plasma 6 on Wayland), with Annie's own look and
apps on top.

## What you get

- **A unicorn and rainbow wallpaper**, a light purple theme, and the
  **Annie Rainbow** mouse cursor (a rainbow arrow with a sparkle, a pointing
  hand, a rainbow text cursor and a spinning rainbow ring while things load).
- **One bar along the bottom: apps on the left, search on the right.**
  - **Left:** the unicorn button opens the app menu, followed by your pinned
    apps: Firefox, Files, Steam, Annie Paint, Disk Writer, The Fixer, DC,
    Terminal and Settings. Right-click any app and choose **Add to bar** to
    pin it.
  - **Right:** a **search box** (type and press Enter to search the web, or
    type an address such as `youtube.com`), then the tray and the clock.
- **The Command / Windows key opens the app menu.**
- **Apps you download just run.** AppImages open with a double-click, even
  straight after downloading, before they're marked as allowed to run.
  `.exe` files run too, through [Oliver](../README.md), which picks Wine,
  Proton or Proton-GE.
- **Games:** Steam with Proton, Gamescope, Steam Game Mode, Wine, and **Get
  Proton-GE** in the menu.
- **Annie Paint:** a drawing app with a brush, a rainbow brush, an eraser,
  lines, boxes, circles, a paint bucket, undo and redo, and open and save.
- **Annie Disk Writer:** formats USB drives as **exFAT** or **FAT32** and
  copies files onto them, or writes an `.iso` / `.img` to a drive.
- **The Fixer:** checks every system program for missing or damaged files
  and puts good copies back.
- **DC - Debug Computer:** checks memory, storage, temperature, internet,
  graphics, sound, Bluetooth, Steam, Proton and Wine, explains each problem
  in plain words, and fixes what it can.
- **Driver Installer:** finds your graphics card, Wi-Fi, printers, tablets
  and more, and installs their drivers.
- **Update Annie OS** (in the menu): updates everything, adds anything
  missing, and runs every fix in one go. See [The Annie OS Updater](#the-annie-os-updater).
- **Your own username and password**, made the first time Annie OS starts
  (or choose **Skip** to go straight to the desktop). Admin tasks never ask
  for a password.

## Tested before every release

Each build boots the finished ISO in virtual machines and checks it the way
a person would use it:

| Test | What it checks |
| --- | --- |
| ISO contents | every package, app, menu entry, cursor and setting is there, and programs are allowed to run |
| Boot | the desktop starts, and nothing fails on the way |
| First start | create a username and password, then log in (QEMU, a GNOME Boxes-style PC, and VirtualBox) |
| The Fixer | breaks a program on purpose and checks The Fixer repairs it |
| **Apps** | starts Firefox, Dolphin, Konsole, Settings, Annie Paint and Annie Disk Writer inside the desktop, and runs an AppImage both directly and by double-click |

The apps test exists because the first Annie OS couldn't start apps: it
lacked the FUSE library AppImages need, and was never booted before release.

## Getting it

GitHub builds it on every push that changes `annie-os/`, then publishes the
**Annie OS (newest build)** release. Download:

- the ISO (in two parts if it's over GitHub's 2 GB limit for one file),
- **Annie-OS-USB-Maker-x86_64.AppImage** to put it on a USB drive,
- **Annie-OS-Updater-x86_64.AppImage** to update an Annie OS you already have.

To build it yourself on Linux with Docker (about 20 GB free):

```console
$ docker run --rm --privileged -v "$PWD:/src" -w /src archlinux:latest bash annie-os/build.sh
$ ls annie-os/out/
annie-os-2026.10.10-x86_64.iso  annie-os-2026.10.10-x86_64.iso.sha256  Annie-OS-Updater-x86_64.AppImage
```

Everything comes from the official Arch Linux repositories; nothing is built
from the AUR.

## Annie OS USB Maker

`Annie-OS-USB-Maker-x86_64.AppImage` puts Annie OS on a USB drive that it
then runs from, keeping your files and games. Run it on Linux or SteamOS
with the ISO (whole, its two parts, or a zip of them) in Downloads. It finds
them, joins and checks the parts, and asks only which drive to use.

| Part | Size | |
| --- | --- | --- |
| ANNIE_BOOT | 5 GB, FAT32 | Annie OS itself |
| ANNIE_DATA | up to 20 GB, ext4 | everything you save and install, kept between starts |
| ANNIE_SHARE | the rest, exFAT or FAT32 | opens on Windows, macOS and SteamOS too |

- Drives from **8 GB** up work; under 25 GB, ANNIE_DATA shrinks to fit.
- On a drive that is already FAT32, press **Enter** to copy Annie OS on
  without erasing anything (it then starts fresh every time), or type
  **ERASE** to wipe the drive and set it up as above.
- `Annie-OS-USB-Maker-Zip-x86_64.AppImage` is the same, but takes only a zip.

## The Annie OS Updater

`Annie-OS-Updater-x86_64.AppImage`, and **Update Annie OS** in the menu:

- installs the newest Annie apps and settings
- updates every package, repairing the package keys if they stop the update
- adds anything Annie OS needs that is missing
- puts on the rainbow cursor and sets the Command key to open the app menu
- unsticks installs, reinstalls damaged packages, frees disk space, and
  brings back Wi-Fi, sound and the taskbar

Run on the first Annie OS (the Xfce one), it adds AppImage support, Python
and Tk so apps start again, along with the cursor and the Command key. The
new desktop itself comes with a fresh USB drive made by the USB Maker.

## Installing

1. Boot from the USB drive. On a Steam Deck or Steam Machine, hold
   **Volume Down** and press **Power** to open the boot manager.
2. On the live desktop, double-click **Install Annie OS**.
3. Choose the drive to install onto (not the USB drive you booted from),
   type `ERASE`, then pick a computer name and time zone.
4. Restart. The first start asks for your username and password.

## How it is put together

```
annie-os/
├── build.sh            builds the ISO (inside an Arch Linux container)
├── annie-write         writes an ISO to a drive from a terminal
├── art/                logo and app icons
├── tools/              draws the unicorn wallpaper and the rainbow cursor
├── usb/                Annie OS USB Maker (AppImage)
├── updater/            Annie OS Updater (AppImage)
├── tests/              the virtual-machine tests above
└── profile/
    ├── packages.x86_64            packages from the Arch repositories
    ├── packages-optional.x86_64   added only if the repositories carry them
    └── airootfs/                  files copied into the system
        ├── etc/                   autologin, services, KDE defaults, Command key
        └── usr/
            ├── local/bin/         Annie's apps and helpers
            └── share/plasma/      the bar layout and the search box
```

The ISO is an [archiso](https://wiki.archlinux.org/title/Archiso) image: the
build starts from archiso's `releng` profile for the boot loader, then
layers `profile/` on top.
