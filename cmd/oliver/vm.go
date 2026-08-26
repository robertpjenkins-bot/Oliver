package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/vbox"
)

const vmSynopsis = `oliver vm [start|setup|status|stop|list|remove|desktop] [flags]

Runs Windows itself, in a VirtualBox machine, for the software that cannot be
translated — anything wanting kernel drivers, raw disks or real Windows
services, which "oliver info" flags before you waste an evening on it.

  oliver vm                     start Windows, setting it up the first time
  oliver vm setup -iso win.iso  set up a machine without starting it
  oliver vm status              what the machine is doing
  oliver vm stop                ask Windows to shut down
  oliver vm list                the machines Oliver has made
  oliver vm remove              delete a machine and its disk
  oliver vm desktop             add or remove its applications-menu entry

Setting a machine up also puts it in your applications menu, so Windows starts
with a click like anything else installed. Pass -no-desktop to skip that.

The first run asks four things — how much memory and disk Windows gets, which
installer image to boot, and where to keep it all — suggests an answer to each
from what this machine actually has, and shows the whole plan before creating
anything. Every answer can be given as a flag instead, which is also how a
machine is set up without a terminal.`

// vmOptions is what the user asked for, before any of it is checked.
type vmOptions struct {
	name    string
	iso     string
	dir     string
	ram     string
	disk    string
	cpus    int
	release string

	headless  bool
	dryRun    bool
	yes       bool
	force     bool
	noStart   bool
	keep      bool
	noDesktop bool
	// fromDesktop marks a run started by a menu launcher rather than typed,
	// where questions have to be asked in dialogs or not at all.
	fromDesktop bool
	asJSON      bool
}

func (a *app) cmdVM(args []string) error {
	fs := a.newFlagSet("vm", vmSynopsis)
	var o vmOptions
	fs.StringVar(&o.name, "name", vbox.DefaultName, "name of the machine")
	fs.StringVar(&o.iso, "iso", "", "the Windows installer image to boot from")
	fs.StringVar(&o.dir, "dir", "", "where to keep the machine (default: ~/VirtualBox VMs)")
	fs.StringVar(&o.ram, "ram", "", "memory for Windows, e.g. 8G")
	fs.StringVar(&o.disk, "disk", "", "disk size for Windows, e.g. 64G")
	fs.IntVar(&o.cpus, "cpus", 0, "processors for Windows")
	fs.StringVar(&o.release, "release", "", "which Windows the image holds (win11, win10, win8, win7)")
	fs.BoolVar(&o.headless, "headless", false, "run without a window")
	fs.BoolVar(&o.dryRun, "n", false, "print what would be done and stop")
	fs.BoolVar(&o.dryRun, "dry-run", false, "print what would be done and stop")
	fs.BoolVar(&o.yes, "y", false, "do not ask for confirmation")
	fs.BoolVar(&o.force, "f", false, "carry on despite problems that can be overridden")
	fs.BoolVar(&o.noStart, "no-start", false, "set the machine up but do not start it")
	fs.BoolVar(&o.noDesktop, "no-desktop", false, "do not add the machine to your applications menu")
	// Not advertised: the applications-menu launcher passes this so a machine
	// that no longer exists offers to be set up again rather than failing at a
	// user who has no terminal to read the failure in.
	fs.BoolVar(&o.fromDesktop, "from-desktop", false, "")
	fs.BoolVar(&o.keep, "keep-files", false, "remove: unregister the machine but keep its files")
	fs.BoolVar(&o.asJSON, "json", false, "print as JSON")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	sub := "start"
	rest := fs.Args()
	if len(rest) > 0 && !strings.HasSuffix(strings.ToLower(rest[0]), ".iso") {
		sub, rest = rest[0], rest[1:]
	}
	// An installer image is accepted as a bare argument, since that is what
	// someone has in hand when they come to this command.
	for _, arg := range rest {
		if strings.HasSuffix(strings.ToLower(arg), ".iso") && o.iso == "" {
			o.iso = arg
		}
	}

	switch sub {
	case "start", "run":
		return a.vmStart(o)
	case "setup", "create", "new", "install":
		o.noStart = o.noStart || sub == "setup" || sub == "create" || sub == "new"
		return a.vmSetup(o)
	case "status", "info":
		return a.vmStatus(o)
	case "stop", "shutdown", "halt":
		return a.vmStop(o)
	case "list", "ls":
		return a.vmList(o)
	case "remove", "delete", "rm":
		return a.vmRemove(o)
	case "desktop", "menu":
		return a.vmDesktop(o, rest)
	default:
		return fmt.Errorf("unknown subcommand %q (want start, setup, status, stop, list or remove)", sub)
	}
}

