//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// relaunchElevated runs the installer again through sudo: macOS only lets
// administrators write to a card. It only returns if that can't be done; the
// installer then carries on and can still make image files.
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
	fmt.Println("Etros needs your Mac password to write to the SD card.")
	args := append([]string{"sudo", exe, "-elevated"}, os.Args[1:]...)
	syscall.Exec(sudo, args, os.Environ())
	return false
}

func elevatedWrite(image, id string) (bool, error) { return false, nil }
