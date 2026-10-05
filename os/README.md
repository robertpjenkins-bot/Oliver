# Ollie OS

A desktop Linux made for running Windows programs and Steam games, sized for a
32 GB ("29 GB") drive, and tuned for Valve hardware: the Steam Machine and
Steam Deck.

![wallpaper](art/wallpaper.svg)

## What you get

- **A teal desktop with the Ollie cube.** Right-click any PNG or JPEG and
  choose **Set as wallpaper** to use it instead.
- **One bar along the bottom.**
  - **Left:** the cube button opens the app menu, and next to it is a
    **Google search box**. Type, press Enter, and the results open in Google
    Chrome. Typing a web address such as `youtube.com` goes straight there.
  - **Right:** your pinned apps, then the tray and the clock. To pin an app,
    right-click it and choose **Add to bar**. That works on desktop icons,
    on apps in Dolphin, and on Windows `.exe` files. In the app menu, the
    same option is called **Pin to Task Manager**.
- **`.exe` files just run.** Double-click one and [Oliver](../README.md)
  chooses the best runtime installed (Wine, Proton or Proton-GE) and starts
  it.
- **Preinstalled:** Google Chrome, Steam, Proton-GE, Wine (with Mono and
  Gecko), Oliver, the Dolphin file manager, Konsole, Kate and a screenshot
  tool.
- **Steam Machine and Steam Deck support:** AMD graphics drivers (32- and
  64-bit), Steam's controller rules, Deck audio firmware, and a
  **Steam (Game Mode)** session. Pick it on the login screen to boot straight
  into Big Picture, like a Deck.
- **A login screen** on the installed system. The live USB logs in
  automatically as `ollie` (password `ollie`).

## Built for a 29 GB drive

The installed system takes exactly **25 GB**:

| Partition | Size    | What                                                 |
|-----------|---------|------------------------------------------------------|
| boot      | 512 MB  | kernel and boot loader (systemd-boot)                |
| root      | 24.5 GB | btrfs with zstd compression; `@` and `@home` volumes |
| free      | rest    | about 4 GB left unpartitioned on a 32 GB stick       |

To make those 25 GB go further, there is no swap partition: compressed swap
lives in memory instead (zram). Old package downloads are cleaned up every
week, and system logs are capped at 200 MB.

## Getting the ISO

GitHub builds it: **Actions → Ollie OS ISO → latest run → Artifacts →
`ollie-os-iso`**. The ISO is about 3–4 GB. A build starts on every push that
touches `os/`, and can also be started by hand with **Run workflow**. Pushing
a tag named `os-v1.0` (or any `os-v*`) also publishes a GitHub release.

To build on your own Arch or SteamOS machine (needs Docker and about
20 GB free):

```console
$ docker run --rm --privileged -v "$PWD:/src" -w /src archlinux:latest bash os/build.sh
$ ls os/out/
ollie-os-2026.10.05-x86_64.iso  ollie-os-2026.10.05-x86_64.iso.sha256
```

## Writing it to a drive

`ollie-write` copies the ISO onto a USB stick or drive. It runs on any Linux,
including SteamOS desktop mode, and comes with Ollie OS too, as
**Write ISO to Drive**:

```console
$ sudo ./os/ollie-write ollie-os-2026.10.05-x86_64.iso
==> Image: ollie-os-2026.10.05-x86_64.iso (3412 MB)
==> Drives you can write to:

  1) /dev/sdb         28.9G  removable SanDisk Ultra

Write to which drive? [1-1] 1
EVERYTHING on /dev/sdb will be erased.
Type the drive name (sdb) to continue: sdb
==> Writing (do not unplug the drive)
==> Checking what was written
==> Done. /dev/sdb is ready; boot from it to try or install Ollie OS.
```

It never lists the drive your running system is on. You have to type the
drive's name before anything is erased, and it reads the drive back to check
the copy. On Windows or macOS, write the ISO with balenaEtcher or Rufus
instead.

## Installing

1. Boot from the USB stick. On a Steam Deck or Steam Machine, hold
   **Volume Down** and press **Power** to open the boot manager.
2. On the live desktop, double-click **Install Ollie OS**.
3. Choose the drive to install onto. The USB stick you booted from is not
   offered, so you need a second drive (for example the internal SSD, or
   another USB stick).
4. Type `ERASE`, then pick a username, password, computer name and time
   zone.
5. Restart and log in.

## How it is put together

```
os/
├── build.sh                 builds the ISO (runs inside an Arch container)
├── ollie-write              writes an ISO to a drive
├── art/                     cube logo, wallpaper, boot menu splash (SVG)
└── profile/
    ├── packages.x86_64            packages from the Arch repositories
    ├── packages-optional.x86_64   added only if the repositories carry them
    ├── aur-packages.txt           built from the AUR (Chrome, Proton-GE)
    └── airootfs/                  files copied into the system
        ├── etc/                   login screen, services, defaults
        └── usr/
            ├── local/bin/         ollie-install, ollie-add-to-bar, ...
            └── share/plasma/      the bar layout and the Google search box
```

The ISO is an [archiso](https://wiki.archlinux.org/title/Archiso) image. The
build starts from archiso's `releng` profile for the boot loader setup, then
layers `profile/` on top. The desktop is KDE Plasma 6 on Wayland. Its bar is
the layout script in
`profile/airootfs/usr/share/plasma/look-and-feel/org.ollieos.desktop/`, and
the search box is the small widget in
`profile/airootfs/usr/share/plasma/plasmoids/org.ollieos.googlesearch/`.
