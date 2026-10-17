//go:build darwin

package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

// mountedDirs lists the drives open in Finder.
func mountedDirs() []string {
	m, _ := filepath.Glob("/Volumes/*")
	var out []string
	for _, d := range m {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			out = append(out, d)
		}
	}
	return out
}

func closeCard(dir string) bool { return exec.Command("diskutil", "eject", dir).Run() == nil }
