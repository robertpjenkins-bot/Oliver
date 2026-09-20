//go:build !linux

package device

import "fmt"

// USBPath and ResetUSB need the Linux USB filesystem.
func USBPath(d *Device) (string, bool) { return "", false }

func ResetUSB(d *Device) error {
	return fmt.Errorf("resetting a USB device is only implemented on Linux")
}
