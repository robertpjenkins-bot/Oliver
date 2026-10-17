//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
)

// On Linux the installer runs as you, with no password: the card is opened
// through UDisks, which lets the person at the computer write to removable
// drives (see openDisk). Only if that isn't allowed does writing fall back
// to sudo (elevatedWrite).
func relaunchElevated() bool { return false }

// elevatedWrite writes an image to a card with sudo, for systems where
// UDisks isn't available or won't allow it. sudo is tried without a
// password first (Chromebooks and Raspberry Pi OS set it up that way).
func elevatedWrite(image, id string) (bool, error) {
	if os.Geteuid() == 0 {
		return false, nil
	}
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return false, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return false, nil
	}
	args := []string{exe, "-write", image, "-target", "disk:" + id, "-yes"}
	run := func(extra ...string) error {
		cmd := exec.Command(sudo, append(extra, args...)...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	}
	if exec.Command(sudo, "-n", "true").Run() == nil {
		logf("writing with sudo (no password needed)")
		return true, run("-n")
	}
	fmt.Println("This computer only lets administrators write to cards, so it needs your password once.")
	logf("writing with sudo (password)")
	return true, run()
}
