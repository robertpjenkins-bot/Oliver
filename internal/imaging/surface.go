package imaging

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// A drive can be perfectly writable and still lose what is written to it. Two
// kinds do, and neither shows up in anything a filesystem reports.
//
// A counterfeit stick claims a size its flash does not have: the controller
// takes every write and wraps around, so the last file quietly replaces the
// first, and the drive looks fine until somebody reads back what they stored.
// A worn one has regions that no longer hold what they are given.
//
// Both are found the same way — write something different to every part of the
// drive, read it all back, and see what came back changed. It is the only
// honest answer to "is this drive good", and it costs everything on it.

// SurfaceOptions control a surface test.
type SurfaceOptions struct {
	// Quick samples across the drive rather than covering all of it. It finds
	// a counterfeit in seconds, because a wrapped-around write shows up
	// wherever it is looked for, but it will miss scattered worn blocks.
	Quick bool

	BlockSize int
	Progress  func(Progress)
}

// BadRegion is a place where what was read back was not what was written.
type BadRegion struct {
	Offset int64 `json:"offset"`
	Length int64 `json:"length"`
}

// SurfaceResult is what the drive turned out to be.
type SurfaceResult struct {
	Claimed int64 `json:"claimed_bytes"`
	// Good is how many bytes came back as they were written.
	Good     int64         `json:"good_bytes"`
	Bad      []BadRegion   `json:"bad_regions,omitempty"`
	Checked  int64         `json:"checked_bytes"`
	Quick    bool          `json:"quick"`
	Duration time.Duration `json:"duration"`

	// RealSize is the capacity the drive actually has, when it turned out to
	// have less than it claims. It is worked out from where the data went
	// rather than from where the failures start, because a drive that wraps
	// around destroys its own beginning: the last writes land on top of the
	// first, and every part of it then reads back wrong.
	RealSize int64 `json:"real_size_bytes,omitempty"`
}

// Counterfeit reports whether the drive claims a size its flash does not have.
func (r *SurfaceResult) Counterfeit() bool {
	return r.RealSize > 0 && r.RealSize < r.Claimed
}

// Healthy reports whether everything written came back.
func (r *SurfaceResult) Healthy() bool { return len(r.Bad) == 0 }

// PhaseTesting is the surface test's own phase name.
const PhaseTesting Phase = "testing"

// medium is the part of a drive a surface test needs: somewhere to put bytes
// and somewhere to get them back. Naming it lets the test put a drive that
// lies in front of the same code that will meet a real one.
type medium interface {
	io.ReaderAt
	io.WriterAt
}

// SurfaceTest fills the drive with a pattern that says where it came from,
// reads it all back, and reports what did not survive.
//
// It destroys everything on the drive. There is no version of this that does
// not: a drive can only be proved to hold what it is given by giving it
// something and asking for it back.
func SurfaceTest(ctx context.Context, path string, opt SurfaceOptions) (*SurfaceResult, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, openError(path, err)
	}
	defer f.Close()

	capacity, err := Capacity(f)
	if err != nil {
		return nil, err
	}
	if capacity == 0 {
		return nil, fmt.Errorf("%s reports no capacity", path)
	}
	return surfaceTest(ctx, f, capacity, opt, func() error {
		if err := f.Sync(); err != nil {
			return fmt.Errorf("flushing %s: %w", path, err)
		}
		dropCache(f)
		return nil
	})
}

