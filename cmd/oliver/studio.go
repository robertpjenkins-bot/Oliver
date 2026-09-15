package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/robertpjenkins-bot/oliver/internal/gamekit"
	"github.com/robertpjenkins-bot/oliver/internal/studio"
)

const studioSynopsis = `oliver studio [command] [flags]

A retro video game maker. Describe a game and it writes one — sprites, levels,
enemies, rules and sound — then opens an editor where every part of it can be
taken apart and changed by hand.

  oliver studio                      open the studio
  oliver studio new "<description>"  design a game without opening the editor
  oliver studio list                 list the games on this machine
  oliver studio edit <name> "<what to change>"
  oliver studio play <name>          play a game now
  oliver studio export <name>        write a playable HTML file
  oliver studio export <name> --appimage

Games are JSON files in ~/.local/share/oliver/games. An exported game is one
HTML file with the engine, the artwork and the sound inside it: no server, no
network, nothing to install.

With ANTHROPIC_API_KEY set, designing asks Claude. Without it, the studio's own
designer does the work offline — the editor, the engine and every export work
the same either way.`

func (a *app) cmdStudio(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "new", "create":
			return a.studioNew(args[1:])
		case "list", "ls":
			return a.studioList(args[1:])
		case "edit", "change":
			return a.studioEdit(args[1:])
		case "export":
			return a.studioExport(args[1:])
		case "play":
			return a.studioPlay(args[1:])
		case "-h", "--help", "help":
			fmt.Fprintln(os.Stdout, studioSynopsis)
			return nil
		}
	}
	return a.studioServe(args)
}

// studioServe starts the editor and opens it.
func (a *app) studioServe(args []string) error {
	fs := a.newFlagSet("studio", studioSynopsis)
	addr := fs.String("addr", "127.0.0.1:0", "address to listen on")
	dir := fs.String("dir", "", "folder holding the games (default: ~/.local/share/oliver/games)")
	noOpen := fs.Bool("no-open", false, "do not open a browser")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	u := a.ui
	s := studio.New(*dir)
	s.Log = func(format string, args ...any) { u.Info("· "+format, args...) }

	// Ctrl-C has to leave nothing behind: the studio owns a port and a browser
	// tab, and both should end with the command.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	u.Info("%s", u.Bold("Oliver Studio"))
	u.Info("games in %s", s.Dir)
	if s.Assistant.Available() {
		u.Info("designing with %s", s.Assistant.Model)
	} else {
		u.Info("%s", u.Dim("designing offline — set ANTHROPIC_API_KEY to design with Claude instead"))
	}

	return s.Serve(ctx, *addr, func(url string) {
		u.Printf("%s", url)
		if *noOpen {
			return
		}
		if err := studio.OpenBrowser(url); err != nil {
			u.Warn("%v", err)
		}
	})
}

func (a *app) studioNew(args []string) error {
	fs := a.newFlagSet("studio new", studioSynopsis)
	dir := fs.String("dir", "", "folder holding the games")
	out := fs.String("o", "", "also write a playable HTML file here")
	play := fs.Bool("play", false, "open the game once it is made")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	prompt := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if prompt == "" {
		fs.Usage()
		return fmt.Errorf("say what kind of game to make, for example: oliver studio new \"a haunted castle platformer\"")
	}

	u := a.ui
	s := studio.New(*dir)
	u.Info("designing…")
	g, note, err := s.Assistant.Create(context.Background(), prompt)
	if err != nil {
		return err
	}
	name, err := s.Save("", g)
	if err != nil {
		return err
	}
	u.Info("%s %s (%s)", u.Green("✓"), g.Name, note)
	u.Printf("%s", strings.TrimRight(g.Summary(), "\n"))
	if problems := g.Validate(); len(problems) > 0 {
		for _, p := range problems {
			u.Warn("%s", p)
		}
	}
	path, _ := s.Path(name)
	u.Info("saved %s", path)

	if *out != "" {
		written, err := g.WriteHTML(*out)
		if err != nil {
			return err
		}
		u.Info("%s wrote %s", u.Green("✓"), written)
		if *play {
			return openPlayable(a, written)
		}
		return nil
	}
	if *play {
		return a.studioPlay([]string{name})
	}
	u.Info("%s", u.Dim("open it with: oliver studio play "+name))
	return nil
}

func (a *app) studioList(args []string) error {
	fs := a.newFlagSet("studio list", studioSynopsis)
	dir := fs.String("dir", "", "folder holding the games")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	s := studio.New(*dir)
	games, err := s.List()
	if err != nil {
		return err
	}
	if len(games) == 0 {
		a.ui.Info("no games yet — make one with: oliver studio new \"a space shooter\"")
		return nil
	}
	table := a.ui.NewTable("NAME", "TITLE", "GENRE", "LEVELS", "CHANGED")
	for _, g := range games {
		table.Row(g.Name, g.Title, g.Genre, fmt.Sprintf("%d", g.Levels), g.Modified.Format("2006-01-02 15:04"))
	}
	table.Render()
	return nil
}

