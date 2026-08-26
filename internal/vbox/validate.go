package vbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Level says how much a problem matters.
type Level int

const (
	// Warn is worth saying but not worth stopping for.
	Warn Level = iota
	// Error stops the build unless the user insists.
	Error
	// Blocker cannot be overridden, because the machine could not be built
	// at all: a missing installer image is not a matter of opinion.
	Blocker
)

// Problem is something wrong with a specification, and what to do about it.
type Problem struct {
	Level Level  `json:"level"`
	Text  string `json:"text"`
	Fix   string `json:"fix,omitempty"`
}

func (p Problem) String() string {
	if p.Fix == "" {
		return p.Text
	}
	return p.Text + " — " + p.Fix
}

// Fatal reports whether any problem in the list stops the build.
func Fatal(problems []Problem, force bool) bool {
	for _, p := range problems {
		if p.Level == Blocker || (p.Level == Error && !force) {
			return true
		}
	}
	return false
}

// Check reports everything wrong with a specification before anything is
// created, so the answer arrives in a second rather than forty minutes into a
// Windows installation.
func (s Spec) Check(h Host) []Problem {
	rel := ReleaseOrDefault(s.Release)
	var out []Problem
	add := func(l Level, fix, format string, args ...any) {
		out = append(out, Problem{Level: l, Text: fmt.Sprintf(format, args...), Fix: fix})
	}

	if strings.TrimSpace(s.Name) == "" || strings.ContainsAny(s.Name, `/\`) {
		add(Blocker, "choose a plain name such as Windows", "%q is not a usable machine name", s.Name)
	}

	// Memory. Too little and the installer refuses; too much and the host it
	// was taken from is the thing that stops working.
	switch {
	case s.RAMMB < rel.MinRAMMB:
		add(Error, fmt.Sprintf("%s needs at least %s", rel.Name, FormatMB(rel.MinRAMMB)),
			"%s of memory is below the minimum for %s", FormatMB(s.RAMMB), rel.Name)
	case h.RAMMB > 0 && s.RAMMB > h.RAMMB-1024:
		add(Error, fmt.Sprintf("this machine has %s in total; leave at least 1 GB for Linux", FormatMB(h.RAMMB)),
			"%s of memory leaves the host almost none", FormatMB(s.RAMMB))
	case h.RAMMB > 0 && s.RAMMB > h.RAMMB/2:
		add(Warn, "the desktop may slow down while Windows is running",
			"%s is more than half this machine's %s", FormatMB(s.RAMMB), FormatMB(h.RAMMB))
	}

	switch {
	case s.CPUs < 1:
		add(Blocker, "give it at least one", "a machine needs at least one processor")
	case h.CPUs > 0 && s.CPUs > h.CPUs:
		add(Error, fmt.Sprintf("this machine has %d", h.CPUs), "%d processors is more than the host has", s.CPUs)
	case h.CPUs > 0 && s.CPUs > h.CPUs/2:
		add(Warn, "VirtualBox runs a guest slower once it has more than half the host's cores",
			"%d of %d processors is a large share", s.CPUs, h.CPUs)
	}

	// Disk. The image grows as Windows fills it, so the size is a ceiling and
	// not a reservation — which is why running out of real space later is the
	// thing worth warning about, not the number itself.
	if s.DiskGB < rel.MinDiskGB {
		add(Error, fmt.Sprintf("%s wants at least %d GB", rel.Name, rel.MinDiskGB),
			"a %d GB disk is below the minimum for %s", s.DiskGB, rel.Name)
	}
	if free := FreeGB(s.Dir); free > 0 {
		switch {
		case free < 25:
			add(Error, fmt.Sprintf("only %d GB is free at %s", free, s.Dir),
				"there is not enough room for a Windows installation")
		case free < s.DiskGB:
			add(Warn, fmt.Sprintf("the disk starts small and grows, so this is fine until Windows actually fills %d GB", s.DiskGB),
				"%s has %d GB free, less than the %d GB disk", s.Dir, free, s.DiskGB)
		}
	}

	if !writable(s.Dir) {
		add(Blocker, "choose a folder you can write to", "%s cannot be written to", s.Dir)
	}
	if entries, err := os.ReadDir(s.MachineDir()); err == nil && len(entries) > 0 {
		add(Blocker, "choose another name or another folder, or remove that one",
			"%s already exists and is not empty", s.MachineDir())
	}

	out = append(out, checkISO(s.ISO, rel)...)
	return out
}

// checkISO reports on the installer image itself.
func checkISO(path string, rel Release) []Problem {
	if strings.TrimSpace(path) == "" {
		return []Problem{{Level: Blocker, Text: "no Windows installer image was given",
			Fix: "download an ISO from microsoft.com and pass it with -iso"}}
	}
	st, err := os.Stat(path)
	if err != nil {
		return []Problem{{Level: Blocker, Text: fmt.Sprintf("cannot read the installer image %s", path), Fix: err.Error()}}
	}
	if st.IsDir() {
		return []Problem{{Level: Blocker, Text: fmt.Sprintf("%s is a folder, not an installer image", path),
			Fix: "point -iso at the .iso file itself"}}
	}

	var out []Problem
	iso, err := InspectISO(path)
	if err != nil {
		return append(out, Problem{Level: Error, Text: err.Error(),
			Fix: "an interrupted download is the usual cause; check the file size against the one you downloaded"})
	}
	if !iso.Bootable {
		out = append(out, Problem{
			Level: Error,
			Text:  fmt.Sprintf("%s carries no boot record, so a machine cannot start from it", filepath.Base(path)),
			Fix:   "this is usually a repacked or partly downloaded image; use the original ISO",
		})
	}
	if !iso.LooksLikeWindows() {
		out = append(out, Problem{
			Level: Warn,
			Text:  fmt.Sprintf("nothing in %s identifies it as a Windows installer", filepath.Base(path)),
			Fix:   "carry on if you know what it is",
		})
	}
	// A Windows image is several gigabytes. Anything much smaller is a
	// recovery or network-install disc that will ask for the real one.
	if iso.Size < 2<<30 {
		out = append(out, Problem{
			Level: Warn,
			Text:  fmt.Sprintf("%s is only %s, small for a Windows installer", filepath.Base(path), humanSize(iso.Size)),
			Fix:   "a full installer is 4 GB or more",
		})
	}
	if iso.Release != "" && iso.Release != rel.ID {
		if found, ok := ReleaseByID(iso.Release); ok {
			out = append(out, Problem{
				Level: Warn,
				Text:  fmt.Sprintf("the image looks like %s but the machine is being prepared for %s", found.Name, rel.Name),
				Fix:   fmt.Sprintf("pass -release %s to set up for what the image holds", found.ID),
			})
		}
	}
	return out
}

// writable reports whether a directory can be written to, walking up to the
// nearest parent that exists — the chosen folder is usually about to be made.
func writable(dir string) bool {
	path, err := filepath.Abs(dir)
	if err != nil {
		path = dir
	}
	for {
		if st, err := os.Stat(path); err == nil {
			return st.IsDir() && syscall.Access(path, 2) == nil
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d bytes", n)
	}
}

// ParseMB reads a memory size such as "8G", "8 GB", "4096M" or "8".
//
// A bare number is read as gibibytes below 256 and mebibytes at or above it,
// because "8" always means 8 GB of memory and "4096" always means 4096 MB —
// nobody means the other thing by either.
func ParseMB(s string) (int, error) { return parseSize(s, 256) }

// ParseGB reads a disk size. A bare number is always gibibytes here, since a
// Windows disk measured in megabytes is not a thing anyone wants.
func ParseGB(s string) (int, error) {
	mb, err := parseSize(s, 1<<30)
	if err != nil {
		return 0, err
	}
	if mb < 1024 {
		return 0, fmt.Errorf("%q is too small for a disk", s)
	}
	return mb / 1024, nil
}

func parseSize(s string, bareIsMBAbove int) (int, error) {
	text := strings.ToLower(strings.TrimSpace(s))
	text = strings.TrimSuffix(text, "b") // gb -> g
	text = strings.TrimSuffix(text, "i") // gib -> g
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("no size given")
	}

	multiplier := 0 // megabytes per unit; 0 means "decide from the number"
	switch text[len(text)-1] {
	case 'g':
		multiplier, text = 1024, strings.TrimSpace(text[:len(text)-1])
	case 'm':
		multiplier, text = 1, strings.TrimSpace(text[:len(text)-1])
	case 't':
		multiplier, text = 1024*1024, strings.TrimSpace(text[:len(text)-1])
	}

	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a size (try 8G, 8192M or 8)", s)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%q is not a size", s)
	}
	if multiplier == 0 {
		multiplier = 1024
		if value >= float64(bareIsMBAbove) {
			multiplier = 1
		}
	}
	return int(value * float64(multiplier)), nil
}
