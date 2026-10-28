package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

const desktopSynopsis = `ziptransform desktop [install|uninstall]

Adds Zip Transform to the applications menu, and to the "Open With" menu for
ZIP files, so a project ZIP can be opened with it from the file manager.`

func cmdDesktop(u *ui.UI, args []string) error {
	fs := newFlagSet(u, "desktop", desktopSynopsis)
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch fs.Arg(0) {
	case "install", "":
		return desktopInstall(u)
	case "uninstall", "remove":
		return desktopUninstall(u)
	default:
		return fmt.Errorf("unknown subcommand %q (want install or uninstall)", fs.Arg(0))
	}
}

func desktopInstall(u *ui.UI) error {
	data := dataHome()
	if data == "" {
		return fmt.Errorf("cannot determine your home directory")
	}
	apps := filepath.Join(data, "applications")
	icons := filepath.Join(data, "icons", "hicolor", "scalable", "apps")
	for _, d := range []string{apps, icons} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(icons, "ziptransform.svg"), []byte(iconSVG), 0o644); err != nil {
		u.Warn("could not install the icon: %v", err)
	}

	launcher := quoteExec(selfPath())
	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Zip Transform
GenericName=MCreator ZIP to mod jar
Comment=Turn an MCreator project ZIP into a Minecraft mod .jar
Exec=%s %%f
TryExec=%s
Icon=ziptransform
Terminal=false
Categories=Development;Game;
MimeType=application/zip;application/x-zip-compressed;
Keywords=mcreator;minecraft;mod;jar;zip;forge;neoforge;fabric;
`, launcher, launcher)
	path := filepath.Join(apps, "ziptransform.desktop")
	if err := os.WriteFile(path, []byte(entry), 0o644); err != nil {
		return err
	}
	if _, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command("update-desktop-database", apps).Run()
	}
	u.Info("installed %s", path)
	return nil
}

func desktopUninstall(u *ui.UI) error {
	data := dataHome()
	for _, p := range []string{
		filepath.Join(data, "applications", "ziptransform.desktop"),
		filepath.Join(data, "icons", "hicolor", "scalable", "apps", "ziptransform.svg"),
	} {
		if err := os.Remove(p); err == nil {
			u.Info("removed %s", p)
		} else if !os.IsNotExist(err) {
			u.Warn("could not remove %s: %v", p, err)
		}
	}
	return nil
}

// selfPath is the path a menu entry should launch. Inside an AppImage the
// binary lives in a mount that disappears, so the AppImage file is the only
// path that lasts.
func selfPath() string {
	if p := os.Getenv("APPIMAGE"); p != "" {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
		return p
	}
	exe, err := os.Executable()
	if err != nil {
		return "ziptransform"
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

func dataHome() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share")
}

// quoteExec escapes a path for a .desktop Exec line.
func quoteExec(p string) string {
	if !strings.ContainsAny(p, ` "'\`+"\t") {
		return p
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(p) + `"`
}