// surfaceTest is the test itself, against anything that behaves like a drive.
func surfaceTest(ctx context.Context, f medium, capacity int64, opt SurfaceOptions, settle func() error) (*SurfaceResult, error) {
	block := opt.BlockSize
	if block <= 0 {
		block = 1 << 20
	}
	spots := testSpots(capacity, int64(block), opt.Quick)

	started := time.Now()
	result := &SurfaceResult{Claimed: capacity, Quick: opt.Quick}
	report := throttle(opt.Progress)
	buf := make([]byte, block)
	read := make([]byte, block)

	// Writing everything before reading anything is what catches a
	// counterfeit: a drive that wraps around overwrites its earlier blocks
	// with later ones, and only a read after the last write can see that.
	for i, offset := range spots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		length := blockAt(capacity, offset, int64(block))
		fillPattern(buf[:length], offset)
		if _, err := f.WriteAt(buf[:length], offset); err != nil {
			// A refusal part way through is the drive giving out, which is
			// itself the answer.
			result.Bad = append(result.Bad, BadRegion{Offset: offset, Length: length})
			break
		}
		report(Progress{Phase: PhaseWriting, Done: int64(i+1) * int64(block),
			Total: int64(len(spots)) * int64(block), Elapsed: time.Since(started)}, false)
	}
	if settle != nil {
		if err := settle(); err != nil {
			return nil, err
		}
	}

	var strides []int64
	for i, offset := range spots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		length := blockAt(capacity, offset, int64(block))
		fillPattern(buf[:length], offset)
		n, err := f.ReadAt(read[:length], offset)
		switch {
		case err != nil && int64(n) < length:
			result.addBad(offset, length)
		case !sameBytes(read[:length], buf[:length]):
			result.addBad(offset, length)
			// A block that came back holding another part of the drive is the
			// signature of a counterfeit: the controller mapped two addresses
			// onto one piece of flash. The distance between them is a
			// multiple of the real size.
			if came, ok := patternOffset(read[:length]); ok && came != offset {
				strides = append(strides, abs64(offset-came))
			}
		default:
			result.Good += length
		}
		result.Checked += length
		report(Progress{Phase: PhaseVerifying, Done: int64(i+1) * int64(block),
			Total: int64(len(spots)) * int64(block), Elapsed: time.Since(started)}, false)
	}
	report(Progress{Phase: PhaseVerifying, Done: int64(len(spots)) * int64(block),
		Total: int64(len(spots)) * int64(block), Elapsed: time.Since(started)}, true)

	// Every distance is a multiple of the real capacity, so what they have in
	// common is the capacity itself.
	if len(strides) > 0 {
		real := strides[0]
		for _, s := range strides[1:] {
			real = gcd64(real, s)
		}
		if real > 0 && real < capacity {
			result.RealSize = real
		}
	}
	result.Duration = time.Since(started)
	return result, nil
}

// patternOffset reads back which part of the drive a block was written for.
func patternOffset(buf []byte) (int64, bool) {
	if len(buf) < 16 {
		return 0, false
	}
	first := binary.LittleEndian.Uint64(buf[0:])
	second := binary.LittleEndian.Uint64(buf[8:])
	// The pattern steps by a known amount, which is what tells it apart from
	// whatever else might be on the drive. The step is computed rather than
	// written as a constant because it is meant to wrap, and a constant that
	// overflows will not compile.
	step := patternStep
	if second-first != 8*step {
		return 0, false
	}
	return int64(first), true
}

func gcd64(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func (r *SurfaceResult) addBad(offset, length int64) {
	// Neighbouring failures are one region: a worn drive fails in runs, and a
	// list of ten thousand megabytes helps nobody.
	if n := len(r.Bad); n > 0 && r.Bad[n-1].Offset+r.Bad[n-1].Length == offset {
		r.Bad[n-1].Length += length
		return
	}
	r.Bad = append(r.Bad, BadRegion{Offset: offset, Length: length})
}

// testSpots returns the offsets to test, in order.
func testSpots(capacity, block int64, quick bool) []int64 {
	if !quick {
		var out []int64
		for offset := int64(0); offset < capacity; offset += block {
			out = append(out, offset)
		}
		return out
	}
	// Enough places, spread evenly, that a drive wrapping around cannot avoid
	// being caught, and few enough that the whole thing takes seconds.
	const samples = 64
	step := capacity / samples
	if step < block {
		step = block
	}
	var out []int64
	for offset := int64(0); offset < capacity; offset += step {
		out = append(out, offset)
	}
	// The last block matters most: it is the first place a counterfeit fails.
	if last := (capacity - block) &^ 511; last > 0 && out[len(out)-1] != last {
		out = append(out, last)
	}
	return out
}

func blockAt(capacity, offset, block int64) int64 {
	if left := capacity - offset; left < block {
		return left
	}
	return block
}

// fillPattern writes a block that says where on the drive it belongs, so a
// block that turns up somewhere else is recognisable as the drive lying about
// its size rather than as ordinary corruption.
// patternStep is how much the pattern advances per byte. It is odd and large
// so that no two places on a drive get the same block.
const patternStep uint64 = 0x9E3779B97F4A7C15

func fillPattern(buf []byte, offset int64) {
	for i := 0; i+8 <= len(buf); i += 8 {
		binary.LittleEndian.PutUint64(buf[i:], uint64(offset)+uint64(i)*patternStep)
	}
	for i := len(buf) &^ 7; i < len(buf); i++ {
		buf[i] = byte(offset + int64(i))
	}
}

func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
