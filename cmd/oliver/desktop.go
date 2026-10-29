package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/compat"
	"github.com/robertpjenkins-bot/oliver/internal/pe"
)

// exeMimeTypes are the MIME types desktops use for Windows programs. Different
// distributions and file managers disagree, so all of them are claimed.
var exeMimeTypes = []string{
	"application/x-ms-dos-executable",
	"application/x-msdownload",
	"application/vnd.microsoft.portable-executable",
	"application/x-msi",
}

// writerMimeTypes are the MIME types desktops give disk images. The shared MIME
// database has renamed them over the years, so old and new names are claimed.
var writerMimeTypes = []string{
	"application/x-cd-image",
	"application/x-iso9660-image",
	"application/vnd.efi.iso",
	"application/x-raw-disk-image",
	"application/vnd.efi.img",
}

const desktopSynopsis = `oliver desktop [install|uninstall|status]

Registers Oliver with your desktop so .exe and .msi files can be opened by
double-clicking them, and so Oliver appears in the "Open With" menu.

It also adds a Disk Image Writer to the application menu: drop an .iso or
.img onto it, or open one with it, to write it to a drive or partition.

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
	case "clean":
		return a.desktopClean()
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
		if _, err := os.Stat(writerFilePath()); err == nil {
			a.ui.Printf("Disk Image Writer at %s", writerFilePath())
		}
		a.ui.Printf("%s", a.ui.Dim("launcher: "+selfPath()))
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q (want install, uninstall, status or clean)", sub)
	}
}

// cmdDesktopWelcome runs when Oliver is opened with no arguments from a
// desktop, where there is no terminal to print help into.
//
// Someone who double-clicks expects to be able to do something. Telling them
// to open a terminal is a dead end, particularly on a machine whose whole
// point is that you do not need one.
func (a *app) cmdDesktopWelcome(args []string) error {
	kit, ok := findDialogKit()
	if !ok {
		fmt.Fprint(os.Stdout, usage)
		return nil
	}

	registered := false
	if p := desktopFilePath(); p != "" {
		if _, err := os.Stat(p); err == nil {
			registered = true
		}
	}

	choices := []menuChoice{
		{ID: "run", Label: "Run a Windows program", Note: "choose an .exe or .msi to launch"},
		{ID: "pack", Label: "Package a program as an AppImage", Note: "make it launchable like any other app"},
		{ID: "write", Label: "Write a disk image to a drive", Note: "put an .iso or .img onto a USB stick or a partition"},
		{ID: "doctor", Label: "Check this system", Note: "what is installed and what is missing"},
		{ID: "log", Label: "Show the last run", Note: "why something failed to start"},
		{ID: "clean", Label: "Fix broken menu entries", Note: "remove launcher icons that no longer work"},
		{ID: "report", Label: "Save a diagnostic report", Note: "one file describing everything, saved to your desktop"},
	}
	if !registered {
		choices = append([]menuChoice{
			{ID: "setup", Label: "Set up double-click for .exe files", Note: "recommended"},
		}, choices...)
	}

	choice, ok := kit.menu("Oliver", "What would you like to do?", choices)
	if !ok || choice == "" {
		return nil
	}

	switch choice {
	case "setup":
		if err := a.desktopInstall(true); err != nil {
			kit.errorBox("Setup failed", fmtErr(err))
			return err
		}
		kit.info("Oliver is ready", "You can now open .exe and .msi files from your file manager.")
	case "run":
		return a.guiRun(kit)
	case "pack":
		return a.guiPack(kit)
	case "write":
		return a.guiWrite(kit, "", true)
	case "doctor":
		kit.showText("Oliver - system check", a.captureText(func() { _ = a.cmdDoctor(nil) }))
	case "clean":
		// The prompt inside desktopClean needs a terminal, so decide here and
		// report what happened in a dialog.
		report := a.captureText(func() { _ = a.desktopCleanForce(kit) })
		kit.showText("Oliver - menu entries", report)
	case "report":
		var err error
		msg := a.captureText(func() { err = a.cmdReport(nil) })
		if err != nil {
			kit.errorBox("Could not save the report", fmtErr(err))
		} else {
			kit.info("Report saved", strings.TrimSpace(msg)+"\n\nYou can attach this file when asking for help.")
		}
	case "log":
		body := a.captureText(func() { _ = a.cmdLog(nil) })
		if strings.TrimSpace(body) == "" {
			body = "No runs have been recorded yet."
		}
		kit.showText("Oliver - last run", body)
	}
	return nil
}

// guiRun picks a program and launches it.
func (a *app) guiRun(kit dialogKit) error {
	path, ok := kit.openFile("Choose a Windows program", "Windows programs", windowsProgramPatterns)
	if !ok || path == "" {
		return nil
	}

	// Say up front when a program is known not to work here, rather than
	// letting it fail in a way that looks like Oliver's fault.
	if info, err := pe.Inspect(path); err == nil {
		if caveats := compat.Check(info); compat.Worst(caveats) == compat.Blocking {
			c := caveats[0]
			body := c.Title
			if c.Symptom != "" {
				body += "\n\n" + c.Symptom
			}
			if c.Workaround != "" {
				body += "\n\n" + c.Workaround
			}
			// A disk writer is a job Oliver does itself, so offer that first.
			if c.BuiltIn == "write" {
				choice, ok := kit.menu("This program cannot do its job here", body, []menuChoice{
					{ID: "write", Label: "Write the image with Oliver instead", Note: "choose the ISO, then the drive or partition"},
					{ID: "run", Label: "Run it anyway", Note: "it will open, but find no drives"},
				})
				switch {
				case !ok || choice == "":
					return nil
				case choice == "write":
					return a.guiWrite(kit, "", true)
				}
			} else if !kit.confirm("This program cannot do its job here", body+"\n\nTry it anyway?") {
				return nil
			}
		}
	}

	// A failed launch raises its own dialog, so nothing more is needed here.
	_ = a.cmdRun([]string{path})
	return nil
}

// guiPack wraps a program into an AppImage.
func (a *app) guiPack(kit dialogKit) error {
	path, ok := kit.openFile("Choose a Windows program to package", "Windows programs", windowsProgramPatterns)
	if !ok || path == "" {
		return nil
	}

	info, err := pe.Inspect(path)
	if err != nil {
		kit.errorBox("Cannot package this file", fmtErr(err))
		return nil
	}

	name := info.VersionInfo.ProductName
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	name, ok = kit.entry("Package as AppImage", "Name for the application:", name)
	if !ok || name == "" {
		return nil
	}

	args := []string{path, "-name", name}

	// An installer packaged as-is would reinstall itself on every launch, so
	// ask what it installs.
	if info.IsInstaller {
		target, ok := kit.entry("Package as AppImage",
			"This is an installer. Which program does it install? "+
				"(the .exe name, for example MinecraftLauncher.exe)", "")
		if !ok {
			return nil
		}
		if target != "" {
			args = append(args, "-launches", target)
		}
	}

	home, _ := os.UserHomeDir()
	out := filepath.Join(home, "Desktop", name+"-x86_64.AppImage")
	if st, err := os.Stat(filepath.Dir(out)); err != nil || !st.IsDir() {
		out = filepath.Join(home, name+"-x86_64.AppImage")
	}
	args = append(args, "-o", out)

	var packErr error
	text := a.captureText(func() { packErr = a.cmdPack(args) })
	if packErr != nil {
		kit.errorBox("Packaging failed", fmtErr(packErr)+"\n\n"+text)
		return nil
	}
	// Offer to add it to the menu now. Leaving this to the user means they
	// meet the bundled desktop entry first, which cannot work on its own.
	if kit.confirm("Packaged", name+" was saved to:\n"+out+"\n\nAdd it to your application menu?") {
		if err := installMenuEntry(name, slugFor(name), out, []byte(iconSVG)); err != nil {
			kit.errorBox("Could not add it to the menu", fmtErr(err))
		} else {
			kit.info("Added", name+" is now in your application menu.")
		}
	} else {
		kit.info("Packaged", name+" was saved to:\n"+out)
	}
	return nil
}

// slugFor derives the identifier used for desktop files and icons. A name
// carrying a Windows extension is trimmed, so "Thing.exe" does not become an
// icon called "thing.exe".
func slugFor(name string) string {
	for _, ext := range []string{".exe", ".msi", ".bat", ".cmd"} {
		if strings.HasSuffix(strings.ToLower(name), ext) {
			name = name[:len(name)-len(ext)]
			break
		}
	}
	s := strings.ToLower(strings.Trim(packableName.ReplaceAllString(name, "-"), "-"))
	if s == "" {
		return "windows-app"
	}
	return s
}

// captureText runs a command with its output redirected into a string, so a
// terminal-shaped report can be shown in a dialog.
func (a *app) captureText(fn func()) string {
	var buf strings.Builder
	prevOut, prevErr, prevColor := a.ui.Out, a.ui.Err, a.ui.Color
	a.ui.Out, a.ui.Err, a.ui.Color = &buf, &buf, false
	defer func() { a.ui.Out, a.ui.Err, a.ui.Color = prevOut, prevErr, prevColor }()
	fn()
	return buf.String()
}

// desktopCleanForce runs the cleanup from the desktop menu, asking through a
// dialog rather than a terminal prompt.
func (a *app) desktopCleanForce(kit dialogKit) error {
	broken := findBrokenEntries()
	if len(broken) == 0 {
		a.ui.Printf("Every menu entry points at something that exists.")
		return nil
	}
	var ours []brokenEntry
	for _, b := range broken {
		a.ui.Printf("%s\n  runs %s\n  problem: %s", b.Name, b.Command, b.Reason)
		if b.Ours {
			ours = append(ours, b)
		} else {
			a.ui.Printf("  left alone: not created by Oliver")
		}
		a.ui.Printf("")
	}
	if len(ours) == 0 {
		a.ui.Printf("None of these were created by Oliver, so nothing was removed.")
		return nil
	}
	if !kit.confirm("Fix broken menu entries",
		fmt.Sprintf("%d launcher entries no longer work. Remove them?", len(ours))) {
		a.ui.Printf("Nothing was removed.")
		return nil
	}
	for _, b := range ours {
		if err := os.Remove(b.Path); err != nil {
			a.ui.Printf("could not remove %s: %v", b.Path, err)
			continue
		}
		a.ui.Printf("removed %s", filepath.Base(b.Path))
	}
	if _, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command("update-desktop-database", filepath.Join(dataHome(), "applications")).Run()
	}
	return nil
}

// entryDirs are the places a launcher icon can live.
//
// The applications directory is the menu, but a desktop shortcut is a
// .desktop file sitting in the desktop folder itself, and on KDE that is
// where an icon the user made by hand ends up. Cleanup that skips it leaves
// exactly the icon they are clicking.
func entryDirs() []string {
	dirs := []string{filepath.Join(dataHome(), "applications")}
	if d := os.Getenv("XDG_DESKTOP_DIR"); d != "" {
		dirs = append(dirs, d)
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Desktop"))
	}

	seen := map[string]bool{}
	var out []string
	for _, d := range dirs {
		real := d
		if r, err := filepath.EvalSymlinks(d); err == nil {
			real = r
		}
		if seen[real] {
			continue
		}
		seen[real] = true
		out = append(out, d)
	}
	return out
}

// findBrokenEntries lists launcher entries whose command cannot be run.
func findBrokenEntries() []brokenEntry {
	var broken []brokenEntry
	for _, dir := range entryDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
				continue
			}
			if b, bad := inspectEntry(filepath.Join(dir, e.Name())); bad {
				broken = append(broken, b)
			}
		}
	}
	return broken
}

// brokenEntry is a desktop menu entry whose command cannot be run.
type brokenEntry struct {
	Path    string
	Name    string
	Command string
	Reason  string
	// Ours reports an entry Oliver created, which is safe to remove.
	Ours bool
}

// desktopClean finds menu entries that cannot launch anything.
//
// An entry left behind by an earlier packaging attempt keeps its icon in the
// launcher and keeps failing, however many times the application is repacked:
// each new package writes a new entry rather than repairing the old one.
func (a *app) desktopClean() error {
	broken := findBrokenEntries()

	u := a.ui
	if len(broken) == 0 {
		u.Printf("%s", u.Green("Every menu entry points at something that exists."))
		return nil
	}

	u.Printf("%s", u.Bold("Launcher entries that cannot start anything:"))
	u.Printf("")
	var removable int
	for _, b := range broken {
		u.Printf("  %s", u.Bold(b.Name))
		u.Printf("    %s %s", u.Dim("runs   "), b.Command)
		u.Printf("    %s %s", u.Dim("problem"), u.Red(b.Reason))
		u.Printf("    %s %s", u.Dim("file   "), b.Path)
		if b.Ours {
			removable++
		} else {
			u.Printf("    %s", u.Dim("not created by Oliver, so left alone"))
		}
		u.Printf("")
	}

	if removable == 0 {
		u.Printf("%s", u.Dim("None of these were created by Oliver, so nothing was removed."))
		return nil
	}
	if !u.Confirm(fmt.Sprintf("Remove %d entry/entries Oliver created?", removable), true) {
		u.Info("nothing removed")
		return nil
	}
	for _, b := range broken {
		if !b.Ours {
			continue
		}
		if err := os.Remove(b.Path); err != nil {
			u.Warn("could not remove %s: %v", b.Path, err)
			continue
		}
		u.Info("removed %s", filepath.Base(b.Path))
	}
	if _, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command("update-desktop-database", filepath.Join(dataHome(), "applications")).Run()
	}
	return nil
}

// inspectEntry reports whether a desktop entry's command can actually be run.
func inspectEntry(path string) (brokenEntry, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return brokenEntry{}, false
	}
	text := string(data)

	b := brokenEntry{Path: path, Name: filepath.Base(path)}
	// Oliver stamps what it writes, so its own entries can be told apart from
	// everything else in the menu.
	b.Ours = strings.Contains(text, "packaged with Oliver") ||
		strings.Contains(text, "Comment=Run Windows programs with the best runtime") ||
		strings.Contains(text, "\nX-Oliver=")

	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "Name=") && b.Name == filepath.Base(path):
			b.Name = strings.TrimPrefix(line, "Name=")
		case strings.HasPrefix(line, "Exec="):
			b.Command = strings.TrimSpace(strings.TrimPrefix(line, "Exec="))
		}
	}
	if b.Command == "" {
		return brokenEntry{}, false
	}

	program := execProgram(b.Command)
	switch {
	case program == "":
		return brokenEntry{}, false
	case program == "AppRun":
		// The AppDir convention, which only resolves inside an AppImage.
		b.Reason = "points at AppRun, which exists only inside an AppImage"
		b.Ours = true
		return b, true
	case strings.ContainsRune(program, '/'):
		if _, err := os.Stat(program); err != nil {
			b.Reason = "the file it runs no longer exists"
			return b, true
		}
	default:
		if _, err := exec.LookPath(program); err != nil {
			b.Reason = "no program called " + program + " is installed"
			return b, true
		}
	}
	return brokenEntry{}, false
}

// execProgram extracts the program from a desktop Exec line, dropping the
// quoting and the field codes the specification allows.
func execProgram(exec string) string {
	exec = strings.TrimSpace(exec)
	if exec == "" {
		return ""
	}
	if exec[0] == '"' || exec[0] == '\'' {
		quote := exec[0]
		if end := strings.IndexByte(exec[1:], quote); end >= 0 {
			return exec[1 : end+1]
		}
	}
	first := strings.Fields(exec)[0]
	// env-style prefixes name no program of their own.
	if first == "env" || strings.Contains(first, "=") {
		for _, f := range strings.Fields(exec)[1:] {
			if !strings.Contains(f, "=") {
				return f
			}
		}
		return ""
	}
	return first
}

// installMenuEntry adds an application to the desktop menu.
//
// The entry records an absolute path, because nothing inside an AppImage
// survives it closing: an entry pointing into the mount is the cause of
// "could not find" errors when it is later clicked.
func installMenuEntry(name, slug, execPath string, icon []byte) error {
	apps := filepath.Join(dataHome(), "applications")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		return err
	}

	if len(icon) > 0 {
		iconDir := filepath.Join(dataHome(), "icons", "hicolor", "scalable", "apps")
		if err := os.MkdirAll(iconDir, 0o755); err == nil {
			_ = os.WriteFile(filepath.Join(iconDir, slug+".svg"), icon, 0o644)
		}
	}

	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=Windows program packaged with Oliver
Exec=%s %%U
TryExec=%s
Icon=%s
Terminal=false
Categories=Utility;
`, name, quoteExec(execPath), execPath, slug)

	path := filepath.Join(apps, slug+".desktop")
	if err := os.WriteFile(path, []byte(entry), 0o644); err != nil {
		return err
	}
	if _, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command("update-desktop-database", apps).Run()
	}
	return nil
}

