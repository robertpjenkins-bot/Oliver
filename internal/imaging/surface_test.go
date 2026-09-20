package imaging

import (
	"context"
	"testing"
)

func TestASoundDrivePassesTheSurfaceTest(t *testing.T) {
	drive := blankDrive(t, 16<<20)
	res, err := SurfaceTest(context.Background(), drive, SurfaceOptions{BlockSize: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Healthy() {
		t.Errorf("a sound drive failed: %+v", res.Bad)
	}
	if res.Counterfeit() {
		t.Error("a sound drive was called counterfeit")
	}
	if res.Good != res.Claimed {
		t.Errorf("good = %d, claimed = %d", res.Good, res.Claimed)
	}
	if res.Checked != 16<<20 {
		t.Errorf("checked %d bytes of %d", res.Checked, 16<<20)
	}
}

// aliasingDrive is a counterfeit: it claims a size, but its flash is smaller
// and every address past the end wraps round to the start. Reads wrap too,
// which is what makes the lie invisible until somebody reads back what they
// stored.
type aliasingDrive struct {
	data []byte
	real int64
}

func (d *aliasingDrive) WriteAt(p []byte, off int64) (int, error) {
	for i, b := range p {
		d.data[(off+int64(i))%d.real] = b
	}
	return len(p), nil
}

func (d *aliasingDrive) ReadAt(p []byte, off int64) (int, error) {
	for i := range p {
		p[i] = d.data[(off+int64(i))%d.real]
	}
	return len(p), nil
}

func TestACounterfeitDriveIsCaughtAndItsRealSizeReported(t *testing.T) {
	// A stick sold as 32 MiB whose flash is 8 MiB. The sizes are small so the
	// test is quick; the arithmetic is the same at a thousand times the size.
	const claimed, real = 32 << 20, 8 << 20
	drive := &aliasingDrive{data: make([]byte, real), real: real}

	res, err := surfaceTest(context.Background(), drive, claimed,
		SurfaceOptions{BlockSize: 1 << 20}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if res.Healthy() {
		t.Fatal("a drive that wraps around was called sound")
	}
	if !res.Counterfeit() {
		t.Errorf("not recognised as a counterfeit: real=%d bad=%+v", res.RealSize, res.Bad)
	}
	// Worked out from where the data went, not from where the failures start:
	// a wrapping drive destroys its own beginning, so the first bad block is
	// at nought and says nothing.
	if res.RealSize != real {
		t.Errorf("real size worked out as %s, want %s", HumanBytes(res.RealSize), HumanBytes(real))
	}
	if res.Good >= claimed {
		t.Errorf("a counterfeit was credited with %s of good storage", HumanBytes(res.Good))
	}
}

func TestASoundDriveInMemoryPassesToo(t *testing.T) {
	// The same code against a drive that keeps what it is given.
	const size = 16 << 20
	drive := &aliasingDrive{data: make([]byte, size), real: size}
	res, err := surfaceTest(context.Background(), drive, size,
		SurfaceOptions{BlockSize: 1 << 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Healthy() || res.Counterfeit() {
		t.Errorf("a sound drive was failed: %+v", res)
	}
	if res.Good != size {
		t.Errorf("good = %s, want the whole drive", HumanBytes(res.Good))
	}
}

func TestNeighbouringFailuresAreOneRegion(t *testing.T) {
	res := &SurfaceResult{}
	res.addBad(0, 1<<20)
	res.addBad(1<<20, 1<<20)
	res.addBad(4<<20, 1<<20)

	if len(res.Bad) != 2 {
		t.Fatalf("regions = %+v, want the first two joined", res.Bad)
	}
	if res.Bad[0].Length != 2<<20 {
		t.Errorf("the joined region is %d bytes", res.Bad[0].Length)
	}
	if res.Bad[1].Offset != 4<<20 {
		t.Errorf("the separate region moved to %d", res.Bad[1].Offset)
	}
}

func TestTheQuickTestStillLooksAtTheEnd(t *testing.T) {
	// The last block is where a counterfeit fails first, so a quick test that
	// skipped it would miss the thing it is for.
	spots := testSpots(64<<30, 1<<20, true)
	if len(spots) < 2 {
		t.Fatalf("only %d places sampled", len(spots))
	}
	last := spots[len(spots)-1]
	if last < (64<<30)-(2<<20) {
		t.Errorf("the last sample is at %s, nowhere near the end", HumanBytes(last))
	}
	if len(spots) > 200 {
		t.Errorf("%d samples is not quick", len(spots))
	}
}

func TestTheFullTestCoversEverything(t *testing.T) {
	spots := testSpots(10<<20, 1<<20, false)
	if len(spots) != 10 {
		t.Errorf("%d blocks for a 10 MiB drive", len(spots))
	}
	if spots[0] != 0 {
		t.Error("the first block was skipped")
	}
}

func TestThePatternDiffersEverywhereOnTheDrive(t *testing.T) {
	// If two places held the same bytes, a drive that wrapped between them
	// would pass.
	a := make([]byte, 4096)
	b := make([]byte, 4096)
	fillPattern(a, 0)
	fillPattern(b, 8<<20)
	if sameBytes(a, b) {
		t.Error("two different places got the same pattern")
	}
}

func TestASurfaceTestCanBeStopped(t *testing.T) {
	drive := blankDrive(t, 64<<20)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := SurfaceTest(ctx, drive, SurfaceOptions{
		BlockSize: 64 << 10,
		Progress:  func(Progress) { cancel() },
	})
	if err != context.Canceled {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
