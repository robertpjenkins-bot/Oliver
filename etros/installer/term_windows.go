//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// setupTerminal switches the console to UTF-8 and turns on colours, which
// Windows 10 and later support once asked.
func setupTerminal() {
	windows.SetConsoleOutputCP(65001)
	windows.SetConsoleCP(65001)
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) == nil &&
		windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil {
		setColours(true)
	}
}

// pauseOnExit keeps a double-clicked window open long enough to read.
func pauseOnExit() bool { return true }
