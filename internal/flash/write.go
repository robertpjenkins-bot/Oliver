package flash

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// BlockSize is how much is written, hashed and read back per step.
const BlockSize = 4 << 20

// syncEvery bounds how far reported progress can run ahead of the drive.
// Without it the page cache absorbs gigabytes, the bar races to 100% and then
// sits there for minutes while the data actually reaches a slow USB stick.
const syncEvery = 32 << 20

// Phase is the stage a write is in.
type Phase string

const (
	Writing   Phase = "write"
	Verifying Phase = "verify"
)

// Progress is how far a phase has got.
type Progress struct {
	Phase Phase
	Done  int64
	Total int64
}

// Percent is the progress as a whole percentage.
func (p Progress) Percent() int {
	if p.Total <= 0 {
		return 0
	}
	return int(p.Done * 100 / p.Total)
}

// Job is one image written to one drive or partition.
type Job struct {
	// Target is the device node to write, such as /dev/sdb or /dev/sdb1.
	Target string
	// Source supplies exactly Size bytes of image.
	Source io.Reader
	Size   int64
	// Verify reads everything back afterwards and compares it.
	Verify bool
}

// Run writes the image and, if asked, verifies it. report is called after
// every block.
func Run(ctx context.Context, j Job, report func(Progress)) error {
	if report == nil {
		report = func(Progress) {}
	}
	dst, err := Open(j.Target)
	if err != nil {
		return err
	}
	sums, err := Write(ctx, dst, j.Source, j.Size, report)
	if err != nil {
		dst.Close()
		return err
	}
	// Have the kernel pick up the partition table that was just written, so
	// the new partitions appear without unplugging the drive. This only
	// applies to whole drives, needs privilege, and udev usually does it
	// anyway on close, so failure is not an error.
	_ = rereadPartitions(dst)
	if err := dst.Close(); err != nil {
		return fmt.Errorf("finishing the write to %s: %w", j.Target, err)
	}
	if !j.Verify {
		return nil
	}
	return Verify(ctx, j.Target, j.Size, sums, report)
}

// Open opens a drive or partition for writing.
//
// O_EXCL asks the kernel for exclusive use of a block device: the open fails
// if a filesystem on it is mounted or it is part of an active encrypted or
// LVM volume. It is the last line of defence against writing over something
// in use, whatever the checks before it concluded.
func Open(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_EXCL, 0)
	if err != nil {
		return nil, explainOpen(path, err)
	}
	return f, nil
}

func explainOpen(path string, err error) error {
	switch {
	case errors.Is(err, syscall.EBUSY):
		return fmt.Errorf("%s is in use: a filesystem on it is still mounted, or another program has it open", path)
	case errors.Is(err, syscall.EROFS):
		return fmt.Errorf("%s is write-protected (check the lock switch on an SD card)", path)
	case errors.Is(err, syscall.ENOMEDIUM):
		return fmt.Errorf("there is no card or disc in %s", path)
	case errors.Is(err, os.ErrNotExist), errors.Is(err, syscall.ENXIO):
		return fmt.Errorf("%s has gone; was the drive unplugged?", path)
	case errors.Is(err, os.ErrPermission):
		return fmt.Errorf("you do not have permission to write to %s: %w", path, os.ErrPermission)
	}
	return fmt.Errorf("opening %s: %w", path, err)
}

// Write copies size bytes from src to dst, syncing as it goes, and returns a
// hash of every block so the result can be checked against the drive later.
// The image is read once, as a stream, so it can arrive through a pipe.
func Write(ctx context.Context, dst *os.File, src io.Reader, size int64, report func(Progress)) ([][32]byte, error) {
	buf := make([]byte, BlockSize)
	sums := make([][32]byte, 0, (size+BlockSize-1)/BlockSize)
	var done, unsynced int64

	for done < size {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n := int(min(BlockSize, size-done))
		if got, err := io.ReadFull(src, buf[:n]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, fmt.Errorf("the image stopped arriving after %d of %d bytes", done+int64(got), size)
			}
			return nil, fmt.Errorf("reading the image: %w", err)
		}
		sums = append(sums, sha256.Sum256(buf[:n]))
		if _, err := dst.Write(buf[:n]); err != nil {
			return nil, explainWrite(dst.Name(), done, err)
		}
		done += int64(n)
		unsynced += int64(n)

		if unsynced >= syncEvery || done == size {
			if err := dst.Sync(); err != nil {
				return nil, explainWrite(dst.Name(), done, err)
			}
			unsynced = 0
		}
		report(Progress{Phase: Writing, Done: done, Total: size})
	}
	return sums, nil
}

