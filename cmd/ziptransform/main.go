// Command ziptransform turns an MCreator project, exported as a shareable
// ZIP, into a mod jar — with one of the project's worlds inside it if you
// want, which then appears in the world list of anyone who installs the mod.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/mcreator"
	"github.com/robertpjenkins-bot/oliver/internal/minecraft"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `ziptransform - turn an MCreator project ZIP into a mod .jar

USAGE
  ziptransform                        open the Zip Transform window
  ziptransform <project.zip>          build it (opens the window when started
                                      from a file manager rather than a terminal)
  ziptransform build <project.zip>    build from the terminal
  ziptransform info <project.zip>     show what is in the ZIP
  ziptransform setup                  show the MCreator and Java builds will use
  ziptransform install <mod.jar>      put a mod into Minecraft, switched on
  ziptransform instances              list the Minecraft instances mods can go into
  ziptransform desktop install        add Zip Transform to the applications menu

Make the ZIP in MCreator, from the File menu:
  Export workspace to a shareable ZIP...        the mod on its own
  Export to a shareable ZIP (with run dir)...   the mod and the worlds you
                                                played while testing it

Zip Transform builds with the MCreator installed on this computer: it borrows
MCreator's build files, its Java and its download cache, so the jar is the one
MCreator's own "Export the mod for distribution" makes. Drop a .jar on the
window instead and it goes into the mods folder of the Minecraft you pick, in
Prism Launcher, the Minecraft Launcher and others.

Run "ziptransform build -h" for the build options.
`

func main() {
	u := ui.New()
	u.ErrorTitle = "Zip Transform could not finish"
	if err := run(u, os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		u.Error(err)
		var ge *mcreator.GradleError
		if errors.As(err, &ge) {
			if hint := ge.Hint(); hint != "" {
				u.Info("%s", hint)
			}
		}
		os.Exit(1)
	}
}

func run(u *ui.UI, args []string) error {
	if len(args) == 0 {
		return cmdWindow(u, nil)
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(u.Out, usage)
		return nil
	case "-V", "--version", "version":
		fmt.Fprintf(u.Out, "ziptransform %s\n", version)
		return nil
	case "window", "gui":
		return cmdWindow(u, args[1:])
	case "build":
		return cmdBuild(u, args[1:])
	case "info":
		return cmdInfo(u, args[1:])
	case "setup":
		return cmdSetup(u, args[1:])
	case "install":
		return cmdInstall(u, args[1:])
	case "instances":
		return cmdInstances(u, args[1:])
	case "desktop":
		return cmdDesktop(u, args[1:])
	}

	// Flags alone are the window's: `ziptransform -mcreator ~/MCreator20263`.
	if strings.HasPrefix(args[0], "-") {
		return cmdWindow(u, args)
	}
	// A bare file is what a file manager passes when a ZIP is dropped on the
	// app or opened with it. With nobody at a terminal to answer questions,
	// that means the window.
	if _, err := os.Stat(args[0]); err != nil {
		if strings.Contains(args[0], ".") || strings.ContainsRune(args[0], filepath.Separator) {
			return err // meant as a file
		}
		return fmt.Errorf("unknown command %q (run `ziptransform help`)", args[0])
	}
	if !u.Interactive {
		return cmdWindow(u, args)
	}
	if isModPath(args[0]) {
		return cmdInstall(u, args)
	}
	return cmdBuild(u, args)
}

// buildFlags are shared by the commands that need to find MCreator.
type buildFlags struct {
	mcreator string
}

func (b *buildFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&b.mcreator, "mcreator", "", "the MCreator `folder` to build with (remembered for next time)")
}

