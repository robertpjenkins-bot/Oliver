package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The studio from the command line: make a game, list it, change it, export
// it. This is the path someone takes who never opens the editor at all.
func TestCLIMakesListsAndExportsAGame(t *testing.T) {
	s := newSystem(t)
	games := s.path("home", "user", "games")
	t.Setenv("OLIVER_GAMES_DIR", games)
	// No key: the studio's own designer does the work, with no network at all.
	t.Setenv("ANTHROPIC_API_KEY", "")

	out, err := s.run("studio", "new", "a haunted castle platformer with ghosts")
	if err != nil {
		t.Fatalf("studio new: %v\n%s", err, out)
	}
	if !strings.Contains(out, "platformer") || !strings.Contains(out, "levels") {
		t.Errorf("the summary does not describe the game:\n%s", out)
	}

	entries, err := os.ReadDir(games)
	if err != nil {
		t.Fatal(err)
	}
	var name string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".game.json") {
			name = strings.TrimSuffix(e.Name(), ".game.json")
		}
	}
	if name == "" {
		t.Fatalf("no game was written to %s", games)
	}

	out, err = s.run("studio", "list")
	if err != nil || !strings.Contains(out, name) {
		t.Fatalf("studio list: %v\n%s", err, out)
	}

	if out, err := s.run("studio", "edit", name, "add a boss and make it harder"); err != nil {
		t.Fatalf("studio edit: %v\n%s", err, out)
	}
	project, err := os.ReadFile(filepath.Join(games, name+".game.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(project), `"id": "boss"`) {
		t.Error("the edit did not reach the saved game")
	}

	html := s.path("home", "user", "game.html")
	if out, err := s.run("studio", "export", name, "-o", html); err != nil {
		t.Fatalf("studio export: %v\n%s", err, out)
	}
	page, err := os.ReadFile(html)
	if err != nil {
		t.Fatal(err)
	}
	// An exported game has to carry everything: the engine, the project, and
	// no reference to anything outside itself.
	for _, want := range []string{"OliverEngine", "var GAME =", "<canvas"} {
		if !strings.Contains(string(page), want) {
			t.Errorf("the exported page is missing %q", want)
		}
	}
	for _, unwanted := range []string{"http://", "src=\"/", "__ENGINE__", "__GAME__"} {
		if strings.Contains(string(page), unwanted) {
			t.Errorf("the exported page still refers to %q", unwanted)
		}
	}
	if len(page) < 20000 {
		t.Errorf("the exported page is only %d bytes, which cannot hold a game", len(page))
	}
}

// Packing a game produces a directory that a person could run, with the game,
// the project it was made from, and a launcher that does not depend on where
// the AppImage happens to be mounted.
func TestCLIPacksAGameAsAnAppDir(t *testing.T) {
	s := newSystem(t)
	t.Setenv("OLIVER_GAMES_DIR", s.path("home", "user", "games"))

	if out, err := s.run("studio", "new", "a space shooter"); err != nil {
		t.Fatalf("studio new: %v\n%s", err, out)
	}
	entries, _ := os.ReadDir(s.path("home", "user", "games"))
	var name string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".game.json") {
			name = strings.TrimSuffix(e.Name(), ".game.json")
		}
	}

	target := s.path("home", "user", "Game.AppImage")
	if out, err := s.run("studio", "export", name, "--appimage", "--appdir", "-o", target); err != nil {
		t.Fatalf("export --appimage: %v\n%s", err, out)
	}
	appDir := s.path("home", "user", "Game.AppDir")
	for _, want := range []string{"AppRun", "usr/bin/oliver"} {
		if _, err := os.Stat(filepath.Join(appDir, want)); err != nil {
			t.Errorf("the AppDir has no %s: %v", want, err)
		}
	}
	appRun, err := os.ReadFile(filepath.Join(appDir, "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(appRun), "oliver") || !strings.Contains(string(appRun), "--install") {
		t.Error("the launcher cannot start the game or register it in a menu")
	}
	var html, project, icon bool
	_ = filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		switch {
		case strings.HasSuffix(path, ".html"):
			html = true
		case strings.HasSuffix(path, ".game.json"):
			project = true
		case strings.HasSuffix(path, ".svg"):
			icon = true
		}
		return nil
	})
	if !html || !project || !icon {
		t.Errorf("the AppDir is missing something: page=%v project=%v icon=%v", html, project, icon)
	}
}
