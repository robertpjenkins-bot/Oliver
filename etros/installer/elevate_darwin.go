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

// elevatedWrite writes the image with sudo; macOS asks for your password.
func elevatedWrite(image, id string) (bool, error) {
	if os.Geteuid() == 0 {
		return false, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return false, nil
	}
	fmt.Println("macOS only lets administrators write to drives: enter your Mac password.")
	cmd := exec.Command("sudo", exe, "-write", image, "-target", "disk:"+id, "-yes")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return true, cmd.Run()
}
