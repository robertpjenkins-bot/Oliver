package device

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// blkROSet sets or clears the kernel's read-only flag for a block device. It
// is the ioctl behind `blockdev --setro` and `--setrw`.
const blkROSet = 0x125D

// clearReadOnly tells the kernel this device may be written to again.
func clearReadOnly(path string) error {
	// The device has to be opened to be told, and a read-only device cannot be
	// opened for writing — which is the point — so this opens it read-only.
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return openHint(path, err)
	}
	defer f.Close()

	var off int32 // 0 means writable
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), blkROSet,
		uintptr(unsafe.Pointer(&off))); errno != 0 {
		if errno == syscall.EACCES || errno == syscall.EPERM {
			return fmt.Errorf("permission denied: clearing this flag needs root")
		}
		return errno
	}
	return nil
}
