package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProCoderWritesTheLauncherToAFile(t *testing.T) {
	s := newSystem(t)
	page := filepath.Join(t.TempDir(), "procoder.html")
	if _, err := s.run("procoder", "-write", page); err != nil {
		t.Fatalf("procoder -write: %v", err)
	}
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Pro Coder", "MakeCode Arcade", "Claude Coding Program"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("the written page is missing %q", want)
		}
	}
}

func TestProCoderConvertWritesAPlayablePage(t *testing.T) {
	s := newSystem(t)
	out := filepath.Join(t.TempDir(), "game.html")
	if _, err := s.run("procoder", "convert", "https://scratch.mit.edu/projects/60917032", "-o", out); err != nil {
		t.Fatalf("convert: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// The wrapper has to point at the player, not at the project page: the
	// page refuses to be framed and would show an empty rectangle.
	if !strings.Contains(string(data), "https://scratch.mit.edu/projects/60917032/embed") {
		t.Errorf("the converted page does not play the project:\n%s", data)
	}
}

func TestProCoderConvertBuildsAnAppDir(t *testing.T) {
	s := newSystem(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "game.AppImage")

	// Without appimagetool the AppDir is still produced, which is the part
	// worth testing: packaging it is one external command.
	t.Setenv("APPIMAGETOOL", "")
	t.Setenv("PATH", filepath.Join(dir, "nothing"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, "cache"))

	if _, err := s.run("procoder", "convert", "https://arcade.makecode.com/_gVFuJf2Y3cVW",
		"-format", "appimage", "-name", "Space Game", "-o", out); err != nil {
		t.Fatalf("convert: %v", err)
	}

	appDir := filepath.Join(dir, "game.AppDir")
	for _, want := range []string{"AppRun", "usr/bin/space-game", "space-game.desktop", "space-game.svg"} {
		if _, err := os.Stat(filepath.Join(appDir, want)); err != nil {
			t.Errorf("converted AppDir missing %s: %v", want, err)
		}
	}

	appRun, err := os.ReadFile(filepath.Join(appDir, "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(appRun), "arcade.makecode.com/---run?id=_gVFuJf2Y3cVW") {
		t.Errorf("AppRun does not open the game:\n%s", appRun)
	}
	if !strings.Contains(string(appRun), "--app=") {
		t.Error("AppRun should open a window of its own, not a browser tab")
	}

	desktop, err := os.ReadFile(filepath.Join(appDir, "space-game.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(desktop), "Name=Space Game") {
		t.Errorf("desktop entry should carry the given name:\n%s", desktop)
	}
	// As with `oliver pack`: Exec must name something that resolves on PATH,
	// and AppRun is a file inside the AppDir rather than a command.
	if strings.Contains(string(desktop), "Exec=AppRun") {
		t.Errorf("Exec must not point at AppRun:\n%s", desktop)
	}
}

func TestProCoderConvertRejectsAnUnknownFormat(t *testing.T) {
	s := newSystem(t)
	if _, err := s.run("procoder", "convert", "https://example.org/", "-format", "exe"); err == nil {
		t.Error("an unknown format should be refused")
	}
}

// A machine with no browser on PATH is the normal case on an immutable system
// like a Steam Deck, where browsers are flatpaks. Pro Coder has to find one
// there too, or it opens nothing on the machine it is most likely to run on.
func TestProCoderFindsAFlatpakBrowser(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	// A flatpak that reports Chromium installed and nothing else.
	fake := "#!/bin/sh\ncase \"$2\" in org.chromium.Chromium) exit 0 ;; esac\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "flatpak"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	cmd := appWindowCommand("http://127.0.0.1:1234/")
	if cmd == nil {
		t.Fatal("no way to open a window was found")
	}
	got := strings.Join(cmd.Args, " ")
	for _, want := range []string{"flatpak", "run org.chromium.Chromium", "--app=http://127.0.0.1:1234/"} {
		if !strings.Contains(got, want) {
			t.Errorf("command %q is missing %q", got, want)
		}
	}
	// A flatpak cannot reach a profile directory outside its own sandbox.
	if strings.Contains(got, "--user-data-dir") {
		t.Errorf("a flatpak browser must not be pointed at an outside profile: %s", got)
	}
}

func TestProCoderPrefersABrowserOnPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"chromium", "flatpak"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)

	cmd := appWindowCommand("http://127.0.0.1:1234/")
	if cmd == nil {
		t.Fatal("no way to open a window was found")
	}
	if !strings.HasSuffix(cmd.Args[0], "/chromium") {
		t.Errorf("an installed browser should beat a flatpak: %v", cmd.Args)
	}
}

func TestProCoderReportsWhenThereIsNoBrowser(t *testing.T) {
	t.Setenv("PATH", filepath.Join(t.TempDir(), "empty"))
	if cmd := appWindowCommand("http://127.0.0.1:1234/"); cmd != nil {
		t.Errorf("with no browser installed there is nothing to run: %v", cmd.Args)
	}
}

// The address goes into a shell script, so a quote in it must not end up
// ending the string it lives in.
func TestSiteAppRunQuotesTheAddress(t *testing.T) {
	script := siteAppRun("game", "Game", "https://example.org/?q='; rm -rf /", "https://example.org/")
	if strings.Contains(script, "URL='https://example.org/?q='; rm") {
		t.Errorf("the address escaped its quoting:\n%s", script)
	}
	if !strings.Contains(script, `'\''`) {
		t.Errorf("a quote in the address should be escaped:\n%s", script)
	}
}
