// Package compat reports known limitations of running a particular Windows
// program under a compatibility layer.
//
// Some programs cannot work no matter which runtime is chosen, because they
// need something Linux will not hand to a translated Windows process: raw
// block devices, kernel drivers, or real administrative privilege. Saying so
// before the launch is more useful than letting the program fail in a way that
// looks like a bug in the runner.
package compat

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/pe"
)

// Severity ranks how much a caveat matters.
type Severity int

const (
	// Note is informational: the program should still work.
	Note Severity = iota
	// Degraded means it will run with reduced function.
	Degraded
	// Blocking means the program's main purpose cannot work here. It usually
	// still launches, which is why the wording matters: saying "will not work"
	// invites the reply "but it opened fine".
	Blocking
)

func (s Severity) String() string {
	switch s {
	case Blocking:
		return "cannot do its job"
	case Degraded:
		return "limited"
	}
	return "note"
}

// Alternative is a native Linux tool that does the job properly.
type Alternative struct {
	Name string `json:"name"`
	// Command is the executable to look for on PATH. Empty when the tool is
	// not normally installed system-wide, such as one run from an extracted
	// download.
	Command   string `json:"command,omitempty"`
	Note      string `json:"note,omitempty"`
	Installed bool   `json:"installed"`
}

// lookPath is a variable so tests can describe a system without touching the
// real one.
var lookPath = exec.LookPath

// resolve marks the alternatives that are present on this machine.
func resolve(in []Alternative) []Alternative {
	if len(in) == 0 {
		return nil
	}
	out := make([]Alternative, len(in))
	copy(out, in)
	for i := range out {
		if out[i].Command == "" {
			continue
		}
		_, err := lookPath(out[i].Command)
		out[i].Installed = err == nil
	}
	// Installed tools first: the point is to end on something runnable now.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Installed && !out[j].Installed
	})
	return out
}

// Caveat is one known limitation.
type Caveat struct {
	Severity Severity `json:"-"`
	Level    string   `json:"severity"`
	Title    string   `json:"title"`
	// Symptom describes what the user will observe, so they can tell this
	// diagnosis apart from an unrelated fault.
	Symptom    string `json:"symptom,omitempty"`
	Detail     string `json:"detail"`
	Workaround string `json:"workaround,omitempty"`
	// Alternatives are native tools that do the same job, with those present
	// on this system marked.
	Alternatives []Alternative `json:"alternatives,omitempty"`
	// BuiltIn names the Oliver command that does the program's job natively,
	// such as "write" for a disk image writer.
	BuiltIn string `json:"built_in,omitempty"`
}

// need describes what a known program requires that a compatibility layer
// cannot supply.
type need struct {
	// match is a regular expression tested against the lower-cased executable
	// name without its extension. Patterns are anchored at the start so that
	// an unrelated program merely containing a known name is not caught.
	match        *regexp.Regexp
	program      string
	severity     Severity
	symptom      string
	detail       string
	instead      string
	alternatives []Alternative
	builtin      string
}

// versioned builds a pattern matching a program name on its own or followed by
// anything that does not begin with a letter.
//
// That covers every way these tools actually arrive — "rufus.exe",
// "rufus-3.10.exe", "rufus3.10.exe", "rufus-3.10 (1).exe" from a second
// download — while still refusing a genuinely different program whose name
// merely starts the same way, such as "rufusclone-tool.exe".
func versioned(base string) *regexp.Regexp {
	return regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `(?:[^a-z].*)?$`)
}

