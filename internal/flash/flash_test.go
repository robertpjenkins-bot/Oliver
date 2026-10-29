package flash

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isoImage builds the start of an ISO 9660 image, optionally with the
// partition table a hybrid ISO carries.
func isoImage(label string, hybrid bool) []byte {
	b := make([]byte, 0x8000+2048+4096)
	if hybrid {
		b[446+4] = 0x17 // a partition entry
		b[510], b[511] = 0x55, 0xAA
	}
	copy(b[0x8001:], "CD001")
	copy(b[0x8028:], label+strings.Repeat(" ", 32-len(label)))
	return b
}

func writeTemp(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInspectTellsHybridFromPlainISOs(t *testing.T) {
	img, err := Inspect(writeTemp(t, "fedora.iso", isoImage("Fedora-WS-Live-40", true)))
	if err != nil {
		t.Fatal(err)
	}
	if img.Format != HybridISO || !img.Bootable() {
		t.Errorf("format = %v, want a bootable hybrid ISO", img.Format)
	}
	if img.Label != "Fedora-WS-Live-40" {
		t.Errorf("label = %q", img.Label)
	}

	// A Windows installation ISO has no partition table: dd'd to a stick it
	// does not boot, which is the warning this exists to give.
	img, err = Inspect(writeTemp(t, "Win11.iso", isoImage("CCCOMA_X64FRE_EN-US_DV9", false)))
	if err != nil {
		t.Fatal(err)
	}
	if img.Format != ISO || img.Bootable() {
		t.Errorf("format = %v, want a plain ISO that will not boot from USB", img.Format)
	}
}

func TestInspectRecognisesDiskImages(t *testing.T) {
	gpt := make([]byte, 64<<10)
	gpt[446+4] = 0xEE // protective MBR
	gpt[510], gpt[511] = 0x55, 0xAA
	copy(gpt[512:], "EFI PART")
	img, err := Inspect(writeTemp(t, "pi.img", gpt))
	if err != nil {
		t.Fatal(err)
	}
	if img.Format != DiskImage || img.PartitionTable != "GPT" {
		t.Errorf("got %v %q, want a GPT disk image", img.Format, img.PartitionTable)
	}

	// A bare FAT filesystem ends its boot sector with 0x55AA too, but has no
	// partition table.
	fat := make([]byte, 64<<10)
	fat[510], fat[511] = 0x55, 0xAA
	img, err = Inspect(writeTemp(t, "floppy.img", fat))
	if err != nil {
		t.Fatal(err)
	}
	if img.Format != Raw {
		t.Errorf("a bare filesystem should be raw, got %v", img.Format)
	}

	// Smaller than the ISO volume descriptor offset: still fine.
	img, err = Inspect(writeTemp(t, "tiny.img", []byte("hello")))
	if err != nil || img.Format != Raw || img.Size != 5 {
		t.Errorf("tiny image = %+v, %v", img, err)
	}
}

func TestInspectCatchesCompressedImages(t *testing.T) {
	data := append([]byte{0xFD, '7', 'z', 'X', 'Z', 0x00}, make([]byte, 100)...)
	img, err := Inspect(writeTemp(t, "raspios.img.xz", data))
	if err != nil {
		t.Fatal(err)
	}
	if img.Compression != "xz" {
		t.Fatalf("compression = %q, want xz", img.Compression)
	}
	if cmd := img.UnpackCommand(); !strings.HasPrefix(cmd, "xz -dk ") || !strings.HasSuffix(cmd, "raspios.img.xz") {
		t.Errorf("UnpackCommand = %q", cmd)
	}
}

func TestInspectRefusesWhatCannotBeAnImage(t *testing.T) {
	if _, err := Inspect(writeTemp(t, "partial.iso", nil)); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("an empty file should be refused as empty, got %v", err)
	}
	if _, err := Inspect(t.TempDir()); err == nil || !strings.Contains(err.Error(), "folder") {
		t.Errorf("a folder should be refused, got %v", err)
	}
	if _, err := Inspect(filepath.Join(t.TempDir(), "gone.iso")); err == nil {
		t.Error("a missing file should be refused")
	}
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(1)).Read(b)
	return b
}

