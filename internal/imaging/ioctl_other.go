//go:build !linux

package imaging

import "os"

// Elsewhere there is nothing portable to ask for: verification reads whatever
// the operating system chooses to hand back, and partitions appear when the
// drive is replugged.
func dropCache(*os.File) {}

func rereadPartitions(f *os.File) { _ = f.Sync() }
