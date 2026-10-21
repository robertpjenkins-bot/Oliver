package main

// Looking everywhere on this computer for Batocera for PCs and Arch Linux
// before downloading them: Downloads, Desktop, Documents, home, the folder
// the installer is in, its own downloads, every drive plugged in (a USB
// drive from another computer included) and, on a Mac, Spotlight.

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// A localKind is something the installer can use from a file here.
type localKind struct {
	what    string
	match   func(name string) bool
	maybe   func(name string) bool // a name that could be it, if verify says so
	verify  func(path string) bool
	minSize int64
	spot    string // Spotlight name pattern
}

var (
	localBatocera = localKind{
		what: "Batocera for PCs",
		match: func(n string) bool {
			n = strings.ToLower(n)
			return strings.Contains(n, "x86_64") && strings.Contains(n, "boot") &&
				(strings.HasSuffix(n, ".tar.xz") || strings.HasSuffix(n, ".xz"))
		},
		// As batocera.org names it: boot.tar.xz, boot (1).tar.xz...
		maybe: func(n string) bool {
			n = strings.ToLower(n)
			return strings.HasPrefix(n, "boot") && strings.HasSuffix(n, ".tar.xz")
		},
		verify:  isPCBatocera,
		minSize: 500 << 20,
		spot:    "boot*.tar.xz",
	}
	localArch = localKind{
		what: "Arch Linux",
		match: func(n string) bool {
			n = strings.ToLower(n)
			return strings.HasPrefix(n, "archlinux-bootstrap") && strings.HasSuffix(n, ".zst")
		},
		minSize: 50 << 20,
		spot:    "archlinux-bootstrap*zst",
	}
)

// searchPlaces are the folders looked through, nearest first, each with how
// deep to look.
func searchPlaces() []struct {
	dir   string
	depth int
} {
	home := homeDir()
	type place = struct {
		dir   string
		depth int
	}
	ps := []place{{downloadsDir(), 3}, {defaultCache(), 1}}
	if exe, err := os.Executable(); err == nil {
		// The folder the app (or program) is in.
		d := filepath.Dir(exe)
		if i := strings.Index(d, ".app/"); i >= 0 {
			d = filepath.Dir(d[:i+4])
		}
		ps = append(ps, place{d, 2})
	}
	ps = append(ps, place{filepath.Join(home, "Desktop"), 3}, place{filepath.Join(home, "Documents"), 3}, place{home, 1})
	vols := os.Getenv("ETROS_TEST_VOLUMES") // for tests
	if vols == "" {
		vols = "/Volumes"
		if runtime.GOOS == "linux" {
			vols = "/run/media/" + filepath.Base(home)
		}
	}
	if entries, err := os.ReadDir(vols); err == nil {
		for _, e := range entries {
			ps = append(ps, place{filepath.Join(vols, e.Name()), 3})
		}
	}
	return ps
}

// findLocal is the newest usable file of that kind on this computer, or "".
func findLocal(k localKind) string {
	var found []string
	seen := map[string]bool{}
	consider := func(p string) {
		name := filepath.Base(p)
		check := false
		if seen[p] || strings.HasSuffix(p, ".part") {
			return
		}
		if !k.match(name) {
			if k.maybe == nil || !k.maybe(name) {
				return
			}
			check = true
		}
		seen[p] = true
		min := k.minSize
		if os.Getenv("ETROS_TEST_SMALL_FILES") != "" { // for tests' stand-ins
			min = 0
		}
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() && st.Size() >= min && (!check || k.verify(p)) {
			found = append(found, p)
		}
	}
	for _, pl := range searchPlaces() {
		base := strings.Count(filepath.Clean(pl.dir), string(os.PathSeparator))
		filepath.WalkDir(pl.dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if p != pl.dir && (strings.HasPrefix(name, ".") || name == "Library" || strings.HasSuffix(name, ".app") ||
					strings.Count(p, string(os.PathSeparator))-base >= pl.depth) {
					return filepath.SkipDir
				}
				return nil
			}
			consider(p)
			return nil
		})
	}
	if runtime.GOOS == "darwin" && os.Getenv("ETROS_TEST_VOLUMES") == "" {
		if out, err := exec.Command("mdfind", "-name", strings.Trim(k.spot, "*")).Output(); err == nil {
			for _, p := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				if p != "" {
					consider(p)
				}
			}
		}
	}
	if len(found) == 0 {
		return ""
	}
	// The newest.
	sort.Slice(found, func(i, j int) bool {
		a, _ := os.Stat(found[i])
		b, _ := os.Stat(found[j])
		return a.ModTime().After(b.ModTime())
	})
	logf("found %s: %s", k.what, strings.Join(found, ", "))
	return found[0]
}

// isPCBatocera tells Batocera for PCs' boot.tar.xz from another board's,
// by what's in it: the PC one starts from EFI and has boot/initrd.gz.
func isPCBatocera(path string) bool {
	efi, initrd := false, false
	errFound := errors.New("found")
	err := eachTarFile(path, func(h *tarHeader, r io.Reader) error {
		switch h.Name {
		case "EFI/BOOT/BOOTX64.EFI":
			efi = true
		case "boot/initrd.gz":
			initrd = true
		}
		if efi && initrd {
			return errFound
		}
		return nil
	})
	return errors.Is(err, errFound) || (err == nil && efi && initrd)
}
