package main

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

// Disk is a removable drive the card can be written to.
type Disk struct {
	ID    string `json:"id"`   // what to pass back to openDisk
	Path  string `json:"path"` // device path, for display
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Label string `json:"label"` // "SD card reader (31.9 GB)"
}

func describe(d *Disk) {
	d.Label = fmt.Sprintf("%s (%.1f GB) — %s", d.Name, float64(d.Size)/1e9, d.Path)
}

// diskTarget is an open disk being written.
type diskTarget struct {
	*os.File
	size   int64
	udisks *dbus.Conn // Linux: the connection the card was opened through
}

// errNeedsAdmin means writing to the card needs administrator rights after
// all.
type errNeedsAdmin struct{ error }

func (e errNeedsAdmin) Unwrap() error { return e.error }
