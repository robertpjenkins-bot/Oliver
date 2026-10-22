//go:build !windows

package main

// Opening Etros Mac Installer (the app) when it may be running already:
// macOS can start a second copy (opened from another place, say), which
// then just shows the first one's page. Otherwise the installer runs in the
// app itself, and ends a while after its page is closed (macweb.go), so the
// next open starts it afresh.

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// macLaunch is what opening the app does. serve runs the installer; open
// shows a page in Safari.
func macLaunch(serve func() error, open func(url string) error) error {
	if url := runningMacPage(); url != "" {
		logf("the installer is running already: showing its page")
		return open(url)
	}
	return serve()
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
