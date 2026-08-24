package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/backend"
	"github.com/robertpjenkins-bot/oliver/internal/compat"
	"github.com/robertpjenkins-bot/oliver/internal/pe"
	"github.com/robertpjenkins-bot/oliver/internal/prefix"
	"github.com/robertpjenkins-bot/oliver/internal/runner"
)

// runFlags are the options shared by `run` and `why`.
type runFlags struct {
	backendName string
	prefixName  string
	profile     string
	winver      string
	workdir     string
	env         stringList
	dryRun      bool
	verbose     bool
	quiet       bool
	asJSON      bool
}

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

func (a *app) bindRunFlags(fs *flag.FlagSet, f *runFlags) {
	fs.StringVar(&f.backendName, "backend", "", "runtime to use, by name or id (default: chosen automatically)")
	fs.StringVar(&f.backendName, "b", "", "shorthand for -backend")
	fs.StringVar(&f.prefixName, "prefix", "", "name or path of the Windows environment to use")
	fs.StringVar(&f.prefixName, "p", "", "shorthand for -prefix")
	fs.StringVar(&f.profile, "profile", "", "auto, game or app (default: auto)")
	fs.StringVar(&f.winver, "winver", "", "Windows version to report, or auto (default: auto)")
	fs.StringVar(&f.workdir, "workdir", "", "working directory (default: the program's own directory)")
	fs.Var(&f.env, "env", "extra environment variable as KEY=VALUE (repeatable)")
	fs.Var(&f.env, "e", "shorthand for -env")
	fs.BoolVar(&f.dryRun, "dry-run", false, "print the exact commands instead of running them")
	fs.BoolVar(&f.dryRun, "n", false, "shorthand for -dry-run")
	fs.BoolVar(&f.verbose, "verbose", false, "keep Wine's diagnostic output")
	fs.BoolVar(&f.verbose, "v", false, "shorthand for -verbose")
	fs.BoolVar(&f.quiet, "quiet", false, "suppress Oliver's own progress output")
	fs.BoolVar(&f.asJSON, "json", false, "print the resolved launch plan as JSON")
}

const runSynopsis = `oliver run [flags] <program.exe> [program arguments...]

Runs a Windows program. Oliver inspects the executable, ranks every
compatibility runtime installed on this system, and launches the best match in
an isolated Windows environment of its own.`

func (a *app) cmdRun(args []string) error {
	var f runFlags
	fs := a.newFlagSet("run", runSynopsis)
	a.bindRunFlags(fs, &f)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("no program given")
	}

	a.ui.Quiet = f.quiet
	target := fs.Arg(0)
	programArgs := fs.Args()[1:]

	res, err := a.resolve(target, &f)
	if err != nil {
		return err
	}
	res.opts.Args = programArgs

	plan, err := runner.Build(res.opts)
	if err != nil {
		return err
	}

	if f.asJSON {
		return a.printJSON(map[string]any{
			"executable": res.info,
			"backend":    res.candidate.Backend,
			"reasons":    res.candidate.Reasons,
			"caveats":    compat.Check(res.info),
			"prefix":     res.prefix.Dir,
			"plan":       plan,
		})
	}

	if f.dryRun {
		a.ui.Printf("%s %s", a.ui.Bold("program:"), res.info.Path)
		a.ui.Printf("%s %s", a.ui.Bold("runtime:"), res.candidate.Backend.String())
		a.ui.Printf("%s %s", a.ui.Bold("prefix: "), res.prefix.Dir)
		a.ui.Printf("")
		for _, s := range plan.Setup {
			a.ui.Printf("%s %s", a.ui.Dim("#"), a.ui.Dim(s.Desc))
			a.ui.Printf("%s", s.Pretty())
		}
		a.ui.Printf("%s %s", a.ui.Dim("#"), a.ui.Dim(plan.Launch.Desc))
		a.ui.Printf("%s", plan.Launch.Pretty())
		return nil
	}

	// Say what is known to be broken before launching, not after it fails in a
	// way that looks like the runner's fault.
	if caveats := compat.Check(res.info); len(caveats) > 0 {
		a.printCaveats(caveats)
		if compat.Worst(caveats) == compat.Blocking {
			a.ui.Printf("")
			a.ui.Printf("%s", a.ui.Dim("Launching anyway, since you asked. Press Ctrl-C to stop."))
		}
		a.ui.Printf("")
	}

	a.ui.Info("%s %s  %s", a.ui.Green("→"), res.info.Name, a.ui.Dim("via "+res.candidate.Backend.String()))
	if plan.NewPrefix {
		a.ui.Info("· creating a new Windows environment at %s", res.prefix.Dir)
	}
	for _, note := range res.candidate.Backend.Notes {
		a.ui.Info("%s %s", a.ui.Dim("·"), a.ui.Dim(note))
	}

	// Record how this program was launched before running it, so a crash still
	// leaves the environment documented.
	if res.prefix.Meta.Created.IsZero() {
		res.prefix.Meta.Created = time.Now()
	}
	res.prefix.Meta.Source = res.info.Path
	res.prefix.Meta.LastBackend = res.candidate.Backend.ID
	res.prefix.Meta.LastRun = time.Now()
	if plan.WineArch != "" && res.prefix.Meta.Arch == "" {
		res.prefix.Meta.Arch = plan.WineArch
	}
	res.prefix.Meta.WindowsVer = plan.WinVer
	if err := res.prefix.SaveMeta(); err != nil {
		a.ui.Warn("could not record prefix metadata: %v", err)
	}

	logPath := runner.LogPathFor(res.prefix.Name)
	result, err := runner.Execute(plan, res.prefix, logPath, a.ui.Info)
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			a.reportFailure(res, result)
			// The program ran and chose to fail. That is not Oliver's error.
			return &exitError{code: ee.ExitCode(), err: err}
		}
		return fmt.Errorf("launching %s: %w", res.info.Name, err)
	}
	return nil
}