func (a *app) studioEdit(args []string) error {
	fs := a.newFlagSet("studio edit", studioSynopsis)
	dir := fs.String("dir", "", "folder holding the games")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		fs.Usage()
		return fmt.Errorf("say which game to change and what to change about it")
	}
	s := studio.New(*dir)
	name := fs.Arg(0)
	g, err := s.Load(name)
	if err != nil {
		return err
	}
	instruction := strings.Join(fs.Args()[1:], " ")
	edited, note, err := s.Assistant.Edit(context.Background(), g, instruction)
	if err != nil {
		return err
	}
	if _, err := s.Save(name, edited); err != nil {
		return err
	}
	a.ui.Info("%s %s", a.ui.Green("✓"), note)
	for _, p := range edited.Validate() {
		a.ui.Warn("%s", p)
	}
	return nil
}

func (a *app) studioExport(args []string) error {
	fs := a.newFlagSet("studio export", studioSynopsis)
	dir := fs.String("dir", "", "folder holding the games")
	out := fs.String("o", "", "where to write it")
	appimage := fs.Bool("appimage", false, "package the game as an AppImage instead of an HTML file")
	appDirOnly := fs.Bool("appdir", false, "with --appimage: build the AppDir but do not package it")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("say which game to export (see: oliver studio list)")
	}
	s := studio.New(*dir)
	g, err := s.Load(fs.Arg(0))
	if err != nil {
		return err
	}
	if *appimage {
		return a.packGame(g, *out, *appDirOnly)
	}
	written, err := g.WriteHTML(*out)
	if err != nil {
		return err
	}
	a.ui.Info("%s wrote %s", a.ui.Green("✓"), written)
	a.ui.Info("%s", a.ui.Dim("that one file is the whole game: open it in any browser, or put it on a web host"))
	return nil
}

func (a *app) studioPlay(args []string) error {
	fs := a.newFlagSet("studio play", studioSynopsis)
	dir := fs.String("dir", "", "folder holding the games")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("say which game to play (see: oliver studio list)")
	}
	s := studio.New(*dir)
	g, err := s.Load(fs.Arg(0))
	if err != nil {
		return err
	}
	// The playable file lives somewhere stable rather than in a temporary
	// directory, so the browser can still read it after this command exits.
	path := filepath.Join(s.Dir, "played", g.Slug()+".html")
	written, err := g.WriteHTML(path)
	if err != nil {
		return err
	}
	return openPlayable(a, written)
}

// openPlayable hands a finished game to the desktop.
func openPlayable(a *app, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	url := "file://" + abs
	a.ui.Info("opening %s", abs)
	if err := studio.OpenBrowser(url); err != nil {
		a.ui.Warn("%v", err)
		a.ui.Printf("%s", url)
	}
	return nil
}

