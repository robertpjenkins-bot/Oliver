package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/config"
	"github.com/robertpjenkins-bot/oliver/internal/petest"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

// system is a synthetic Linux install with compatibility runtimes on it,
// used to drive the command-line interface end to end.
type system struct {
	t    *testing.T
	root string
	home string
	out  *bytes.Buffer
}

func newSystem(t *testing.T) *system {
	t.Helper()
	root := t.TempDir()
	s := &system{t: t, root: root, home: filepath.Join(root, "home", "user"), out: &bytes.Buffer{}}

	s.mkdir("home", "user")
	s.mkdir("usr", "bin")

	// Wine on PATH, 64-bit only, so 32-bit programs must go elsewhere.
	s.script("usr", "bin", "wine")
	s.mkdir("usr", "lib", "wine", "x86_64-windows")
	s.mkdir("usr", "lib", "wine", "x86_64-unix")

	// A Lutris runner that does support 32-bit.
	lutris := []string{"home", "user", ".local", "share", "lutris", "runners", "wine", "wine-9.2-staging"}
	s.script(append(append([]string{}, lutris...), "bin", "wine")...)
	s.mkdir(append(append([]string{}, lutris...), "lib", "wine", "i386-windows")...)
	s.mkdir(append(append([]string{}, lutris...), "lib", "wine", "x86_64-windows")...)

	// A Proton-GE build, plus the launcher that makes it usable.
	ge := []string{"home", "user", ".local", "share", "Steam", "compatibilitytools.d", "GE-Proton9-20"}
	s.script(append(append([]string{}, ge...), "proton")...)
	s.write("1710000000 GE-Proton9-20", append(append([]string{}, ge...), "version")...)
	s.mkdir(append(append([]string{}, ge...), "files", "lib", "wine", "i386-unix")...)
	s.mkdir(append(append([]string{}, ge...), "files", "lib", "wine", "x86_64-unix")...)
	s.script("usr", "bin", "umu-run")

	t.Setenv("HOME", s.home)
	t.Setenv("PATH", s.path("usr", "bin"))
	t.Setenv("XDG_DATA_HOME", s.path("home", "user", ".local", "share"))
	t.Setenv("XDG_CONFIG_HOME", s.path("home", "user", ".config"))
	t.Setenv("OLIVER_CONFIG", s.path("home", "user", ".config", "oliver", "config.json"))
	return s
}

func (s *system) path(parts ...string) string {
	return filepath.Join(append([]string{s.root}, parts...)...)
}

func (s *system) mkdir(parts ...string) string {
	s.t.Helper()
	p := s.path(parts...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		s.t.Fatal(err)
	}
	return p
}

func (s *system) write(content string, parts ...string) string {
	s.t.Helper()
	p := s.path(parts...)
	s.t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		s.t.Fatal(err)
	}
	return p
}

func (s *system) script(parts ...string) string {
	s.t.Helper()
	p := s.write("#!/bin/sh\nexit 0\n", parts...)
	if err := os.Chmod(p, 0o755); err != nil {
		s.t.Fatal(err)
	}
	return p
}

// program writes a Windows executable into the synthetic system.
func (s *system) program(name string, o petest.Options, dir ...string) string {
	s.t.Helper()
	parts := append(append([]string{}, dir...), name)
	p := s.path(parts...)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		s.t.Fatal(err)
	}
	if err := petest.WriteFile(p, o); err != nil {
		s.t.Fatal(err)
	}
	return p
}

// run executes a command line and returns everything written to stdout.
func (s *system) run(args ...string) (string, error) {
	s.t.Helper()
	s.out.Reset()
	// Capture output by swapping the UI's writers.
	orig := newUI
	newUI = func() *ui.UI {
		return &ui.UI{Out: s.out, Err: &bytes.Buffer{}}
	}
	defer func() { newUI = orig }()
	err := run(args)
	return s.out.String(), err
}

