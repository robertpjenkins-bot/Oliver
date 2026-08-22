// Package runner turns a chosen backend and prefix into the exact command
// that launches a Windows program.
package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/backend"
	"github.com/robertpjenkins-bot/oliver/internal/pe"
	"github.com/robertpjenkins-bot/oliver/internal/prefix"
)

// Step is one command in a launch plan.
type Step struct {
	Desc string   `json:"description"`
	Argv []string `json:"argv"`
	Env  []string `json:"env,omitempty"`
	Dir  string   `json:"dir,omitempty"`
	// Optional steps may fail without failing the launch.
	Optional bool `json:"optional,omitempty"`
}

// String renders the step as a shell-ish line for --dry-run.
func (s Step) String() string {
	var b strings.Builder
	for _, e := range s.Env {
		b.WriteString(shellQuote(e))
		b.WriteByte(' ')
	}
	for i, a := range s.Argv {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(shellQuote(a))
	}
	return b.String()
}

// Pretty renders the step across several lines, one environment variable per
// line with shell continuations, so a long launch stays readable and can still
// be pasted into a terminal as-is.
func (s Step) Pretty() string {
	var b strings.Builder
	for _, e := range s.Env {
		b.WriteString("  ")
		b.WriteString(shellQuote(e))
		b.WriteString(" \\\n")
	}
	if len(s.Env) > 0 {
		b.WriteString("  ")
	}
	for i, a := range s.Argv {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(shellQuote(a))
	}
	return b.String()
}

// Options describes what to launch and how.
type Options struct {
	Info    *pe.Info
	Backend *backend.Backend
	Prefix  *prefix.Prefix
	Args    []string
	WorkDir string

	// WinVer sets the Windows version the prefix reports. "auto" derives it
	// from the executable's own subsystem version.
	WinVer string
	// Verbose keeps Wine's diagnostic output instead of silencing it.
	Verbose bool
	// Env holds extra variables, as KEY=VALUE.
	Env []string
	// SteamRoot is used for Proton's client-install path.
	SteamRoot string
}

// Plan is a fully resolved launch: setup steps, then the program itself.
type Plan struct {
	Setup  []Step `json:"setup,omitempty"`
	Launch Step   `json:"launch"`
	// NewPrefix reports that the prefix did not exist and will be created.
	NewPrefix bool   `json:"new_prefix"`
	WineArch  string `json:"wine_arch,omitempty"`
	WinVer    string `json:"windows_version,omitempty"`
}

// Build resolves options into a plan without running anything, so the same
// code path serves both --dry-run and a real launch.
func Build(o Options) (*Plan, error) {
	if o.Backend == nil {
		return nil, fmt.Errorf("no backend selected")
	}
	if o.Info == nil {
		return nil, fmt.Errorf("no executable to run")
	}
	if o.Prefix == nil {
		return nil, fmt.Errorf("no prefix selected")
	}

	plan := &Plan{NewPrefix: !o.Prefix.Exists()}

	// A prefix's architecture is fixed when it is created. A 32-bit-only
	// prefix is the reliable way to run a 32-bit program, but it can never be
	// upgraded, so only choose one for a genuinely 32-bit target.
	if plan.NewPrefix && o.Info.Arch == pe.ArchX86 && o.Backend.Kind != backend.KindProton {
		plan.WineArch = "win32"
	} else if plan.NewPrefix {
		plan.WineArch = "win64"
	} else if o.Prefix.Meta.Arch != "" {
		plan.WineArch = o.Prefix.Meta.Arch
	}

	plan.WinVer = resolveWinVer(o)

	env := buildEnv(o, plan)

	workDir := o.WorkDir
	if workDir == "" {
		// Windows programs overwhelmingly expect to start in their own
		// directory; running from anywhere else breaks asset loading.
		workDir = filepath.Dir(o.Info.Path)
	}

	if plan.NewPrefix && o.Backend.Kind != backend.KindProton {
		// Initialise the prefix explicitly so the sandbox and Windows version
		// can be applied before the program ever runs.
		boot := append(wineArgv(o.Backend), "wineboot", "--init")
		plan.Setup = append(plan.Setup, Step{
			Desc: "create the Wine prefix",
			Argv: boot,
			Env:  env,
			Dir:  workDir,
		})
	}
	if plan.WinVer != "" && o.Backend.Kind != backend.KindProton {
		reg := append(wineArgv(o.Backend), "reg", "add", `HKCU\Software\Wine`, "/v", "Version", "/d", plan.WinVer, "/f")
		plan.Setup = append(plan.Setup, Step{
			Desc:     "report Windows " + plan.WinVer + " to the program",
			Argv:     reg,
			Env:      env,
			Dir:      workDir,
			Optional: true,
		})
	}

	launchArgv, err := launchCommand(o)
	if err != nil {
		return nil, err
	}
	plan.Launch = Step{
		Desc: "run " + o.Info.Name,
		Argv: launchArgv,
		Env:  env,
		Dir:  workDir,
	}
	return plan, nil
}

// wineArgv is the command prefix that invokes this backend's wine binary.
func wineArgv(b *backend.Backend) []string {
	switch {
	case b.Source == backend.SourceFlatpak && len(b.Launcher) > 0:
		// The flatpak wrapper already names the binary to run.
		return append([]string{}, b.Launcher...)
	case b.Kind == backend.KindBox && len(b.Launcher) > 0:
		// A translation layer wraps the real binary.
		return append(append([]string{}, b.Launcher...), b.Exec)
	default:
		return []string{b.Exec}
	}
}