// dialogError reports a failure on the desktop, for callers outside the
// welcome flow.
func dialogError(title, body string) {
	if kit, ok := findDialogKit(); ok {
		kit.errorBox(title, body)
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

// writerFilePath is the menu entry for the disk image writer.
func writerFilePath() string {
	d := dataHome()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "applications", "oliver-write.desktop")
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

	// The writer gets an entry of its own: "Open With" lists it for disk
	// images, and an image dropped onto it in a dock or on the desktop is
	// handed to it. It is not made the default, since double-clicking an ISO
	// usually means looking inside it.
	writer := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Disk Image Writer
GenericName=USB Image Writer
Comment=Write an ISO or disk image to a drive or a partition
Exec=%s write --gui %%f
TryExec=%s
Icon=drive-removable-media
Terminal=false
Categories=System;Utility;
MimeType=%s;
Keywords=iso;img;usb;flash;burn;partition;dd;
X-Oliver=writer
`, quoteExec(launcher), quoteExec(launcher), strings.Join(writerMimeTypes, ";"))
	if err := os.WriteFile(writerFilePath(), []byte(writer), 0o644); err != nil {
		return err
	}
	a.ui.Info("installed %s", writerFilePath())

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
	for _, p := range []string{desktopFilePath(), writerFilePath(), iconPath()} {
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