func TestCLIListsDiscoveredRuntimes(t *testing.T) {
	s := newSystem(t)
	out, err := s.run("backends")
	if err != nil {
		t.Fatalf("backends: %v", err)
	}
	for _, want := range []string{"GE-Proton9-20", "wine-9.2-staging", "proton", "wine"} {
		if !strings.Contains(out, want) {
			t.Errorf("backends output missing %q:\n%s", want, out)
		}
	}
}

func TestCLIChoosesProtonForGamesAndWineForApplications(t *testing.T) {
	s := newSystem(t)

	game := s.program("Game.exe", petest.Options{Machine: petest.MachineAMD64, PE32Plus: true},
		"home", "user", "games", "Cool")
	// Storefront SDK next to the binary is what marks it as a game.
	s.write("x", "home", "user", "games", "Cool", "steam_api64.dll")

	out, err := s.run("run", "-n", game)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "GE-Proton9-20") {
		t.Errorf("expected a game to use Proton:\n%s", out)
	}
	if !strings.Contains(out, "umu-run") {
		t.Errorf("expected the launch to go through umu-run:\n%s", out)
	}

	app := s.program("Tool.exe", petest.Options{Machine: petest.MachineAMD64, PE32Plus: true, Subsystem: petest.SubsystemConsole},
		"home", "user", "apps", "Tool")
	out, err = s.run("run", "-n", app)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if strings.Contains(out, "GE-Proton9-20") {
		t.Errorf("an ordinary application should not use Proton:\n%s", out)
	}
}

func TestCLIRoutesThirtyTwoBitProgramsToACapableRuntime(t *testing.T) {
	s := newSystem(t)
	old := s.program("Legacy.exe", petest.Options{Machine: petest.MachineI386}, "home", "user", "apps", "Legacy")

	out, err := s.run("run", "-n", old)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	// The system Wine is newer but 64-bit only, so it must be passed over.
	if !strings.Contains(out, "wine-9.2-staging") {
		t.Errorf("expected the 32-bit-capable runtime to be chosen:\n%s", out)
	}
	if !strings.Contains(out, "WINEARCH=win32") {
		t.Errorf("expected a 32-bit prefix to be created:\n%s", out)
	}
}

func TestCLIExplainsItsChoice(t *testing.T) {
	s := newSystem(t)
	old := s.program("Legacy.exe", petest.Options{Machine: petest.MachineI386}, "home", "user", "apps", "Legacy")

	out, err := s.run("why", old)
	if err != nil {
		t.Fatalf("why: %v\n%s", err, out)
	}
	if !strings.Contains(out, "no 32-bit support") {
		t.Errorf("expected the incompatible runtime to be explained:\n%s", out)
	}
	if !strings.Contains(out, "cannot run this program") {
		t.Errorf("expected a blocked runtime to be marked:\n%s", out)
	}
}

func TestCLIConfigRuleOverridesTheAutomaticChoice(t *testing.T) {
	s := newSystem(t)
	game := s.program("Game.exe", petest.Options{Machine: petest.MachineAMD64, PE32Plus: true},
		"home", "user", "games", "Cool")
	s.write("x", "home", "user", "games", "Cool", "steam_api64.dll")

	s.write(`{"rules":[{"match":"**/games/Cool/*.exe","backend":"wine-9.2-staging"}]}`,
		"home", "user", ".config", "oliver", "config.json")

	out, err := s.run("run", "-n", game)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "wine-9.2-staging") {
		t.Errorf("configuration rule was not applied:\n%s", out)
	}
}

func TestCLIRejectsAnImpossibleBackendChoice(t *testing.T) {
	s := newSystem(t)
	old := s.program("Legacy.exe", petest.Options{Machine: petest.MachineI386}, "home", "user", "apps", "Legacy")

	// Forcing the 64-bit-only runtime onto a 32-bit program must fail loudly
	// rather than silently substituting a different runtime.
	_, err := s.run("run", "-b", "system-wine", old)
	if err == nil {
		t.Fatal("expected an error when the requested runtime cannot run the program")
	}
	if !strings.Contains(err.Error(), "32-bit") {
		t.Errorf("error should explain the problem, got: %v", err)
	}
}

