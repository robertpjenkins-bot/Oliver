package main

import (
	"fmt"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/backend"
	"github.com/robertpjenkins-bot/oliver/internal/pe"
)

const infoSynopsis = `oliver info <program.exe>

Reports what an executable is, what it needs to run, and which runtime Oliver
would choose for it. Nothing is executed.`

func (a *app) cmdInfo(args []string) error {
	fs := a.newFlagSet("info", infoSynopsis)
	asJSON := fs.Bool("json", false, "print the report as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("no program given")
	}

	info, err := pe.Inspect(fs.Arg(0))
	if err != nil {
		return err
	}
	if *asJSON {
		return a.printJSON(info)
	}

	u := a.ui
	u.Printf("%s", u.Bold(info.Name))
	u.Printf("  %s %s", u.Dim(field("path")), info.Path)
	u.Printf("  %s %s", u.Dim(field("type")), describeKind(info))
	u.Printf("  %s %s", u.Dim(field("size")), humanBytes(info.Size))

	if info.Kind == pe.KindPE {
		u.Printf("  %s %s", u.Dim(field("architecture")), string(info.Arch))
		u.Printf("  %s %s", u.Dim(field("interface")), describeSubsystem(info))
		if info.TargetWindows != "" {
			u.Printf("  %s %s", u.Dim(field("targets")), info.TargetWindows)
		}
		if info.DotNET {
			runtime := ".NET Framework (needs Wine Mono, or install .NET in the prefix)"
			if info.DotNETCore {
				runtime = ".NET Core / .NET 5+"
			}
			u.Printf("  %s %s", u.Dim(field("runtime")), runtime)
		}
		if len(info.GraphicsAPIs) > 0 {
			u.Printf("  %s %s", u.Dim(field("graphics")), strings.Join(info.GraphicsAPIs, ", "))
		}
	}
	if info.IsInstaller {
		u.Printf("  %s %s", u.Dim(field("installer")), string(info.Installer))
	}
	if info.LikelyGame {
		u.Printf("  %s %s", u.Dim(field("classified")), "game")
		for _, h := range info.GameHints {
			u.Printf("    %s %s", u.Dim("·"), u.Dim(h))
		}
	}
	for _, w := range info.Warnings {
		u.Printf("  %s %s", u.Yellow("!"), w)
	}

	// Close the loop: knowing what it is only helps if you know what will run it.
	scanner := backend.NewScanner()
	scanner.ExtraRoots = a.cfg.RunnerDirs
	backends := scanner.Discover()
	profile, err := backend.ParseProfile(a.cfg.Profile)
	if err != nil {
		return err
	}
	req := backend.Request{Info: info, Profile: profile, Prefer: a.cfg.Backend}
	u.Printf("")
	if best, err := backend.Best(backends, req); err == nil {
		u.Printf("%s %s %s", u.Bold("would run with"), u.Green(best.Backend.String()),
			u.Dim(fmt.Sprintf("(%s profile)", req.EffectiveProfile())))
		u.Printf("%s", u.Dim("run `oliver why "+info.Name+"` to see how that was decided"))
	} else {
		u.Printf("%s %v", u.Red("cannot run:"), err)
	}
	return nil
}

// field pads an info label so every value starts in the same column.
func field(name string) string { return pad(name, 12) }

func describeKind(i *pe.Info) string {
	switch i.Kind {
	case pe.KindPE:
		if i.Is64 {
			return "Windows executable (64-bit)"
		}
		return "Windows executable (32-bit)"
	case pe.KindMSI:
		return "Windows Installer package"
	case pe.KindDOS:
		return "16-bit DOS program"
	case pe.KindBatch:
		return "Windows batch script"
	}
	return string(i.Kind)
}

func describeSubsystem(i *pe.Info) string {
	switch {
	case i.GUI:
		return "graphical"
	case i.Console:
		return "console"
	}
	return fmt.Sprintf("subsystem %d", i.Subsystem)
}

const whySynopsis = `oliver why <program.exe>

Shows how every installed runtime scores for a program, and why. Useful when
Oliver's choice is not the one you expected.`

func (a *app) cmdWhy(args []string) error {
	var f runFlags
	fs := a.newFlagSet("why", whySynopsis)
	a.bindRunFlags(fs, &f)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("no program given")
	}

	res, err := a.resolve(fs.Arg(0), &f)
	if err != nil {
		return err
	}
	if f.asJSON {
		return a.printJSON(map[string]any{
			"executable": res.info,
			"chosen":     res.candidate.Backend.ID,
			"candidates": res.ranked,
		})
	}

	u := a.ui
	profile := backend.Request{Info: res.info, Profile: mustProfile(a, &f)}.EffectiveProfile()
	u.Printf("%s %s", u.Bold("program:"), res.info.Path)
	u.Printf("%s %s %s", u.Bold("profile:"), profile, u.Dim(profileReason(res.info, profile)))
	u.Printf("")

	for i, c := range res.ranked {
		marker := "  "
		name := c.Backend.String()
		switch {
		case c.Backend == res.candidate.Backend:
			marker, name = u.Green("→ "), u.Green(name)
		case !c.Compatible:
			name = u.Dim(name)
		}
		verdict := fmt.Sprintf("%d points", c.Score)
		if !c.Compatible {
			verdict = u.Red("cannot run this program")
		}
		u.Printf("%s%s  %s", marker, name, u.Dim(verdict))
		u.Printf("    %s", u.Dim(fmt.Sprintf("%s · %s · %s", c.Backend.Kind, c.Backend.Source, c.Backend.Bitness())))
		for _, line := range c.Explain() {
			u.Printf("  %s", u.Dim(line))
		}
		if i < len(res.ranked)-1 {
			u.Printf("")
		}
	}
	return nil
}

func mustProfile(a *app, f *runFlags) backend.Profile {
	name := f.profile
	if name == "" {
		name = a.cfg.Profile
	}
	p, err := backend.ParseProfile(name)
	if err != nil {
		return backend.ProfileAuto
	}
	return p
}

func profileReason(i *pe.Info, p backend.Profile) string {
	switch {
	case i.IsInstaller:
		return "(installer)"
	case p == backend.ProfileGame && len(i.GameHints) > 0:
		return "(" + i.GameHints[0] + ")"
	case p == backend.ProfileGame:
		return "(graphics APIs detected)"
	}
	return "(no game indicators found)"
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < 3 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}
