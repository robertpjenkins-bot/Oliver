//go:build linux

package main

// On Linux, run the installer with sudo to write to a card; without it the
// window still opens and can save an image file.
func relaunchElevated(port int) bool { return false }
