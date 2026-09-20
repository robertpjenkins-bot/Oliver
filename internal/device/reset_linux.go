package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// usbDevFSReset makes the kernel reset a USB device: the same thing that
// happens when it is unplugged and plugged back in, without the unplugging.
const usbDevFSReset = 0x5514

// USBPath finds the USB device behind a block device, by walking up from the
// block device to the first ancestor that has a bus and device number — which
// is what a USB device has and its interfaces do not.
func USBPath(d *Device) (string, bool) {
	link, err := filepath.EvalSymlinks(filepath.Join("/sys/block", d.Name))
	if err != nil {
		return "", false
	}
	for dir := link; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		busRaw, err := os.ReadFile(filepath.Join(dir, "busnum"))
		if err != nil {
			continue
		}
		devRaw, err := os.ReadFile(filepath.Join(dir, "devnum"))
		if err != nil {
			continue
		}
		bus, err1 := strconv.Atoi(strings.TrimSpace(string(busRaw)))
		num, err2 := strconv.Atoi(strings.TrimSpace(string(devRaw)))
		if err1 != nil || err2 != nil {
			continue
		}
		return fmt.Sprintf("/dev/bus/usb/%03d/%03d", bus, num), true
	}
	return "", false
}

// ResetUSB unplugs and replugs a USB drive without anybody touching it.
//
// A stick that has hit an error can be left read-only by its own controller
// until it is power-cycled, and this is the software way to do that. The
// device disappears and comes back, so the caller must look it up again
// afterwards — it may even come back with a different name.
func ResetUSB(d *Device) error {
	path, ok := USBPath(d)
	if !ok {
		return fmt.Errorf("%s is not a USB device, so there is nothing to reset", d.Path)
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return openHint(path, err)
	}
	defer f.Close()

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), usbDevFSReset, 0); errno != 0 {
		return fmt.Errorf("the kernel refused to reset %s: %w", path, errno)
	}
	return nil
}