func TestRunWritesVerifiesAndReportsEveryPhase(t *testing.T) {
	// More than two blocks, ending part-way through one, written over a
	// "drive" that is larger and full of something else.
	image := randomBytes(2*BlockSize + 12345)
	drive := bytes.Repeat([]byte{0xAB}, 3*BlockSize)
	target := writeTemp(t, "sdz", drive)

	var reports []Progress
	err := Run(context.Background(), Job{Target: target, Source: bytes.NewReader(image), Size: int64(len(image)), Verify: true},
		func(p Progress) { reports = append(reports, p) })
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, _ := os.ReadFile(target)
	if !bytes.Equal(got[:len(image)], image) {
		t.Error("the drive does not hold the image")
	}
	if len(got) != len(drive) || got[len(image)] != 0xAB {
		t.Error("nothing past the end of the image should be touched")
	}

	var wrote, checked bool
	for _, p := range reports {
		if p.Total != int64(len(image)) || p.Done > p.Total {
			t.Fatalf("bad progress report %+v", p)
		}
		if p.Done == p.Total {
			wrote = wrote || p.Phase == Writing
			checked = checked || p.Phase == Verifying
		}
	}
	if !wrote || !checked {
		t.Errorf("expected both phases to reach 100%%: %+v", reports)
	}
}

func TestVerifyCatchesADriveThatDoesNotHoldWhatWasWritten(t *testing.T) {
	image := randomBytes(3 * BlockSize)
	target := writeTemp(t, "sdz", nil)
	dst, err := Open(target)
	if err != nil {
		t.Fatal(err)
	}
	sums, err := Write(context.Background(), dst, bytes.NewReader(image), int64(len(image)), func(Progress) {})
	dst.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(context.Background(), target, int64(len(image)), sums, nil); err != nil {
		t.Fatalf("an intact write failed verification: %v", err)
	}

	// What a counterfeit stick does: data past its real capacity comes back
	// as something else.
	f, _ := os.OpenFile(target, os.O_WRONLY, 0)
	_, _ = f.WriteAt([]byte{0}, BlockSize+5)
	f.Close()
	err = Verify(context.Background(), target, int64(len(image)), sums, nil)
	if err == nil {
		t.Fatal("verification passed although the drive returned different data")
	}
	if !strings.Contains(err.Error(), "4.0 MB and 8.0 MB") {
		t.Errorf("the error should say where the difference is: %v", err)
	}

	// A drive that is shorter than the image.
	if err := os.Truncate(target, BlockSize); err != nil {
		t.Fatal(err)
	}
	if err := Verify(context.Background(), target, int64(len(image)), sums, nil); err == nil || !strings.Contains(err.Error(), "ended") {
		t.Errorf("a truncated drive should fail verification as too short, got %v", err)
	}
}

func TestWriteNoticesAnImageThatEndsEarly(t *testing.T) {
	target := writeTemp(t, "sdz", nil)
	err := Run(context.Background(), Job{Target: target, Source: bytes.NewReader(make([]byte, 1000)), Size: 5000}, nil)
	if err == nil || !strings.Contains(err.Error(), "after 1000 of 5000 bytes") {
		t.Errorf("expected a short image to be reported, got %v", err)
	}
}

func TestWriteStopsWhenCancelled(t *testing.T) {
	target := writeTemp(t, "sdz", nil)
	ctx, cancel := context.WithCancel(context.Background())
	image := randomBytes(3 * BlockSize)
	err := Run(ctx, Job{Target: target, Source: bytes.NewReader(image), Size: int64(len(image))}, func(Progress) { cancel() })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected cancellation, got %v", err)
	}
	if st, _ := os.Stat(target); st.Size() > BlockSize {
		t.Errorf("writing went on after cancellation: %d bytes", st.Size())
	}
}

func TestOpenExplainsAMissingDevice(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "sdz"))
	if err == nil || !strings.Contains(err.Error(), "unplugged") {
		t.Errorf("expected a missing device to be explained, got %v", err)
	}
}

func TestHelperMessagesSurviveTheTrip(t *testing.T) {
	for _, m := range []Message{
		{Progress: &Progress{Phase: Writing, Done: 4194304, Total: 2147483648}},
		{Progress: &Progress{Phase: Verifying, Done: 0, Total: 1}},
		{Error: "cannot write to /dev/sdb: it is \"busy\"\nreally"},
		{Done: true},
	} {
		line := m.String()
		if strings.Contains(line, "\n") {
			t.Errorf("a message must stay on one line: %q", line)
		}
		got, err := ParseMessage(line + "\n")
		if err != nil {
			t.Fatalf("ParseMessage(%q): %v", line, err)
		}
		switch {
		case m.Progress != nil:
			if got.Progress == nil || *got.Progress != *m.Progress {
				t.Errorf("progress %+v came back as %+v", *m.Progress, got)
			}
		case got.Error != m.Error || got.Done != m.Done:
			t.Errorf("%+v came back as %+v", m, got)
		}
	}
	for _, junk := range []string{"", "progress write lots", "[sudo] password for u:"} {
		if _, err := ParseMessage(junk); err == nil {
			t.Errorf("ParseMessage(%q) should fail", junk)
		}
	}
}
