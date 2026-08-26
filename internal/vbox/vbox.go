// Package vbox creates and runs a Windows virtual machine under VirtualBox.
//
// Some Windows software cannot be translated: it wants kernel drivers, raw
// disks or a real Windows kernel, and no compatibility layer can provide those
// (`oliver info` says so when it recognises such a program). The honest answer
// for that software is a real Windows, and a virtual machine is the least
// disruptive way to have one.
//
// Standing one up by hand means a dozen VBoxManage invocations whose flags
// have to agree with each other, plus firmware settings that Windows 11 will
// silently refuse to install without. This package asks for the four decisions
// that are genuinely the user's — memory, disk, installer image, and where it
// all lives — and derives the rest.
package vbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultName is the name of the machine when the user does not choose one.
const DefaultName = "Windows"

// Release is a generation of Windows. It decides both the resources the
// installer will accept and the firmware it demands: Windows 11 refuses to
// install without EFI, Secure Boot and a TPM, and reports it as "this PC
// doesn't meet the minimum system requirements" long after the user has
// committed to the download.
type Release struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OSType    string `json:"os_type"` // what VirtualBox calls it
	MinRAMMB  int    `json:"min_ram_mb"`
	MinDiskGB int    `json:"min_disk_gb"`
	NeedsEFI  bool   `json:"needs_efi"`
	NeedsTPM  bool   `json:"needs_tpm"`
	// MinVBox is the oldest VirtualBox that can host this release at all.
	MinVBox [2]int `json:"-"`
}

// Releases are the Windows versions Oliver knows how to prepare a machine for.
// The minimums are Microsoft's own published figures, not opinions.
var Releases = []Release{
	{ID: "win11", Name: "Windows 11", OSType: "Windows11_64", MinRAMMB: 4096, MinDiskGB: 64, NeedsEFI: true, NeedsTPM: true, MinVBox: [2]int{7, 0}},
	{ID: "win10", Name: "Windows 10", OSType: "Windows10_64", MinRAMMB: 2048, MinDiskGB: 32},
	{ID: "win8", Name: "Windows 8.1", OSType: "Windows81_64", MinRAMMB: 2048, MinDiskGB: 32},
	{ID: "win7", Name: "Windows 7", OSType: "Windows7_64", MinRAMMB: 2048, MinDiskGB: 32},
}

// DefaultRelease is assumed when nothing in the installer image says which
// Windows it holds. Windows 10's requirements are the ones an unknown image is
// most likely to satisfy, and its firmware settings are the ones every later
// release can also boot from.
const DefaultRelease = "win10"

// ReleaseByID looks up a release by its identifier.
func ReleaseByID(id string) (Release, bool) {
	for _, r := range Releases {
		if strings.EqualFold(r.ID, id) {
			return r, true
		}
	}
	return Release{}, false
}

// ReleaseOrDefault resolves an identifier, falling back to the default.
func ReleaseOrDefault(id string) Release {
	if r, ok := ReleaseByID(id); ok {
		return r
	}
	r, _ := ReleaseByID(DefaultRelease)
	return r
}

// Spec is the machine to build: everything the user chose, plus what Oliver
// derived from it. It is written beside the machine so a later run can say how
// it was set up.
type Spec struct {
	Name    string `json:"name"`
	RAMMB   int    `json:"ram_mb"`
	CPUs    int    `json:"cpus"`
	VRAMMB  int    `json:"vram_mb"`
	DiskGB  int    `json:"disk_gb"`
	ISO     string `json:"iso"`
	Dir     string `json:"dir"`
	Release string `json:"release"`
	EFI     bool   `json:"efi"`
	TPM     bool   `json:"tpm"`
}

// MachineDir is the folder VirtualBox will put this machine's files in.
func (s Spec) MachineDir() string { return filepath.Join(s.Dir, s.Name) }

// DiskPath is the virtual disk file.
func (s Spec) DiskPath() string { return filepath.Join(s.MachineDir(), s.Name+".vdi") }

// Step is one VBoxManage invocation.
type Step struct {
	Desc string   `json:"description"`
	Argv []string `json:"argv"`
	// Optional steps improve the machine but are not required for it to run,
	// so a VirtualBox that spells the flag differently must not fail the
	// build over comfort settings like audio or a shared clipboard.
	Optional bool `json:"optional,omitempty"`
}

// String renders the step as a line that can be pasted into a shell.
func (s Step) String() string {
	parts := make([]string, len(s.Argv))
	for i, a := range s.Argv {
		parts[i] = shellQuote(a)
	}
	return strings.Join(parts, " ")
}

var shellSafe = regexp.MustCompile(`^[a-zA-Z0-9_@%+=:,./-]+$`)

