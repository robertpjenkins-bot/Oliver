//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCopyTreeBackupAndRestore(t *testing.T) {
	src, backup, back := t.TempDir(), t.TempDir(), t.TempDir()
	put := func(rel, data string) {
		p := filepath.Join(src, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(data), 0o644)
	}
	put("roms/snes/Super Metroid (USA).sfc", "rom")
	put("saves/snes/Super Metroid (USA).srm", "save")
	put("system/batocera.conf", "global.videooutput=HDMI-A-1\n")
	put("downloads/My ROMs/READ ME.txt", "hi")
	put(ollieFileOnShare, "big ollie file")
	os.MkdirAll(filepath.Join(src, "lost+found"), 0o700)
	os.Symlink("roms/snes", filepath.Join(src, "snes-link"))
	skip := func(rel string) bool { return rel == "lost+found" || rel == ollieFileOnShare }

	n, bad, err := copyTree(src, backup, skip, nopReporter{})
	if err != nil || bad != 0 || n != 4 {
		t.Fatalf("backup: copied %d, unreadable %d, err %v", n, bad, err)
	}
	if _, err := os.Stat(filepath.Join(backup, ollieFileOnShare)); err == nil {
		t.Error("Ollie OS's file was backed up; it's rebuilt instead")
	}
	if _, _, err := copyTree(backup, back, func(string) bool { return false }, nopReporter{}); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(src, ollieFileOnShare))
	os.Remove(filepath.Join(src, "lost+found"))
	if out, err := exec.Command("diff", "-r", "--no-dereference", src, back).CombinedOutput(); err != nil {
		t.Fatalf("restored copy differs:\n%s", out)
	}
}
