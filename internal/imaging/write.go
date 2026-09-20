package imaging

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"syscall"
	"time"
)

// DefaultBlockSize is the transfer size. Large enough that a USB drive is not
// starved between writes, small enough that progress stays responsive and a
// cancelled write stops promptly.
const DefaultBlockSize = 4 << 20

// Phase names the stage a Progress report belongs to.
type Phase string

const (
	PhaseWriting   Phase = "writing"
	PhaseVerifying Phase = "verifying"
	PhaseErasing   Phase = "erasing"
)

// Progress is one report from a long operation.
type Progress struct {
	Phase   Phase
	Done    int64
	Total   int64
	Elapsed time.Duration
}

// Rate returns the bytes per second achieved so far.
func (p Progress) Rate() float64 {
	if p.Elapsed <= 0 {
		return 0
	}
	return float64(p.Done) / p.Elapsed.Seconds()
}

// Options control a write.
type Options struct {
	// BlockSize is the transfer size; zero means DefaultBlockSize.
	BlockSize int

	// Verify reads the drive back after writing and compares it with the
	// source. This is the only way to know the drive holds what was sent to
	// it: a failing stick accepts every write and returns something else.
	Verify bool

	// Progress, if set, is called periodically. It must not block.
	Progress func(Progress)
}

// Result reports what a write did.
type Result struct {
	Bytes    int64         `json:"bytes_written"`
	Duration time.Duration `json:"duration"`
	Source   string        `json:"source_sha256,omitempty"`
	Target   string        `json:"target_sha256,omitempty"`
	Verified bool          `json:"verified"`
}

// Rate returns the average throughput of the write.
func (r *Result) Rate() float64 {
	if r.Duration <= 0 {
		return 0
	}
	return float64(r.Bytes) / r.Duration.Seconds()
}

// ErrVerifyFailed is returned when the drive did not read back what was
// written to it. The drive is left in whatever state it reached.
var ErrVerifyFailed = errors.New("the drive did not read back what was written to it")

// Write copies an image onto a device and, if asked, verifies it.
//
// The caller is expected to have checked that the device is the right one:
// this function does not second-guess the target, it only refuses a write that
// cannot physically fit.
func Write(ctx context.Context, imagePath, devicePath string, opt Options) (*Result, error) {
	src, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	st, err := src.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()
	if size == 0 {
		return nil, fmt.Errorf("%s is empty", imagePath)
	}

	dst, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
	if err != nil {
		return nil, openError(devicePath, err)
	}
	defer dst.Close()

	if capacity, err := Capacity(dst); err == nil && capacity > 0 && capacity < size {
		return nil, fmt.Errorf("%s holds %s, which is smaller than the %s image",
			devicePath, HumanBytes(capacity), HumanBytes(size))
	}

	started := time.Now()
	digest := sha256.New()
	written, err := copyBlocks(ctx, dst, src, size, opt, PhaseWriting, digest)
	if err != nil {
		return nil, err
	}

	// Data sits in the kernel's cache until this returns; without it the
	// program would report success while the drive is still being written,
	// and pulling the stick would corrupt it.
	if err := dst.Sync(); err != nil {
		return nil, fmt.Errorf("flushing %s: %w", devicePath, err)
	}

	res := &Result{
		Bytes:    written,
		Duration: time.Since(started),
		Source:   hex.EncodeToString(digest.Sum(nil)),
	}

	if opt.Verify {
		sum, err := readBack(ctx, devicePath, written, opt)
		if err != nil {
			return res, err
		}
		res.Target = sum
		res.Verified = sum == res.Source
		res.Duration = time.Since(started)
		if !res.Verified {
			return res, fmt.Errorf("%w: wrote %s but read back %s",
				ErrVerifyFailed, short(res.Source), short(res.Target))
		}
	}

	// The kernel is still holding the partition table the drive had before the
	// write. Asking it to re-read makes the new partitions appear without a
	// replug; failing to is not an error worth stopping for.
	rereadPartitions(dst)
	return res, nil
}

