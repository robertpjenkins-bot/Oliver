package prefix

import (
	"os"
	"path/filepath"
	"testing"
)

func preparedPrefix(t *testing.T) *Prefix {
	t.Helper()
	m := &Manager{Root: t.TempDir()}
	p := m.For("", "test")
	if err := os.MkdirAll(p.dosDevices(), 0o755); err != nil {
		t.Fatal(err)
	}
	// Every real prefix has these two.
	if err := os.Symlink("../drive_c", filepath.Join(p.dosDevices(), "c:")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/", filepath.Join(p.dosDevices(), "z:")); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMapDeviceCreatesBothLinks(t *testing.T) {
	p := preparedPrefix(t)
	if err := p.MapDevice("e", "/dev/sda", "/run/media/deck/USB"); err != nil {
		t.Fatal(err)
	}

	// The "::" link is what a program opening a drive for raw access follows.
	raw, err := os.Readlink(filepath.Join(p.dosDevices(), "e::"))
	if err != nil || raw != "/dev/sda" {
		t.Errorf("e:: = %q, %v; want /dev/sda", raw, err)
	}
	dir, err := os.Readlink(filepath.Join(p.dosDevices(), "e:"))
	if err != nil || dir != "/run/media/deck/USB" {
		t.Errorf("e: = %q, %v", dir, err)
	}
}

func TestMapDeviceWithoutAMountPointExposesNothingElse(t *testing.T) {
	p := preparedPrefix(t)
	if err := p.MapDevice("e", "/dev/sda", ""); err != nil {
		t.Fatal(err)
	}
	dir, err := os.Readlink(filepath.Join(p.dosDevices(), "e:"))
	if err != nil {
		t.Fatal(err)
	}
	// The letter must not fall back to the real filesystem root.
	if dir == "/" {
		t.Fatal("an unmounted device must not expose the whole filesystem")
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		t.Errorf("expected an empty directory inside the prefix, got %q", dir)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("the placeholder directory should be empty, has %d entries", len(entries))
	}
}

func TestMapDeviceRejectsBadLetters(t *testing.T) {
	p := preparedPrefix(t)
	for _, letter := range []string{"", "ee", "1", "c"} {
		if err := p.MapDevice(letter, "/dev/sda", ""); err == nil {
			t.Errorf("MapDevice(%q) was allowed", letter)
		}
	}
	// c: is the prefix's own Windows drive; remapping it breaks the prefix.
	if _, err := os.Readlink(filepath.Join(p.dosDevices(), "c:")); err != nil {
		t.Error("c: was damaged by a rejected mapping")
	}
}

func TestDevicesListsOnlyRealDeviceMappings(t *testing.T) {
	p := preparedPrefix(t)
	if err := p.MapDevice("e", "/dev/sda", "/mnt/usb"); err != nil {
		t.Fatal(err)
	}
	if err := p.MapDevice("f", "/dev/sdb", ""); err != nil {
		t.Fatal(err)
	}

	got, err := p.Devices()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("Devices() = %+v, want 2 (c: and z: are not devices)", got)
	}
	if got[0].Letter != "e" || got[0].Device != "/dev/sda" || got[0].Mount != "/mnt/usb" {
		t.Errorf("first mapping = %+v", got[0])
	}
	if got[1].Letter != "f" || got[1].Device != "/dev/sdb" {
		t.Errorf("second mapping = %+v", got[1])
	}
}

func TestFreeLetterSkipsUsedAndReservedLetters(t *testing.T) {
	p := preparedPrefix(t)
	first, err := p.FreeLetter()
	if err != nil {
		t.Fatal(err)
	}
	if first == "c" || first == "z" {
		t.Errorf("FreeLetter returned a reserved letter %q", first)
	}
	if err := p.MapDevice(first, "/dev/sda", ""); err != nil {
		t.Fatal(err)
	}
	second, err := p.FreeLetter()
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Errorf("FreeLetter returned %q twice", second)
	}
}

func TestUnmapRemovesBothLinksAndProtectsReserved(t *testing.T) {
	p := preparedPrefix(t)
	if err := p.MapDevice("e", "/dev/sda", ""); err != nil {
		t.Fatal(err)
	}
	if err := p.UnmapDevice("e"); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{":", "::"} {
		if _, err := os.Lstat(filepath.Join(p.dosDevices(), "e"+suffix)); !os.IsNotExist(err) {
			t.Errorf("e%s survived unmapping", suffix)
		}
	}

	if err := p.UnmapDevice("e"); err == nil {
		t.Error("unmapping an unmapped letter should report an error")
	}
	for _, letter := range []string{"c", "z"} {
		if err := p.UnmapDevice(letter); err == nil {
			t.Errorf("UnmapDevice(%q) was allowed", letter)
		}
	}
	if _, err := os.Readlink(filepath.Join(p.dosDevices(), "z:")); err != nil {
		t.Error("z: was removed despite being reserved")
	}
}

func TestRemappingALetterReplacesIt(t *testing.T) {
	p := preparedPrefix(t)
	if err := p.MapDevice("e", "/dev/sda", ""); err != nil {
		t.Fatal(err)
	}
	if err := p.MapDevice("e", "/dev/sdb", ""); err != nil {
		t.Fatalf("remapping should replace, not fail: %v", err)
	}
	got, _ := p.Devices()
	if len(got) != 1 || got[0].Device != "/dev/sdb" {
		t.Errorf("Devices() = %+v, want a single mapping to /dev/sdb", got)
	}
}
