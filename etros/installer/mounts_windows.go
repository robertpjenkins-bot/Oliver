//go:build windows

package main

import "os"

// mountedDirs lists the drive letters in use.
func mountedDirs() []string {
	var out []string
	for c := 'D'; c <= 'Z'; c++ {
		d := string(c) + `:\`
		if _, err := os.Stat(d); err == nil {
			out = append(out, d)
		}
	}
	return out
}

func closeCard(dir string) bool { return false }