func shellQuote(s string) string {
	if s != "" && shellSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Manager drives VirtualBox.
type Manager struct {
	// Bin is the VBoxManage binary.
	Bin string
	// Major and Minor are VirtualBox's version, used where flags changed.
	Major, Minor int
	// Root is where Oliver records the machines it made.
	Root string
	// Run executes a VBoxManage invocation. It is a field so tests can
	// describe a VirtualBox without one being installed.
	Run func(argv []string) (string, error)
}

// ErrNotInstalled reports that VirtualBox is not on this system.
var ErrNotInstalled = errors.New("VirtualBox is not installed")

// searchPaths are where distributions and Oracle's own installer put
// VBoxManage when it is not on PATH.
var searchPaths = []string{
	"/usr/bin/VBoxManage",
	"/usr/local/bin/VBoxManage",
	"/usr/lib/virtualbox/VBoxManage",
	"/opt/VirtualBox/VBoxManage",
	"/var/lib/flatpak/exports/bin/org.virtualbox.VirtualBox",
}

// Find locates VirtualBox and reads its version.
func Find() (*Manager, error) {
	bin, err := exec.LookPath("VBoxManage")
	if err != nil {
		for _, p := range searchPaths {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				bin = p
				break
			}
		}
	}
	if bin == "" {
		return nil, ErrNotInstalled
	}

	m := &Manager{Bin: bin, Root: DefaultRoot()}
	m.Run = func(argv []string) (string, error) {
		out, err := exec.Command(m.Bin, argv...).CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("%s %s: %w\n%s", filepath.Base(m.Bin), strings.Join(argv, " "), err, strings.TrimSpace(string(out)))
		}
		return string(out), nil
	}
	if out, err := m.Run([]string{"--version"}); err == nil {
		m.Major, m.Minor = parseVersion(out)
	}
	return m, nil
}

// parseVersion reads the leading numbers of a VirtualBox version string, which
// carries a build suffix such as "7.0.14r161095" or "7.1.4_Debianr165100".
func parseVersion(s string) (int, int) {
	m := regexp.MustCompile(`(\d+)\.(\d+)`).FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major, minor
}

// AtLeast reports whether VirtualBox is at least the given version. An
// unreadable version is treated as new enough, since refusing to work over a
// version string nobody can parse would be worse than trying.
func (m *Manager) AtLeast(major, minor int) bool {
	if m.Major == 0 {
		return true
	}
	return m.Major > major || (m.Major == major && m.Minor >= minor)
}

// Version renders VirtualBox's version for display.
func (m *Manager) Version() string {
	if m.Major == 0 {
		return "unknown version"
	}
	return fmt.Sprintf("%d.%d", m.Major, m.Minor)
}

// KernelModuleLoaded reports whether VirtualBox's kernel driver is present.
//
// This is the failure that wastes an evening: everything installs, the machine
// is created, and starting it dies with a driver error. A secure-boot system
// that has not enrolled VirtualBox's signing key hits it every kernel update.
//
// It is a variable so tests can describe a system whose driver is loaded.
var KernelModuleLoaded = func() bool {
	_, err := os.Stat("/dev/vboxdrv")
	return err == nil
}

// DefaultRoot is where Oliver records the machines it made. The machines
// themselves live wherever the user chose to put them.
func DefaultRoot() string {
	if d := os.Getenv("OLIVER_VM_ROOT"); d != "" {
		return d
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "oliver", "vms")
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "oliver", "vms")
}

// DefaultVMDir is where VirtualBox already keeps machines, and so the least
// surprising place to offer to put a new one.
func DefaultVMDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "VirtualBox VMs")
	}
	return filepath.Join(home, "VirtualBox VMs")
}