// vmManager locates VirtualBox, or explains how to get it.
func (a *app) vmManager() (*vbox.Manager, error) {
	m, err := vbox.Find()
	if errors.Is(err, vbox.ErrNotInstalled) {
		return nil, fmt.Errorf("VirtualBox is not installed, and Oliver needs it to run a real Windows.\n" +
			"  Debian, Ubuntu, Mint:  sudo apt install virtualbox\n" +
			"  Fedora:                sudo dnf install VirtualBox\n" +
			"  Arch:                  sudo pacman -S virtualbox\n" +
			"  openSUSE:              sudo zypper install virtualbox\n" +
			"  anything else:         https://www.virtualbox.org/wiki/Downloads")
	}
	return m, err
}

// vmStart runs the machine, setting it up first if this is the first time.
func (a *app) vmStart(o vmOptions) error {
	m, err := a.vmManager()
	if err != nil {
		return err
	}

	_, known, err := m.Load(o.name)
	if err != nil {
		return err
	}
	if !known {
		// The first run is the one that asks the questions — in dialogs when
		// there is no terminal to ask in.
		if o.fromDesktop && !a.ui.Interactive {
			if kit, ok := findDialogKit(); ok {
				return a.guiVM(kit)
			}
		}
		return a.vmSetup(o)
	}

	registered, err := m.Registered(o.name)
	if err != nil {
		return err
	}
	if !registered {
		return fmt.Errorf("Oliver has a record of %q but VirtualBox does not know it any more.\n"+
			"It was probably removed in VirtualBox itself. Clear the record with `oliver vm remove -name %s`, "+
			"then set it up again with `oliver vm setup`", o.name, o.name)
	}

	if state, err := m.State(o.name); err == nil && state == "running" {
		a.ui.Info("%s is already running", o.name)
		return nil
	}
	return a.vmBoot(m, o)
}

// vmBoot starts a machine that already exists.
func (a *app) vmBoot(m *vbox.Manager, o vmOptions) error {
	if o.dryRun {
		kind := "gui"
		if o.headless {
			kind = "headless"
		}
		a.ui.Printf("%s startvm %s --type %s", filepath.Base(m.Bin), shellish(o.name), kind)
		return nil
	}
	if !vbox.KernelModuleLoaded() {
		return errors.New(vboxDriverAdvice)
	}
	a.ui.Info("starting %s", o.name)
	if err := m.Start(o.name, o.headless); err != nil {
		return err
	}
	return m.MarkStarted(o.name)
}

const vboxDriverAdvice = `VirtualBox is installed but its kernel driver is not loaded, so no machine can start.
/dev/vboxdrv does not exist. Load it with:

  sudo modprobe vboxdrv

If that fails the module needs rebuilding for the running kernel:

  sudo /sbin/vboxconfig      (or: sudo dpkg-reconfigure virtualbox-dkms)

On a system with Secure Boot enabled the rebuilt module must also be signed and
its key enrolled, or the kernel will refuse to load it.`

