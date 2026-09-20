// Command oliver runs Windows programs on Linux by choosing the best of the
// compatibility runtimes already installed on the system.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/config"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

// newUI constructs the output layer. It is a variable so tests can capture
// what the commands write.
var newUI = ui.New

// app carries state shared by every subcommand.
type app struct {
	ui   *ui.UI
	cfg  *config.Config
	json bool
}

const usage = `oliver - run Windows programs with the best runtime you have installed

USAGE
  oliver [command] [flags] [arguments]
  oliver <program.exe> [arguments]     shorthand for "oliver run"

COMMANDS
  run          run a Windows program, choosing a runtime automatically
  info         show what an executable is and what it needs
  why          explain how every installed runtime scores for a program
  backends     list the compatibility runtimes found on this system
  prefix       list, inspect and remove the Windows environments Oliver made
  ui           open this machine's drives in a browser
  drives       show this machine's drives and what is on them
  write        write a disc image to a drive, and verify it arrived
  erase        erase a drive and leave an empty one behind
  format       put one empty FAT32 volume on a drive
  verify       check a drive against the image that should be on it
  device       expose one of this machine's drives to a Windows program
  doctor       check this system and report what is missing
  log          show the output of the last program that ran
  report       write everything needed to diagnose a problem to one file
  pack         wrap a Windows program into an AppImage
  which        find an installed program inside a Windows environment
  desktop      register Oliver so .exe files open with a double-click
  config       show or create the configuration file
  version      print the version

Run "oliver <command> -h" for the flags a command accepts.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		if errors.Is(err, errCancelled) {
			// Declining a prompt is not a failure to report as one.
			fmt.Fprintln(os.Stderr, "cancelled")
			os.Exit(1)
		}
		var ee *exitError
		if errors.As(err, &ee) {
			// The program itself failed; pass its status through unchanged so
			// scripts calling oliver see what they would have seen on Windows.
			os.Exit(ee.code)
		}
		newUI().Error(err)
		os.Exit(1)
	}
}

// exitError propagates a launched program's exit status.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

func run(args []string) error {
	a := &app{ui: newUI()}

	if len(args) == 0 {
		fmt.Fprint(os.Stdout, usage)
		return nil
	}

	cmd := args[0]
	rest := args[1:]

	// Being handed a file directly is the file-manager case: treat it as a run.
	if looksLikeProgram(cmd) {
		cmd, rest = "run", args
	}

	switch cmd {
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return nil
	case "-V", "--version", "version":
		fmt.Printf("oliver %s\n", version)
		return nil
	}

	cfg, err := config.Load(os.Getenv("OLIVER_CONFIG"))
	if err != nil {
		// A broken config must not silently change how programs are launched.
		return fmt.Errorf("reading configuration: %w", err)
	}
	a.cfg = cfg

	switch cmd {
	case "run":
		return a.cmdRun(rest)
	case "info":
		return a.cmdInfo(rest)
	case "why":
		return a.cmdWhy(rest)
	case "backends", "runners":
		return a.cmdBackends(rest)
	case "prefix", "prefixes":
		return a.cmdPrefix(rest)
	case "device", "devices":
		return a.cmdDevice(rest)
	case "ui", "gui", "web":
		return a.cmdUI(rest)
	case "drives", "disks":
		return a.deviceList(a.wantsJSON(rest))
	case "write", "burn", "flash":
		return a.cmdWrite(rest)
	case "erase", "wipe":
		return a.cmdErase(rest)
	case "format":
		return a.cmdFormat(rest)
	case "verify":
		return a.cmdVerify(rest)
	case "doctor":
		return a.cmdDoctor(rest)
	case "log", "logs":
		return a.cmdLog(rest)
	case "report":
		return a.cmdReport(rest)
	case "pack":
		return a.cmdPack(rest)
	case "which":
		return a.cmdWhich(rest)
	case "desktop":
		return a.cmdDesktop(rest)
	case "desktop-welcome":
		// Not advertised: the AppImage launcher uses it when opened with no
		// arguments from a desktop menu.
		return a.cmdDesktopWelcome(rest)
	case "config":
		return a.cmdConfig(rest)
	default:
		return fmt.Errorf("unknown command %q (run `oliver help`)", cmd)
	}
}

// looksLikeProgram reports whether an argument is a program to run rather than
// a subcommand name.
func looksLikeProgram(arg string) bool {
	if strings.HasPrefix(arg, "-") {
		return false
	}
	switch strings.ToLower(filepath.Ext(arg)) {
	case ".exe", ".msi", ".bat", ".cmd":
		return true
	}
	// An existing path that is not one of our command names.
	if _, err := os.Stat(arg); err == nil && strings.ContainsRune(arg, filepath.Separator) {
		return true
	}
	return false
}

// parseFlagsAnywhere parses flags that appear after positional arguments too.
//
// Go's flag package stops at the first non-flag argument, so `oliver pack
// app.exe -o out` would silently ignore -o. That is the order people naturally
// type. Commands that forward trailing arguments to the Windows program —
// `run` above all — must not use this, since there the trailing flags belong
// to the program.
func parseFlagsAnywhere(fs *flag.FlagSet, args []string) error {
	var flags, positional []string
	defined := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) { defined[f.Name] = true })

	// A flag that takes a value may be written as "-o out" or "-o=out"; only
	// the first form consumes the next argument.
	takesValue := func(name string) bool {
		f := fs.Lookup(name)
		if f == nil {
			return false
		}
		// Boolean flags are the ones that do not consume a following value.
		bf, ok := f.Value.(interface{ IsBoolFlag() bool })
		return !ok || !bf.IsBoolFlag()
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		name := strings.TrimLeft(arg, "-")
		value := ""
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			name, value = name[:idx], name[idx:]
		}
		flags = append(flags, arg)
		if value == "" && takesValue(name) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return fs.Parse(append(flags, positional...))
}

// newFlagSet builds a flag set that prints a command-specific usage block.
func (a *app) newFlagSet(name, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\nFLAGS\n", synopsis)
		fs.PrintDefaults()
	}
	return fs
}

// wantsJSON reports whether -json appears among a command's arguments, for the
// aliases that take no other flags.
func (a *app) wantsJSON(args []string) bool {
	for _, arg := range args {
		if arg == "-json" || arg == "--json" {
			return true
		}
	}
	return false
}
