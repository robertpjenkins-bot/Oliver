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

func TestPlanThisMac(t *testing.T) {
	disk := func(parts ...map[string]any) any {
		var ps []any
		for _, p := range parts {
			ps = append(ps, p)
		}
		return map[string]any{"AllDisksAndPartitions": []any{map[string]any{"DeviceIdentifier": "disk0", "Size": int64(251e9), "Partitions": ps}}}
	}
	part := func(id, content, name string) map[string]any {
		return map[string]any{"DeviceIdentifier": id, "Content": content, "VolumeName": name, "Size": int64(50e9)}
	}
	efi, apfs := part("disk0s1", "EFI", "EFI"), part("disk0s2", "Apple_APFS", "")

	pl, err := planThisMac(disk(efi, apfs, part("disk0s3", "Microsoft Basic Data", "BOOTCAMP")))
	if err != nil || len(pl.bootCamp) != 1 || pl.store != "disk0s2" || pl.isEtros {
		t.Errorf("Boot Camp: %+v %v", pl, err)
	}
	// A partition made by hand in Disk Utility (MS-DOS or ExFAT) is used too.
	if pl, err := planThisMac(disk(efi, apfs, part("disk0s3", "Microsoft Basic Data", "ETROS"))); err != nil || len(pl.bootCamp) != 1 {
		t.Errorf("hand-made: %+v %v", pl, err)
	}
	// No Boot Camp: room comes from APFS.
	if pl, err := planThisMac(disk(efi, apfs)); err != nil || len(pl.bootCamp) != 0 || pl.store != "disk0s2" {
		t.Errorf("no Boot Camp: %+v %v", pl, err)
	}
	if _, err := planThisMac(disk(efi, apfs, part("disk0s3", "Linux Filesystem", ""))); err == nil {
		t.Error("touched a Linux partition")
	}
	if pl, err := planThisMac(disk(efi, apfs, part("disk0s3", "Microsoft Basic Data", "BATOCERA"), part("disk0s4", "Microsoft Basic Data", "SHARE"))); err != nil || !pl.isEtros {
		t.Errorf("Etros: %+v %v", pl, err)
	}
}
