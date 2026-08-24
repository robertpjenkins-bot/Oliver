package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/backend"
	"github.com/robertpjenkins-bot/oliver/internal/config"
	"github.com/robertpjenkins-bot/oliver/internal/prefix"
)

const backendsSynopsis = `oliver backends

Lists every Windows-compatibility runtime installed on this system: system
Wine packages, Proton builds from Steam, and runners belonging to Lutris,
Bottles, Heroic, PlayOnLinux and CrossOver.`

func (a *app) cmdBackends(args []string) error {
	fs := a.newFlagSet("backends", backendsSynopsis)
	asJSON := fs.Bool("json", false, "print the list as JSON")
	verbose := fs.Bool("v", false, "show install paths and notes")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	scanner := backend.NewScanner()
	scanner.ExtraRoots = a.cfg.RunnerDirs
	backends := scanner.Discover()

	if *asJSON {
		return a.printJSON(backends)
	}

	u := a.ui
	if len(backends) == 0 {
		u.Printf("No Windows-compatibility runtime found.")
		u.Printf("")
		u.Printf("%s", u.Dim("Install one and Oliver will pick it up automatically:"))
		u.Printf("%s", u.Dim("  Debian/Ubuntu  sudo apt install wine"))
		u.Printf("%s", u.Dim("  Fedora         sudo dnf install wine"))
		u.Printf("%s", u.Dim("  Arch           sudo pacman -S wine"))
		return nil
	}

	t := u.NewTable("RUNTIME", "KIND", "VERSION", "BITS", "SOURCE")
	for _, b := range backends {
		name := b.Name
		if b.Variant != backend.VariantStable {
			name += " " + u.Cyan("["+string(b.Variant)+"]")
		}
		version := b.Version.String()
		if b.Version.IsZero() {
			version = u.Dim("unknown")
		}
		bits := b.Bitness()
		if !b.Has32 {
			bits = u.Yellow(bits)
		}
		t.Row(name, string(b.Kind), version, bits, string(b.Source))
	}
	t.Render()

	if *verbose {
		u.Printf("")
		for _, b := range backends {
			u.Printf("%s %s", u.Bold(b.ID), u.Dim(b.Exec))
			for _, n := range b.Notes {
				u.Printf("  %s %s", u.Dim("·"), u.Dim(n))
			}
		}
	}

	u.Printf("")
	u.Printf("%s", u.Dim(fmt.Sprintf("%d runtimes. Use `oliver run -b <name>` to force one.", len(backends))))
	return nil
}

const prefixSynopsis = `oliver prefix [list|path|remove] [name]

Manages the isolated Windows environments Oliver creates. Each program
directory gets its own, so one program can never disturb another's settings.`

func (a *app) cmdPrefix(args []string) error {
	fs := a.newFlagSet("prefix", prefixSynopsis)
	asJSON := fs.Bool("json", false, "print as JSON")
	force := fs.Bool("f", false, "remove without confirmation")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	mgr := prefix.NewManager()
	if a.cfg.PrefixRoot != "" {
		mgr.Root = a.cfg.PrefixRoot
	}

	sub := "list"
	if fs.NArg() > 0 {
		sub = fs.Arg(0)
	}

	switch sub {
	case "list", "ls":
		return a.prefixList(mgr, *asJSON)
	case "path":
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: oliver prefix path <name|program.exe>")
		}
		arg := fs.Arg(1)
		p := mgr.For(arg, "")
		if _, err := os.Stat(arg); err != nil {
			p = mgr.For("", arg) // treat it as a prefix name
		}
		a.ui.Printf("%s", p.WinePrefix)
		return nil
	case "remove", "rm", "delete":
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: oliver prefix remove <name>")
		}
		name := fs.Arg(1)
		if !*force {
			p := mgr.For("", name)
			if !a.ui.Confirm(fmt.Sprintf("Delete %s and everything installed in it?", p.Dir), false) {
				a.ui.Info("cancelled")
				return nil
			}
		}
		if err := mgr.Remove(name); err != nil {
			return err
		}
		a.ui.Info("removed %s", name)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q (want list, path or remove)", sub)
	}
}

func (a *app) prefixList(mgr *prefix.Manager, asJSON bool) error {
	list, err := mgr.List()
	if err != nil {
		return err
	}
	if asJSON {
		out := make([]map[string]any, 0, len(list))
		for _, p := range list {
			out = append(out, map[string]any{
				"name": p.Name, "path": p.Dir, "wine_prefix": p.WinePrefix,
				"meta": p.Meta, "bytes": p.DiskUsage(),
			})
		}
		return a.printJSON(out)
	}

	u := a.ui
	if len(list) == 0 {
		u.Printf("No Windows environments yet. One is created the first time you run a program.")
		u.Printf("%s", u.Dim("They live in "+mgr.Root))
		return nil
	}

	t := u.NewTable("NAME", "SIZE", "RUNTIME", "LAST RUN", "PROGRAM")
	for _, p := range list {
		last := u.Dim("never")
		if !p.Meta.LastRun.IsZero() {
			last = p.Meta.LastRun.Format("2006-01-02 15:04")
		}
		program := p.Meta.Source
		if program == "" {
			program = u.Dim("unknown")
		}
		runtimeName := p.Meta.LastBackend
		if runtimeName == "" {
			runtimeName = u.Dim("-")
		}
		t.Row(p.Name, humanBytes(p.DiskUsage()), runtimeName, last, program)
	}
	t.Render()
	u.Printf("")
	u.Printf("%s", u.Dim("in "+mgr.Root))
	return nil
}

const configSynopsis = `oliver config [path|init|show]

Shows where the configuration file lives, or writes a starter one.`

func (a *app) cmdConfig(args []string) error {
	fs := a.newFlagSet("config", configSynopsis)
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	sub := "show"
	if fs.NArg() > 0 {
		sub = fs.Arg(0)
	}

	switch sub {
	case "path":
		a.ui.Printf("%s", a.cfg.Path())
		return nil
	case "show":
		return a.printJSON(a.cfg)
	case "init":
		if _, err := os.Stat(a.cfg.Path()); err == nil {
			return fmt.Errorf("%s already exists", a.cfg.Path())
		}
		example := config.Example()
		example.SetPath(a.cfg.Path())
		if err := example.Save(); err != nil {
			return err
		}
		a.ui.Info("wrote %s", a.cfg.Path())
		a.ui.Printf("%s", strings.TrimSpace(configHelp))
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q (want path, show or init)", sub)
	}
}

const configHelp = `
The file supports:
  profile      auto | game | app        how to weigh runtimes by default
  backend      a runtime name           force one runtime for everything
  winver       auto | win7 | winxp ...  the Windows version programs see
  prefix_root  a directory              where Windows environments are kept
  env          ["KEY=VALUE", ...]       variables added to every launch
  runner_dirs  ["/path", ...]           extra places to look for runtimes
  rules        per-program overrides, matched in order against the full path
`
