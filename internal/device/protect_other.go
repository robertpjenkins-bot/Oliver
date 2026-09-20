//go:build !linux

package device

import "fmt"

// Elsewhere there is no equivalent flag to clear.
func clearReadOnly(path string) error {
	return fmt.Errorf("clearing a device's read-only flag is only implemented on Linux")
}