func TestCLIJSONOutputIsMachineReadable(t *testing.T) {
	s := newSystem(t)
	out, err := s.run("backends", "-json")
	if err != nil {
		t.Fatalf("backends -json: %v", err)
	}
	var parsed []map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if len(parsed) != 3 {
		t.Errorf("expected 3 runtimes, got %d", len(parsed))
	}

	prog := s.program("App.exe", petest.Options{Machine: petest.MachineAMD64, PE32Plus: true}, "home", "user", "apps", "A")
	out, err = s.run("info", "-json", prog)
	if err != nil {
		t.Fatalf("info -json: %v", err)
	}
	var info map[string]any
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatalf("info output is not valid JSON: %v\n%s", err, out)
	}
	if info["arch"] != "x86_64" {
		t.Errorf("arch = %v, want x86_64", info["arch"])
	}
}

func TestCLIReportsAMissingProgramClearly(t *testing.T) {
	s := newSystem(t)
	if _, err := s.run("run", "/nonexistent/program.exe"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if _, err := s.run("run"); err == nil {
		t.Fatal("expected an error when no program is given")
	}
	if _, err := s.run("nonsense-command"); err == nil {
		t.Fatal("expected an error for an unknown command")
	}
}

func TestCLIRejectsABrokenConfigurationFile(t *testing.T) {
	s := newSystem(t)
	s.write("{ this is not json", "home", "user", ".config", "oliver", "config.json")
	_, err := s.run("backends")
	if err == nil {
		t.Fatal("expected a broken configuration file to be reported")
	}
}

func TestProgramPathIsTreatedAsARunRequest(t *testing.T) {
	// This is what a file manager does: hand the executable over as argv[1].
	for _, arg := range []string{"game.exe", "/x/y/thing.MSI", "/x/setup.bat"} {
		if !looksLikeProgram(arg) {
			t.Errorf("looksLikeProgram(%q) = false, want true", arg)
		}
	}
	for _, arg := range []string{"run", "backends", "-json", "doctor"} {
		if looksLikeProgram(arg) {
			t.Errorf("looksLikeProgram(%q) = true, want false", arg)
		}
	}
}

func TestDoctorReportsSystemState(t *testing.T) {
	s := newSystem(t)
	out, err := s.run("doctor")
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	for _, want := range []string{"compatibility runtime", "32-bit programs", "Proton support"} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output missing %q:\n%s", want, out)
		}
	}
}

func TestDesktopInstallWritesAValidEntry(t *testing.T) {
	s := newSystem(t)
	if _, err := s.run("desktop", "install"); err != nil {
		t.Fatalf("desktop install: %v", err)
	}

	data, err := os.ReadFile(desktopFilePath())
	if err != nil {
		t.Fatalf("desktop entry not written: %v", err)
	}
	entry := string(data)
	for _, want := range []string{
		"[Desktop Entry]",
		"application/x-ms-dos-executable",
		"application/x-msi",
		"Exec=",
		"%f", // the file the user opened must be passed through
	} {
		if !strings.Contains(entry, want) {
			t.Errorf("desktop entry missing %q:\n%s", want, entry)
		}
	}

	if _, err := os.Stat(iconPath()); err != nil {
		t.Errorf("icon not installed: %v", err)
	}

	if _, err := s.run("desktop", "uninstall"); err != nil {
		t.Fatalf("desktop uninstall: %v", err)
	}
	if _, err := os.Stat(desktopFilePath()); !os.IsNotExist(err) {
		t.Error("desktop entry survived uninstall")
	}
}

func TestQuoteExecEscapesAwkwardPaths(t *testing.T) {
	if got := quoteExec("/opt/oliver"); got != "/opt/oliver" {
		t.Errorf("quoteExec = %q, want unquoted", got)
	}
	got := quoteExec("/home/u/My Apps/Oliver.AppImage")
	if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
		t.Errorf("a path with spaces must be quoted, got %s", got)
	}
}

