package main

import "testing"

func TestParsePlist(t *testing.T) {
	out := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>Content</key>
			<string>GUID_partition_scheme</string>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>OSInternal</key>
			<false/>
			<key>Partitions</key>
			<array>
				<dict>
					<key>Content</key>
					<string>EFI</string>
					<key>DeviceIdentifier</key>
					<string>disk0s1</string>
					<key>Size</key>
					<integer>209715200</integer>
					<key>VolumeName</key>
					<string>EFI</string>
				</dict>
				<dict>
					<key>Content</key>
					<string>Apple_APFS</string>
					<key>DeviceIdentifier</key>
					<string>disk0s2</string>
					<key>Size</key>
					<integer>180000000000</integer>
				</dict>
				<dict>
					<key>Content</key>
					<string>Microsoft Basic Data</string>
					<key>DeviceIdentifier</key>
					<string>disk0s3</string>
					<key>Size</key>
					<integer>70000000000</integer>
					<key>VolumeName</key>
					<string>BOOTCAMP</string>
				</dict>
			</array>
			<key>Size</key>
			<integer>251000193024</integer>
		</dict>
	</array>
	<key>WholeDisks</key>
	<array><string>disk0</string></array>
	<key>Internal</key>
	<true/>
</dict>
</plist>`)
	v, err := parsePlist(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := plistString(v, "AllDisksAndPartitions", 0, "Partitions", 2, "VolumeName"); got != "BOOTCAMP" {
		t.Errorf("VolumeName %q", got)
	}
	if got := plistInt(v, "AllDisksAndPartitions", 0, "Size"); got != 251000193024 {
		t.Errorf("Size %d", got)
	}
	if plistGet(v, "Internal") != true || plistGet(v, "AllDisksAndPartitions", 0, "OSInternal") != false {
		t.Error("booleans")
	}
	if plistString(v, "AllDisksAndPartitions", 5, "x") != "" || partNum("disk0s12") != 12 {
		t.Error("missing paths / partNum")
	}
}

func TestMacTargets(t *testing.T) {
	part := func(id, content, name string, gb int64) map[string]any {
		return map[string]any{"DeviceIdentifier": id, "Content": content, "VolumeName": name, "Size": gb << 30}
	}
	list := map[string]any{"AllDisksAndPartitions": []any{
		map[string]any{"DeviceIdentifier": "disk0", "Size": int64(251) << 30, "Partitions": []any{
			part("disk0s1", "EFI", "EFI", 0), part("disk0s2", "Apple_APFS", "", 180), part("disk0s3", "Microsoft Basic Data", "ETROS", 70),
		}},
		map[string]any{"DeviceIdentifier": "disk2", "Size": int64(64) << 30, "Partitions": []any{
			part("disk2s1", "Microsoft Basic Data", "USB", 64),
		}},
		map[string]any{"DeviceIdentifier": "disk3", "Size": int64(4) << 30, "Partitions": []any{}},
	}}
	info := func(id string) any {
		return map[string]any{"Internal": id == "disk0", "MediaName": map[string]string{"disk0": "APPLE SSD", "disk2": "SanDisk"}[id]}
	}
	disks := readDisks(list, info)
	if len(disks) != 3 || !disks[0].internal || disks[1].internal || disks[0].name != "APPLE SSD" || len(disks[0].parts) != 3 {
		t.Fatalf("disks: %+v", disks)
	}
	ts := macTargets(disks, "disk0s2")
	got := map[string]macTarget{}
	for _, x := range ts {
		key := x.part.id
		if x.kind == "whole" {
			key = x.disk.id
		}
		got[key] = x
	}
	if x := got["disk0s1"]; x.why == "" {
		t.Error("EFI usable")
	}
	if x := got["disk0s2"]; x.kind != "room" || x.why != "" {
		t.Errorf("macOS: %+v", x)
	}
	if x := got["disk0s3"]; x.kind != "part" || x.why != "" || x.after != "disk0s2" {
		t.Errorf("ExFAT: %+v", x)
	}
	if _, ok := got["disk0"]; ok {
		t.Error("whole internal drive offered")
	}
	if x := got["disk2"]; x.kind != "whole" || x.why != "" {
		t.Errorf("USB drive: %+v", x)
	}
	if x := got["disk2s1"]; x.why == "" {
		t.Error("first partition of a USB drive usable on its own")
	}
	if x := got["disk3"]; x.why == "" {
		t.Error("4 GB drive usable")
	}
	// Etros there already: BATOCERA takes its SHARE with it.
	disks[0].parts = append(disks[0].parts[:2], dPart{"disk0s3", "Microsoft Basic Data", "BATOCERA", 8 << 30}, dPart{"disk0s4", "Linux Filesystem", "", 60 << 30})
	for _, x := range macTargets(disks, "disk0s2") {
		if x.part.id == "disk0s3" && (len(x.extra) != 1 || x.why != "") {
			t.Errorf("Etros: %+v", x)
		}
	}
	// Not knowing which macOS is running: no APFS is ever erased.
	for _, x := range macTargets(disks, "") {
		if x.part.id == "disk0s2" && x.kind != "room" {
			t.Errorf("APFS without bootStore: %+v", x)
		}
	}
}
