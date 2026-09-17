package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/robertpjenkins-bot/oliver/internal/procoder"
)

const procoderSynopsis = `oliver procoder [flags]
oliver procoder convert <project-url> [flags]

Pro Coder: Scratch, MakeCode Arcade and CCP in one window.

Opening it with no arguments shows the menu. Choosing something opens it inside
Pro Coder with its address on show and a way back out.

CCP — the Claude Coding Program — is Oliver's own block coder: Scratch's blocks
and MakeCode Arcade's game blocks in one palette, with a few of its own.

"convert" is the Convert button from the menu, on the command line. Give it the
address of a project and it writes either a standalone HTML file or an AppImage
that opens that project as an application of its own.`

func (a *app) cmdProCoder(args []string) error {
	if len(args) > 0 && args[0] == "convert" {
		return a.cmdProCoderConvert(args[1:])
	}

	fs := a.newFlagSet("procoder", procoderSynopsis)
	port := fs.Int("port", 0, "serve on this port instead of a free one")
	noOpen := fs.Bool("no-open", false, "print the address instead of opening a window")
	write := fs.String("write", "", "write Pro Coder to this HTML file and exit")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	u := a.ui
	if *write != "" {
		if err := os.WriteFile(*write, procoder.App(), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", *write, err)
		}
		u.Info("%s wrote %s", u.Green("✓"), *write)
		u.Info("%s", u.Dim("Open it by double-clicking it. Convert can make HTML files from there;"))
		u.Info("%s", u.Dim("AppImages need `oliver procoder` itself, which this file will explain."))
		return nil
	}

	srv := &procoder.Server{Pack: a.packSite}
	ln, addr, err := procoder.Listen(*port)
	if err != nil {
		return err
	}
	defer ln.Close()

	server := &http.Server{Handler: srv.Handler()}
	errs := make(chan error, 1)
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()

	u.Info("%s Pro Coder is at %s", u.Green("▸"), u.Cyan(addr))

	// Waiting on the browser is what makes this feel like an application
	// rather than a server: closing the window ends the command. When the
	// window cannot be owned — an already-running browser took the address,
	// or the desktop opened it for us — there is nothing to wait for, so
	// Ctrl-C is the way out and the address stays printed.
	var window *exec.Cmd
	if !*noOpen {
		window, err = openAppWindow(addr)
		if err != nil {
			u.Warn("%v", err)
			u.Info("%s", u.Dim("Open that address in a browser yourself."))
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	closed := make(chan struct{})
	if window != nil {
		go func() {
			_ = window.Wait()
			close(closed)
		}()
	} else {
		u.Info("%s", u.Dim("Press Ctrl-C to stop Pro Coder."))
	}

	select {
	case err := <-errs:
		return err
	case <-stop:
	case <-closed:
	}
	_ = server.Shutdown(context.Background())
	if window != nil && window.Process != nil {
		_ = window.Process.Kill()
	}
	return nil
}

const procoderConvertSynopsis = `oliver procoder convert <project-url> [flags]

Turns a shared project into something you keep: an HTML file that plays it, or
an AppImage that opens it as an application with its own menu entry.

  oliver procoder convert https://scratch.mit.edu/projects/60917032
  oliver procoder convert https://arcade.makecode.com/_gVFuJf -format appimage

Scratch and MakeCode projects are recognised, so the wrapper points at the
player rather than at the editor page. Any other web address is wrapped as it
is. Converting copies nothing: the project still lives on the site, and the
wrapper needs the internet to reach it.`

func (a *app) cmdProCoderConvert(args []string) error {
	fs := a.newFlagSet("procoder convert", procoderConvertSynopsis)
	out := fs.String("o", "", "output path (default: alongside the current directory)")
	name := fs.String("name", "", "what to call it")
	format := fs.String("format", "html", "html or appimage")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("no project address given")
	}

	target, err := procoder.Recognise(fs.Arg(0))
	if err != nil {
		return err
	}
	appName := *name
	if appName == "" {
		appName = target.Title
	}

	switch strings.ToLower(*format) {
	case "html":
		page, err := procoder.WrapperHTML(target, appName)
		if err != nil {
			return err
		}
		path := *out
		if path == "" {
			path = target.Name() + ".html"
		}
		if err := os.WriteFile(path, page, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		abs, _ := filepath.Abs(path)
		a.ui.Info("%s built %s", a.ui.Green("✓"), abs)
		a.ui.Info("%s", a.ui.Dim("Plays "+target.Play))
		return nil
	case "appimage":
		res, err := a.packSite(procoder.PackRequest{URL: fs.Arg(0), Name: appName, Format: "appimage", Out: *out})
		if err != nil {
			return err
		}
		a.ui.Info("%s built %s", a.ui.Green("✓"), res.Path)
		if res.Note != "" {
			a.ui.Info("%s", a.ui.Dim(res.Note))
		}
		a.ui.Info("")
		a.ui.Info("%s", a.ui.Bold("To add it to your application menu:"))
		a.ui.Info("  %s --install", shellQuoteArg(res.Path))
		return nil
	default:
		return fmt.Errorf("unknown format %q: use html or appimage", *format)
	}
}

// packSite builds an AppImage that opens one web project as an application.
//
// Unlike `oliver pack` there is no program to carry and no Wine involved: what
// goes inside is a launcher that finds a browser on the machine it lands on
// and gives the project a window of its own, with its own browser profile so
// it keeps its logins and its saved work separately from everything else.
func (a *app) packSite(req procoder.PackRequest) (procoder.PackResult, error) {
	target, err := procoder.Recognise(req.URL)
	if err != nil {
		return procoder.PackResult{}, err
	}
	appName := strings.TrimSpace(req.Name)
	if appName == "" {
		appName = target.Title
	}
	slug := slugFor(appName)
	if slug == "" {
		slug = target.Name()
	}

	work, err := os.MkdirTemp("", "oliver-procoder-")
	if err != nil {
		return procoder.PackResult{}, err
	}
	defer os.RemoveAll(work)

	appDir := filepath.Join(work, appName+".AppDir")
	if err := os.MkdirAll(filepath.Join(appDir, "usr", "share", "applications"), 0o755); err != nil {
		return procoder.PackResult{}, err
	}
	if err := os.MkdirAll(filepath.Join(appDir, "usr", "bin"), 0o755); err != nil {
		return procoder.PackResult{}, err
	}

	appRun := siteAppRun(slug, appName, target.Play, target.Page)
	if err := os.WriteFile(filepath.Join(appDir, "AppRun"), []byte(appRun), 0o755); err != nil {
		return procoder.PackResult{}, err
	}
	// As in `oliver pack`: the desktop entry's Exec has to name something on
	// PATH, and AppRun is not.
	if err := os.WriteFile(filepath.Join(appDir, "usr", "bin", slug), []byte(appRun), 0o755); err != nil {
		return procoder.PackResult{}, err
	}

	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=Opened with Pro Coder
Exec=%s %%U
Icon=%s
Terminal=false
Categories=Development;Education;
NoDisplay=true
`, appName, slug, slug)
	for _, p := range []string{
		filepath.Join(appDir, slug+".desktop"),
		filepath.Join(appDir, "usr", "share", "applications", slug+".desktop"),
	} {
		if err := os.WriteFile(p, []byte(desktop), 0o644); err != nil {
			return procoder.PackResult{}, err
		}
	}
	if err := os.WriteFile(filepath.Join(appDir, slug+".svg"), []byte(proCoderIconSVG), 0o644); err != nil {
		return procoder.PackResult{}, err
	}

	output := strings.TrimSpace(req.Out)
	if output == "" {
		// Named by what the person called it when they said so, and after the
		// project otherwise.
		base := target.Name()
		if strings.TrimSpace(req.Name) != "" && slug != "" {
			base = slug
		}
		output = base + "-x86_64.AppImage"
	}
	if abs, err := filepath.Abs(output); err == nil {
		output = abs
	}

	tool, err := findAppImageTool()
	if err != nil {
		// Same bargain as `oliver pack`: the directory is complete and
		// runnable, only the single-file step is missing, so keep it.
		final := strings.TrimSuffix(output, ".AppImage") + ".AppDir"
		if err := os.RemoveAll(final); err != nil {
			return procoder.PackResult{}, err
		}
		if copyErr := copyTree(appDir, final); copyErr != nil {
			return procoder.PackResult{}, copyErr
		}
		return procoder.PackResult{
			Path: final,
			Note: "appimagetool is not installed, so this is an unpackaged AppDir: run " + final + "/AppRun",
		}, nil
	}

	if err := runAppImageTool(tool, appDir, output); err != nil {
		return procoder.PackResult{}, err
	}
	return procoder.PackResult{Path: output, Note: "Opens " + target.Page}, nil
}

// runAppImageTool packages an AppDir, falling back to extracting appimagetool
// when FUSE is missing, which is the usual reason it fails.
func runAppImageTool(tool, appDir, output string) error {
	for _, args := range [][]string{{appDir, output}, {"--appimage-extract-and-run", appDir, output}} {
		cmd := exec.Command(tool, args...)
		cmd.Env = append(os.Environ(), "ARCH=x86_64")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		} else if args[0] != appDir {
			return fmt.Errorf("packaging failed: %w", err)
		}
	}
	return nil
}

// siteAppRun is the launcher inside a converted project's AppImage.
func siteAppRun(slug, appName, play, page string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu

HERE="$(dirname "$(readlink -f "${0}")")"
URL=%[3]s
PAGE=%[4]s

# A menu entry has to record where the AppImage actually is: anything pointing
# inside the mount stops working the moment the AppImage closes.
if [ "${1:-}" = "--install" ] || [ "${1:-}" = "--uninstall" ]; then
	APPS="${XDG_DATA_HOME:-${HOME}/.local/share}/applications"
	ICONS="${XDG_DATA_HOME:-${HOME}/.local/share}/icons/hicolor/scalable/apps"
	ENTRY="${APPS}/%[1]s.desktop"
	if [ "${1}" = "--uninstall" ]; then
		rm -f "${ENTRY}" "${ICONS}/%[1]s.svg"
		echo "Removed %[2]s from your application menu."
	else
		SELF="${APPIMAGE:-$(readlink -f "${0}")}"
		mkdir -p "${APPS}" "${ICONS}"
		[ -f "${HERE}/%[1]s.svg" ] && cp "${HERE}/%[1]s.svg" "${ICONS}/%[1]s.svg" || true
		cat > "${ENTRY}" <<DESKTOP_ENTRY
[Desktop Entry]
Type=Application
Name=%[2]s
Comment=Opened with Pro Coder
Exec="${SELF}" %%U
TryExec=${SELF}
Icon=%[1]s
Terminal=false
Categories=Development;Education;
DESKTOP_ENTRY
		echo "Added %[2]s to your application menu."
	fi
	command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "${APPS}" || true
	exit 0
fi

if [ "${1:-}" = "--url" ]; then
	echo "${PAGE}"
	exit 0
fi

# Its own browser profile, so the project's saved work and logins are its own
# and closing this window never touches anybody's ordinary browsing.
DATA="${XDG_DATA_HOME:-${HOME}/.local/share}/oliver/procoder/%[1]s"
mkdir -p "${DATA}"

for browser in chromium chromium-browser google-chrome google-chrome-stable brave-browser microsoft-edge vivaldi-stable; do
	if command -v "${browser}" >/dev/null 2>&1; then
		exec "${browser}" --app="${URL}" --user-data-dir="${DATA}/browser" --class="%[1]s" "$@"
	fi
done

# A system with no package manager — a Steam Deck, say — keeps its browsers as
# flatpaks, where nothing lands on PATH. A flatpak cannot reach a profile
# directory outside its sandbox, so it keeps its own.
if command -v flatpak >/dev/null 2>&1; then
	for id in org.chromium.Chromium com.google.Chrome com.brave.Browser com.microsoft.Edge; do
		if flatpak info "${id}" >/dev/null 2>&1; then
			exec flatpak run "${id}" --app="${URL}" --class="%[1]s" "$@"
		fi
	done
fi

# Firefox has no application mode; a plain window is the closest it gets.
if command -v firefox >/dev/null 2>&1; then
	exec firefox --new-window "${URL}" "$@"
fi
if command -v flatpak >/dev/null 2>&1 && flatpak info org.mozilla.firefox >/dev/null 2>&1; then
	exec flatpak run org.mozilla.firefox --new-window "${URL}" "$@"
fi
if command -v xdg-open >/dev/null 2>&1; then
	exec xdg-open "${URL}"
fi

echo "No browser found. Open this address yourself:" >&2
echo "  ${PAGE}" >&2
exit 1
`, slug, appName, shellLiteral(play), shellLiteral(page))
}

// chromiumBrowsers are the browsers with an application mode, in the order
// they are worth trying.
var chromiumBrowsers = []string{
	"chromium", "chromium-browser", "google-chrome", "google-chrome-stable",
	"brave-browser", "microsoft-edge", "vivaldi-stable",
}

// chromiumFlatpaks are the same browsers installed the way an immutable
// system installs them. A Steam Deck has no package manager to speak of and
// often no browser on PATH at all, so skipping these would mean Pro Coder
// could not open a window on the machine it is most likely to be used on.
var chromiumFlatpaks = []string{
	"org.chromium.Chromium", "com.google.Chrome", "com.brave.Browser", "com.microsoft.Edge",
}

// openAppWindow opens an address in a window with no browser furniture around
// it, which is what turns a page into something that reads as an application.
func openAppWindow(addr string) (*exec.Cmd, error) {
	cmd := appWindowCommand(addr)
	if cmd == nil {
		// Anything else at all is better than nothing, but a browser opened
		// this way belongs to the desktop and cannot be waited on.
		if path, err := exec.LookPath("xdg-open"); err == nil {
			if err := exec.Command(path, addr).Start(); err == nil {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("no browser found to open Pro Coder")
	}
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", cmd.Path, err)
	}
	return cmd, nil
}

func appWindowCommand(addr string) *exec.Cmd {
	profile := filepath.Join(os.TempDir(), "oliver-procoder-window")
	for _, browser := range chromiumBrowsers {
		if path, err := exec.LookPath(browser); err == nil {
			return exec.Command(path, "--app="+addr, "--user-data-dir="+profile, "--class=oliver-procoder")
		}
	}
	if flatpak, err := exec.LookPath("flatpak"); err == nil {
		for _, id := range chromiumFlatpaks {
			if !flatpakInstalled(flatpak, id) {
				continue
			}
			// No --user-data-dir here: a flatpak cannot reach a path outside
			// its own sandbox, and it keeps a profile of its own anyway.
			return exec.Command(flatpak, "run", id, "--app="+addr, "--class=oliver-procoder")
		}
	}
	// Firefox has no application mode, so this is a plain window — which is
	// still Pro Coder, just with an address bar above it.
	if path, err := exec.LookPath("firefox"); err == nil {
		return exec.Command(path, "--new-window", addr)
	}
	if flatpak, err := exec.LookPath("flatpak"); err == nil && flatpakInstalled(flatpak, "org.mozilla.firefox") {
		return exec.Command(flatpak, "run", "org.mozilla.firefox", "--new-window", addr)
	}
	return nil
}

func flatpakInstalled(flatpak, id string) bool {
	cmd := exec.Command(flatpak, "info", id)
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Run() == nil
}

// shellLiteral quotes a string so a POSIX shell sees it exactly as written.
func shellLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// proCoderIconSVG is the icon a converted project carries.
const proCoderIconSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="128" height="128">
  <defs>
    <linearGradient id="p" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#7c5cff"/>
      <stop offset="1" stop-color="#2bb3ff"/>
    </linearGradient>
  </defs>
  <rect x="8" y="8" width="112" height="112" rx="26" fill="url(#p)"/>
  <rect x="26" y="34" width="52" height="18" rx="6" fill="#fff" opacity=".95"/>
  <rect x="34" y="58" width="60" height="18" rx="6" fill="#fff" opacity=".8"/>
  <rect x="26" y="82" width="44" height="18" rx="6" fill="#fff" opacity=".65"/>
</svg>
`