// launchCommand builds the argv that starts the target program.
func launchCommand(o Options) ([]string, error) {
	target := o.Info.Path

	if o.Backend.Kind == backend.KindProton {
		if len(o.Backend.Launcher) > 0 {
			// umu-run takes the executable directly and sets up the Steam
			// runtime container itself.
			return append(append(append([]string{}, o.Backend.Launcher...), target), o.Args...), nil
		}
		// Without umu, Proton's own entry point is the next best thing.
		return append([]string{o.Backend.Exec, "run", target}, o.Args...), nil
	}

	argv := wineArgv(o.Backend)
	switch o.Info.Kind {
	case pe.KindMSI:
		// An MSI is data, not a program: it has to be handed to the installer.
		return append(append(argv, "msiexec", "/i", target), o.Args...), nil
	case pe.KindBatch:
		return append(append(argv, "cmd", "/c", target), o.Args...), nil
	case pe.KindDOS:
		return nil, fmt.Errorf("%s is a 16-bit DOS program; Wine cannot run it (try DOSBox)", o.Info.Name)
	}
	return append(append(argv, target), o.Args...), nil
}

// resolveWinVer decides which Windows version the prefix should report.
func resolveWinVer(o Options) string {
	switch strings.ToLower(o.WinVer) {
	case "", "none", "off":
		return ""
	case "auto":
		if o.Info.Packed {
			// The header belongs to the packer stub, not the program, so its
			// target version says nothing about what the program expects.
			return ""
		}
		// Only worth overriding when the program targets something older than
		// the modern default; claiming to be newer than the program expects is
		// what causes "unsupported Windows version" dialogs.
		switch o.Info.TargetWindows {
		case "winxp", "vista", "win7":
			return o.Info.TargetWindows
		}
		return ""
	default:
		return o.WinVer
	}
}

// buildEnv assembles the environment for every step of the launch.
func buildEnv(o Options, plan *Plan) []string {
	env := map[string]string{
		"WINEPREFIX": o.Prefix.WinePrefix,
	}

	if plan.NewPrefix && plan.WineArch != "" && o.Backend.Kind != backend.KindProton {
		// WINEARCH is only meaningful while the prefix is being created, and
		// setting it against an existing prefix makes Wine refuse to start.
		env["WINEARCH"] = plan.WineArch
	}

	if o.Verbose {
		env["WINEDEBUG"] = "fixme-all"
	} else {
		env["WINEDEBUG"] = "-all"
	}

	// winemenubuilder writes a .desktop file into the real application menu
	// for every shortcut a Windows installer creates. Disabling it is the
	// difference between a tidy system and a menu full of junk entries.
	overrides := []string{"winemenubuilder.exe=d"}
	if !o.Info.DotNET {
		// Suppress the Mono and Gecko install prompts, which otherwise block
		// the first launch of any program that does not need them.
		overrides = append(overrides, "mscoree=d", "mshtml=d")
	}
	env["WINEDLLOVERRIDES"] = strings.Join(overrides, ";")

	if o.Backend.Kind == backend.KindProton {
		env["STEAM_COMPAT_DATA_PATH"] = o.Prefix.Dir
		steamRoot := o.SteamRoot
		if steamRoot == "" {
			steamRoot = o.Prefix.Dir
		}
		env["STEAM_COMPAT_CLIENT_INSTALL_PATH"] = steamRoot
		env["PROTONPATH"] = o.Backend.Root
		// umu identifies the title being run; 0 is its documented default for
		// programs that are not Steam games.
		env["GAMEID"] = "0"
		env["PROTON_VERB"] = "run"
	}

	for _, kv := range o.Env {
		if k, v, ok := strings.Cut(kv, "="); ok {
			env[k] = v
		}
	}

	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	sort.Strings(out) // deterministic, which makes --dry-run diffable
	return out
}

// Execute runs a plan, streaming output to the caller's terminal.
func Execute(plan *Plan, p *prefix.Prefix, log func(string, ...any)) error {
	if err := p.Prepare(); err != nil {
		return fmt.Errorf("preparing prefix: %w", err)
	}

	for _, step := range plan.Setup {
		log("· %s", step.Desc)
		if err := runStep(step); err != nil {
			if step.Optional {
				log("  (skipped: %v)", err)
				continue
			}
			return fmt.Errorf("%s: %w", step.Desc, err)
		}
	}

	// Isolate the prefix from the real home directory as soon as it exists,
	// and before the program itself gets a chance to write anything.
	if plan.NewPrefix {
		if isolated, err := p.Sandbox(); err != nil {
			log("  (could not isolate home folders: %v)", err)
		} else if len(isolated) > 0 {
			log("· isolated %d folders from your real home directory", len(isolated))
		}
	}

	cmd := exec.Command(plan.Launch.Argv[0], plan.Launch.Argv[1:]...)
	cmd.Env = mergeEnv(os.Environ(), plan.Launch.Env)
	cmd.Dir = plan.Launch.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runStep(s Step) error {
	cmd := exec.Command(s.Argv[0], s.Argv[1:]...)
	cmd.Env = mergeEnv(os.Environ(), s.Env)
	cmd.Dir = s.Dir
	cmd.Stdout = os.Stderr // setup chatter must not pollute the program's stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// mergeEnv overlays overrides onto a base environment, replacing duplicates.
func mergeEnv(base, overrides []string) []string {
	index := make(map[string]int, len(base))
	out := append([]string{}, base...)
	for i, kv := range out {
		if k, _, ok := strings.Cut(kv, "="); ok {
			index[k] = i
		}
	}
	for _, kv := range overrides {
		k, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if i, exists := index[k]; exists {
			out[i] = kv
			continue
		}
		index[k] = len(out)
		out = append(out, kv)
	}
	return out
}

var shellSafe = func(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("-_./=:+,@", r):
		default:
			return false
		}
	}
	return true
}

func shellQuote(s string) string {
	if shellSafe(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