func explainWrite(path string, offset int64, err error) error {
	switch {
	case errors.Is(err, syscall.ENOSPC):
		return fmt.Errorf("%s filled up after %s: it is smaller than the image", path, sizeString(offset))
	case errors.Is(err, syscall.EIO):
		return fmt.Errorf("%s reported a hardware error near %s; the drive may be failing, or was unplugged", path, sizeString(offset))
	case errors.Is(err, syscall.EROFS), errors.Is(err, syscall.EPERM):
		return fmt.Errorf("%s refused the write; it may be write-protected", path)
	}
	return fmt.Errorf("writing %s: %w", path, err)
}

// Verify reads the image back from the drive and checks every block against
// the hashes recorded while writing.
//
// The read bypasses the page cache where the device allows it. Otherwise it
// would compare the data with the copy still sitting in memory and prove
// nothing about the drive — which is precisely how a counterfeit stick that
// reports more space than it has gets past a naive check.
func Verify(ctx context.Context, path string, size int64, sums [][32]byte, report func(Progress)) error {
	if report == nil {
		report = func(Progress) {}
	}
	f, direct, err := openUncached(path)
	if err != nil {
		return fmt.Errorf("opening %s to check it: %w", path, err)
	}
	defer f.Close()

	buf := alignedBuffer(BlockSize, directAlign)
	for i, want := range sums {
		if err := ctx.Err(); err != nil {
			return err
		}
		offset := int64(i) * BlockSize
		n := int(min(BlockSize, size-offset))
		// Direct reads must cover whole sectors, so the last short block is
		// read in full and only the part that belongs to the image compared.
		length := n
		if direct {
			length = (n + directAlign - 1) / directAlign * directAlign
		}
		got, err := io.ReadFull(f, buf[:length])
		if got < n {
			if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return fmt.Errorf("%s ended after %s, before the whole image could be read back", path, sizeString(offset+int64(got)))
			}
			return fmt.Errorf("reading %s back near %s: %w", path, sizeString(offset), err)
		}
		if sha256.Sum256(buf[:n]) != want {
			return fmt.Errorf("verification failed: %s returned different data between %s and %s than was written. "+
				"The drive may be failing, or be a counterfeit that reports more space than it has",
				path, sizeString(offset), sizeString(offset+int64(n)))
		}
		report(Progress{Phase: Verifying, Done: offset + int64(n), Total: size})
	}
	return nil
}

// directAlign is the alignment O_DIRECT needs: 4096 covers drives with 512
// and 4096 byte sectors alike.
const directAlign = 4096

// openUncached opens a file for reading around the page cache, falling back to
// an ordinary read on filesystems that do not support that.
func openUncached(path string) (*os.File, bool, error) {
	if f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_DIRECT, 0); err == nil {
		return f, true, nil
	}
	f, err := os.Open(path)
	return f, false, err
}

// alignedBuffer returns a buffer whose start is aligned for direct I/O.
func alignedBuffer(size, align int) []byte {
	buf := make([]byte, size+align)
	off := int(uintptr(unsafe.Pointer(&buf[0])) & uintptr(align-1))
	if off != 0 {
		off = align - off
	}
	return buf[off : off+size]
}

// blkrrpart is the ioctl that asks the kernel to re-read a partition table.
const blkrrpart = 0x125F

func rereadPartitions(f *os.File) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), blkrrpart, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func sizeString(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < 3 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

// The privileged helper.
//
// When the user cannot write to a drive, the write runs in a separate copy of
// Oliver started through pkexec or sudo. Only the helper runs as root: it is
// handed the image on standard input — so root never opens a file in the
// user's home — and it reports progress on standard output, one line per
// step, for the unprivileged side to draw.

// Message is one line from the helper.
type Message struct {
	Progress *Progress
	// Error is set when the write failed, and says why.
	Error string
	// Done is set once everything succeeded.
	Done bool
}

func (m Message) String() string {
	switch {
	case m.Done:
		return "done"
	case m.Error != "":
		return "error " + strconv.Quote(m.Error)
	case m.Progress != nil:
		return fmt.Sprintf("progress %s %d %d", m.Progress.Phase, m.Progress.Done, m.Progress.Total)
	}
	return ""
}

// ParseMessage decodes a line written by the helper.
func ParseMessage(line string) (Message, error) {
	line = strings.TrimSpace(line)
	kind, rest, _ := strings.Cut(line, " ")
	switch kind {
	case "done":
		return Message{Done: true}, nil
	case "error":
		text, err := strconv.Unquote(rest)
		if err != nil {
			text = rest
		}
		return Message{Error: text}, nil
	case "progress":
		f := strings.Fields(rest)
		if len(f) == 3 {
			done, err1 := strconv.ParseInt(f[1], 10, 64)
			total, err2 := strconv.ParseInt(f[2], 10, 64)
			if err1 == nil && err2 == nil {
				return Message{Progress: &Progress{Phase: Phase(f[0]), Done: done, Total: total}}, nil
			}
		}
	}
	return Message{}, fmt.Errorf("unrecognised line from the writer: %q", line)
}
