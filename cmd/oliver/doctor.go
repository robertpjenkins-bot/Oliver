package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/backend"
	"github.com/robertpjenkins-bot/oliver/internal/prefix"
	"github.com/robertpjenkins-bot/oliver/internal/vbox"
)

const doctorSynopsis = `oliver doctor

Checks this system and reports what is present, what is missing, and what to
install to fix it.`

type checkStatus int

const (
	statusOK checkStatus = iota
	statusWarn
	statusFail
)

type check struct {
	Name   string      `json:"name"`
	Status checkStatus `json:"-"`
	Result string      `json:"result"`
	Advice string      `json:"advice,omitempty"`
	State  string      `json:"status"`
}

func (a *app) cmdDoctor(args []string) error {
	fs := a.newFlagSet("doctor", doctorSynopsis)
	asJSON := fs.Bool("json", false, "print the report as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	scanner := backend.NewScanner()
	scanner.ExtraRoots = a.cfg.RunnerDirs
	backends := scanner.Discover()

	checks := []check{
		checkRuntimes(backends),
		checkPlainWine(backends),
		check32Bit(backends),
		checkProtonSupport(backends),
		checkGraphics(),
		checkPrefixRoot(a),
		checkDesktopIntegration(),
		checkVirtualBox(),
		checkArch(),
	}

	if *asJSON {
		for i := range checks {
			checks[i].State = [...]string{"ok", "warning", "problem"}[checks[i].Status]
		}
		return a.printJSON(checks)
	}

	u := a.ui
	worst := statusOK
	for _, c := range checks {
		var mark string
		switch c.Status {
		case statusOK:
			mark = u.Green("ok  ")
		case statusWarn:
			mark = u.Yellow("note")
		case statusFail:
			mark = u.Red("fail")
		}
		if c.Status > worst {
			worst = c.Status
		}
		u.Printf("%s  %s  %s", mark, u.Bold(pad(c.Name, 24)), c.Result)
		if c.Advice != "" {
			u.Printf("      %s", u.Dim(c.Advice))
		}
	}

	u.Printf("")
	switch worst {
	case statusOK:
		u.Printf("%s", u.Green("Everything Oliver needs is in place."))
	case statusWarn:
		u.Printf("%s", u.Yellow("Usable, with the caveats noted above."))
	case statusFail:
		u.Printf("%s", u.Red("Something essential is missing; see the advice above."))
	}
	return nil
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// checkVirtualBox reports on the machinery behind `oliver vm`. It is a note
// rather than a failure: most programs never need a real Windows, and the ones
// that do say so themselves.
func checkVirtualBox() check {
	m, err := vbox.Find()
	if err != nil {
		return check{
			Name:   "virtualbox",
			Status: statusWarn,
			Result: "not installed",
			Advice: "only needed for `oliver vm`, which runs Windows itself for software no runtime can translate",
		}
	}
	if !vbox.KernelModuleLoaded() {
		return check{
			Name:   "virtualbox",
			Status: statusWarn,
			Result: fmt.Sprintf("%s installed, but its kernel driver is not loaded", m.Version()),
			Advice: "run `sudo modprobe vboxdrv`, or `sudo /sbin/vboxconfig` to rebuild it; no machine can start until it loads",
		}
	}
	return check{
		Name:   "virtualbox",
		Status: statusOK,
		Result: fmt.Sprintf("%s, kernel driver loaded", m.Version()),
	}
}

func checkRuntimes(bs []*backend.Backend) check {
	if len(bs) == 0 {
		return check{
			Name:   "compatibility runtime",
			Status: statusFail,
			Result: "none installed",
			Advice: "install Wine: `sudo apt install wine`, `sudo dnf install wine` or `sudo pacman -S wine`",
		}
	}
	kinds := map[backend.Kind]int{}
	for _, b := range bs {
		kinds[b.Kind]++
	}
	parts := make([]string, 0, len(kinds))
	for _, k := range []backend.Kind{backend.KindWine, backend.KindProton, backend.KindCrossOver, backend.KindBox} {
		if n := kinds[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, k))
		}
	}
	return check{
		Name:   "compatibility runtime",
		Status: statusOK,
		Result: fmt.Sprintf("%d found (%s)", len(bs), strings.Join(parts, ", ")),
	}
}

