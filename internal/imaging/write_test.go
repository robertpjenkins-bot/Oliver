package imaging

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

// fakeImage writes a file of pseudo-random bytes, which is the worst case for
// anything that might silently deduplicate or compress on the way to a drive.
func fakeImage(t *testing.T, size int64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "image.iso")
	data := make([]byte, size)
	rand.New(rand.NewSource(size)).Read(data)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWriteCopiesTheImageAndProvesItArrived(t *testing.T) {
	image := fakeImage(t, 5<<20)
	drive := blankDrive(t, 32<<20)

	var phases []Phase
	res, err := Write(context.Background(), image, drive, Options{
		Verify:    true,
		BlockSize: 1 << 20,
		Progress: func(p Progress) {
			if len(phases) == 0 || phases[len(phases)-1] != p.Phase {
				phases = append(phases, p.Phase)
			}
			if p.Done > p.Total {
				t.Errorf("progress overran: %d of %d", p.Done, p.Total)
			}
		},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !res.Verified {
		t.Error("write reported success without verifying")
	}
	if res.Bytes != 5<<20 {
		t.Errorf("wrote %d bytes, want %d", res.Bytes, 5<<20)
	}
	if res.Source != res.Target {
		t.Errorf("digests differ:\n source %s\n target %s", res.Source, res.Target)
	}

	want, _ := os.ReadFile(image)
	got := readAt(t, drive, 0, len(want))
	if !bytes.Equal(want, got) {
		t.Error("the drive does not hold the image")
	}

	if len(phases) != 2 || phases[0] != PhaseWriting || phases[1] != PhaseVerifying {
		t.Errorf("progress phases = %v, want writing then verifying", phases)
	}
}

func TestVerifyCatchesADriveThatLostWhatItWasGiven(t *testing.T) {
	image := fakeImage(t, 2<<20)
	drive := blankDrive(t, 8<<20)

	if _, err := Write(context.Background(), image, drive, Options{}); err != nil {
		t.Fatal(err)
	}

	// A failing stick accepts every write and returns something else. One
	// flipped byte a megabyte in stands for that.
	f, err := os.OpenFile(drive, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte{0x00, 0xFF}, 1<<20); err != nil {
		t.Fatal(err)
	}
	f.Close()

	res, err := Verify(context.Background(), image, drive, Options{})
	if !errors.Is(err, ErrVerifyFailed) {
		t.Fatalf("Verify accepted a corrupted drive: err = %v", err)
	}
	if res == nil || res.Verified {
		t.Error("result should record the failure rather than claim success")
	}
}

func TestVerifyAcceptsAGoodDrive(t *testing.T) {
	image := fakeImage(t, 3<<20)
	drive := blankDrive(t, 8<<20)
	if _, err := Write(context.Background(), image, drive, Options{}); err != nil {
		t.Fatal(err)
	}
	res, err := Verify(context.Background(), image, drive, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !res.Verified {
		t.Error("a drive holding the image was reported as not matching")
	}
}

func TestWriteRefusesAnImageThatCannotFit(t *testing.T) {
	image := fakeImage(t, 4<<20)
	drive := blankDrive(t, 1<<20)

	_, err := Write(context.Background(), image, drive, Options{})
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if !strings.Contains(err.Error(), "smaller than") {
		t.Errorf("unhelpful error: %v", err)
	}
	// Nothing may have been written: a partial image on a drive is worse than
	// no image, because it looks like one.
	if got := readAt(t, drive, 0, 64); !bytes.Equal(got, make([]byte, 64)) {
		t.Error("the drive was written to despite the refusal")
	}
}

func TestWriteStopsWhenCancelled(t *testing.T) {
	image := fakeImage(t, 8<<20)
	drive := blankDrive(t, 16<<20)

	ctx, cancel := context.WithCancel(context.Background())
	_, err := Write(ctx, image, drive, Options{
		BlockSize: 64 << 10,
		Progress:  func(p Progress) { cancel() },
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled write returned %v, want context.Canceled", err)
	}
}

func TestQuickWipeClearsTheDescriptionsAndKeepsTheRest(t *testing.T) {
	const size = 64 << 20
	drive := blankDrive(t, size)

	f, err := os.OpenFile(drive, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	filler := bytes.Repeat([]byte{0xA5}, size)
	if _, err := f.WriteAt(filler, 0); err != nil {
		t.Fatal(err)
	}
	f.Close()

	res, err := Wipe(context.Background(), drive, WipeOptions{})
	if err != nil {
		t.Fatalf("Wipe: %v", err)
	}
	if want := int64(4<<20 + 1<<20); res.Bytes != want {
		t.Errorf("cleared %d bytes, want %d", res.Bytes, want)
	}

	head := readAt(t, drive, 0, 4<<20)
	if !bytes.Equal(head, make([]byte, 4<<20)) {
		t.Error("the partition table area was not cleared")
	}
	tail := readAt(t, drive, size-(1<<20), 1<<20)
	if !bytes.Equal(tail, make([]byte, 1<<20)) {
		t.Error("the backup partition table at the end of the drive survived")
	}
	// The middle is deliberately left alone: a quick wipe is about making the
	// drive look empty, not about destroying data.
	middle := readAt(t, drive, size/2, 4096)
	if !bytes.Equal(middle, bytes.Repeat([]byte{0xA5}, 4096)) {
		t.Error("a quick wipe touched the data area")
	}
}

func TestFullWipeClearsEveryByte(t *testing.T) {
	const size = 8 << 20
	drive := blankDrive(t, size)
	f, _ := os.OpenFile(drive, os.O_RDWR, 0)
	f.WriteAt(bytes.Repeat([]byte{0xFF}, size), 0)
	f.Close()

	res, err := Wipe(context.Background(), drive, WipeOptions{Full: true, BlockSize: 1 << 20})
	if err != nil {
		t.Fatalf("Wipe: %v", err)
	}
	if res.Bytes != size {
		t.Errorf("cleared %d of %d bytes", res.Bytes, size)
	}
	if got := readAt(t, drive, 0, size); !bytes.Equal(got, make([]byte, size)) {
		t.Error("the drive still holds data after a full wipe")
	}
}

func TestWipedDriveLooksEmptyToAProbe(t *testing.T) {
	drive := blankDrive(t, 2<<30)
	if _, err := FormatFAT32(drive, FormatOptions{Label: "Old"}); err != nil {
		t.Fatal(err)
	}
	if got := Probe(drive); got.Empty() {
		t.Fatal("a formatted drive probed as empty")
	}
	if _, err := Wipe(context.Background(), drive, WipeOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := Probe(drive); !got.Empty() {
		t.Errorf("a wiped drive still probes as %v", got)
	}
}

func TestMissingDriveIsExplainedRatherThanReported(t *testing.T) {
	image := fakeImage(t, 1<<20)
	_, err := Write(context.Background(), image, filepath.Join(t.TempDir(), "sdz"), Options{})
	if err == nil || !strings.Contains(err.Error(), "unplugged") {
		t.Errorf("error for a missing drive = %v", err)
	}
}

func TestAQuickWipeReportsWhatItClearedNotWhereItWent(t *testing.T) {
	const size = 512 << 20
	drive := blankDrive(t, size)

	var last Progress
	res, err := Wipe(context.Background(), drive, WipeOptions{
		Progress: func(p Progress) {
			last = p
			if p.Done > p.Total {
				t.Errorf("progress overran: %d of %d", p.Done, p.Total)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Counting position on the drive would report a quick wipe of a large
	// drive as having erased all of it, which is not what happened.
	want := int64(4<<20 + 1<<20)
	if res.Bytes != want {
		t.Errorf("cleared %d bytes, want %d", res.Bytes, want)
	}
	if last.Total != want {
		t.Errorf("progress counted up to %d, want %d", last.Total, want)
	}
	if last.Done != want {
		t.Errorf("progress ended at %d of %d", last.Done, last.Total)
	}
}

func TestAFullWipeStillCountsTheWholeDrive(t *testing.T) {
	const size = 8 << 20
	drive := blankDrive(t, size)
	var last Progress
	if _, err := Wipe(context.Background(), drive, WipeOptions{
		Full: true, BlockSize: 1 << 20,
		Progress: func(p Progress) { last = p },
	}); err != nil {
		t.Fatal(err)
	}
	if last.Total != size || last.Done != size {
		t.Errorf("progress ended at %d of %d, want %d", last.Done, last.Total, size)
	}
}