// vmSetup asks the four questions, shows what it would build, and builds it.
func (a *app) vmSetup(o vmOptions) error {
	m, err := a.vmManager()
	if err != nil {
		return err
	}

	if _, known, err := m.Load(o.name); err != nil {
		return err
	} else if known {
		return fmt.Errorf("a machine called %q has already been set up; start it with `oliver vm`, "+
			"or set up another one with `oliver vm setup -name <other>`", o.name)
	}

	spec, err := a.vmAsk(m, o)
	if err != nil {
		return err
	}

	host := vbox.ProbeHost()
	problems := spec.Check(host)
	a.printVMSummary(m, spec, host, problems)

	if vbox.Fatal(problems, o.force) {
		for _, p := range problems {
			if p.Level == vbox.Blocker {
				return fmt.Errorf("%s", p.String())
			}
		}
		return fmt.Errorf("%s (pass -f to build it anyway)", firstError(problems))
	}

	if o.dryRun {
		a.ui.Printf("")
		a.ui.Printf("%s", a.ui.Bold("Would run:"))
		for _, step := range m.Plan(spec) {
			a.ui.Printf("  %s %s", filepath.Base(m.Bin), step.String())
		}
		return nil
	}

	if !o.yes && !a.ui.Confirm("Create this machine?", true) {
		a.ui.Info("cancelled; nothing was created")
		return nil
	}

	warnings, err := m.Create(spec, func(s vbox.Step) {
		a.ui.Info("%s %s", a.ui.Dim("·"), s.Desc)
	})
	for _, w := range warnings {
		a.ui.Warn("%s", w)
	}
	if err != nil {
		return err
	}
	a.ui.Info("%s %s is ready at %s", a.ui.Green("✓"), spec.Name, spec.MachineDir())

	// A machine that can only be started from a terminal is not much of an
	// application, so it goes in the menu unless the user says otherwise.
	if !o.noDesktop {
		if _, err := installVMEntry(spec); err != nil {
			a.ui.Warn("could not add %s to your applications menu: %v", spec.Name, err)
		} else {
			a.ui.Info("%s %s is in your applications menu", a.ui.Green("✓"), spec.Name)
		}
	}

	if o.noStart {
		a.ui.Info("start it with: oliver vm")
		return nil
	}
	if err := a.vmBoot(m, o); err != nil {
		return err
	}
	a.ui.Info("%s", a.ui.Dim("Windows is installing in the VirtualBox window. It restarts itself a few times; that is normal."))
	return nil
}

// vmAsk builds the specification from flags, and asks for whatever is missing.
//
// Every question suggests an answer derived from this machine's own memory,
// processors and free space, so accepting each suggestion gives a machine that
// fits the host it is running on.
func (a *app) vmAsk(m *vbox.Manager, o vmOptions) (vbox.Spec, error) {
	if _, ok := vbox.ReleaseByID(o.release); !ok && o.release != "" {
		return vbox.Spec{}, fmt.Errorf("unknown release %q (want win11, win10, win8 or win7)", o.release)
	}
	dir := expandPath(o.dir)
	if dir == "" {
		dir = vbox.DefaultVMDir()
	}
	host := vbox.ProbeHost()

	interactive := a.ui.Interactive && !o.yes
	if interactive {
		a.ui.Info("%s", a.ui.Bold("Setting up Windows in VirtualBox."))
		a.ui.Info("%s", a.ui.Dim("Nothing is created until you have seen the whole plan."))
		a.ui.Info("")
		a.ui.Info("  this machine has %s of memory and %d processors",
			vbox.FormatMB(host.RAMMB), host.CPUs)
		a.ui.Info("")
	}

	// The installer image comes first: it decides which Windows this is, and
	// therefore what the other answers are allowed to be.
	iso := expandPath(o.iso)
	if iso == "" {
		if !interactive {
			return vbox.Spec{}, errors.New("no Windows installer image given; pass -iso <file.iso>")
		}
		answer, err := a.askUntil("Windows installer image (.iso)", "", func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("an installer image is needed; download one from microsoft.com")
			}
			_, err := os.Stat(expandPath(s))
			return err
		})
		if err != nil {
			return vbox.Spec{}, err
		}
		iso = expandPath(answer)
	}

	release := o.release
	if inspected, err := vbox.InspectISO(iso); err == nil {
		a.describeISO(inspected)
		if release == "" {
			release = inspected.Release
		}
	}
	if release == "" {
		release = vbox.DefaultRelease
		if interactive {
			a.ui.Info("  %s", a.ui.Dim("nothing in the image says which Windows it holds; assuming "+
				vbox.ReleaseOrDefault(release).Name+" (change with -release)"))
		}
	}
	rel := vbox.ReleaseOrDefault(release)
	if rel.MinVBox[0] > 0 && !m.AtLeast(rel.MinVBox[0], rel.MinVBox[1]) {
		return vbox.Spec{}, fmt.Errorf("%s needs a TPM and Secure Boot, which VirtualBox %d.%d cannot provide; "+
			"this system has VirtualBox %s. Upgrade it, or set up %s instead with -release win10",
			rel.Name, rel.MinVBox[0], rel.MinVBox[1], m.Version(), rel.Name)
	}

	// Where the machine goes is asked before how big it is, since the free
	// space there is what makes a disk size sensible or not.
	if o.dir == "" && interactive {
		dir = expandPath(a.ui.Ask("Where to keep the machine", dir))
		a.ui.Info("  %s", a.ui.Dim(fmt.Sprintf("%d GB free there", vbox.FreeGB(dir))))
	}

	spec := host.Recommend(rel, dir)
	spec.Name = o.name
	spec.ISO = iso

	if o.ram != "" {
		mb, err := vbox.ParseMB(o.ram)
		if err != nil {
			return vbox.Spec{}, err
		}
		spec.RAMMB = mb
	} else if interactive {
		answer, err := a.askUntil("Memory for Windows", vbox.FormatMB(spec.RAMMB), func(s string) error {
			_, err := vbox.ParseMB(s)
			return err
		})
		if err != nil {
			return vbox.Spec{}, err
		}
		spec.RAMMB, _ = vbox.ParseMB(answer)
	}

	if o.cpus > 0 {
		spec.CPUs = o.cpus
	} else if interactive {
		answer, err := a.askUntil("Processors", strconv.Itoa(spec.CPUs), func(s string) error {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || n < 1 {
				return fmt.Errorf("%q is not a number of processors", s)
			}
			return nil
		})
		if err != nil {
			return vbox.Spec{}, err
		}
		spec.CPUs, _ = strconv.Atoi(strings.TrimSpace(answer))
	}

	if o.disk != "" {
		gb, err := vbox.ParseGB(o.disk)
		if err != nil {
			return vbox.Spec{}, err
		}
		spec.DiskGB = gb
	} else if interactive {
		answer, err := a.askUntil("Disk size for Windows", fmt.Sprintf("%d GB", spec.DiskGB), func(s string) error {
			_, err := vbox.ParseGB(s)
			return err
		})
		if err != nil {
			return vbox.Spec{}, err
		}
		spec.DiskGB, _ = vbox.ParseGB(answer)
	}

	return spec, nil
}

