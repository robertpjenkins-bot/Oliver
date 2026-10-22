//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Opening the app: the installer runs in it, unless one's running already,
// whose page is shown instead; a page left by one that has ended counts for
// nothing.
func TestMacLaunch(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	os.MkdirAll(filepath.Join(cache, "etros-installer"), 0o755)
	var opened []string
	served := 0
	open := func(url string) error { opened = append(opened, url); return nil }
	serve := func() error { served++; return nil }

	if err := macLaunch(serve, open); err != nil || served != 1 || len(opened) != 0 {
		t.Fatalf("first open: %v served=%d opened=%v", err, served, opened)
	}
	// Another installer running (a stand-in process).
	other := exec.Command("sleep", "30")
	if err := other.Start(); err != nil {
		t.Skip("needs sleep")
	}
	defer other.Process.Kill()
	os.WriteFile(macPageFile(), []byte(fmt.Sprintf("%d http://127.0.0.1:1/page/\n", other.Process.Pid)), 0o600)
	if err := macLaunch(serve, open); err != nil || served != 1 || len(opened) != 1 || opened[0] != "http://127.0.0.1:1/page/" {
		t.Fatalf("while running: %v served=%d opened=%v", err, served, opened)
	}
	other.Process.Kill()
	other.Wait()
	if err := macLaunch(serve, open); err != nil || served != 2 {
		t.Fatalf("after it ended: %v served=%d", err, served)
	}
}
