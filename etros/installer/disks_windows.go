//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func powershell(script string) ([]byte, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.CombinedOutput()
}

type winDisk struct {
	Number       int    `json:"Number"`
	FriendlyName string `json:"FriendlyName"`
	Size         int64  `json:"Size"`
	BusType      string `json:"BusType"`
	IsBoot       bool   `json:"IsBoot"`
	IsSystem     bool   `json:"IsSystem"`
}

func listDisks() ([]Disk, error) {
	out, err := powershell(`@(Get-Disk | Select-Object Number,FriendlyName,Size,@{n='BusType';e={"$($_.BusType)"}},IsBoot,IsSystem) | ConvertTo-Json -Compress`)
	if err != nil {
		return nil, fmt.Errorf("listing disks: %s", out)
	}
	var raw []winDisk
	if err := json.Unmarshal(out, &raw); err != nil {
		var one winDisk
		if json.Unmarshal(out, &one) != nil {
			return nil, fmt.Errorf("listing disks: %w", err)
		}
		raw = []winDisk{one}
	}
	var disks []Disk
	for _, w := range raw {
		bus := strings.ToUpper(w.BusType)
		if w.IsBoot || w.IsSystem || !(bus == "USB" || bus == "SD" || bus == "MMC") {
			continue
		}
		path := fmt.Sprintf(`\\.\PhysicalDrive%d`, w.Number)
		d := Disk{ID: fmt.Sprint(w.Number), Path: path, Name: strings.TrimSpace(w.FriendlyName), Size: w.Size}
		if d.Name == "" {
			d.Name = "Removable drive"
		}
		describe(&d)
		disks = append(disks, d)
	}
	return disks, nil
}

func canWriteDisks() (bool, string) {
	if windows.GetCurrentProcessToken().IsElevated() {
		return true, ""
	}
	return false, "Writing to a card needs administrator rights. Close this and run the installer again, choosing Yes when Windows asks."
}

func openDisk(id string) (*diskTarget, error) {
	disks, err := listDisks()
	if err != nil {
		return nil, err
	}
	var disk *Disk
	for i := range disks {
		if disks[i].ID == id {
			disk = &disks[i]
		}
	}
	if disk == nil {
		return nil, fmt.Errorf("disk %s is not a removable drive", id)
	}
	// Remove the card's partitions first, so Windows lets go of its drive
	// letters and lets us write the whole disk. A blank card has none,
	// which Clear-Disk reports as an error we can ignore.
	powershell(fmt.Sprintf(`Clear-Disk -Number %s -RemoveData -RemoveOEM -Confirm:$false`, id))
	f, err := os.OpenFile(disk.Path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", disk.Path, err)
	}
	return &diskTarget{File: f, size: disk.Size}, nil
}

func (d *diskTarget) Finish() error {
	d.Sync()
	err := d.Close()
	powershell(`Update-HostStorageCache`)
	return err
}
