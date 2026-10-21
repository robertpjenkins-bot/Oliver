//go:build !linux && !darwin

package main

import "errors"

// A Mac's drive is only changed without a password through Linux's UDisks.
func macInstallSupported() bool { return false }

func (t *tui) macLoop() bool {
	note(cRed + "Etros Mac Installer needs Linux (like a Steam Machine or Steam Deck)." + cReset)
	return false
}

func runMacInstaller(o Options) error {
	return errors.New("Etros Mac Installer needs Linux (like a Steam Machine or Steam Deck)")
}
