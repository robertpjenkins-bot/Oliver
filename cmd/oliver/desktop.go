package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// exeMimeTypes are the MIME types desktops use for Windows programs. Different
// distributions and file managers disagree, so all of them are claimed.
var exeMimeTypes = []string{
	"application/x-ms-dos-executable",
	"application/x-msdownload",
	"application/vnd.microsoft.portable-executable",
	"application/x-msi",
}

const desktopSynopsis = `oliver desktop [install|uninstall|status]

Registers Oliver with your desktop so .exe and .msi files can be opened by
double-clicking them, and so Oliver appears in the "Open With" menu.

An AppImage is a single file that nothing knows about until you say so; this
is the step that connects it to your file manager.`

func (a *app) cmdDesktop(args []string) error {
	fs := a.newFlagSet("desktop", desktopSynopsis)
	setDefault := fs.Bool("default", false, "also make Oliver the default handler for .exe files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sub := "status"
	if fs.NArg() > 0 {
		sub = fs.Arg(0)
	}

	switch sub {
	case "install":
		return a.desktopInstall(*setDefault)
	case "uninstall", "remove":
		return a.desktopUninstall()
	case "status":
		path := desktopFilePath()
		if _, err := os.Stat(path); err != nil {
			a.ui.Printf("Not registered. Run `oliver desktop install`.")
			return nil
		}
		a.ui.Printf("Registered at %s", path)
		a.ui.Printf("%s", a.ui.Dim("launcher: "+selfPath()))
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q (want install, uninstall or status)", sub)
	}
}

// cmdDesktopWelcome runs when the AppImage is opened with no arguments from a
// desktop menu, where there is no terminal to print help into. It offers the
// one thing that is worth doing at that moment: connecting Oliver to the file
// manager so .exe files can be double-clicked.
func (a *app) cmdDesktopWelcome(args []string) error {
	registered := false
	if p := desktopFilePath(); p != "" {
		if _, err := os.Stat(p); err == nil {
			registered = true
		}
	}

	if registered {
		dialogInfo("Oliver is ready",
			"Oliver is already set up.\n\nRight-click a .exe file and choose Open With → Oliver, or run `oliver` in a terminal.")
		return nil
	}

	if !dialogConfirm("Set up Oliver",
		"Register Oliver so you can open Windows programs by double-clicking them?") {
		return nil
	}
	if err := a.desktopInstall(true); err != nil {
		dialogError("Setup failed", err.Error())
		return err
	}
	dialogInfo("Oliver is ready", "You can now open .exe and .msi files from your file manager.")
	return nil
}

// dialogConfirm asks a yes/no question on the desktop. With no dialog tool
// available it declines, since silently changing the user's file associations
// without asking would be worse than doing nothing.
func dialogConfirm(title, body string) bool {
	if bin, err := exec.LookPath("zenity"); err == nil {
		return exec.Command(bin, "--question", "--title", title, "--text", body).Run() == nil
	}
	if bin, err := exec.LookPath("kdialog"); err == nil {
		return exec.Command(bin, "--yesno", body, "--title", title).Run() == nil
	}
	return false
}

func dialogInfo(title, body string) {
	if bin, err := exec.LookPath("zenity"); err == nil {
		_ = exec.Command(bin, "--info", "--title", title, "--text", body).Run()
		return
	}
	if bin, err := exec.LookPath("kdialog"); err == nil {
		_ = exec.Command(bin, "--msgbox", body, "--title", title).Run()
		return
	}
	fmt.Fprintf(os.Stderr, "%s\n%s\n", title, body)
}

func dialogError(title, body string) {
	if bin, err := exec.LookPath("zenity"); err == nil {
		_ = exec.Command(bin, "--error", "--title", title, "--text", body).Run()
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, body)
}

// selfPath returns the path users should launch. Inside an AppImage the
// running binary lives in a temporary mount that disappears, so the AppImage
// file itself is the only durable path.
func selfPath() string {
	if p := os.Getenv("APPIMAGE"); p != "" {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
		return p
	}
	exe, err := os.Executable()
	if err != nil {
		return "oliver"
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

func desktopFilePath() string {
	d := dataHome()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "applications", "oliver.desktop")
}

func iconPath() string {
	d := dataHome()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "icons", "hicolor", "scalable", "apps", "oliver.svg")
}

func (a *app) desktopInstall(setDefault bool) error {
	launcher := selfPath()
	path := desktopFilePath()
	if path == "" {
		return fmt.Errorf("cannot determine your home directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(iconPath()), 0o755); err == nil {
		if err := os.WriteFile(iconPath(), []byte(iconSVG), 0o644); err != nil {
			a.ui.Warn("could not install the icon: %v", err)
		}
	}

	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Oliver
GenericName=Windows Program Runner
Comment=Run Windows programs with the best runtime installed
Exec=%s run %%f
TryExec=%s
Icon=oliver
Terminal=false
Categories=System;Emulator;Utility;
MimeType=%s;
Keywords=wine;proton;windows;exe;msi;
NoDisplay=false
`, quoteExec(launcher), quoteExec(launcher), strings.Join(exeMimeTypes, ";"))

	if err := os.WriteFile(path, []byte(entry), 0o644); err != nil {
		return err
	}
	a.ui.Info("installed %s", path)

	// Refresh the desktop's MIME cache; harmless when the tool is absent.
	if _, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command("update-desktop-database", filepath.Dir(path)).Run()
	}

	if setDefault {
		if _, err := exec.LookPath("xdg-mime"); err != nil {
			a.ui.Warn("xdg-mime is not installed, so the default handler was not changed")
		} else {
			for _, mime := range exeMimeTypes {
				if err := exec.Command("xdg-mime", "default", "oliver.desktop", mime).Run(); err != nil {
					a.ui.Warn("could not set the default handler for %s: %v", mime, err)
				}
			}
			a.ui.Info("Oliver is now the default handler for Windows programs")
		}
	} else {
		a.ui.Info("%s", a.ui.Dim("run with -default to also make Oliver the default for .exe files"))
	}
	return nil
}

func (a *app) desktopUninstall() error {
	var removed int
	for _, p := range []string{desktopFilePath(), iconPath()} {
		if p == "" {
			continue
		}
		if err := os.Remove(p); err == nil {
			removed++
			a.ui.Info("removed %s", p)
		} else if !os.IsNotExist(err) {
			a.ui.Warn("could not remove %s: %v", p, err)
		}
	}
	if removed == 0 {
		a.ui.Info("nothing to remove")
	}
	if _, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command("update-desktop-database", filepath.Join(dataHome(), "applications")).Run()
	}
	return nil
}

// quoteExec escapes a path for a .desktop Exec line.
func quoteExec(p string) string {
	if !strings.ContainsAny(p, ` "'\`+"\t") {
		return p
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(p)
	return `"` + escaped + `"`
}

// iconSVG is the application icon, kept inline so the binary stays a single
// self-contained file.
const iconSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="128" height="128">
  <defs>
    <linearGradient id="g" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#4a90d9"/>
      <stop offset="1" stop-color="#2a5d8f"/>
    </linearGradient>
  </defs>
  <rect x="8" y="8" width="112" height="112" rx="24" fill="url(#g)"/>
  <path d="M34 40h60M34 64h60M34 88h38" stroke="#fff" stroke-width="8" stroke-linecap="round" opacity=".35"/>
  <path d="M48 34l32 30-32 30z" fill="#fff"/>
</svg>
`
