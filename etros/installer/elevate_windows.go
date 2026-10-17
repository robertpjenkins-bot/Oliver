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
