package main

// A log of every run, saved to Downloads, so that when something goes wrong
// there is something to send.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"
)

var logFile io.Writer = io.Discard

func openLog() string {
	path := filepath.Join(downloadsDir(), "etros-installer-log.txt")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return ""
	}
	chownToUser(path)
	logFile = f
	logf("=== Etros Installer %s, %s/%s, started %s, args %q",
		version, runtime.GOOS, runtime.GOARCH, time.Now().Format(time.RFC3339), os.Args[1:])
	return path
}

func logf(format string, a ...any) {
	fmt.Fprintf(logFile, time.Now().Format("15:04:05 ")+format+"\n", a...)
}

// crashGuard turns a crash into a message, a log entry and a pause, instead
// of a window that vanishes.
func crashGuard(logPath string) {
	r := recover()
	if r == nil {
		return
	}
	logf("CRASH: %v\n%s", r, debug.Stack())
	fmt.Printf("\n%sTHE INSTALLER CRASHED:%s %v\n", cRed, cReset, r)
	if logPath != "" {
		fmt.Println("Please send this file, it says what happened:")
		fmt.Println("  " + logPath)
	}
	waitForEnter()
	os.Exit(2)
}

func waitForEnter() {
	fmt.Print("\nPress Enter to close.")
	var b [1]byte
	os.Stdin.Read(b[:])
}
