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
	"path/filepath"
	"regexp"
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
	// Blocking means the program's main purpose cannot work here.
	Blocking
)

func (s Severity) String() string {
	switch s {
	case Blocking:
		return "will not work"
	case Degraded:
		return "limited"
	}
	return "note"
}

// Caveat is one known limitation.
type Caveat struct {
	Severity   Severity `json:"-"`
	Level      string   `json:"severity"`
	Title      string   `json:"title"`
	Detail     string   `json:"detail"`
	Workaround string   `json:"workaround,omitempty"`
}

// need describes what a known program requires that a compatibility layer
// cannot supply.
type need struct {
	// match is a regular expression tested against the lower-cased executable
	// name without its extension. Patterns are anchored at the start so that
	// an unrelated program merely containing a known name is not caught.
	match    *regexp.Regexp
	program  string
	severity Severity
	detail   string
	instead  string
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
		detail: "Rufus writes boot records to raw USB devices. That needs direct access to " +
			"physical drives and the Windows storage ioctls, neither of which a translated " +
			"Windows process gets, so the device list stays empty and most controls stay disabled.",
		instead: "Use a native writer instead: Ventoy, GNOME Disks, KDE ISO Image Writer, " +
			"Fedora Media Writer, or `dd`. Running Rufus under Wine is unsupported upstream.",
	},
	{
		match: versioned("win32diskimager"), program: "Win32 Disk Imager", severity: Blocking,
		detail:  "It reads and writes raw disk images to physical devices, which requires block-device access.",
		instead: "Use GNOME Disks, or `dd` with the device path.",
	},
	{
		match: versioned("balenaetcher"), program: "balenaEtcher", severity: Blocking,
		detail:  "It writes images directly to removable devices.",
		instead: "balenaEtcher ships a native Linux build; use that.",
	},
	{
		match: regexp.MustCompile(`^cpu-?z(?:[-_. ].*)?$`), program: "CPU-Z", severity: Blocking,
		detail:  "It loads a kernel driver to read hardware registers. Wine cannot load Windows kernel drivers.",
		instead: "Use `lscpu`, `hwinfo` or CPU-X.",
	},
	{
		match: regexp.MustCompile(`^hwmonitor`), program: "HWMonitor", severity: Blocking,
		detail:  "It reads sensors through a kernel driver, which Wine cannot load.",
		instead: "Use `lm-sensors`, `psensor` or Mission Center.",
	},
	{
		match: regexp.MustCompile(`^crystaldiskinfo`), program: "CrystalDiskInfo", severity: Blocking,
		detail:  "It queries drive SMART data through direct device access.",
		instead: "Use `smartctl` from smartmontools, or GNOME Disks.",
	},
	{
		match: regexp.MustCompile(`^daemon[-_. ]?tools`), program: "DAEMON Tools", severity: Blocking,
		detail:  "It creates virtual drives using a kernel driver.",
		instead: "Mount the image directly: `mount -o loop`, or use fuseiso.",
	},
	{
		match: regexp.MustCompile(`^aida64`), program: "AIDA64", severity: Degraded,
		detail:  "Hardware probing goes through a kernel driver, so most sensor and low-level pages stay empty.",
		instead: "Use `inxi -Fxz` or hwinfo for the same information.",
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
			Severity:   matched.severity,
			Title:      fmt.Sprintf("%s needs hardware access that no compatibility layer provides", matched.program),
			Detail:     matched.detail,
			Workaround: matched.instead,
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
