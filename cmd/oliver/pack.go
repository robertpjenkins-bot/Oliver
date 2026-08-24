package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/pe"
)

const packSynopsis = `oliver pack <program.exe> [flags]

Wraps a Windows program into an AppImage that runs it on any Linux machine.

The AppImage carries the program and Oliver, not a copy of Wine: on each
machine it uses whichever compatibility runtime is installed there. That keeps
it small, and means it improves as the host's runtimes improve.

Packing does not change whether a program works. If it fails under Wine now, it
will fail the same way packed: run it with "oliver run" first.`

// packableName strips characters that are awkward in a file name.
var packableName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func (a *app) cmdPack(args []string) error {
	fs := a.newFlagSet("pack", packSynopsis)
	out := fs.String("o", "", "output path (default: <name>-x86_64.AppImage)")
	name := fs.String("name", "", "application name (default: taken from the executable)")
	iconPath := fs.String("icon", "", "PNG or SVG icon to use")
	payloadDir := fs.String("dir", "", "also bundle this directory, for programs that need their files alongside")
	launches := fs.String("launches", "", "for an installer: the program to run once it has installed, e.g. MinecraftLauncher.exe")
	appDirOnly := fs.Bool("appdir", false, "build the AppDir but do not package it")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("no program given")
	}

	info, err := pe.Inspect(fs.Arg(0))
	if err != nil {
		return err
	}

	// A name from the executable's own version resource beats its file name.
	appName := *name
	if appName == "" {
		appName = info.VersionInfo.ProductName
	}
	if appName == "" {
		appName = strings.TrimSuffix(info.Name, filepath.Ext(info.Name))
	}
	slug := slugFor(appName)

	// Packing an installer without saying what it installs produces an
	// AppImage that reinstalls the program every time it is opened, which is
	// almost never what someone means by packaging an application.
	if info.IsInstaller && *launches == "" {
		u := a.ui
		u.Warn("%s is an installer (%s)", info.Name, info.Installer)
		u.Info("%s", u.Dim("Packed as-is, the AppImage will run the installer on every launch."))
		u.Info("%s", u.Dim("Use -launches to name the program it installs, for example:"))
		u.Info("%s", u.Dim(fmt.Sprintf("  oliver pack %s -launches TheProgram.exe", info.Name)))
		u.Info("")
	}

	self, err := oliverBinary()
	if err != nil {
		return err
	}

	work, err := os.MkdirTemp("", "oliver-pack-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	appDir := filepath.Join(work, appName+".AppDir")

	u := a.ui
	u.Info("packing %s", appName)

	for _, d := range []string{
		filepath.Join(appDir, "usr", "bin"),
		filepath.Join(appDir, "usr", "share", "oliver", "payload"),
		filepath.Join(appDir, "usr", "share", "applications"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	if err := copyFile(self, filepath.Join(appDir, "usr", "bin", "oliver"), 0o755); err != nil {
		return fmt.Errorf("bundling oliver: %w", err)
	}

	payload := filepath.Join(appDir, "usr", "share", "oliver", "payload")
	if *payloadDir != "" {
		if err := copyTree(*payloadDir, payload); err != nil {
			return fmt.Errorf("bundling %s: %w", *payloadDir, err)
		}
		u.Info("· bundled %s", *payloadDir)
	}
	if err := copyFile(info.Path, filepath.Join(payload, info.Name), 0o755); err != nil {
		return fmt.Errorf("bundling %s: %w", info.Name, err)
	}
	u.Info("· bundled %s", info.Name)

	appRun := appRunFor(slug, info.Name, appName)
	if info.IsInstaller && *launches != "" {
		appRun = installerAppRunFor(slug, info.Name, *launches, appName)
	}
	if err := os.WriteFile(filepath.Join(appDir, "AppRun"), []byte(appRun), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(appDir, "usr", "bin", slug), []byte(appRun), 0o755); err != nil {
		return err
	}

	// Exec must name something that resolves on PATH. "AppRun" does not: it is
	// a file inside the AppDir, so any desktop entry using it fails with
	// "could not find AppRun" the moment it is used outside the mount. The
	// launcher is therefore also installed as usr/bin/<slug>, which AppRun
	// puts on PATH.
	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=Windows program packaged with Oliver
Exec=%s %%U
Icon=%s
Terminal=false
Categories=Utility;
`, appName, slug, slug)
	for _, p := range []string{
		filepath.Join(appDir, slug+".desktop"),
		filepath.Join(appDir, "usr", "share", "applications", slug+".desktop"),
	} {
		if err := os.WriteFile(p, []byte(desktop), 0o644); err != nil {
			return err
		}
	}

	iconExt := ".svg"
	iconData := []byte(iconSVG)
	if *iconPath != "" {
		data, err := os.ReadFile(*iconPath)
		if err != nil {
			return fmt.Errorf("reading icon: %w", err)
		}
		iconData = data
		iconExt = strings.ToLower(filepath.Ext(*iconPath))
	}
	if err := os.WriteFile(filepath.Join(appDir, slug+iconExt), iconData, 0o644); err != nil {
		return err
	}

	output := *out
	if output == "" {
		output = fmt.Sprintf("%s-x86_64.AppImage", appName)
	}
	if abs, err := filepath.Abs(output); err == nil {
		output = abs
	}

	if *appDirOnly {
		final := strings.TrimSuffix(output, ".AppImage") + ".AppDir"
		if err := os.RemoveAll(final); err != nil {
			return err
		}
		if err := copyTree(appDir, final); err != nil {
			return err
		}
		u.Info("%s built %s", u.Green("✓"), final)
		return nil
	}

	tool, err := findAppImageTool()
	if err != nil {
		// The AppDir is complete and usable; only the final packaging step
		// needs the tool, so keep the work rather than discarding it.
		final := strings.TrimSuffix(output, ".AppImage") + ".AppDir"
		if copyErr := copyTree(appDir, final); copyErr == nil {
			u.Warn("%v", err)
			u.Info("the unpackaged AppDir is at %s; run it with %s/AppRun", final, final)
			return nil
		}
		return err
	}

	cmd := exec.Command(tool, appDir, output)
	cmd.Env = append(os.Environ(), "ARCH=x86_64")
	cmd.Stdout, cmd.Stderr = io.Discard, os.Stderr
	if err := cmd.Run(); err != nil {
		// appimagetool is itself an AppImage and needs FUSE; without it, it
		// can unpack itself instead.
		cmd = exec.Command(tool, "--appimage-extract-and-run", appDir, output)
		cmd.Env = append(os.Environ(), "ARCH=x86_64")
		cmd.Stdout, cmd.Stderr = io.Discard, os.Stderr
		if err2 := cmd.Run(); err2 != nil {
			return fmt.Errorf("packaging failed: %w", err)
		}
	}

	u.Info("%s built %s", u.Green("✓"), output)
	u.Info("")
	u.Info("%s", u.Bold("To add it to your application menu:"))
	u.Info("  %s --install", shellQuoteArg(output))
	u.Info("%s", u.Dim("Do not use the .desktop file inside the AppImage: it cannot work on its own."))
	if info.RequiresAdmin || info.Packed {
		u.Info("%s", u.Dim("run `oliver info` on the program for anything that might stop it working"))
	}
	return nil
}

// oliverBinary returns a path to this executable's real binary, which is what
// gets bundled. Inside an AppImage that is the copy on the mount, which is a
// genuine binary and safe to copy out.
func oliverBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating this program: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// appRunFor builds the launcher script placed inside the AppImage.
//
// Two things it has to get right. The payload is copied to a writable location
// on first run, because an AppImage is mounted read-only and many Windows
// programs write next to their own executable. And the prefix is named
// explicitly, because the mount path changes on every launch and a
// path-derived name would build a fresh Windows environment every time.
func appRunFor(slug, exeName, appName string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu

HERE="$(dirname "$(readlink -f "${0}")")"
OLIVER="${HERE}/usr/bin/oliver"
export PATH="${HERE}/usr/bin:${PATH}"

# Adding this application to the desktop menu has to record where the AppImage
# actually is: a menu entry pointing at anything inside the mount breaks as
# soon as the AppImage is closed.
if [ "${1:-}" = "--install" ] || [ "${1:-}" = "--uninstall" ]; then
	APPS="${XDG_DATA_HOME:-${HOME}/.local/share}/applications"
	ICONS="${XDG_DATA_HOME:-${HOME}/.local/share}/icons/hicolor/scalable/apps"
	ENTRY="${APPS}/%[1]s.desktop"
	if [ "${1}" = "--uninstall" ]; then
		rm -f "${ENTRY}" "${ICONS}/%[1]s.svg"
		echo "Removed %[4]s from your application menu."
	else
		SELF="${APPIMAGE:-$(readlink -f "${0}")}"
		mkdir -p "${APPS}" "${ICONS}"
		[ -f "${HERE}/%[1]s.svg" ] && cp "${HERE}/%[1]s.svg" "${ICONS}/%[1]s.svg" || true
		cat > "${ENTRY}" <<DESKTOP_ENTRY
[Desktop Entry]
Type=Application
Name=%[4]s
Comment=Windows program packaged with Oliver
Exec="${SELF}" %%U
TryExec=${SELF}
Icon=%[1]s
Terminal=false
Categories=Utility;
DESKTOP_ENTRY
		echo "Added %[4]s to your application menu."
	fi
	command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "${APPS}" || true
	exit 0
fi

DATA="${XDG_DATA_HOME:-${HOME}/.local/share}/oliver/packed/%[1]s"
PAYLOAD="${HERE}/usr/share/oliver/payload"

# An AppImage is read-only and its mount point changes every launch. Copy the
# program somewhere stable and writable so it can keep its own files.
if [ ! -d "${DATA}" ]; then
	mkdir -p "${DATA}"
	cp -a "${PAYLOAD}/." "${DATA}/"
fi

exec "${OLIVER}" run --prefix %[1]s "${DATA}/%[2]s" "$@"
`, slug, exeName, "", appName)
}

// installerAppRunFor builds the launcher for a packed installer.
//
// The installer runs once, into the AppImage's own Windows environment; every
// later launch starts the program it installed. Without this an installer
// packed as an AppImage would simply reinstall itself on every open.
func installerAppRunFor(slug, installerName, launches, appName string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu

HERE="$(dirname "$(readlink -f "${0}")")"
OLIVER="${HERE}/usr/bin/oliver"
export PATH="${HERE}/usr/bin:${PATH}"

# Adding this application to the desktop menu has to record where the AppImage
# actually is: a menu entry pointing at anything inside the mount breaks as
# soon as the AppImage is closed.
if [ "${1:-}" = "--install" ] || [ "${1:-}" = "--uninstall" ]; then
	APPS="${XDG_DATA_HOME:-${HOME}/.local/share}/applications"
	ICONS="${XDG_DATA_HOME:-${HOME}/.local/share}/icons/hicolor/scalable/apps"
	ENTRY="${APPS}/%[1]s.desktop"
	if [ "${1}" = "--uninstall" ]; then
		rm -f "${ENTRY}" "${ICONS}/%[1]s.svg"
		echo "Removed %[4]s from your application menu."
	else
		SELF="${APPIMAGE:-$(readlink -f "${0}")}"
		mkdir -p "${APPS}" "${ICONS}"
		[ -f "${HERE}/%[1]s.svg" ] && cp "${HERE}/%[1]s.svg" "${ICONS}/%[1]s.svg" || true
		cat > "${ENTRY}" <<DESKTOP_ENTRY
[Desktop Entry]
Type=Application
Name=%[4]s
Comment=Windows program packaged with Oliver
Exec="${SELF}" %%U
TryExec=${SELF}
Icon=%[1]s
Terminal=false
Categories=Utility;
DESKTOP_ENTRY
		echo "Added %[4]s to your application menu."
	fi
	command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "${APPS}" || true
	exit 0
fi

DATA="${XDG_DATA_HOME:-${HOME}/.local/share}/oliver/packed/%[1]s"
STAMP="${DATA}/.installed"
PAYLOAD="${HERE}/usr/share/oliver/payload"

if [ ! -d "${DATA}" ]; then
	mkdir -p "${DATA}"
	cp -a "${PAYLOAD}/." "${DATA}/"
fi

# First launch: install into this AppImage's own Windows environment.
if [ ! -f "${STAMP}" ]; then
	echo "Installing %[2]s. This happens only once." >&2
	# Installers routinely report a non-zero status after succeeding, so the
	# proof of success is that the program appeared, not the exit code.
	"${OLIVER}" run --prefix %[1]s "${DATA}/%[2]s" "$@" || true
	if "${OLIVER}" which --prefix %[1]s %[3]s >/dev/null 2>&1; then
		touch "${STAMP}"
	fi
fi

# Every launch after that: start the installed program.
TARGET="$("${OLIVER}" which --prefix %[1]s %[3]s 2>/dev/null || true)"
if [ -z "${TARGET}" ]; then
	echo "Installed, but %[3]s was not found in the Windows environment." >&2
	echo "List what is there with: ${OLIVER} which --prefix %[1]s %[3]s --all" >&2
	exit 1
fi

exec "${OLIVER}" run --prefix %[1]s "${TARGET}" "$@"
`, slug, installerName, launches, appName)
}

// findAppImageTool locates appimagetool, which turns an AppDir into a file.
func findAppImageTool() (string, error) {
	if p := os.Getenv("APPIMAGETOOL"); p != "" {
		return p, nil
	}
	if p, err := exec.LookPath("appimagetool"); err == nil {
		return p, nil
	}
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cache = filepath.Join(home, ".cache")
		}
	}
	if cache != "" {
		p := filepath.Join(cache, "oliver", "appimagetool-x86_64.AppImage")
		if st, err := os.Stat(p); err == nil && st.Mode().Perm()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("appimagetool not found: install it, or set APPIMAGETOOL to a copy " +
		"(https://github.com/AppImage/appimagetool/releases)")
}

// shellQuoteArg quotes a path for display as a runnable command.
func shellQuoteArg(p string) string {
	if !strings.ContainsAny(p, " \t'\"\\") {
		return p
	}
	return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// copyTree copies a directory recursively, following the same rules as cp -a
// for the cases that matter here: regular files, directories and symlinks.
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case fi.IsDir():
			return os.MkdirAll(target, fi.Mode().Perm()|0o700)
		case fi.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			_ = os.Remove(target)
			return os.Symlink(link, target)
		case fi.Mode().IsRegular():
			return copyFile(path, target, fi.Mode().Perm())
		}
		return nil // sockets, devices and the like are not worth carrying
	})
}
