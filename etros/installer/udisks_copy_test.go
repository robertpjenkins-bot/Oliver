//go:build linux

package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// The copy steps of the password-free install, checked against what mtools
// and dd read out of the same image. Set ETROS_TEST_IMAGE to an Etros image.
func TestFilesInstallCopies(t *testing.T) {
	image := os.Getenv("ETROS_TEST_IMAGE")
	if image == "" {
		t.Skip("set ETROS_TEST_IMAGE")
	}
	dir := t.TempDir()
	raw := filepath.Join(dir, "card.img")
	cmd := exec.Command("sh", "-c", "zcat \"$0\" > \"$1\"", image, raw)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	f, _ := os.Open(raw)
	mbr := make([]byte, 512)
	io.ReadFull(f, mbr)
	f.Close()
	parts := readMBR(mbr)

	// Boot partition files.
	img := &imageSource{path: image}
	if err := img.scan(); err != nil {
		t.Fatal(err)
	}
	got := filepath.Join(dir, "boot")
	if err := copyImageFiles(img, got, nopReporter{}); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "want")
	os.Mkdir(want, 0o755)
	mc := exec.Command("mcopy", "-s", "-n", "-i", raw+"@@"+itoa(int64(parts[0].start)*512), "::*", want+"/")
	mc.Env = append(os.Environ(), "MTOOLS_SKIP_CHECK=1")
	if out, err := mc.CombinedOutput(); err != nil {
		t.Fatalf("mcopy: %v %s", err, out)
	}
	if out, err := exec.Command("diff", "-r", want, got).CombinedOutput(); err != nil {
		t.Fatalf("boot partition copy differs:\n%s", out)
	}
	if st, err := os.Stat(filepath.Join(got, "boot")); err != nil || !st.IsDir() {
		t.Error("empty boot directory missing")
	}

	// Ollie OS partition into a file.
	if len(parts) >= 3 {
		dest := filepath.Join(dir, "share", ollieFileOnShare)
		start, size := int64(parts[2].start)*512, int64(parts[2].count)*512
		if err := copyOllie(image, start, size, dest, nopReporter{}); err != nil {
			t.Fatal(err)
		}
		a, _ := os.ReadFile(dest)
		r, _ := os.Open(raw)
		b := make([]byte, size)
		r.ReadAt(b, start)
		r.Close()
		if !bytes.Equal(a, b) {
			t.Error("Ollie OS file differs from the partition")
		}
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
