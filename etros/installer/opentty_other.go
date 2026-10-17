//go:build !linux

package main

// Windows and macOS already open a terminal window for a double-clicked
// terminal program.
func openInTerminal() bool { return false }