// knownPrograms lists software whose core function depends on hardware access
// that Wine and Proton do not implement. These are not bugs in the runner:
// the Windows APIs involved talk to devices that only the Linux kernel owns.
var knownPrograms = []need{
	{
		match: versioned("rufus"), program: "Rufus", severity: Blocking,
		symptom: "Rufus opens and the window works. The Device list stays empty, and because " +
			"everything below it depends on a selected device, START and the partition, file " +
			"system, cluster size and volume label controls stay greyed out. Selecting an ISO " +
			"still works, which is why it looks like it is almost working.",
		detail: "Rufus enumerates drives through setupapi and writes boot records to raw " +
			"physical devices. A translated Windows process gets neither real device " +
			"enumeration nor the Windows storage ioctls, so no device is ever found.",
		instead: "Mapping the drive into the prefix with a dosdevices symlink does not help " +
			"here: that exposes a drive letter, not the device enumeration Rufus uses. Use a " +
			"native writer instead — `oliver write` does the job itself, as do Ventoy, GNOME " +
			"Disks, KDE ISO Image Writer, Fedora Media Writer and `dd`. Running Rufus under " +
			"Wine is unsupported upstream.",
		builtin: "write",
		alternatives: []Alternative{
			{Name: "Ventoy", Note: "copy ISOs onto the stick and pick one at boot; handles Windows ISOs"},
			{Name: "GNOME Disks", Command: "gnome-disks", Note: "Restore Disk Image from the menu"},
			{Name: "KDE ISO Image Writer", Command: "isoimagewriter"},
			{Name: "Fedora Media Writer", Command: "mediawriter"},
			{Name: "balenaEtcher", Command: "balena-etcher"},
			{Name: "dd", Command: "dd", Note: "for a single Linux ISO; check the device name first"},
		},
	},
	{
		match: versioned("win32diskimager"), program: "Win32 Disk Imager", severity: Blocking,
		detail:  "It reads and writes raw disk images to physical devices, which requires block-device access.",
		instead: "Use `oliver write`, GNOME Disks, or `dd` with the device path.",
		builtin: "write",
		alternatives: []Alternative{
			{Name: "Ventoy", Note: "copy ISOs onto the stick and pick one at boot; handles Windows ISOs"},
			{Name: "GNOME Disks", Command: "gnome-disks", Note: "Restore Disk Image from the menu"},
			{Name: "KDE ISO Image Writer", Command: "isoimagewriter"},
			{Name: "Fedora Media Writer", Command: "mediawriter"},
			{Name: "balenaEtcher", Command: "balena-etcher"},
			{Name: "dd", Command: "dd", Note: "for a single Linux ISO; check the device name first"},
		},
	},
	{
		match: versioned("balenaetcher"), program: "balenaEtcher", severity: Blocking,
		detail:  "It writes images directly to removable devices.",
		instead: "balenaEtcher ships a native Linux build; use that, or `oliver write`.",
		builtin: "write",
		alternatives: []Alternative{
			{Name: "Ventoy", Note: "copy ISOs onto the stick and pick one at boot; handles Windows ISOs"},
			{Name: "GNOME Disks", Command: "gnome-disks", Note: "Restore Disk Image from the menu"},
			{Name: "KDE ISO Image Writer", Command: "isoimagewriter"},
			{Name: "Fedora Media Writer", Command: "mediawriter"},
			{Name: "balenaEtcher", Command: "balena-etcher"},
			{Name: "dd", Command: "dd", Note: "for a single Linux ISO; check the device name first"},
		},
	},
	{
		match: regexp.MustCompile(`^cpu-?z(?:[-_. ].*)?$`), program: "CPU-Z", severity: Blocking,
		detail:  "It loads a kernel driver to read hardware registers. Wine cannot load Windows kernel drivers.",
		instead: "Use `lscpu`, `hwinfo` or CPU-X.",
		alternatives: []Alternative{
			{Name: "lscpu", Command: "lscpu"},
			{Name: "CPU-X", Command: "cpu-x", Note: "the closest equivalent, with the same layout"},
			{Name: "hwinfo", Command: "hwinfo"},
			{Name: "inxi", Command: "inxi", Note: "`inxi -Fxz` for a full summary"},
		},
	},
	{
		match: regexp.MustCompile(`^hwmonitor`), program: "HWMonitor", severity: Blocking,
		detail:  "It reads sensors through a kernel driver, which Wine cannot load.",
		instead: "Use `lm-sensors`, `psensor` or Mission Center.",
		alternatives: []Alternative{
			{Name: "lm-sensors", Command: "sensors", Note: "run `sudo sensors-detect` once first"},
			{Name: "psensor", Command: "psensor"},
			{Name: "Mission Center", Command: "missioncenter"},
		},
	},
	{
		match: regexp.MustCompile(`^crystaldiskinfo`), program: "CrystalDiskInfo", severity: Blocking,
		detail:  "It queries drive SMART data through direct device access.",
		instead: "Use `smartctl` from smartmontools, or GNOME Disks.",
		alternatives: []Alternative{
			{Name: "smartctl", Command: "smartctl", Note: "`sudo smartctl -a /dev/sda`"},
			{Name: "GNOME Disks", Command: "gnome-disks", Note: "SMART Data from the menu"},
		},
	},
	{
		match: regexp.MustCompile(`^daemon[-_. ]?tools`), program: "DAEMON Tools", severity: Blocking,
		detail:  "It creates virtual drives using a kernel driver.",
		instead: "Mount the image directly: `mount -o loop`, or use fuseiso.",
		alternatives: []Alternative{
			{Name: "mount", Command: "mount", Note: "`sudo mount -o loop image.iso /mnt`"},
			{Name: "fuseiso", Command: "fuseiso", Note: "mounts without root"},
		},
	},
	{
		match: regexp.MustCompile(`^aida64`), program: "AIDA64", severity: Degraded,
		detail:  "Hardware probing goes through a kernel driver, so most sensor and low-level pages stay empty.",
		instead: "Use `inxi -Fxz` or hwinfo for the same information.",
		alternatives: []Alternative{
			{Name: "inxi", Command: "inxi", Note: "`inxi -Fxz`"},
			{Name: "hwinfo", Command: "hwinfo"},
		},
	},
}

