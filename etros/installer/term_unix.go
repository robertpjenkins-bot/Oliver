//go:build !windows

package main

import (
	"os"
)

func setupTerminal() {
	st, err := os.Stdout.Stat()
	setColours(err == nil && st.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb")
}

// pauseOnExit is set when the installer opened its own terminal window,
// which would otherwise close as soon as it ends.
func pauseOnExit() bool { return os.Getenv("ETROS_PAUSE") == "1" }
