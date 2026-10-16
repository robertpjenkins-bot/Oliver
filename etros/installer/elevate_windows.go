//go:build windows

package main

import (
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// relaunchElevated starts the installer again with administrator rights,
// which writing to a card needs; Windows asks first. It reports whether a
// new copy took over. If the request is turned down this copy carries on,
// able to save image files only.
func relaunchElevated(port int) bool {
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
	err = windows.ShellExecute(0, verb, file, params, nil, windows.SW_SHOWNORMAL)
	return err == nil
}