// describeISO reports what the image turned out to be, since a mistyped path
// that happens to exist is otherwise only discovered when Windows fails to
// install.
func (a *app) describeISO(iso *vbox.ISO) {
	parts := []string{humanBytes(iso.Size)}
	if iso.Label != "" {
		parts = append(parts, "label "+iso.Label)
	}
	if iso.Bootable {
		parts = append(parts, "bootable")
	} else {
		parts = append(parts, a.ui.Yellow("no boot record"))
	}
	name := "installer image"
	if rel, ok := vbox.ReleaseByID(iso.Release); ok {
		name = rel.Name + " installer"
	}
	a.ui.Info("  %s %s (%s)", a.ui.Green("✓"), name, strings.Join(parts, ", "))
}

// printVMSummary shows the whole machine before it is built.
func (a *app) printVMSummary(m *vbox.Manager, spec vbox.Spec, host vbox.Host, problems []vbox.Problem) {
	u := a.ui
	rel := vbox.ReleaseOrDefault(spec.Release)

	u.Printf("")
	u.Printf("%s", u.Bold(fmt.Sprintf("%s in a VirtualBox machine called %q", rel.Name, spec.Name)))
	rows := [][2]string{
		{"memory", fmt.Sprintf("%s of this machine's %s", vbox.FormatMB(spec.RAMMB), vbox.FormatMB(host.RAMMB))},
		{"processors", fmt.Sprintf("%d of %d", spec.CPUs, host.CPUs)},
		{"disk", fmt.Sprintf("%d GB, growing as Windows fills it rather than taken now", spec.DiskGB)},
		{"installer", spec.ISO},
		{"location", spec.MachineDir()},
	}
	if spec.EFI || spec.TPM {
		rows = append(rows, [2]string{"firmware", "EFI, Secure Boot and a TPM 2.0, which " + rel.Name + " refuses to install without"})
	} else {
		rows = append(rows, [2]string{"firmware", "BIOS"})
	}
	rows = append(rows, [2]string{"virtualbox", m.Version()})
	for _, r := range rows {
		u.Printf("  %-11s %s", u.Dim(r[0]), r[1])
	}

	if len(problems) > 0 {
		u.Printf("")
		for _, p := range problems {
			switch p.Level {
			case vbox.Warn:
				u.Printf("  %s %s", u.Yellow("!"), p.String())
			default:
				u.Printf("  %s %s", u.Red("✗"), p.String())
			}
		}
	}
	u.Printf("")
}

func firstError(problems []vbox.Problem) string {
	for _, p := range problems {
		if p.Level == vbox.Error {
			return p.Text
		}
	}
	return "the machine cannot be built as specified"
}

