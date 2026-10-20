package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoAutoresize(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "batocera-boot.conf")
	os.WriteFile(p, []byte("# Batocera\nautoresize=true\nsharedevice=INTERNAL\n"), 0o644)
	if err := noAutoresize(dir); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	if strings.Contains(s, "\nautoresize=true") || !strings.Contains(s, "sharedevice=INTERNAL") || !strings.Contains(s, "#autoresize=true") {
		t.Errorf("got:\n%s", s)
	}
	if err := noAutoresize(t.TempDir()); err != nil { // no conf: nothing to do
		t.Error(err)
	}
}