// readBack re-reads the first n bytes of the device and returns their digest.
func readBack(ctx context.Context, devicePath string, n int64, opt Options) (string, error) {
	f, err := os.Open(devicePath)
	if err != nil {
		return "", openError(devicePath, err)
	}
	defer f.Close()

	// Reading back through the page cache would compare the data with itself
	// and pass on a drive that stored nothing.
	dropCache(f)

	digest := sha256.New()
	if _, err := copyBlocks(ctx, digest, f, n, opt, PhaseVerifying, nil); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// copyBlocks moves exactly total bytes, reporting progress as it goes and
// optionally digesting what passes through.
func copyBlocks(ctx context.Context, dst io.Writer, src io.Reader, total int64, opt Options, phase Phase, digest hash.Hash) (int64, error) {
	block := opt.BlockSize
	if block <= 0 {
		block = DefaultBlockSize
	}
	buf := make([]byte, block)

	started := time.Now()
	var done int64
	report := throttle(opt.Progress)
	for done < total {
		if err := ctx.Err(); err != nil {
			return done, err
		}
		want := int64(block)
		if left := total - done; left < want {
			want = left
		}
		n, err := io.ReadFull(src, buf[:want])
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return done, fmt.Errorf("%s at %s: %w", phase, HumanBytes(done), werr)
			}
			if digest != nil {
				digest.Write(buf[:n])
			}
			done += int64(n)
			report(Progress{Phase: phase, Done: done, Total: total, Elapsed: time.Since(started)}, false)
		}
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return done, fmt.Errorf("%s at %s: %w", phase, HumanBytes(done), err)
		}
	}
	report(Progress{Phase: phase, Done: done, Total: total, Elapsed: time.Since(started)}, true)

	if done != total {
		return done, fmt.Errorf("%s ended early: %s of %s", phase, HumanBytes(done), HumanBytes(total))
	}
	return done, nil
}

// throttle limits progress reports to a rate a terminal can keep up with,
// while always letting the final one through.
func throttle(fn func(Progress)) func(Progress, bool) {
	if fn == nil {
		return func(Progress, bool) {}
	}
	const every = 100 * time.Millisecond
	var last time.Time
	return func(p Progress, force bool) {
		now := time.Now()
		if force || now.Sub(last) >= every {
			last = now
			fn(p)
		}
	}
}

// WipeOptions control an erase.
type WipeOptions struct {
	// Full zeroes every byte rather than only the areas that describe the
	// drive's contents. It takes as long as a write of the same size, and is
	// what to use before passing a drive on to somebody else.
	Full bool

	BlockSize int
	Progress  func(Progress)
}

// Wipe destroys the partition tables and filesystem signatures on a device, so
// the next tool to look at it sees an empty drive rather than the remains of
// the last one.
//
// A quick wipe clears the start and the end: the MBR, any GPT and its backup
// at the far end of the drive, and the superblocks of the filesystems that
// lived in between. The data is still on the platters afterwards; only the
// descriptions of it are gone.
func Wipe(ctx context.Context, devicePath string, opt WipeOptions) (*Result, error) {
	f, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
	if err != nil {
		return nil, openError(devicePath, err)
	}
	defer f.Close()

	capacity, err := Capacity(f)
	if err != nil {
		return nil, err
	}
	if capacity == 0 {
		return nil, fmt.Errorf("%s reports no capacity", devicePath)
	}

	started := time.Now()
	var cleared int64

	if opt.Full {
		cleared, err = zeroRange(ctx, f, 0, capacity, opt, capacity, 0)
		if err != nil {
			return nil, err
		}
	} else {
		// 4 MiB covers the MBR, a GPT with a full entry array, and the
		// superblocks of every common filesystem.
		head := min64(4<<20, capacity)
		// The backup GPT lives in the last sectors, and a drive left holding
		// one comes back with its old partitions.
		tail := min64(1<<20, capacity-head)

		// Progress counts the bytes being cleared, not where on the drive they
		// are. Counting position would report a quick wipe of a 512 GB drive
		// as having erased 512 GB, which is neither true nor what was asked.
		total := head + tail
		cleared, err = zeroRange(ctx, f, 0, head, opt, total, 0)
		if err != nil {
			return nil, err
		}
		if tail > 0 {
			n, err := zeroRange(ctx, f, capacity-tail, tail, opt, total, cleared)
			if err != nil {
				return nil, err
			}
			cleared += n
		}
	}

	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("flushing %s: %w", devicePath, err)
	}
	rereadPartitions(f)
	return &Result{Bytes: cleared, Duration: time.Since(started)}, nil
}