// checkPlainWine reports a Proton-only system. Proton is built to launch
// games from Steam; installers and ordinary desktop applications work far more
// reliably under plain Wine, and a machine with none is a common cause of
// programs that will not start.
func checkPlainWine(bs []*backend.Backend) check {
	for _, b := range bs {
		if b.Kind == backend.KindWine || b.Kind == backend.KindCrossOver {
			return check{Name: "plain Wine", Status: statusOK, Result: "available (" + b.Name + ")"}
		}
	}
	if len(bs) == 0 {
		return check{Name: "plain Wine", Status: statusFail, Result: "no runtime installed"}
	}
	return check{
		Name:   "plain Wine",
		Status: statusWarn,
		Result: "none: only Proton is installed",
		Advice: "installers and desktop applications will go through Proton, which is built for " +
			"launching games from Steam and is less reliable for anything else. Install Wine, or " +
			"install Bottles from Discover and let it download a Wine runner — Oliver will find it.",
	}
}

func check32Bit(bs []*backend.Backend) check {
	for _, b := range bs {
		if b.Has32 {
			return check{Name: "32-bit programs", Status: statusOK, Result: "supported by " + b.Name}
		}
	}
	if len(bs) == 0 {
		return check{Name: "32-bit programs", Status: statusFail, Result: "no runtime installed"}
	}
	return check{
		Name:   "32-bit programs",
		Status: statusWarn,
		Result: "no installed runtime can run 32-bit programs",
		Advice: "install your distribution's 32-bit Wine libraries (e.g. wine32, wine-core.i686) — many older programs and installers are still 32-bit",
	}
}

func checkProtonSupport(bs []*backend.Backend) check {
	var hasProton bool
	for _, b := range bs {
		if b.IsProton() {
			hasProton = true
			if len(b.Launcher) > 0 {
				return check{Name: "Proton support", Status: statusOK, Result: "Proton builds found, umu-launcher present"}
			}
		}
	}
	if !hasProton {
		return check{
			Name:   "Proton support",
			Status: statusWarn,
			Result: "no Proton builds found",
			Advice: "Proton gives the best results for games; install it through Steam, or add a Proton-GE build to compatibilitytools.d",
		}
	}
	return check{
		Name:   "Proton support",
		Status: statusWarn,
		Result: "Proton found, but umu-launcher is missing",
		Advice: "install umu-launcher for reliable Proton launches outside Steam: https://github.com/Open-Wine-Components/umu-launcher",
	}
}

func checkGraphics() check {
	// Vulkan is what DXVK and VKD3D translate Direct3D into; without it,
	// Direct3D falls back to a far slower OpenGL path.
	for _, p := range []string{
		"/usr/share/vulkan/icd.d", "/usr/local/share/vulkan/icd.d",
		"/etc/vulkan/icd.d",
	} {
		if entries, err := os.ReadDir(p); err == nil && len(entries) > 0 {
			return check{Name: "Vulkan drivers", Status: statusOK, Result: fmt.Sprintf("%d driver(s) registered", len(entries))}
		}
	}
	return check{
		Name:   "Vulkan drivers",
		Status: statusWarn,
		Result: "no Vulkan drivers found",
		Advice: "install your GPU's Vulkan driver (mesa-vulkan-drivers, vulkan-radeon, nvidia driver) for Direct3D performance",
	}
}

func checkPrefixRoot(a *app) check {
	root := prefix.DefaultRoot()
	if a.cfg.PrefixRoot != "" {
		root = a.cfg.PrefixRoot
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return check{Name: "environment storage", Status: statusFail, Result: "cannot create " + root, Advice: err.Error()}
	}
	probe := filepath.Join(root, ".oliver-write-test")
	if err := os.WriteFile(probe, []byte("x"), 0o644); err != nil {
		return check{Name: "environment storage", Status: statusFail, Result: root + " is not writable", Advice: err.Error()}
	}
	os.Remove(probe)
	return check{Name: "environment storage", Status: statusOK, Result: root}
}

func checkDesktopIntegration() check {
	path := desktopFilePath()
	if _, err := os.Stat(path); err != nil {
		return check{
			Name:   "double-click .exe files",
			Status: statusWarn,
			Result: "not registered with the desktop",
			Advice: "run `oliver desktop install` to open .exe and .msi files from your file manager",
		}
	}
	return check{Name: "double-click .exe files", Status: statusOK, Result: "registered"}
}

func checkArch() check {
	switch runtime.GOARCH {
	case "amd64":
		return check{Name: "processor", Status: statusOK, Result: "x86-64, natively supported"}
	case "386":
		return check{Name: "processor", Status: statusOK, Result: "x86, natively supported"}
	case "arm64":
		if _, err := exec.LookPath("box64"); err == nil {
			return check{Name: "processor", Status: statusOK, Result: "ARM64 with box64 translation"}
		}
		return check{
			Name:   "processor",
			Status: statusWarn,
			Result: "ARM64 without an x86 translation layer",
			Advice: "install box64 to run x86 Windows programs on this machine",
		}
	}
	return check{Name: "processor", Status: statusWarn, Result: runtime.GOARCH + ": untested"}
}
