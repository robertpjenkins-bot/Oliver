package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFATVolumeChecksAndReadsBack(t *testing.T) {
	for _, tool := range []string{"fsck.fat", "mcopy"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " not installed")
		}
	}
	v := NewFATVolume("BATOCERA")
	v.AddBytes("config.txt", []byte("kernel=etros/pi5/linux\n"))
	v.AddBytes("overlays/vc4-kms-v3d-pi5.dtbo", []byte("dtbo"))
	v.AddBytes("my-roms/snes/README with a long name.txt", []byte("hello"))
	v.AddBytes("empty", nil)
	v.Dir("boot")
	big := bytes.Repeat([]byte("squash"), 200000)
	v.AddStreamed("etros/pi5/batocera.update", int64(len(big)), "pi5")
	v.AddStreamed("etros/pi5/batocera.board", 8, "pi5")
	v.Layout(64 << 20)
	v.SetHiddenSectors(32768)

	img := filepath.Join(t.TempDir(), "fat.img")
	f, err := os.Create(img)
	if err != nil {
		t.Fatal(err)
	}
	err = v.WriteTo(f, map[string]func(func(string, io.Reader) error) error{
		"pi5": func(emit func(string, io.Reader) error) error {
			if err := emit("batocera.update", bytes.NewReader(big)); err != nil {
				return err
			}
			return emit("batocera.board", strings.NewReader("bcm2712\n"))
		},
	})
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(img)
	if st.Size() != v.Size() {
		t.Fatalf("wrote %d bytes, volume is %d", st.Size(), v.Size())
	}
	if out, err := exec.Command("fsck.fat", "-n", "-v", img).CombinedOutput(); err != nil {
		t.Fatalf("fsck.fat: %v\n%s", err, out)
	}
	read := func(p string) string {
		cmd := exec.Command("mtype", "-i", img, "::/"+p)
		cmd.Env = append(os.Environ(), "MTOOLS_SKIP_CHECK=1")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("mtype %s: %v", p, err)
		}
		return string(out)
	}
	if got := read("config.txt"); got != "kernel=etros/pi5/linux\n" {
		t.Errorf("config.txt = %q", got)
	}
	if got := read("my-roms/snes/README with a long name.txt"); got != "hello" {
		t.Errorf("long name file = %q", got)
	}
	if got := read("etros/pi5/batocera.update"); got != string(big) {
		t.Errorf("streamed file differs (%d bytes)", len(got))
	}
	if got := read("etros/pi5/batocera.board"); got != "bcm2712\n" {
		t.Errorf("batocera.board = %q", got)
	}
	cmd := exec.Command("mdir", "-/", "-b", "-i", img, "::")
	cmd.Env = append(os.Environ(), "MTOOLS_SKIP_CHECK=1")
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "::/boot/") {
		t.Errorf("empty boot directory missing:\n%s", out)
	}
}
