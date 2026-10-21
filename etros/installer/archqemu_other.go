//go:build !(linux && amd64)

package main

// ARM computers run the Pi's programs themselves; other systems don't make
// Ollie OS at all.
var qemuAarch64 []byte
