package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const whichSynopsis = `oliver which -p <prefix> <program.exe>

Finds an installed program inside a Windows environment and prints its path.

After an installer has run, the program it installed lives somewhere under the
prefix's C: drive. This locates it, which is how a packed installer knows what
to launch on later runs.`

func (a *app) cmdWhich(args []string) error {
	fs := a.newFlagSet("which", whichSynopsis)
	prefixName := fs.String("p", "", "prefix to search")
	fs.StringVar(prefixName, "prefix", "", "prefix to search")
	all := fs.Bool("all", false, "print every match rather than the best one")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("no program name given")
	}
	if *prefixName == "" {
		return fmt.Errorf("a prefix is required (-p)")
	}

	p := a.manager().For("", *prefixName)
	// Search whenever there is a C: drive to search. Requiring a fully
	// initialised prefix would refuse one that is mid-creation, or that
	// another tool built, even though the files are plainly there.
	driveC := filepath.Join(p.WinePrefix, "drive_c")
	if st, err := os.Stat(driveC); err != nil || !st.IsDir() {
		return fmt.Errorf("prefix %q has no Windows drive yet; has anything been installed into it?", *prefixName)
	}

	matches := findInPrefix(driveC, fs.Arg(0))
	if len(matches) == 0 {
		return fmt.Errorf("%s is not installed in prefix %q", fs.Arg(0), *prefixName)
	}
	if *all {
		for _, m := range matches {
			a.ui.Printf("%s", m)
		}
		return nil
	}
	a.ui.Printf("%s", matches[0])
	return nil
}

// findInPrefix locates executables matching a name under a prefix's C: drive,
// best candidate first.
func findInPrefix(driveC, want string) []string {
	want = strings.ToLower(want)
	// A bare name matches on the file name; a path fragment matches the tail.
	wantBase := strings.ToLower(filepath.Base(want))

	var matches []string
	_ = filepath.WalkDir(driveC, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if name != wantBase {
			return nil
		}
		// An uninstaller shares the directory but is never what to launch.
		if strings.HasPrefix(name, "unins") {
			return nil
		}
		if want != wantBase && !strings.HasSuffix(strings.ToLower(filepath.ToSlash(path)), strings.ToLower(want)) {
			return nil
		}
		matches = append(matches, path)
		return nil
	})

	// Prefer the usual install locations over caches and temporary copies.
	score := func(p string) int {
		l := strings.ToLower(p)
		switch {
		case strings.Contains(l, "/program files"):
			return 0
		case strings.Contains(l, "/users/"):
			return 1
		case strings.Contains(l, "/windows/"):
			return 3
		}
		return 2
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if a, b := score(matches[i]), score(matches[j]); a != b {
			return a < b
		}
		return len(matches[i]) < len(matches[j])
	})
	return matches
}