// Plan builds the exact sequence of commands that creates the machine.
//
// It is separate from running them so that --dry-run can print precisely what
// would happen, and so the sequence can be tested without VirtualBox.
func (m *Manager) Plan(spec Spec) []Step {
	rel := ReleaseOrDefault(spec.Release)
	name := spec.Name
	diskPath := spec.DiskPath()

	steps := []Step{{
		Desc: "register the machine",
		Argv: []string{"createvm", "--name", name, "--basefolder", spec.Dir, "--ostype", rel.OSType, "--register"},
	}, {
		Desc: fmt.Sprintf("give it %s of memory and %d processor(s)", FormatMB(spec.RAMMB), spec.CPUs),
		Argv: []string{"modifyvm", name,
			"--memory", strconv.Itoa(spec.RAMMB),
			"--cpus", strconv.Itoa(spec.CPUs),
			"--vram", strconv.Itoa(spec.VRAMMB),
			"--graphicscontroller", "vboxsvga",
			"--nic1", "nat",
			// Booting from the disk first means the installer runs only while
			// the disk is still empty. Once Windows is installed the machine
			// starts it directly, so the installer image can stay attached
			// without the "press any key" prompt reappearing every boot.
			"--boot1", "disk", "--boot2", "dvd", "--boot3", "none", "--boot4", "none",
		},
	}}

	if spec.EFI {
		steps = append(steps, Step{
			Desc: "switch the firmware to EFI",
			Argv: []string{"modifyvm", name, "--firmware", "efi"},
		})
	}
	if spec.TPM {
		steps = append(steps, Step{
			Desc: "add a TPM 2.0, which Windows 11 requires",
			Argv: []string{"modifyvm", name, "--tpm-type", "2.0"},
		})
		steps = append(steps, Step{
			Desc:     "enrol Microsoft's Secure Boot keys",
			Argv:     []string{"modifynvram", name, "enrollmssignatures"},
			Optional: true,
		})
	}

	steps = append(steps,
		Step{
			Desc: fmt.Sprintf("create a %d GB disk that grows as Windows fills it", spec.DiskGB),
			Argv: []string{"createmedium", "disk", "--filename", diskPath,
				"--size", strconv.Itoa(spec.DiskGB * 1024), "--format", "VDI", "--variant", "Standard"},
		},
		Step{
			Desc: "add a disk controller",
			Argv: []string{"storagectl", name, "--name", "SATA", "--add", "sata", "--controller", "IntelAhci", "--portcount", "2", "--bootable", "on"},
		},
		Step{
			Desc: "attach the disk",
			Argv: []string{"storageattach", name, "--storagectl", "SATA", "--port", "0", "--device", "0", "--type", "hdd", "--medium", diskPath},
		},
		Step{
			Desc: "add a DVD drive",
			Argv: []string{"storagectl", name, "--name", "IDE", "--add", "ide", "--controller", "PIIX4"},
		},
		Step{
			Desc: "insert the Windows installer",
			Argv: []string{"storageattach", name, "--storagectl", "IDE", "--port", "0", "--device", "0", "--type", "dvddrive", "--medium", spec.ISO},
		},
	)

	// Comfort settings. Every one of these has been spelled differently by
	// some VirtualBox release, so none of them may fail the build.
	audio := []string{"modifyvm", name, "--audio-driver", "default", "--audio-enabled", "on"}
	if !m.AtLeast(7, 0) {
		audio = []string{"modifyvm", name, "--audio", "pulse"}
	}
	steps = append(steps,
		Step{Desc: "enable sound", Argv: audio, Optional: true},
		Step{Desc: "enable USB", Argv: []string{"modifyvm", name, "--usb-ohci", "on"}, Optional: true},
		Step{
			Desc:     "share the clipboard with the desktop",
			Argv:     []string{"modifyvm", name, "--clipboard-mode", "bidirectional", "--draganddrop", "bidirectional"},
			Optional: true,
		},
	)
	return steps
}

// Create builds the machine, reporting each step as it runs.
//
// A half-built machine is worse than none: it registers a name that then
// cannot be reused and leaves a disk image behind. So a failure part-way
// through unregisters and deletes what was made before returning.
func (m *Manager) Create(spec Spec, onStep func(Step)) (warnings []string, err error) {
	if err := os.MkdirAll(spec.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating %s: %w", spec.Dir, err)
	}

	registered := false
	for _, step := range m.Plan(spec) {
		if onStep != nil {
			onStep(step)
		}
		if _, err := m.Run(step.Argv); err != nil {
			if step.Optional {
				warnings = append(warnings, fmt.Sprintf("%s: this VirtualBox would not (%v)", step.Desc, err))
				continue
			}
			if registered {
				_, _ = m.Run([]string{"unregistervm", spec.Name, "--delete"})
			}
			return warnings, fmt.Errorf("%s failed: %w", step.Desc, err)
		}
		if step.Argv[0] == "createvm" {
			registered = true
		}
	}

	if err := m.Save(Record{Spec: spec, Created: time.Now()}); err != nil {
		return warnings, fmt.Errorf("recording the machine: %w", err)
	}
	return warnings, nil
}

// Start boots the machine. The call returns as soon as the window is up,
// leaving VirtualBox owning the machine's lifetime rather than Oliver.
func (m *Manager) Start(name string, headless bool) error {
	kind := "gui"
	if headless {
		kind = "headless"
	}
	_, err := m.Run([]string{"startvm", name, "--type", kind})
	return err
}

// Stop shuts the machine down: by asking Windows to shut down, or by pulling
// the power when asked to be forceful.
func (m *Manager) Stop(name string, force bool) error {
	action := "acpipowerbutton"
	if force {
		action = "poweroff"
	}
	_, err := m.Run([]string{"controlvm", name, action})
	return err
}

// Remove deletes the machine and everything in it.
func (m *Manager) Remove(name string, keepFiles bool) error {
	argv := []string{"unregistervm", name}
	if !keepFiles {
		argv = append(argv, "--delete")
	}
	if _, err := m.Run(argv); err != nil {
		return err
	}
	return m.Forget(name)
}

// Registered reports whether VirtualBox knows a machine by this name. Oliver's
// own record is not enough: the machine may have been deleted in VirtualBox's
// interface, and a stale record would then hide the real state.
func (m *Manager) Registered(name string) (bool, error) {
	out, err := m.Run([]string{"list", "vms"})
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), `"`+name+`"`) {
			return true, nil
		}
	}
	return false, nil
}

// State reports what the machine is doing: running, poweroff, saved, aborted.
func (m *Manager) State(name string) (string, error) {
	out, err := m.Run([]string{"showvminfo", name, "--machinereadable"})
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "VMState="); ok {
			return strings.Trim(v, `"`), nil
		}
	}
	return "unknown", nil
}
