# Annie OS

![Annie OS wallpaper](artwork/wallpaper.png)

Annie OS is an Arch Linux system made to run in VirtualBox. It comes with:

- the **KDE Plasma** desktop, in dark mode, with a magic wand wallpaper
- **Steam** (with the 32-bit libraries games need) and **Firefox**
- **Oliver**, from this repository, for running Windows programs
- VirtualBox Guest Additions, so the screen resizes with the window and the
  clipboard works between your computer and the VM
- a login screen with your own username and password

You get it as a bootable ISO. Boot it in a VM and you're in a live desktop;
double-click **Install Annie OS** to put it on the VM's disk.

## Getting the ISO: Annie OS Maker

On a Linux or SteamOS (desktop mode) computer:

1. Get `Annie-OS-Maker.tar.gz` and unpack it.
2. Double-click **Annie OS Maker** in the folder (or run `./annie-os-maker`).
3. Wait 20 minutes to an hour. It downloads Arch Linux (about 2.5 GB), builds
   the ISO, and saves it as `Downloads/annie-os-<date>-x86_64.iso`.

It never asks for a password. It works in a user namespace of its own, so
nothing outside `~/.cache/annie-os-maker` and the finished ISO is touched.
Failed downloads and builds are tried again by themselves (three times
each). It needs 20 GB free in your home folder. Everything it prints is
saved in `~/.cache/annie-os-maker/last-build.log`.

To make `Annie-OS-Maker.tar.gz` from this repository (needs Go):

```sh
./annie-os/maker/package.sh       # writes annie-os/out/Annie-OS-Maker.tar.gz
```

On Arch Linux you can also build the ISO directly with
`sudo ./annie-os/build.sh`, or with Docker on any Linux machine with
`./annie-os/build-in-docker.sh`. Both write it to `annie-os/out/`.

## Setting up the VirtualBox machine

1. **New** machine. Name: `Annie OS`. ISO Image: the Annie OS ISO.
   Type: **Linux**, Version: **Arch Linux (64-bit)**. Tick **Skip Unattended
   Installation**.
2. **Hardware**: at least **4096 MB** of memory (8192 MB if you'll play games)
   and **2 or more** CPUs. Leaving **Enable EFI** off or on both work.
3. **Hard disk**: create one of at least **30 GB**; 60 GB or more if you'll
   install games.
4. Before starting, open **Settings > Display**: set Graphics Controller to
   **VMSVGA**, Video Memory to **128 MB**, and tick **Enable 3D Acceleration**.
5. Start the machine. It boots straight to the Annie OS desktop.

## Installing

The installer opens by itself when the live desktop starts (or double-click
**Install Annie OS** on the desktop). It asks for:

- the disk to install on (everything on it is erased)
- your username and password
- a name for the computer
- your timezone

It then copies the system to the disk in a terminal window, which takes 5 to
15 minutes. When it says it's done, click **Shut down**, remove the ISO under
**Settings > Storage** in VirtualBox, and start the machine again. You'll get
the login screen with your username already filled in.

The installer copies the system you're running from the ISO, so it doesn't
need an internet connection.

## After installing

- **Updates**: open Konsole and run `sudo pacman -Syu`.
- **Steam**: the first launch downloads Steam's own runtime (a few hundred MB).
  To play Windows games, go to Steam > Settings > Compatibility and turn on
  Steam Play.
- **Windows programs**: right-click an `.exe` and open it with Oliver, or run
  `oliver run program.exe` in Konsole.
- **Clipboard sharing**: in VirtualBox, Devices > Shared Clipboard >
  Bidirectional.

### About games in a VM

VirtualBox gives the VM only basic 3D support, and none for Vulkan, which
most modern games and Proton use. Annie OS includes a software Vulkan driver
so those games can at least start, but it runs on the CPU and is slow.
Expect older and lighter 2D/3D games to play well and recent 3D games to
struggle.

## What's in this folder

| Path | What it is |
| --- | --- |
| `packages.x86_64` | Every package in the image |
| `profiledef.sh`, `pacman.conf` | The [archiso](https://wiki.archlinux.org/title/Archiso) profile settings; `pacman.conf` enables the multilib repository Steam needs |
| `airootfs/` | Files laid over the system: branding, theme, live session setup, installer |
| `airootfs/usr/local/bin/annie-installer` | The installer's questions (runs as the live user) |
| `airootfs/usr/local/bin/annie-install-backend` | The installer's disk work (runs as root) |
| `syslinux/`, `efiboot/`, `grub/` | Boot menus for BIOS and UEFI |
| `artwork/` | The wallpaper, boot splash and logo. Edit `make-artwork.py`, then run it and `render.sh` to redraw them |
| `maker/` | Annie OS Maker: builds the ISO on a Linux/SteamOS computer without a password |
| `prepare-profile.sh` | Copies the profile and adds Oliver; used by the Maker and `build.sh` |
| `build.sh`, `build-in-docker.sh` | Build the ISO as root, or in Docker |