func (a *app) vmList(o vmOptions) error {
	m, err := a.vmManager()
	if err != nil {
		return err
	}
	records, err := m.List()
	if err != nil {
		return err
	}
	if o.asJSON {
		return a.printJSON(records)
	}
	if len(records) == 0 {
		a.ui.Printf("No machines yet. Create one with: oliver vm setup -iso <windows.iso>")
		return nil
	}

	t := a.ui.NewTable("NAME", "WINDOWS", "MEMORY", "DISK", "STATE", "LOCATION")
	for _, r := range records {
		state, err := m.State(r.Spec.Name)
		if err != nil {
			state = a.ui.Red("not in VirtualBox")
		} else if state == "running" {
			state = a.ui.Green(state)
		}
		t.Row(r.Spec.Name, vbox.ReleaseOrDefault(r.Spec.Release).Name, vbox.FormatMB(r.Spec.RAMMB),
			fmt.Sprintf("%d GB", r.Spec.DiskGB), state, r.Spec.MachineDir())
	}
	t.Render()
	return nil
}

func (a *app) vmStatus(o vmOptions) error {
	m, err := a.vmManager()
	if err != nil {
		return err
	}
	record, known, err := m.Load(o.name)
	if err != nil {
		return err
	}
	if !known {
		return fmt.Errorf("no machine called %q yet; create one with `oliver vm setup -iso <windows.iso>`", o.name)
	}
	state, stateErr := m.State(o.name)
	if stateErr != nil {
		state = "not registered with VirtualBox"
	}

	if o.asJSON {
		return a.printJSON(struct {
			vbox.Record
			State string `json:"state"`
		}{record, state})
	}

	u := a.ui
	rel := vbox.ReleaseOrDefault(record.Spec.Release)
	u.Printf("%s %s", u.Bold(record.Spec.Name), u.Dim("("+rel.Name+")"))
	u.Printf("  %-11s %s", u.Dim("state"), state)
	u.Printf("  %-11s %s, %d processors, %d GB disk", u.Dim("hardware"),
		vbox.FormatMB(record.Spec.RAMMB), record.Spec.CPUs, record.Spec.DiskGB)
	u.Printf("  %-11s %s", u.Dim("location"), record.Spec.MachineDir())
	u.Printf("  %-11s %s", u.Dim("installer"), record.Spec.ISO)
	u.Printf("  %-11s %s", u.Dim("created"), record.Created.Format("2006-01-02 15:04"))
	if _, err := os.Stat(vmEntryPath(o.name)); err == nil {
		u.Printf("  %-11s %s", u.Dim("menu"), "in your applications menu")
	} else {
		u.Printf("  %-11s %s", u.Dim("menu"), "not in your applications menu (`oliver vm desktop install` adds it)")
	}
	if record.Starts == 0 {
		u.Printf("  %-11s %s", u.Dim("started"), "never")
	} else {
		u.Printf("  %-11s %d times, last %s", u.Dim("started"), record.Starts, record.LastStarted.Format("2006-01-02 15:04"))
	}
	return nil
}

func (a *app) vmStop(o vmOptions) error {
	m, err := a.vmManager()
	if err != nil {
		return err
	}
	if o.dryRun {
		a.ui.Printf("%s controlvm %s acpipowerbutton", filepath.Base(m.Bin), shellish(o.name))
		return nil
	}
	if err := m.Stop(o.name, o.force); err != nil {
		return err
	}
	if o.force {
		a.ui.Info("pulled the power on %s", o.name)
	} else {
		a.ui.Info("asked %s to shut down (use -f to pull the power)", o.name)
	}
	return nil
}

func (a *app) vmRemove(o vmOptions) error {
	m, err := a.vmManager()
	if err != nil {
		return err
	}
	record, known, err := m.Load(o.name)
	if err != nil {
		return err
	}
	if !known {
		return fmt.Errorf("no machine called %q", o.name)
	}

	registered, _ := m.Registered(o.name)
	if !registered {
		// VirtualBox no longer has it; all that is left is Oliver's record.
		if err := m.Forget(o.name); err != nil {
			return err
		}
		removeVMEntry(o.name)
		a.ui.Info("forgot %s, which VirtualBox no longer had", o.name)
		return nil
	}

	what := "the machine and its disk at " + record.Spec.MachineDir()
	if o.keep {
		what = "the machine, keeping its files at " + record.Spec.MachineDir()
	}
	if !o.yes && !a.ui.Confirm(fmt.Sprintf("Delete %s?", what), false) {
		a.ui.Info("cancelled")
		return nil
	}
	if err := m.Remove(o.name, o.keep); err != nil {
		return err
	}
	if removeVMEntry(o.name) {
		a.ui.Info("took %s out of your applications menu", o.name)
	}
	a.ui.Info("%s removed %s", a.ui.Green("✓"), o.name)
	return nil
}

