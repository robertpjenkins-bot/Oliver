//go:build !linux

package main

import "errors"

func addR36SSupported() bool { return false }

func (t *tui) remakeWithR36S(dir string) error {
	return errors.New("adding the R36S to a card needs Linux; make a new card with Etros Maker instead")
}

// shareFillsCard can only be told on Linux; elsewhere the card is left as is.
func shareFillsCard(dir string) bool { return false }

// Proton goes on SHARE, which only Linux can open without a password.
func (t *tui) offerProton(dir string) {}

func (t *tui) remakeWithoutWindows(dir string) error {
	return errors.New("removing the Windows 11 partitions needs Linux")
}

func cardHasWindows(dir string) bool { return false }

// Arch Linux goes on SHARE, which only Linux can open without a password.
func (t *tui) offerArch(dir string) error { return nil }

func (t *tui) getArchTarball() (string, error) { return "", errors.New("needs Linux") }

func putArchOnCard(disk, tarball string, rep Reporter) error { return nil }
