package main

// Looking in Downloads (and next to the installer) for what it can use:
// a finished Etros image, Batocera downloads, an Ollie OS image, games.

import (
	"archive/zip"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type foundFile struct {
	Path string
	Size int64
	Mod  time.Time
}

// Found is what a scan turned up; each list is newest first.
type Found struct {
	Etros   []foundFile // finished Etros images
	Pi5     []foundFile // Batocera for the Raspberry Pi 5
	R36S    []foundFile // Batocera for the R36S
	Ollie   []foundFile // Ollie OS / Raspberry Pi OS images
	PCOllie []foundFile // Ollie OS for PCs (x86 ISOs), which can't run here
	Games   []string    // my-roms folders
}

// homeDir is the home of the person running the installer, even under sudo.
func homeDir() string {
	if name := os.Getenv("SUDO_USER"); name != "" && os.Geteuid() == 0 {
		if u, err := user.Lookup(name); err == nil {
			return u.HomeDir
		}
	}
	h, _ := os.UserHomeDir()
	return h
}

func downloadsDir() string {
	d := filepath.Join(homeDir(), "Downloads")
	if st, err := os.Stat(d); err == nil && st.IsDir() {
		return d
	}
	return homeDir()
}

// scanDirs are the folders searched: Downloads, the folder the installer
// was unzipped into, and the current folder.
func scanDirs() []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(d string) {
		if d == "" {
			return
		}
		if abs, err := filepath.Abs(d); err == nil && !seen[abs] {
			seen[abs] = true
			dirs = append(dirs, abs)
		}
	}
	add(downloadsDir())
	if exe, err := os.Executable(); err == nil {
		add(filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		add(wd)
	}
	return dirs
}

func hasSuffix(name string, exts ...string) bool {
	for _, e := range exts {
		if strings.HasSuffix(name, e) {
			return true
		}
	}
	return false
}

var imageExts = []string{".img", ".img.gz", ".img.xz"}

// classify says what a file looks like from its name.
func classify(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".part") || strings.HasSuffix(n, ".crdownload"):
		return ""
	case strings.Contains(n, "installer") || strings.Contains(n, "maker"):
		return "" // Etros Maker or the Installer itself, not an image
	case strings.HasPrefix(n, "etros") && hasSuffix(n, imageExts...):
		return "etros"
	case strings.HasPrefix(n, "etros") && strings.HasSuffix(n, ".zip"):
		return "etros-zip"
	case strings.Contains(n, "bcm2712") && hasSuffix(n, append(imageExts, ".tar.xz")...):
		return "pi5"
	case strings.Contains(n, "r36s") && hasSuffix(n, append(imageExts, ".tar.xz")...):
		return "r36s"
	case strings.Contains(n, "ollie") && strings.HasSuffix(n, ".iso"):
		return "pc-ollie"
	case strings.Contains(n, "ollie") && hasSuffix(n, imageExts...):
		return "ollie"
	case strings.Contains(n, "raspios") && strings.Contains(n, "arm64") && !strings.Contains(n, "lite") && hasSuffix(n, imageExts...):
		return "ollie"
	}
	return ""
}

func scan() Found {
	var f Found
	seen := map[string]bool{}
	for _, dir := range scanDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if seen[path] {
				continue
			}
			seen[path] = true
			if e.IsDir() {
				n := strings.ToLower(e.Name())
				if n == "my-roms" || n == "etros games" || n == "my games" {
					f.Games = append(f.Games, path)
				}
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			ff := foundFile{path, info.Size(), info.ModTime()}
			switch classify(e.Name()) {
			case "etros":
				f.Etros = append(f.Etros, ff)
			case "etros-zip":
				// Only a zip that really has an Etros image in it.
				if zipHasImage(path) {
					f.Etros = append(f.Etros, ff)
				}
			case "pi5":
				f.Pi5 = append(f.Pi5, ff)
			case "r36s":
				f.R36S = append(f.R36S, ff)
			case "ollie":
				f.Ollie = append(f.Ollie, ff)
			case "pc-ollie":
				f.PCOllie = append(f.PCOllie, ff)
			}
		}
	}
	for _, list := range []*[]foundFile{&f.Etros, &f.Pi5, &f.R36S, &f.Ollie} {
		sort.Slice(*list, func(i, j int) bool { return (*list)[i].Mod.After((*list)[j].Mod) })
	}
	return f
}

func zipHasImage(path string) bool {
	z, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	defer z.Close()
	for _, f := range z.File {
		if hasSuffix(strings.ToLower(f.Name), imageExts...) {
			return true
		}
	}
	return false
}
