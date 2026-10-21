//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Opening the app: the first time starts the installer in the background
// and returns; opening it again while that runs shows its page; once it has
// ended, opening it starts it again.
func TestMacLaunch(t *testing.T) {
	if os.Getenv("ETROS_MAC_SERVER") == "1" {
		// The background installer, as started by the test below.
		macLaunch(func() error {
			os.WriteFile(macPageFile(), []byte(fmt.Sprintf("%d http://127.0.0.1:1/page/\n", os.Getpid())), 0o600)
			time.Sleep(3 * time.Second)
			os.Remove(macPageFile())
			return nil
		}, nil)
		return
	}
	// The background copy runs this test only.
	oldArgs := os.Args
	os.Args = []string{os.Args[0], "-test.run=^TestMacLaunch$"}
	defer func() { os.Args = oldArgs }()
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	os.MkdirAll(filepath.Join(cache, "etros-installer"), 0o755)
	var opened []string
	open := func(url string) error { opened = append(opened, url); return nil }
	serveHere := func() error { t.Fatal("served in the app itself"); return nil }

	start := time.Now()
	if err := macLaunch(serveHere, open); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second || runningMacPage() == "" {
		t.Fatalf("first open: took %v, page %q", time.Since(start), runningMacPage())
	}
	if err := macLaunch(serveHere, open); err != nil || len(opened) != 1 || opened[0] != "http://127.0.0.1:1/page/" {
		t.Fatalf("second open: %v, opened %v", err, opened)
	}
	time.Sleep(4 * time.Second)
	if runningMacPage() != "" {
		t.Fatal("page still said to be running after the installer ended")
	}
	// A page file left by an installer that's gone counts for nothing.
	os.WriteFile(macPageFile(), []byte("999999 http://127.0.0.1:1/old/\n"), 0o600)
	if runningMacPage() != "" {
		t.Error("an ended installer's page was used")
	}
}