// reportFailure explains a program that died on startup. Launched from a file
// manager this is otherwise invisible: a spinner appears, then nothing.
func (a *app) reportFailure(res *resolution, result *runner.Result) {
	if result == nil {
		return
	}
	u := a.ui

	if result.Vanished {
		u.Warn("%s started and exited after %s — it failed during startup, it did not run",
			res.info.Name, result.Duration.Round(time.Millisecond))
	}

	lines := runner.Diagnose(result.LogPath, 12)
	if len(lines) > 0 {
		u.Info("")
		u.Info("%s", u.Bold("Last output before it stopped:"))
		for _, l := range lines {
			u.Info("  %s", u.Dim(l))
		}
	}
	u.Info("")
	u.Info("%s", u.Dim("full log: "+result.LogPath))
	u.Info("%s", u.Dim("see it again with: oliver log"))

	// With no terminal attached the above went nowhere the user will look.
	if !u.Interactive {
		body := fmt.Sprintf("%s failed to start.\n\n%s\n\nFull log:\n%s",
			res.info.Name, strings.Join(lines, "\n"), result.LogPath)
		dialogError("Oliver could not run "+res.info.Name, body)
	}
}

// resolution is everything needed to launch, after config, flags and ranking
// have been combined.
type resolution struct {
	info      *pe.Info
	backends  []*backend.Backend
	candidate *backend.Candidate
	prefix    *prefix.Prefix
	opts      runner.Options
	ranked    []*backend.Candidate
}

// resolve inspects the target, applies configuration and flags, and picks a
// runtime. Precedence is: command-line flag, then a matching config rule, then
// the global config default.
func (a *app) resolve(target string, f *runFlags) (*resolution, error) {
	info, err := pe.Inspect(target)
	if err != nil {
		return nil, err
	}
	for _, w := range info.Warnings {
		a.ui.Warn("%s", w)
	}

	rule := a.cfg.RuleFor(info.Path)

	pick := func(flagVal, ruleVal, cfgVal string) string {
		if flagVal != "" {
			return flagVal
		}
		if ruleVal != "" {
			return ruleVal
		}
		return cfgVal
	}

	var ruleBackend, ruleProfile, rulePrefix, ruleWinVer string
	var ruleEnv []string
	if rule != nil {
		ruleBackend, ruleProfile, rulePrefix, ruleWinVer, ruleEnv =
			rule.Backend, rule.Profile, rule.Prefix, rule.WinVer, rule.Env
		a.ui.Info("%s", a.ui.Dim("· applying configuration rule "+rule.Match))
	}

	profileName := pick(f.profile, ruleProfile, a.cfg.Profile)
	profile, err := backend.ParseProfile(profileName)
	if err != nil {
		return nil, err
	}

	scanner := backend.NewScanner()
	scanner.ExtraRoots = a.cfg.RunnerDirs
	backends := scanner.Discover()
	if len(backends) == 0 {
		return nil, fmt.Errorf("no Windows-compatibility runtime is installed; run `oliver doctor` to see what to install")
	}

	prefer := pick(f.backendName, ruleBackend, a.cfg.Backend)
	req := backend.Request{Info: info, Profile: profile, Prefer: prefer}
	ranked := backend.Rank(backends, req)

	var candidate *backend.Candidate
	if prefer != "" {
		// An explicit choice must be honoured or reported, never silently
		// replaced with something else.
		b, err := backend.Find(backends, prefer)
		if err != nil {
			return nil, err
		}
		for _, c := range ranked {
			if c.Backend == b {
				candidate = c
			}
		}
		if candidate != nil && !candidate.Compatible {
			return nil, fmt.Errorf("%s cannot run %s: %s",
				b.Name, info.Name, strings.Join(candidate.Blockers, "; "))
		}
	}
	if candidate == nil {
		candidate, err = backend.Best(backends, req)
		if err != nil {
			return nil, err
		}
	}

	mgr := prefix.NewManager()
	if a.cfg.PrefixRoot != "" {
		mgr.Root = a.cfg.PrefixRoot
	}
	pfx := mgr.For(info.Path, pick(f.prefixName, rulePrefix, ""))

	env := append(append([]string{}, a.cfg.Env...), ruleEnv...)
	env = append(env, f.env...)

	winver := pick(f.winver, ruleWinVer, a.cfg.WinVer)
	if winver == "" {
		winver = "auto"
	}

	return &resolution{
		info:      info,
		backends:  backends,
		candidate: candidate,
		prefix:    pfx,
		ranked:    ranked,
		opts: runner.Options{
			Info:      info,
			Backend:   candidate.Backend,
			Prefix:    pfx,
			WorkDir:   f.workdir,
			WinVer:    winver,
			Verbose:   f.verbose,
			Env:       env,
			SteamRoot: steamRoot(),
		},
	}, nil
}

// steamRoot finds a Steam installation for Proton's client-install path.
func steamRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, c := range []string{".steam/steam", ".local/share/Steam", ".steam/root"} {
		p := home + "/" + c
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

func (a *app) printJSON(v any) error {
	enc := json.NewEncoder(a.ui.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