// packGame wraps a game in an AppImage.
//
// The result carries the game and Oliver, and nothing else: no browser engine,
// no runtime, no node. It opens the game in whatever browser the machine has,
// which is why it is a couple of megabytes rather than a couple of hundred.
func (a *app) packGame(g *gamekit.Game, out string, appDirOnly bool) error {
	u := a.ui
	slug := g.Slug()
	self, err := oliverBinary()
	if err != nil {
		return err
	}

	work, err := os.MkdirTemp("", "oliver-game-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	appDir := filepath.Join(work, g.Name+".AppDir")
	for _, d := range []string{
		filepath.Join(appDir, "usr", "bin"),
		filepath.Join(appDir, "usr", "share", "oliver", "game"),
		filepath.Join(appDir, "usr", "share", "applications"),
		filepath.Join(appDir, "usr", "share", "icons", "hicolor", "scalable", "apps"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	if err := copyFile(self, filepath.Join(appDir, "usr", "bin", "oliver"), 0o755); err != nil {
		return fmt.Errorf("bundling oliver: %w", err)
	}
	page, err := g.ExportHTML()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(appDir, "usr", "share", "oliver", "game", slug+".html"), page, 0o644); err != nil {
		return err
	}
	// The project travels with the game, so an AppImage can be opened in the
	// studio again and changed. A game nobody can edit is a dead end.
	project, err := g.JSON()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(appDir, "usr", "share", "oliver", "game", slug+".game.json"), project, 0o644); err != nil {
		return err
	}

	appRun := gameAppRun(slug, g.Name)
	if err := os.WriteFile(filepath.Join(appDir, "AppRun"), []byte(appRun), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(appDir, "usr", "bin", slug), []byte(appRun), 0o755); err != nil {
		return err
	}

	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=%s
Exec=%s
Icon=%s
Terminal=false
Categories=Game;
NoDisplay=true
`, g.Name, g.Tagline, slug, slug)
	for _, p := range []string{
		filepath.Join(appDir, slug+".desktop"),
		filepath.Join(appDir, "usr", "share", "applications", slug+".desktop"),
	} {
		if err := os.WriteFile(p, []byte(desktop), 0o644); err != nil {
			return err
		}
	}
	icon := g.IconSVG()
	for _, p := range []string{
		filepath.Join(appDir, slug+".svg"),
		filepath.Join(appDir, "usr", "share", "icons", "hicolor", "scalable", "apps", slug+".svg"),
	} {
		if err := os.WriteFile(p, icon, 0o644); err != nil {
			return err
		}
	}

	if out == "" {
		out = fmt.Sprintf("%s-x86_64.AppImage", g.Slug())
	}
	if abs, err := filepath.Abs(out); err == nil {
		out = abs
	}
	if appDirOnly {
		final := strings.TrimSuffix(out, ".AppImage") + ".AppDir"
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
		final := strings.TrimSuffix(out, ".AppImage") + ".AppDir"
		if copyErr := copyTree(appDir, final); copyErr == nil {
			u.Warn("%v", err)
			u.Info("the unpackaged AppDir is at %s; run it with %s/AppRun", final, final)
			return nil
		}
		return err
	}
	cmd := exec.Command(tool, appDir, out)
	cmd.Env = append(os.Environ(), "ARCH=x86_64")
	cmd.Stdout, cmd.Stderr = io.Discard, os.Stderr
	if err := cmd.Run(); err != nil {
		cmd = exec.Command(tool, "--appimage-extract-and-run", appDir, out)
		cmd.Env = append(os.Environ(), "ARCH=x86_64")
		cmd.Stdout, cmd.Stderr = io.Discard, os.Stderr
		if err2 := cmd.Run(); err2 != nil {
			return fmt.Errorf("packaging failed: %w", err)
		}
	}
	u.Info("%s built %s", u.Green("✓"), out)
	u.Info("")
	u.Info("%s", u.Bold("To add it to the games menu:"))
	u.Info("  %s --install", shellQuoteArg(out))
	return nil
}

// gameAppRun is the launcher inside a packed game.
//
// It copies the page out of the read-only mount before opening it: a browser
// opened on a file inside an AppImage may well find that file gone by the time
// it gets there, because the mount belongs to a process that has exited.
func gameAppRun(slug, name string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu

HERE="$(dirname "$(readlink -f "${0}")")"
OLIVER="${HERE}/usr/bin/oliver"
DATA="${XDG_DATA_HOME:-${HOME}/.local/share}/oliver/games/packed/%[1]s"

if [ "${1:-}" = "--install" ] || [ "${1:-}" = "--uninstall" ]; then
	APPS="${XDG_DATA_HOME:-${HOME}/.local/share}/applications"
	ICONS="${XDG_DATA_HOME:-${HOME}/.local/share}/icons/hicolor/scalable/apps"
	ENTRY="${APPS}/%[1]s.desktop"
	if [ "${1}" = "--uninstall" ]; then
		rm -f "${ENTRY}" "${ICONS}/%[1]s.svg"
		echo "Removed %[2]s from your games menu."
	else
		SELF="${APPIMAGE:-$(readlink -f "${0}")}"
		mkdir -p "${APPS}" "${ICONS}"
		[ -f "${HERE}/%[1]s.svg" ] && cp "${HERE}/%[1]s.svg" "${ICONS}/%[1]s.svg" || true
		cat > "${ENTRY}" <<DESKTOP_ENTRY
[Desktop Entry]
Type=Application
Name=%[2]s
Comment=A game made with Oliver Studio
Exec="${SELF}"
TryExec=${SELF}
Icon=%[1]s
Terminal=false
Categories=Game;
DESKTOP_ENTRY
		echo "Added %[2]s to your games menu."
	fi
	command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "${APPS}" || true
	exit 0
fi

# --edit opens the game in the studio instead of playing it.
if [ "${1:-}" = "--edit" ]; then
	GAMES="${XDG_DATA_HOME:-${HOME}/.local/share}/oliver/games"
	mkdir -p "${GAMES}"
	cp -f "${HERE}/usr/share/oliver/game/%[1]s.game.json" "${GAMES}/%[1]s.game.json"
	exec "${OLIVER}" studio
fi

mkdir -p "${DATA}"
cp -f "${HERE}/usr/share/oliver/game/%[1]s.html" "${DATA}/%[1]s.html"
exec "${OLIVER}" play "${DATA}/%[1]s.html"
`, slug, name)
}
