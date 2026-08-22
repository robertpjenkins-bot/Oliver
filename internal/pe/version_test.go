package pe

import (
	"path/filepath"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/petest"
)

func TestVersionResourceIsParsed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "anything.exe")
	err := petest.WriteFile(path, petest.Options{
		Machine: petest.MachineI386,
		VersionStrings: map[string]string{
			"ProductName":      "Rufus",
			"OriginalFilename": "rufus-3.10.exe",
			"CompanyName":      "Akeo Consulting",
			"FileDescription":  "Rufus",
			"FileVersion":      "3.10.1647",
			"ProductVersion":   "3.10.1647",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	v := got.VersionInfo
	if v.ProductName != "Rufus" {
		t.Errorf("ProductName = %q, want Rufus", v.ProductName)
	}
	if v.OriginalFilename != "rufus-3.10.exe" {
		t.Errorf("OriginalFilename = %q", v.OriginalFilename)
	}
	if v.CompanyName != "Akeo Consulting" {
		t.Errorf("CompanyName = %q", v.CompanyName)
	}
	if v.FileVersion != "3.10.1647" {
		t.Errorf("FileVersion = %q", v.FileVersion)
	}
	if want := "Rufus 3.10.1647 (Akeo Consulting)"; v.Describe() != want {
		t.Errorf("Describe() = %q, want %q", v.Describe(), want)
	}
}

func TestVersionResourceAbsentIsEmptyNotAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bare.exe")
	if err := petest.WriteFile(path, petest.Options{Machine: petest.MachineI386}); err != nil {
		t.Fatal(err)
	}
	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.VersionInfo.Empty() {
		t.Errorf("expected an empty version resource, got %+v", got.VersionInfo)
	}
	if got.VersionInfo.Describe() != "" {
		t.Errorf("Describe() on an empty resource = %q, want empty", got.VersionInfo.Describe())
	}
}

func TestDescribeFallsBackToFileDescription(t *testing.T) {
	v := VersionInfo{FileDescription: "Some Tool", FileVersion: "1.2"}
	if got := v.Describe(); got != "Some Tool 1.2" {
		t.Errorf("Describe() = %q", got)
	}
	// Product version is preferred over file version when both are present.
	v = VersionInfo{ProductName: "Thing", FileVersion: "1.0.0.5", ProductVersion: "1.0"}
	if got := v.Describe(); got != "Thing 1.0" {
		t.Errorf("Describe() = %q, want the product version", got)
	}
}

func TestReadUTF16ValueHandlesPaddingAndTruncation(t *testing.T) {
	// A value preceded by padding, as the 4-byte alignment produces.
	data := append([]byte{0, 0}, utf16Bytes("Hello")...)
	data = append(data, 0, 0)
	if got := readUTF16Value(data, 0); got != "Hello" {
		t.Errorf("readUTF16Value = %q, want Hello", got)
	}
	// A run with no terminator must not read off the end.
	if got := readUTF16Value(utf16Bytes("Unterminated"), 0); got != "Unterminated" {
		t.Errorf("readUTF16Value = %q", got)
	}
	if got := readUTF16Value(nil, 0); got != "" {
		t.Errorf("readUTF16Value on empty data = %q", got)
	}
	if got := readUTF16Value([]byte{1, 2}, 99); got != "" {
		t.Errorf("out-of-range offset should be empty, got %q", got)
	}
}
