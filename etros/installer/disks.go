package main

import (
	"fmt"
	"os"
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
	size int64
}
