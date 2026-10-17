//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// relaunchElevated starts the installer again with administrator rights,
// which writing to a card needs; Windows asks first. It reports whether a
// new copy took over. If the request is turned down this copy carries on,
// able to save image files only.
func relaunchElevated() bool {
	if windows.GetCurrentProcessToken().IsElevated() {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	args := append([]string{"-elevated"}, os.Args[1:]...)
	for i, a := range args {
		args[i] = windows.EscapeArg(a)
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	params, _ := windows.UTF16PtrFromString(strings.Join(args, " "))
	fmt.Println("Etros Installer needs administrator rights to write to the SD card.")
	fmt.Println("Choose Yes in the box Windows shows (it may be behind this window).")
	err = windows.ShellExecute(0, verb, file, params, nil, windows.SW_SHOWNORMAL)
	if err != nil {
		logf("administrator rights not given: %v", err)
		fmt.Println("Carrying on without them: you can still make an image file.")
		return false
	}
	fmt.Println("The installer is carrying on in a new window.")
	time.Sleep(3 * time.Second)
	return true
}

// elevatedWrite has a copy of the installer with administrator rights write
// the image (Windows shows a Yes/No box), and waits for it to finish.
func elevatedWrite(image, id string) (bool, error) {
	if windows.GetCurrentProcessToken().IsElevated() {
		return false, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return false, nil
	}
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	argList := strings.Join([]string{
		q("-write"), q(windows.EscapeArg(image)), q("-target"), q("disk:" + id), q("-yes"), q("-pause"),
	}, ",")
	fmt.Println("Windows will ask whether to let Etros make changes: choose Yes.")
	fmt.Println("The writing happens in a new window; this one waits for it.")
	logf("writing %s to disk %s in an elevated copy", image, id)
	out, err := powershell(fmt.Sprintf(
		`$p = Start-Process -FilePath %s -ArgumentList %s -Verb RunAs -Wait -PassThru; exit $p.ExitCode`,
		q(exe), argList))
	if err != nil {
		return true, fmt.Errorf("it didn't finish (did you choose No?) %s", strings.TrimSpace(string(out)))
	}
	return true, nil
}