// toolkit finds MCreator, preferring a folder given on the command line, then
// the one remembered from last time.
func (b *buildFlags) toolkit(u *ui.UI) (*mcreator.Toolkit, error) {
	cfg := loadConfig()
	if b.mcreator != "" {
		inst, err := mcreator.CheckInstall(b.mcreator)
		if err != nil {
			return nil, err
		}
		if cfg.MCreator != inst.Dir {
			cfg.MCreator = inst.Dir
			if err := cfg.save(); err != nil {
				u.Warn("could not remember the MCreator folder: %v", err)
			}
		}
	}
	return mcreator.FindToolkit(cfg.MCreator), nil
}

const buildSynopsis = `ziptransform build [flags] <project.zip>

Builds the mod in an MCreator shareable ZIP into a jar. When the ZIP holds
worlds and neither -world nor -no-world is given, you are asked whether to
include one.`

func cmdBuild(u *ui.UI, args []string) error {
	fs := newFlagSet(u, "build", buildSynopsis)
	var bf buildFlags
	bf.register(fs)
	world := fs.String("world", "", "bundle the world with this `name` into the jar")
	noWorld := fs.Bool("no-world", false, "build the mod without a world, without asking")
	out := fs.String("o", "", "write the jar to this `file` (default: beside the ZIP)")
	install := fs.String("install", "", "then add the mod to the Minecraft instance with this `name` (\"auto\" for the one that fits)")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return flag.ErrHelp
	}

	p, err := mcreator.Open(fs.Arg(0))
	if err != nil {
		return err
	}
	defer p.Close()
	t, err := bf.toolkit(u)
	if err != nil {
		return err
	}
	defer t.Close()

	describe(u, p)

	choice := *world
	switch {
	case *noWorld || choice != "":
	case len(p.Worlds) > 0 && p.CanBundleWorld():
		choice = askWorld(u, p)
	}
	if choice != "" {
		if _, ok := p.World(choice); !ok {
			return fmt.Errorf("the ZIP has no world called %q (it has: %s)", choice, worldNames(p))
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	res, err := mcreator.Build(ctx, p, t, mcreator.BuildOptions{World: choice, Output: *out, Log: u.Err})
	if err != nil {
		return err
	}

	fmt.Fprintln(u.Err)
	u.Printf("%s %s", u.Green("✓ built"), res.Jar)
	if res.World != "" {
		u.Info("  %q is added to the world list of anyone who plays with this mod, in the folder %q.", choice, res.World)
	}
	if *install == "" {
		return nil
	}
	m, err := minecraft.ReadMod(res.Jar)
	if err != nil {
		return err
	}
	if *install == "auto" {
		*install = ""
	}
	return installMod(u, res.Jar, m, *install)
}

// askWorld asks which world to bundle. Without a terminal there is nobody to
// ask, so no world is bundled.
func askWorld(u *ui.UI, p *mcreator.Project) string {
	if !u.Interactive {
		u.Info("not bundling a world (use -world NAME to include one)")
		return ""
	}
	if len(p.Worlds) == 1 {
		w := p.Worlds[0]
		if u.Confirm(fmt.Sprintf("Include your world %q (%s) in the jar?", w.Name, mcreator.HumanSize(w.Size)), false) {
			return w.Name
		}
		return ""
	}
	fmt.Fprintln(u.Err, "Include a world in the jar?")
	fmt.Fprintln(u.Err, "  0) no, just the mod")
	for i, w := range p.Worlds {
		fmt.Fprintf(u.Err, "  %d) %s (%s)\n", i+1, w.Name, mcreator.HumanSize(w.Size))
	}
	for {
		fmt.Fprint(u.Err, "Choose a number: ")
		var answer string
		if _, err := fmt.Fscanln(os.Stdin, &answer); err != nil {
			return ""
		}
		n, err := strconv.Atoi(strings.TrimSpace(answer))
		if err == nil && n >= 0 && n <= len(p.Worlds) {
			if n == 0 {
				return ""
			}
			return p.Worlds[n-1].Name
		}
	}
}

func worldNames(p *mcreator.Project) string {
	if len(p.Worlds) == 0 {
		return "no worlds"
	}
	names := make([]string, len(p.Worlds))
	for i, w := range p.Worlds {
		names[i] = strconv.Quote(w.Name)
	}
	return strings.Join(names, ", ")
}

func describe(u *ui.UI, p *mcreator.Project) {
	u.Info("%s %s  %s", u.Bold(p.ModName), p.Version, u.Dim(p.GeneratorLabel()))
}

const infoSynopsis = `ziptransform info <project.zip>

Shows the mod in an MCreator shareable ZIP and the worlds it holds.`

func cmdInfo(u *ui.UI, args []string) error {
	fs := newFlagSet(u, "info", infoSynopsis)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return flag.ErrHelp
	}
	p, err := mcreator.Open(fs.Arg(0))
	if err != nil {
		return err
	}
	defer p.Close()

	u.Printf("%s", u.Bold(p.ModName))
	u.Printf("  mod ID     %s", p.ModID)
	u.Printf("  version    %s", p.Version)
	u.Printf("  made for   %s (%s)", p.GeneratorLabel(), p.Generator)
	if len(p.APIs) > 0 {
		u.Printf("  mod APIs   %s", strings.Join(p.APIs, ", "))
	}
	if !p.MakesJar() {
		u.Printf("  %s", u.Yellow("this kind of project does not build into a jar"))
		return nil
	}
	u.Printf("  jar        %s", p.JarName())
	u.Printf("")
	if len(p.Worlds) == 0 {
		u.Printf("No worlds. To include one, export with File → Export to a shareable ZIP (with run dir)...")
		return nil
	}
	t := u.NewTable("WORLD", "SIZE", "FILES")
	for _, w := range p.Worlds {
		t.Row(w.Name, mcreator.HumanSize(w.Size), strconv.Itoa(w.Files))
	}
	t.Render()
	return nil
}

