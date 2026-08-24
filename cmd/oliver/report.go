package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/prefix"
	"github.com/robertpjenkins-bot/oliver/internal/runner"
)

const reportSynopsis = `oliver report [-o file]

Writes everything needed to diagnose a problem into one file: the runtimes
found, every launcher entry and whether it can actually start, the Windows
environments that exist, and the last run's output.

Nothing is sent anywhere. The file is yours to read or pass on.`

func (a *app) cmdReport(args []string) error {
	fs := a.newFlagSet("report", reportSynopsis)
	out := fs.String("o", "", "where to write the report (default: the desktop, or your home directory)")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	path := *out
	if path == "" {
		path = defaultReportPath()
	}
	body := a.buildReport()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("writing the report: %w", err)
	}
	a.ui.Info("%s wrote %s", a.ui.Green("✓"), path)
	return nil
}

func defaultReportPath() string {
	name := "oliver-report.txt"
	home, err := os.UserHomeDir()
	if err != nil {
		return name
	}
	if d := os.Getenv("XDG_DESKTOP_DIR"); d != "" {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return filepath.Join(d, name)
		}
	}
	desktop := filepath.Join(home, "Desktop")
	if st, err := os.Stat(desktop); err == nil && st.IsDir() {
		return filepath.Join(desktop, name)
	}
	return filepath.Join(home, name)
}

// buildReport gathers the state that explains why a launch failed.
func (a *app) buildReport() string {
	var b strings.Builder
	section := func(title string) { fmt.Fprintf(&b, "\n===== %s =====\n", title) }

	fmt.Fprintf(&b, "oliver %s report, %s\n", version, time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "%s %s, running from %s\n", runtime.GOOS, runtime.GOARCH, selfPath())
	if v := os.Getenv("APPIMAGE"); v != "" {
		fmt.Fprintf(&b, "APPIMAGE=%s\n", v)
	}
	for _, key := range []string{"XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_DESKTOP_DIR", "DESKTOP_SESSION", "XDG_CURRENT_DESKTOP"} {
		if v := os.Getenv(key); v != "" {
			fmt.Fprintf(&b, "%s=%s\n", key, v)
		}
	}

	section("system check")
	b.WriteString(a.captureText(func() { _ = a.cmdDoctor(nil) }))

	section("runtimes found")
	b.WriteString(a.captureText(func() { _ = a.cmdBackends([]string{"-v"}) }))

	// The launcher entries are the usual cause of "could not find" errors, so
	// record every one along with whether its command actually resolves.
	section("launcher entries")
	for _, dir := range entryDirs() {
		fmt.Fprintf(&b, "\n-- %s\n", dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(&b, "   (cannot read: %v)\n", err)
			continue
		}
		var found int
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
				continue
			}
			found++
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			fmt.Fprintf(&b, "\n   %s\n", e.Name())
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "Name=") || strings.HasPrefix(line, "Exec=") ||
					strings.HasPrefix(line, "TryExec=") || strings.HasPrefix(line, "Icon=") ||
					strings.HasPrefix(line, "NoDisplay=") || strings.HasPrefix(line, "Comment=") {
					fmt.Fprintf(&b, "     %s\n", line)
				}
			}
			if be, bad := inspectEntry(path); bad {
				fmt.Fprintf(&b, "     >>> BROKEN: %s\n", be.Reason)
			} else {
				fmt.Fprintf(&b, "     ok: its command resolves\n")
			}
		}
		if found == 0 {
			fmt.Fprintf(&b, "   (no .desktop files)\n")
		}
	}

	section("files on the desktop and in downloads")
	if home, err := os.UserHomeDir(); err == nil {
		for _, d := range []string{"Desktop", "Downloads"} {
			dir := filepath.Join(home, d)
			fmt.Fprintf(&b, "\n-- %s\n", dir)
			entries, err := os.ReadDir(dir)
			if err != nil {
				fmt.Fprintf(&b, "   (cannot read: %v)\n", err)
				continue
			}
			for _, e := range entries {
				name := e.Name()
				lower := strings.ToLower(name)
				// Only what is relevant: Oliver, packages, and Windows programs.
				if !strings.Contains(lower, "oliver") && !strings.HasSuffix(lower, ".appimage") &&
					!strings.HasSuffix(lower, ".exe") && !strings.HasSuffix(lower, ".msi") &&
					!strings.HasSuffix(lower, ".desktop") {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				mode := "----"
				if info.Mode().Perm()&0o111 != 0 {
					mode = "exec"
				}
				fmt.Fprintf(&b, "   %s  %10d  %s\n", mode, info.Size(), name)
			}
		}
	}

	section("packaged programs")
	if home, err := os.UserHomeDir(); err == nil {
		packed := filepath.Join(dataHome(), "oliver", "packed")
		_ = home
		entries, err := os.ReadDir(packed)
		if err != nil {
			fmt.Fprintf(&b, "(none: %v)\n", err)
		} else {
			for _, e := range entries {
				fmt.Fprintf(&b, "   %s\n", e.Name())
				sub, _ := os.ReadDir(filepath.Join(packed, e.Name()))
				for _, f := range sub {
					fmt.Fprintf(&b, "     %s\n", f.Name())
				}
			}
		}
	}

	section("windows environments")
	mgr := prefix.NewManager()
	if a.cfg != nil && a.cfg.PrefixRoot != "" {
		mgr.Root = a.cfg.PrefixRoot
	}
	b.WriteString(a.captureText(func() { _ = a.prefixList(mgr, false) }))

	section("last run")
	if path, err := runner.LatestLog(); err != nil {
		fmt.Fprintf(&b, "(no runs recorded: %v)\n", err)
	} else {
		fmt.Fprintf(&b, "from %s\n\n", path)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(&b, "(cannot read: %v)\n", err)
		} else {
			// Keep the report readable when a program is very chatty.
			text := string(data)
			const limit = 40000
			if len(text) > limit {
				text = text[:limit] + "\n...(truncated)\n"
			}
			b.WriteString(text)
		}
	}

	return b.String()
}
