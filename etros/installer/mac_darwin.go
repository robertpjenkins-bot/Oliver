//go:build darwin

package main

// Etros on an Intel Mac, from macOS itself (for Macs without Target Disk
// Mode, like the 12-inch MacBook): every drive and partition is listed, and
// the one chosen becomes BATOCERA (FAT32, Batocera for PCs with the Etros
// layer) and SHARE right after it (ExFAT here, as macOS can't make ext4;
// Etros makes it ext4 the first time it starts, see S11aetros-share). It can
// be a partition (Boot Camp, or one made for Etros), free room in macOS, or
// a whole USB drive, which then starts Etros on any Intel Mac. macOS's own
// diskutil does the partitions; if it wants the Mac's password for that,
// it's macOS asking, in its own password window. It all shows in Safari
// (macweb.go): no Terminal.
//
// Ollie OS can't be made in macOS: its Arch Linux goes on BATOCERA, with
// the Wi-Fi this Mac uses, and Etros makes Ollie OS the first time it
// starts (etros-setup-ollie-pc).

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func macInstallSupported() bool { return runtime.GOARCH == "amd64" }

func runMacInstaller(o Options) error {
	// Start Etros Installer.command: everything in its Terminal window.
	if os.Getenv("ETROS_TEXT") == "1" {
		t := &tui{in: bufio.NewReader(os.Stdin), opts: o}
		err := t.macText(darwinMac{})
		if err != nil {
			logf("ERROR: Mac: %v", err)
			fmt.Printf("\n%s  ETROS DIDN'T GO ON%s\n\n", cRed, cReset)
			note(err.Error())
			note("This is also in etros-installer-log.txt in Downloads.")
		}
		return err
	}
	// The app has no window of its own: anything that goes wrong is shown
	// in a message, not just in the log.
	safari := func(url string) error {
		if exec.Command("open", "-a", "Safari", url).Run() == nil {
			return nil
		}
		return exec.Command("open", url).Run()
	}
	// Started from Terminal (Start Etros Installer.command): say so there.
	say := func(url string) error {
		fmt.Println("Etros Installer is open in Safari: " + url)
		fmt.Println("(If Safari didn't open, paste that address into it.)")
		return safari(url)
	}
	err := macLaunch(func() error { return serveMacWeb(darwinMac{}, say) }, say)
	if err != nil {
		fmt.Println("Etros Installer couldn't start: " + err.Error())
		macAlert("Etros Installer couldn't start: " + err.Error())
	}
	return err
}

func macAlert(msg string) {
	exec.Command("osascript", "-e", fmt.Sprintf("display alert %q message %q as critical", "Etros Installer", msg)).Run()
}

// macLoop (Etros Maker's menu on a Mac) opens the installer in Safari.
func (t *tui) macLoop() bool {
	if runtime.GOARCH != "amd64" {
		note(cRed + "This is for Intel Macs; Macs with Apple chips can't start Etros." + cReset)
		return false
	}
	note("Etros Mac Installer opens in Safari.")
	if err := runMacInstaller(t.opts); err != nil {
		note(cRed + err.Error() + cReset)
		return false
	}
	return true
}

// showCrash puts a crash in front of the person, as the app has no window.
func showCrash(msg string) { macAlert(msg) }
