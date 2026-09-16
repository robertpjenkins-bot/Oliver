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