const setupSynopsis = `ziptransform setup [-mcreator DIR]

Shows the MCreator installation and Java that builds will use, and the
Minecraft versions it can build for.`

func cmdSetup(u *ui.UI, args []string) error {
	fs := newFlagSet(u, "setup", setupSynopsis)
	var bf buildFlags
	bf.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	t, err := bf.toolkit(u)
	if err != nil {
		return err
	}
	defer t.Close()

	if len(t.Installs) == 0 {
		u.Printf("%s", u.Yellow("MCreator was not found."))
		u.Printf("Point Zip Transform at the folder you unpacked it to:")
		u.Printf("  ziptransform setup -mcreator ~/MCreator20263")
		return nil
	}
	for i, inst := range t.Installs {
		label := "MCreator"
		if inst.Version != "" {
			label += " " + inst.Version
		}
		marker := " "
		if i == 0 {
			marker = u.Green("→")
		}
		u.Printf("%s %s  %s", marker, u.Bold(label), inst.Dir)
	}
	if java, err := t.Java(); err == nil {
		u.Printf("  Java       %s", java)
	} else {
		u.Printf("  Java       %s", u.Red(err.Error()))
	}
	u.Printf("  cache      %s", t.GradleHome())
	if gens := t.Generators(); len(gens) > 0 {
		u.Printf("  builds for %s", strings.Join(gens, ", "))
	}
	return nil
}

func newFlagSet(u *ui.UI, name, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(u.Err)
	fs.Usage = func() {
		fmt.Fprintf(u.Err, "%s\n\nFLAGS\n", synopsis)
		fs.PrintDefaults()
	}
	return fs
}

// parseFlagsAnywhere parses flags that appear after the ZIP too, since
// `ziptransform build mod.zip -world "New World"` is the order people type.
func parseFlagsAnywhere(fs *flag.FlagSet, args []string) error {
	var flags, positional []string
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
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if strings.Contains(name, "=") {
			continue
		}
		if f := fs.Lookup(name); f != nil {
			if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
				continue
			}
			if i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
		}
	}
	return fs.Parse(append(flags, positional...))
}