// Check returns everything worth telling the user before launching.
func Check(info *pe.Info) []Caveat {
	var out []Caveat
	if info == nil {
		return nil
	}

	var matched *need
	for _, candidate := range identities(info) {
		for i := range knownPrograms {
			if knownPrograms[i].match.MatchString(candidate) {
				matched = &knownPrograms[i]
				break
			}
		}
		if matched != nil {
			break
		}
	}
	if matched != nil {
		out = append(out, Caveat{
			Severity:     matched.severity,
			Title:        fmt.Sprintf("%s needs hardware access that no compatibility layer provides", matched.program),
			Symptom:      matched.symptom,
			Detail:       matched.detail,
			Workaround:   matched.instead,
			Alternatives: resolve(matched.alternatives),
			BuiltIn:      matched.builtin,
		})
	}

	// Wine reports the user as an administrator but the process still runs
	// with ordinary Linux permissions, so anything that genuinely needs
	// privilege fails later, and confusingly.
	if info.RequiresAdmin && matched == nil {
		severity := Note
		detail := "The program's manifest requests " + info.ExecutionLevel + ". Wine grants this " +
			"nominally — it reports the user as an administrator — but the process still runs with " +
			"your ordinary Linux permissions."
		workaround := "Anything that only writes inside its own prefix will work. Anything that " +
			"touches real devices, drivers or system files will not."
		if !info.IsInstaller {
			severity = Degraded
		}
		out = append(out, Caveat{
			Severity: severity, Title: "Requests administrator privileges",
			Detail: detail, Workaround: workaround,
		})
	}

	if info.Packed {
		out = append(out, Caveat{
			Severity: Note,
			Title:    "Compressed with " + info.Packer,
			Detail: "The real program is unpacked at startup, so its imports and target Windows " +
				"version cannot be read ahead of time. Automatic detection is working from the " +
				"packer stub, and may classify this program wrongly.",
			Workaround: "If the chosen runtime is not the one you want, set it explicitly with -b.",
		})
	}

	for i := range out {
		out[i].Level = out[i].Severity.String()
	}
	return out
}

// identities lists the names a program is known by, most trustworthy first.
//
// The file on disk can be called anything, so the executable's own version
// resource is consulted first: it survives renaming, the suffix a browser adds
// to a repeated download, and executable packing.
func identities(info *pe.Info) []string {
	var out []string
	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		s = strings.TrimSuffix(s, filepath.Ext(s))
		if s == "" {
			return
		}
		for _, existing := range out {
			if existing == s {
				return
			}
		}
		out = append(out, s)
	}

	add(info.VersionInfo.ProductName)
	add(info.VersionInfo.OriginalFilename)
	add(info.VersionInfo.FileDescription)
	add(info.Name)
	return out
}

// Worst returns the highest severity among caveats.
func Worst(cs []Caveat) Severity {
	worst := Note
	for _, c := range cs {
		if c.Severity > worst {
			worst = c.Severity
		}
	}
	return worst
}
