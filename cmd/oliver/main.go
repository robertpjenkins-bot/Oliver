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
  device       show this machine's drives, and expose one to a program
  doctor       check this system and report what is missing
  log          show the output of the last program that ran
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
	case "device", "devices", "drives":
		return a.cmdDevice(rest)
	case "doctor":
		return a.cmdDoctor(rest)
	case "log", "logs":
		return a.cmdLog(rest)
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

// newFlagSet builds a flag set that prints a command-specific usage block.
func (a *app) newFlagSet(name, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\nFLAGS\n", synopsis)
		fs.PrintDefaults()
	}
	return fs
}
