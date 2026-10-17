//go:build !linux

package main

import "errors"

// The partitions-and-files install is Linux only (it needs UDisks).
func filesInstallSupported() bool { return false }

func filesInstall(imagePath, id string, rep Reporter) error {
	return errors.New("not available on this system")
}
