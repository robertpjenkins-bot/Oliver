//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func setupTerminal() {
	st, err := os.Stdout.Stat()
	setColours(err == nil && st.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb")
}

// pauseOnExit is set when the installer opened its own terminal window,
// which would otherwise close as soon as it ends.
func pauseOnExit() bool { return os.Getenv("ETROS_PAUSE") == "1" }

// relaunchElevated runs the installer again through sudo, which writing to
// a card needs. It only returns if that can't be done; the installer then
// carries on and can still make image files.
func relaunchElevated() bool {
	if os.Geteuid() == 0 {
		return false
	}
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	fmt.Println("Etros needs your password to write to the SD card.")
	args := append([]string{"sudo", exe, "-elevated"}, os.Args[1:]...)
	syscall.Exec(sudo, args, os.Environ())
	return false
}
