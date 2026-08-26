package vbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isoOptions describes an image to fabricate for a test.
type isoOptions struct {
	label     string
	publisher string
	bootable  bool
	// truncated writes only the reserved system area, as an interrupted
	// download would leave behind.
	truncated bool
	// notISO writes a file with no ISO 9660 signature at all.
	notISO bool
	pad    int64
}

// writeISO fabricates an image with the volume descriptors a real one carries.
func writeISO(t *testing.T, name string, o isoOptions) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)

	descriptor := func(kind byte) []byte {
		b := make([]byte, sectorSize)
		b[0] = kind
		copy(b[1:], "CD001")
		b[6] = 1
		return b
	}

	image := make([]byte, firstDescriptor)
	if o.notISO {
		image = append(image, make([]byte, sectorSize)...)
	} else if !o.truncated {
		if o.bootable {
			boot := descriptor(0)
			copy(boot[7:], "EL TORITO SPECIFICATION")
			image = append(image, boot...)
		}
		primary := descriptor(1)
		copy(primary[40:72], []byte(o.label+strings.Repeat(" ", 32)))
		copy(primary[318:446], []byte(o.publisher+strings.Repeat(" ", 128)))
		image = append(image, primary...)
		image = append(image, descriptor(255)...)
	}
	if o.pad > 0 {
		image = append(image, make([]byte, o.pad)...)
	}

	if err := os.WriteFile(path, image, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInspectISOReadsTheImageRatherThanTheFileName(t *testing.T) {
	path := writeISO(t, "downloaded.iso", isoOptions{
		label: "CCCOMA_X64FRE_EN-US_DV9", publisher: "MICROSOFT CORPORATION", bootable: true,
	})

	iso, err := InspectISO(path)
	if err != nil {
		t.Fatalf("InspectISO: %v", err)
	}
	if iso.Label != "CCCOMA_X64FRE_EN-US_DV9" {
		t.Errorf("label = %q", iso.Label)
	}
	if iso.Publisher != "MICROSOFT CORPORATION" {
		t.Errorf("publisher = %q", iso.Publisher)
	}
	if !iso.Bootable {
		t.Error("an image with a boot record is bootable")
	}
	if !iso.LooksLikeWindows() {
		t.Error("an image published by Microsoft should be recognised as one")
	}
}

// An image with no boot catalogue produces a machine that starts to a black
// screen, which looks like a broken virtual machine rather than a bad file.
func TestInspectISONoticesAnImageNothingCanBootFrom(t *testing.T) {
	iso, err := InspectISO(writeISO(t, "extracted.iso", isoOptions{label: "SOMETHING"}))
	if err != nil {
		t.Fatalf("InspectISO: %v", err)
	}
	if iso.Bootable {
		t.Error("no boot record means not bootable")
	}
}

func TestInspectISORejectsSomethingThatIsNotAnImage(t *testing.T) {
	_, err := InspectISO(writeISO(t, "notes.iso", isoOptions{notISO: true}))
	if err == nil || !strings.Contains(err.Error(), "not an ISO") {
		t.Errorf("err = %v; want a complaint that it is not an ISO", err)
	}
}

func TestInspectISORejectsAnInterruptedDownload(t *testing.T) {
	_, err := InspectISO(writeISO(t, "half.iso", isoOptions{truncated: true}))
	if err == nil || !strings.Contains(err.Error(), "too small") {
		t.Errorf("err = %v; want a complaint about the size", err)
	}
}

func TestDetectReleaseReadsTheNamesInstallersActuallyHave(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"/home/u/Downloads/Win11_23H2_English_x64v2.iso", "win11"},
		{"/home/u/Downloads/Win10_22H2_English_x64.iso", "win10"},
		{"windows-11-pro.iso", "win11"},
		{"Windows 8.1 x64.iso", "win8"},
		{"win7ultimate.iso", "win7"},
		{"ubuntu-24.04-desktop-amd64.iso", ""},
		{"installer.iso", ""},
	} {
		if got := DetectRelease(tc.name, ""); got != tc.want {
			t.Errorf("DetectRelease(%q) = %q; want %q", tc.name, got, tc.want)
		}
	}
}

func TestReleasesCarryMicrosoftsOwnMinimums(t *testing.T) {
	win11, ok := ReleaseByID("win11")
	if !ok {
		t.Fatal("win11 should be known")
	}
	if win11.MinRAMMB != 4096 || win11.MinDiskGB != 64 || !win11.NeedsTPM || !win11.NeedsEFI {
		t.Errorf("Windows 11's requirements are 4 GB, 64 GB, EFI and a TPM: %+v", win11)
	}
	if r := ReleaseOrDefault("nonsense"); r.ID != DefaultRelease {
		t.Errorf("an unknown release should fall back to %s, got %s", DefaultRelease, r.ID)
	}
}

// writeDir creates a file and the directories above it, for tests that need a
// folder to be occupied.
func writeDir(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("<VirtualBox/>"), 0o644)
}
