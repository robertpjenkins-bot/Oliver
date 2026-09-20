package imaging

import (
	"os"
	"syscall"
)

// Linux block device ioctls. Both are advisory here: neither failing means the
// write did not happen, so their errors are deliberately discarded.
const (
	blkRRPart = 0x125F // re-read the partition table
	blkFlsBuf = 0x1261 // discard this device's buffer cache
)

// dropCache asks the kernel to forget what it has cached for a device, so a
// verification pass reads the medium rather than the copy in memory.
func dropCache(f *os.File) {
	ioctl(f, blkFlsBuf)
}

// rereadPartitions asks the kernel to re-read a device's partition table, so
// the partitions just written appear without unplugging the drive.
func rereadPartitions(f *os.File) {
	if err := f.Sync(); err != nil {
		return
	}
	ioctl(f, blkRRPart)
}

func ioctl(f *os.File, request uintptr) {
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), request, 0)
}
