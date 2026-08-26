package vbox

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// ISO is what an installer image says about itself.
//
// Oliver reads the image rather than trusting its file name, for the same
// reason it parses a PE header instead of guessing from an extension: a
// download called `windows.iso` may be an unbootable extract, a Linux image,
// or half a file, and every one of those failures looks identical once the
// virtual machine is already booting into a black screen.
type ISO struct {
	Path      string `json:"path"`
	Size      int64  `json:"size_bytes"`
	Label     string `json:"label,omitempty"`
	Publisher string `json:"publisher,omitempty"`
	// Bootable reports whether the image carries an El Torito boot catalogue.
	// Without one a virtual machine will not start from it at all.
	Bootable bool `json:"bootable"`
	// Release is the Windows generation the image appears to hold, or "" when
	// nothing in it says.
	Release string `json:"release,omitempty"`
}

const (
	sectorSize      = 2048
	firstDescriptor = 16 * sectorSize // ISO 9660 reserves the first 16 sectors
	maxDescriptors  = 32
)

// InspectISO reads an installer image's volume descriptors.
func InspectISO(path string) (*ISO, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading the installer image: %w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not an installer image", path)
	}

	iso := &ISO{Path: path, Size: st.Size()}
	if err := iso.readDescriptors(f); err != nil {
		return nil, err
	}
	iso.Release = DetectRelease(path, iso.Label)
	return iso, nil
}

// readDescriptors walks the volume descriptor set, which begins at a fixed
// offset and ends at a terminator record.
func (iso *ISO) readDescriptors(r io.ReaderAt) error {
	buf := make([]byte, sectorSize)
	for i := 0; i < maxDescriptors; i++ {
		off := int64(firstDescriptor + i*sectorSize)
		if _, err := r.ReadAt(buf, off); err != nil {
			if i == 0 {
				return fmt.Errorf("%s is too small to be an installer image", iso.Path)
			}
			return nil
		}
		if string(buf[1:6]) != "CD001" {
			if i == 0 {
				return fmt.Errorf("%s is not an ISO image (no ISO 9660 signature)", iso.Path)
			}
			return nil
		}
		switch buf[0] {
		case 0: // boot record
			if strings.HasPrefix(strings.TrimSpace(string(buf[7:39])), "EL TORITO SPECIFICATION") {
				iso.Bootable = true
			}
		case 1: // primary volume descriptor
			iso.Label = isoField(buf[40:72])
			iso.Publisher = isoField(buf[318:446])
		case 255: // terminator
			return nil
		}
	}
	return nil
}

// isoField trims the space padding ISO 9660 uses for fixed-width strings.
func isoField(b []byte) string {
	return strings.TrimSpace(strings.TrimRight(string(b), "\x00"))
}

// LooksLikeWindows reports whether anything in the image identifies it as a
// Microsoft installer. It is advice, not a gate: a repacked or localised image
// can be perfectly good and still say nothing recognisable.
func (iso *ISO) LooksLikeWindows() bool {
	haystack := strings.ToLower(iso.Publisher + " " + iso.Label + " " + iso.Path)
	for _, s := range []string{"microsoft", "windows", "win", "cccoma", "cpba", "ccsa"} {
		if strings.Contains(haystack, s) {
			return true
		}
	}
	return false
}

// releasePattern matches the way Windows images are named in practice:
// `Win11_23H2_English_x64.iso`, `windows-10-pro.iso`, `win7ultimate.iso`. The
// trailing check keeps a build number such as `win1000` from reading as
// Windows 10.
var releasePattern = regexp.MustCompile(`(?i)w(?:in)?(?:dows)?[ _.-]*(11|10|8\.1|8|7)(?:[^0-9]|$)`)

// DetectRelease works out which Windows an image holds from its name and
// volume label. It returns "" when neither says, since a wrong guess here sets
// the wrong firmware and the installer then refuses to run at all.
func DetectRelease(path, label string) string {
	for _, s := range []string{label, path} {
		m := releasePattern.FindStringSubmatch(s)
		if m == nil {
			continue
		}
		switch m[1] {
		case "11":
			return "win11"
		case "10":
			return "win10"
		case "8.1", "8":
			return "win8"
		case "7":
			return "win7"
		}
	}
	return ""
}
