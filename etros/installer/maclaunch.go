//go:build !windows

package main

// Opening Etros Mac Installer (the app) when it may be running already.
// macOS doesn't start an app again while it's running, only brings it
// forward, and this app has no window to bring: so the app itself only
// starts the installer in the background (or, if that's running, shows its
// page again) and ends at once, and every open does something.

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// macLaunch is what opening the app does. serve runs the installer (in the
// background copy); open shows a page in Safari.
func macLaunch(serve func() error, open func(url string) error) error {
	if os.Getenv("ETROS_MAC_SERVER") == "1" {
		return serve()
	}
	if url := runningMacPage(); url != "" {
		logf("the installer is running already: showing its page")
		return open(url)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// (The same arguments, minus macOS's -psn_ one: tests pass theirs.)
	var args []string
	for _, a := range os.Args[1:] {
		if !strings.HasPrefix(a, "-psn_") {
			args = append(args, a)
		}
	}
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), "ETROS_MAC_SERVER=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("couldn't start the installer: %w", err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	// Wait for its page, so a problem starting it is still shown here.
	deadline := time.After(20 * time.Second)
	for {
		if runningMacPage() != "" {
			return nil
		}
		select {
		case err := <-exited:
			if runningMacPage() != "" {
				return nil
			}
			return fmt.Errorf("the installer stopped as it started (%v): its log is etros-installer-log.txt in Downloads", err)
		case <-deadline:
			return nil // still starting: it opens Safari itself when ready
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// runningMacPage is the address of a running installer's page, or "".
func runningMacPage() string {
	b, err := os.ReadFile(macPageFile())
	if err != nil {
		return ""
	}
	f := strings.Fields(string(b))
	if len(f) != 2 {
		return ""
	}
	pid, err := strconv.Atoi(f[0])
	if err != nil || pid == os.Getpid() || syscall.Kill(pid, 0) != nil {
		return ""
	}
	return f[1]
}