func TestPackBuildsARunnableAppDir(t *testing.T) {
	s := newSystem(t)
	prog := s.program("DemoApp.exe", petest.Options{
		Machine: petest.MachineAMD64, PE32Plus: true,
		VersionStrings: map[string]string{"ProductName": "Demo App"},
	}, "home", "user", "apps")

	outDir := t.TempDir()
	out := filepath.Join(outDir, "demo.AppImage")
	// Without appimagetool the AppDir is still produced, which is the part
	// worth testing: packaging it is one external command.
	t.Setenv("APPIMAGETOOL", "")
	t.Setenv("PATH", filepath.Join(outDir, "nothing"))

	if _, err := s.run("pack", prog, "-o", out, "-appdir"); err != nil {
		t.Fatalf("pack: %v", err)
	}

	appDir := filepath.Join(outDir, "demo.AppDir")
	for _, want := range []string{
		"AppRun",
		"usr/bin/oliver",
		"usr/share/oliver/payload/DemoApp.exe",
		"demo-app.desktop",
	} {
		if _, err := os.Stat(filepath.Join(appDir, want)); err != nil {
			t.Errorf("packed AppDir missing %s: %v", want, err)
		}
	}

	// The name comes from the version resource, not the file name.
	desktop, err := os.ReadFile(filepath.Join(appDir, "demo-app.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(desktop), "Name=Demo App") {
		t.Errorf("desktop entry should use the product name:\n%s", desktop)
	}
	// Exec must name something that resolves. "AppRun" is a file inside the
	// AppDir, so an entry using it fails with "could not find AppRun" as soon
	// as it is used outside the mount.
	if strings.Contains(string(desktop), "Exec=AppRun") {
		t.Errorf("Exec must not point at AppRun:\n%s", desktop)
	}
	if !strings.Contains(string(desktop), "Exec=demo-app") {
		t.Errorf("Exec should name the bundled launcher:\n%s", desktop)
	}
	// ...and that launcher has to exist on the AppImage's PATH.
	launcher := filepath.Join(appDir, "usr", "bin", "demo-app")
	st, err := os.Stat(launcher)
	if err != nil {
		t.Fatalf("usr/bin/demo-app missing, so Exec would not resolve: %v", err)
	}
	if st.Mode().Perm()&0o111 == 0 {
		t.Error("the bundled launcher must be executable")
	}

	apprun, err := os.ReadFile(filepath.Join(appDir, "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(apprun)
	// An AppImage mounts at a new path every launch, so a path-derived prefix
	// name would rebuild the Windows environment on every run.
	if !strings.Contains(script, "--prefix demo-app") {
		t.Errorf("AppRun must pin the prefix name:\n%s", script)
	}
	// It is also read-only, so the program needs a writable copy of itself.
	if !strings.Contains(script, "oliver/packed/demo-app") {
		t.Errorf("AppRun must copy the payload somewhere writable:\n%s", script)
	}

	info, err := os.Stat(filepath.Join(appDir, "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Error("AppRun must be executable")
	}
}

func TestPackRejectsSomethingThatIsNotAProgram(t *testing.T) {
	s := newSystem(t)
	notExe := s.write("hello", "home", "user", "notes.txt")
	if _, err := s.run("pack", notExe); err == nil {
		t.Fatal("expected pack to reject a non-executable")
	}
	if _, err := s.run("pack"); err == nil {
		t.Fatal("expected pack to require a program")
	}
}

func TestFlagsAreAcceptedAfterPositionalArguments(t *testing.T) {
	// `oliver pack app.exe -o out` is the order people type; Go's flag package
	// would otherwise stop at the first positional and ignore -o entirely.
	s := newSystem(t)
	prog := s.program("Thing.exe", petest.Options{Machine: petest.MachineAMD64, PE32Plus: true},
		"home", "user", "apps")

	outDir := t.TempDir()
	out := filepath.Join(outDir, "chosen.AppImage")
	t.Setenv("APPIMAGETOOL", "")
	t.Setenv("PATH", filepath.Join(outDir, "nothing"))

	if _, err := s.run("pack", prog, "-o", out, "-appdir"); err != nil {
		t.Fatalf("pack: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "chosen.AppDir")); err != nil {
		t.Errorf("the -o path after the positional was ignored: %v", err)
	}
}

func TestPackedInstallerInstallsOnceThenLaunchesTheProgram(t *testing.T) {
	s := newSystem(t)
	// An MSI, as most Windows software is distributed.
	msi := s.path("home", "user", "Setup.msi")
	body := append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, make([]byte, 512)...)
	if err := os.WriteFile(msi, body, 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	t.Setenv("APPIMAGETOOL", "")
	t.Setenv("PATH", filepath.Join(outDir, "nothing"))
	out := filepath.Join(outDir, "app.AppImage")

	if _, err := s.run("pack", msi, "-launches", "TheApp.exe", "-name", "The App", "-o", out, "-appdir"); err != nil {
		t.Fatalf("pack: %v", err)
	}

	script, err := os.ReadFile(filepath.Join(outDir, "app.AppDir", "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(script)
	// The installer must run once, guarded by a marker.
	if !strings.Contains(got, ".installed") {
		t.Errorf("AppRun must not reinstall on every launch:\n%s", got)
	}
	if !strings.Contains(got, "Setup.msi") {
		t.Errorf("AppRun should run the installer:\n%s", got)
	}
	// Later launches must start the installed program, located in the prefix.
	if !strings.Contains(got, "which --prefix the-app TheApp.exe") {
		t.Errorf("AppRun should locate the installed program:\n%s", got)
	}
}

func TestPackWarnsWhenAnInstallerHasNoFollowUpProgram(t *testing.T) {
	s := newSystem(t)
	msi := s.path("home", "user", "Setup.msi")
	body := append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, make([]byte, 512)...)
	if err := os.WriteFile(msi, body, 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	t.Setenv("APPIMAGETOOL", "")
	t.Setenv("PATH", filepath.Join(outDir, "nothing"))

	// Packing an installer with no -launches still works, but the AppRun then
	// simply runs the installer, so the user is told.
	if _, err := s.run("pack", msi, "-o", filepath.Join(outDir, "app.AppImage"), "-appdir"); err != nil {
		t.Fatalf("pack: %v", err)
	}
	script, err := os.ReadFile(filepath.Join(outDir, "app.AppDir", "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(script), ".installed") {
		t.Error("without -launches the AppRun should not pretend to install once")
	}
}

func TestWhichFindsInstalledProgramsAndSkipsUninstallers(t *testing.T) {
	s := newSystem(t)
	prefixRoot := t.TempDir()
	t.Setenv("OLIVER_PREFIX_ROOT", prefixRoot)

	drive := filepath.Join(prefixRoot, "demo", "pfx", "drive_c", "Program Files", "Demo")
	if err := os.MkdirAll(drive, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"TheApp.exe", "unins000.exe"} {
		if err := petest.WriteFile(filepath.Join(drive, name), petest.Options{Machine: petest.MachineAMD64, PE32Plus: true}); err != nil {
			t.Fatal(err)
		}
	}
	// A stale copy somewhere less likely than Program Files.
	temp := filepath.Join(prefixRoot, "demo", "pfx", "drive_c", "windows", "temp")
	if err := os.MkdirAll(temp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := petest.WriteFile(filepath.Join(temp, "TheApp.exe"), petest.Options{Machine: petest.MachineAMD64, PE32Plus: true}); err != nil {
		t.Fatal(err)
	}

	out, err := s.run("which", "-p", "demo", "TheApp.exe")
	if err != nil {
		t.Fatalf("which: %v", err)
	}
	if !strings.Contains(out, filepath.Join("Program Files", "Demo", "TheApp.exe")) {
		t.Errorf("expected the Program Files copy to win:\n%s", out)
	}

	// The uninstaller shares the directory but is never the thing to launch.
	if out, _ := s.run("which", "-p", "demo", "unins000.exe"); strings.Contains(out, "unins000") {
		t.Errorf("uninstallers must not be returned:\n%s", out)
	}

	if _, err := s.run("which", "-p", "demo", "Absent.exe"); err == nil {
		t.Error("expected an error for a program that is not installed")
	}
	if _, err := s.run("which", "TheApp.exe"); err == nil {
		t.Error("expected an error when no prefix is given")
	}
}

func TestWelcomeFallsBackToUsageWithoutADialogTool(t *testing.T) {
	s := newSystem(t)
	// No zenity or kdialog on PATH: the welcome flow must still not fail.
	t.Setenv("PATH", s.path("usr", "bin"))
	if _, err := s.run("desktop-welcome"); err != nil {
		t.Errorf("desktop-welcome should degrade gracefully, got %v", err)
	}
}

func TestCaptureTextRedirectsCommandOutput(t *testing.T) {
	s := newSystem(t)
	a := &app{ui: newUI()}
	cfg, err := config.Load(s.path("home", "user", ".config", "oliver", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	a.cfg = cfg

	got := a.captureText(func() { _ = a.cmdDoctor(nil) })
	if !strings.Contains(got, "compatibility runtime") {
		t.Errorf("captured text missing the report:\n%s", got)
	}
	// Colour must be off, since the text goes into a dialog, not a terminal.
	if strings.Contains(got, "\x1b[") {
		t.Errorf("captured text should carry no escape codes:\n%q", got)
	}
}

func TestPackedLauncherCanRegisterItselfWithAnAbsolutePath(t *testing.T) {
	s := newSystem(t)
	prog := s.program("Thing.exe", petest.Options{Machine: petest.MachineAMD64, PE32Plus: true},
		"home", "user", "apps")

	outDir := t.TempDir()
	t.Setenv("APPIMAGETOOL", "")
	t.Setenv("PATH", filepath.Join(outDir, "nothing"))
	if _, err := s.run("pack", prog, "-name", "Thing", "-o", filepath.Join(outDir, "t.AppImage"), "-appdir"); err != nil {
		t.Fatalf("pack: %v", err)
	}

	script, err := os.ReadFile(filepath.Join(outDir, "t.AppDir", "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(script)
	if !strings.Contains(got, "--install") || !strings.Contains(got, "--uninstall") {
		t.Errorf("the packed launcher should be able to add itself to the menu:\n%s", got)
	}
	// The entry it writes must record where the AppImage is, not a path inside
	// the mount, which disappears when the AppImage closes.
	if !strings.Contains(got, "${APPIMAGE:-$(readlink -f \"${0}\")}") {
		t.Errorf("self-registration must use the AppImage's own path:\n%s", got)
	}
	if !strings.Contains(got, `export PATH="${HERE}/usr/bin:${PATH}"`) {
		t.Errorf("usr/bin must be on PATH so Exec resolves:\n%s", got)
	}
}

func TestSlugForDropsWindowsExtensions(t *testing.T) {
	// A name typed with its extension must not become an icon called
	// "thing.exe" or a desktop file named after a file type.
	cases := map[string]string{
		"Minecraft Launcher":    "minecraft-launcher",
		"MinecraftLauncher.exe": "minecraftlauncher",
		"Setup.MSI":             "setup",
		"Weird  Name!!":         "weird-name",
		"":                      "windows-app",
	}
	for in, want := range cases {
		if got := slugFor(in); got != want {
			t.Errorf("slugFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInstallMenuEntryRecordsAnAbsolutePath(t *testing.T) {
	s := newSystem(t)
	appImage := s.path("home", "user", "Apps", "Thing-x86_64.AppImage")
	if err := os.MkdirAll(filepath.Dir(appImage), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appImage, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := installMenuEntry("Thing", "thing", appImage, []byte("<svg/>")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dataHome(), "applications", "thing.desktop"))
	if err != nil {
		t.Fatalf("entry not written: %v", err)
	}
	entry := string(data)
	if !strings.Contains(entry, "Exec="+appImage) {
		t.Errorf("Exec must be the AppImage's absolute path:\n%s", entry)
	}
	if strings.Contains(entry, "Exec=AppRun") {
		t.Errorf("Exec must never be AppRun:\n%s", entry)
	}
	if _, err := os.Stat(filepath.Join(dataHome(), "icons", "hicolor", "scalable", "apps", "thing.svg")); err != nil {
		t.Errorf("icon not installed: %v", err)
	}
}
