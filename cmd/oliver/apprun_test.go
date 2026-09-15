package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// appRunHarness lays out a fake AppDir around one of the packaging launchers,
// with a stub in place of the real binary that records how it was called.
// Running the script is the only way to know what a double-click does.
func appRunHarness(t *testing.T, script string) (dir string, calls func() string) {
	t.Helper()
	dir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "usr", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join("..", "..", "packaging", script))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AppRun"), source, 0o755); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(dir, "calls")
	stub := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + record + "\n"
	if err := os.WriteFile(filepath.Join(dir, "usr", "bin", "oliver"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, func() string {
		data, err := os.ReadFile(record)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
}

func runAppRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("sh", append([]string{filepath.Join(dir, "AppRun")}, args...)...)
	cmd.Env = append(os.Environ(), "HOME="+dir, "XDG_DATA_HOME="+filepath.Join(dir, "share"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("AppRun %v: %v\n%s", args, err, out)
	}
}

// Double-clicking the studio's AppImage must open the game maker. Anyone who
// downloads a game maker and gets a Windows program runner has been handed the
// wrong program, which is exactly what this package exists to prevent.
func TestStudioAppRunOpensTheGameMaker(t *testing.T) {
	dir, calls := appRunHarness(t, "studio-AppRun")
	runAppRun(t, dir)
	if got := calls(); got != "studio" {
		t.Errorf("double-clicking the studio ran %q, want %q", got, "studio")
	}
}

func TestStudioAppRunHandlesArguments(t *testing.T) {
	game := filepath.Join(t.TempDir(), "frozen.html")
	if err := os.WriteFile(game, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"a game dropped on the icon is played", []string{game}, "play " + game},
		{"a flag belongs to the studio", []string{"--no-open"}, "studio --no-open"},
		{"other commands still reach oliver", []string{"info", "x.exe"}, "info x.exe"},
		{"the studio's own commands pass through", []string{"studio", "list"}, "studio list"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, calls := appRunHarness(t, "studio-AppRun")
			runAppRun(t, dir, c.args...)
			if got := calls(); got != c.want {
				t.Errorf("AppRun %v ran %q, want %q", c.args, got, c.want)
			}
		})
	}
}

// Registering the studio has to record where the AppImage really is: an entry
// pointing inside the mount stops working as soon as the AppImage closes.
func TestStudioAppRunInstallsAMenuEntry(t *testing.T) {
	dir, _ := appRunHarness(t, "studio-AppRun")
	self := filepath.Join(dir, "Studio.AppImage")
	cmd := exec.Command("sh", filepath.Join(dir, "AppRun"), "--install")
	cmd.Env = append(os.Environ(),
		"HOME="+dir,
		"XDG_DATA_HOME="+filepath.Join(dir, "share"),
		"APPIMAGE="+self)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("--install: %v\n%s", err, out)
	}
	entry, err := os.ReadFile(filepath.Join(dir, "share", "applications", "oliver-studio.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(entry)
	for _, want := range []string{"Name=Oliver Studio", "Exec=\"" + self + "\"", "Categories=Development;Game;"} {
		if !strings.Contains(text, want) {
			t.Errorf("the menu entry is missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, dir+"/AppRun") {
		t.Error("the menu entry points inside the AppImage, so it breaks once it closes")
	}

	cmd = exec.Command("sh", filepath.Join(dir, "AppRun"), "--uninstall")
	cmd.Env = append(os.Environ(), "HOME="+dir, "XDG_DATA_HOME="+filepath.Join(dir, "share"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("--uninstall: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "share", "applications", "oliver-studio.desktop")); !os.IsNotExist(err) {
		t.Error("--uninstall left the menu entry behind")
	}
}

// Oliver's own AppImage keeps doing what it has always done: a .exe dropped on
// it runs, and opening it with nothing shows the welcome screen.
func TestOliverAppRunStillRunsWindowsPrograms(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "Setup.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, calls := appRunHarness(t, "AppRun")
	runAppRun(t, dir, exe)
	if got := calls(); got != "run "+exe {
		t.Errorf("dropping a program on Oliver ran %q, want %q", got, "run "+exe)
	}

	dir, calls = appRunHarness(t, "AppRun")
	cmd := exec.Command("sh", "-c", filepath.Join(dir, "AppRun")+" < /dev/null > /dev/null 2>&1")
	cmd.Env = append(os.Environ(), "HOME="+dir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("AppRun with no arguments: %v", err)
	}
	if got := calls(); got != "desktop-welcome" {
		t.Errorf("opening Oliver with nothing ran %q, want %q", got, "desktop-welcome")
	}
}
