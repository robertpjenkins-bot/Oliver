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

func (t *tui) remakeWithWindows(dir string, windowsBytes int64) error {
	return errors.New("adding Windows 11 to a card needs Linux")
}

func cardHasWindows(dir string) bool { return false }