// zeroRange writes zeroes over [offset, offset+length), reporting progress as
// done bytes out of total, counting alreadyDone from earlier ranges.
func zeroRange(ctx context.Context, f *os.File, offset, length int64, opt WipeOptions, total, alreadyDone int64) (int64, error) {
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	block := opt.BlockSize
	if block <= 0 {
		block = DefaultBlockSize
	}
	if int64(block) > length {
		block = int(length)
	}
	zeros := make([]byte, block)

	started := time.Now()
	report := throttle(opt.Progress)
	var done int64
	for done < length {
		if err := ctx.Err(); err != nil {
			return done, err
		}
		want := int64(block)
		if left := length - done; left < want {
			want = left
		}
		n, err := f.Write(zeros[:want])
		done += int64(n)
		if err != nil {
			return done, fmt.Errorf("erasing at %s: %w", HumanBytes(offset+done), err)
		}
		report(Progress{Phase: PhaseErasing, Done: alreadyDone + done, Total: total, Elapsed: time.Since(started)}, false)
	}
	report(Progress{Phase: PhaseErasing, Done: alreadyDone + done, Total: total, Elapsed: time.Since(started)}, true)
	return done, nil
}

// Capacity returns the size in bytes of an open device or file. A block device
// has no size in its inode, so it is measured by seeking to the end.
func Capacity(f *os.File) (int64, error) {
	if st, err := f.Stat(); err == nil && st.Mode().IsRegular() {
		return st.Size(), nil
	}
	cur, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	end, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if _, err := f.Seek(cur, io.SeekStart); err != nil {
		return 0, err
	}
	return end, nil
}

// openError turns the kernel's refusals into the two things that actually
// cause them.
// ErrReadOnlyDevice is the kernel refusing to open a device for writing
// because it is holding it read-only. It is worth recognising because the
// message the kernel gives for it — "read-only file system" — names neither a
// filesystem nor anything a person can act on.
var ErrReadOnlyDevice = errors.New("the kernel is holding this device read-only")

func openError(path string, err error) error {
	switch {
	case errors.Is(err, syscall.EROFS):
		return fmt.Errorf("%w: %s.\n"+
			"Nothing was wrong with the request — a flag on the device is refusing writes before "+
			"they reach the drive. `oliver unprotect %s` clears it", ErrReadOnlyDevice, path, path)
	case errors.Is(err, os.ErrPermission):
		return fmt.Errorf("cannot open %s for writing: %w\n"+
			"Writing to a drive needs privileges your user does not have. "+
			"Either run this one command with sudo, or add yourself to the \"disk\" group", path, err)
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("%s does not exist; the drive may have been unplugged", path)
	}
	return fmt.Errorf("opening %s: %w", path, err)
}

func short(sum string) string {
	if len(sum) > 16 {
		return sum[:16] + "…"
	}
	return sum
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// Verify compares what a device holds with the image that should be on it,
// without writing anything. It is the same check Write performs when asked,
// available on its own for a drive that has been sitting in a drawer.
func Verify(ctx context.Context, imagePath, devicePath string, opt Options) (*Result, error) {
	src, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	st, err := src.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()

	started := time.Now()
	digest := sha256.New()
	if _, err := copyBlocks(ctx, digest, src, size, opt, PhaseVerifying, nil); err != nil {
		return nil, err
	}
	res := &Result{Bytes: size, Source: hex.EncodeToString(digest.Sum(nil))}

	sum, err := readBack(ctx, devicePath, size, opt)
	if err != nil {
		return nil, err
	}
	res.Target = sum
	res.Verified = sum == res.Source
	res.Duration = time.Since(started)
	if !res.Verified {
		return res, fmt.Errorf("%w: expected %s but the drive holds %s",
			ErrVerifyFailed, short(res.Source), short(res.Target))
	}
	return res, nil
}