// askUntil repeats a question until the answer is usable, so a mistyped path
// or size can be corrected in place rather than starting the setup again.
func (a *app) askUntil(question, def string, accept func(string) error) (string, error) {
	for attempt := 0; ; attempt++ {
		answer := a.ui.Ask(question, def)
		err := accept(answer)
		if err == nil {
			return answer, nil
		}
		if !a.ui.Interactive || attempt >= 4 {
			return "", err
		}
		a.ui.Warn("%v", err)
	}
}

// expandPath resolves ~ and strips the quotes a file manager puts around a
// dragged-in path.
func expandPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, `"'`)
	if p == "" {
		return ""
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// shellish quotes a value for the command lines --dry-run prints.
func shellish(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t'\"$&|;<>()*?[]") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// A machine that only starts from a terminal is not much of an application.
// Setting one up therefore writes a launcher for it, so Windows appears in the
// applications menu beside everything else and starts with one click.

// vmEntrySlug names a machine's launcher file and its icon. The prefix keeps
// Oliver's machines together and out of the way of anything else installed.
func vmEntrySlug(name string) string { return "oliver-vm-" + slugFor(name) }

func vmEntryPath(name string) string {
	d := dataHome()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "applications", vmEntrySlug(name)+".desktop")
}

func vmIconFile(name string) string {
	d := dataHome()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "icons", "hicolor", "scalable", "apps", vmEntrySlug(name)+".svg")
}

// installVMEntry writes the launcher and its icon, returning where it went.
//
// The Exec line names the machine explicitly rather than relying on the
// default, so a second machine gets its own working entry — and it records the
// launcher's real path, which is the only durable one when Oliver is an
// AppImage that the user may move.
func installVMEntry(spec vbox.Spec) (string, error) {
	path := vmEntryPath(spec.Name)
	if path == "" {
		return "", fmt.Errorf("cannot determine your home directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if icon := vmIconFile(spec.Name); icon != "" {
		if err := os.MkdirAll(filepath.Dir(icon), 0o755); err == nil {
			_ = os.WriteFile(icon, []byte(vmIconSVG), 0o644)
		}
	}

	launcher := selfPath()
	release := vbox.ReleaseOrDefault(spec.Release).Name
	entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
GenericName=%s virtual machine
Comment=Start %s in VirtualBox, set up by Oliver
Exec=%s vm --from-desktop -name %s
TryExec=%s
Icon=%s
Terminal=false
StartupNotify=false
Categories=System;Emulator;
Keywords=Windows;VirtualBox;virtual machine;VM;
`, spec.Name, release, release, quoteExec(launcher), quoteExec(spec.Name), launcher, vmEntrySlug(spec.Name))

	if err := os.WriteFile(path, []byte(entry), 0o644); err != nil {
		return "", err
	}
	updateDesktopDatabase()
	return path, nil
}

// removeVMEntry takes the launcher away again, reporting whether there was one.
func removeVMEntry(name string) bool {
	var removed bool
	for _, p := range []string{vmEntryPath(name), vmIconFile(name)} {
		if p == "" {
			continue
		}
		if err := os.Remove(p); err == nil {
			removed = true
		}
	}
	if removed {
		updateDesktopDatabase()
	}
	return removed
}

// vmDesktop adds or removes a machine's launcher after the fact.
func (a *app) vmDesktop(o vmOptions, args []string) error {
	m, err := a.vmManager()
	if err != nil {
		return err
	}
	record, known, err := m.Load(o.name)
	if err != nil {
		return err
	}
	if !known {
		return fmt.Errorf("no machine called %q yet; create one with `oliver vm setup -iso <windows.iso>`", o.name)
	}

	sub := "install"
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "install", "add":
		path, err := installVMEntry(record.Spec)
		if err != nil {
			return err
		}
		a.ui.Info("%s %s is in your applications menu", a.ui.Green("✓"), record.Spec.Name)
		a.ui.Info("%s", a.ui.Dim(path))
		return nil
	case "uninstall", "remove", "rm":
		if !removeVMEntry(o.name) {
			a.ui.Info("%s was not in your applications menu", o.name)
			return nil
		}
		a.ui.Info("removed %s from your applications menu", o.name)
		return nil
	case "status":
		path := vmEntryPath(o.name)
		if _, err := os.Stat(path); err != nil {
			a.ui.Printf("%s is not in your applications menu.", o.name)
			return nil
		}
		a.ui.Printf("%s", path)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q (want install, uninstall or status)", sub)
	}
}

// guiVM sets a machine up through desktop dialogs, for someone who opened
// Oliver from a menu and has no terminal to be asked the questions in.
//
// It gathers the same four answers and then hands them to the ordinary setup,
// so the checks, the plan and the machine that comes out are identical.
func (a *app) guiVM(kit dialogKit) error {
	m, err := a.vmManager()
	if err != nil {
		kit.errorBox("VirtualBox is needed for this", fmtErr(err))
		return nil
	}

	// A machine that already exists just starts.
	if _, known, _ := m.Load(vbox.DefaultName); known {
		if err := a.vmBoot(m, vmOptions{name: vbox.DefaultName}); err != nil {
			kit.errorBox("Windows could not start", fmtErr(err))
		}
		return nil
	}

	iso, ok := kit.openFile("Choose the Windows installer", "Windows installer images", []string{"*.iso", "*.ISO"})
	if !ok || iso == "" {
		return nil
	}
	dir, ok := kit.openDir("Where should Windows be kept?", vbox.DefaultVMDir())
	if !ok || dir == "" {
		return nil
	}

	host := vbox.ProbeHost()
	suggestion := host.Recommend(vbox.ReleaseOrDefault(vbox.DetectRelease(iso, "")), dir)
	ram, ok := kit.entry("Memory for Windows",
		fmt.Sprintf("How much memory should Windows get?\nThis machine has %s in total.", vbox.FormatMB(host.RAMMB)),
		vbox.FormatMB(suggestion.RAMMB))
	if !ok {
		return nil
	}
	disk, ok := kit.entry("Disk size for Windows",
		fmt.Sprintf("How large should the Windows disk be?\nIt grows as Windows fills it, so this is a limit rather than space taken now.\n%d GB is free where you chose to keep it.", vbox.FreeGB(dir)),
		fmt.Sprintf("%d GB", suggestion.DiskGB))
	if !ok {
		return nil
	}

	o := vmOptions{name: vbox.DefaultName, iso: iso, dir: dir, ram: ram, disk: disk, yes: true}

	// Show what would be built, with everything wrong with it, before asking.
	spec, err := a.vmAsk(m, o)
	if err != nil {
		kit.errorBox("Windows could not be set up", fmtErr(err))
		return nil
	}
	problems := spec.Check(host)
	summary := a.captureText(func() { a.printVMSummary(m, spec, host, problems) })
	if vbox.Fatal(problems, false) {
		kit.errorBox("This machine cannot be built", strings.TrimSpace(summary))
		return nil
	}
	if !kit.confirm("Set up Windows?", strings.TrimSpace(summary)) {
		return nil
	}

	var setupErr error
	report := a.captureText(func() { setupErr = a.vmSetup(o) })
	if setupErr != nil {
		kit.errorBox("Windows could not be set up", strings.TrimSpace(report+"\n\n"+fmtErr(setupErr)))
		return nil
	}
	kit.info("Windows is starting",
		"Windows is installing in the VirtualBox window. It restarts itself a few times; that is normal.\n\n"+
			"It is now in your applications menu, and starts from there whenever you want it.")
	return nil
}

// vmIconSVG is the launcher icon: a machine within a machine, which is all a
// virtual machine is. It is kept inline so the binary stays self-contained.
const vmIconSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="128" height="128">
  <defs>
    <linearGradient id="g" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#4a90d9"/>
      <stop offset="1" stop-color="#2a5d8f"/>
    </linearGradient>
  </defs>
  <rect x="8" y="8" width="112" height="112" rx="24" fill="url(#g)"/>
  <rect x="28" y="34" width="72" height="52" rx="6" fill="#fff" opacity=".95"/>
  <path d="M64 34v52M28 60h72" stroke="#2a5d8f" stroke-width="5"/>
  <path d="M50 96h28M58 86v10M70 86v10" stroke="#fff" stroke-width="6" stroke-linecap="round" opacity=".75"/>
</svg>
`
