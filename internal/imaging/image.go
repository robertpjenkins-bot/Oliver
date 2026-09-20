// Package imaging writes disk images to physical drives and prepares drives to
// receive them.
//
// Everything here is done directly: the image is streamed to the device node,
// read back and compared, and a fresh FAT32 volume is laid down byte by byte.
// Nothing shells out to dd, mkfs or a Windows utility running under a
// compatibility layer, because writing boot records to a raw device is exactly
// the operation a translated Windows process never gets to perform.
package imaging

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	isoSectorSize = 2048
	// The first 32 KiB of an ISO are the system area, where a hybrid image
	// hides the MBR that lets a USB stick boot from it.
	isoSystemArea = 16 * isoSectorSize
)

// Image is what Inspect could learn about a file before it is written to a
// drive. Every field is reported rather than acted on: the decision to write
// belongs to the caller, and an image Oliver cannot identify is still an image
// the user may have good reason to write.
type Image struct {
	Path   string `json:"path"`
	Size   int64  `json:"size_bytes"`
	Format string `json:"format"`
	Label  string `json:"label,omitempty"`

	// BootMethods names the firmware that can start this image, as the image
	// itself declares: "BIOS" from an El Torito catalogue entry for x86,
	// "UEFI" from one for EFI.
	BootMethods []string `json:"boot_methods,omitempty"`

	// HybridUSB reports whether the image carries a partition table in its
	// system area. An ISO without one is built to be burned to a disc; copied
	// to a USB stick it may present no partition for the firmware to boot.
	HybridUSB bool `json:"hybrid_usb"`

	// WindowsInstaller reports the Microsoft installation layout, which is
	// worth knowing because those ISOs are never hybrid.
	WindowsInstaller bool `json:"windows_installer,omitempty"`

	// Warnings are the reasons this image may not boot once written. They do
	// not stop a write.
	Warnings []string `json:"warnings,omitempty"`
}

// Bootable reports whether the image declares any way to start a machine.
func (i *Image) Bootable() bool { return len(i.BootMethods) > 0 }

// Describe renders the image for a confirmation prompt.
func (i *Image) Describe() string {
	parts := []string{i.Format, HumanBytes(i.Size)}
	if i.Label != "" {
		parts = append(parts, fmt.Sprintf("%q", i.Label))
	}
	if len(i.BootMethods) > 0 {
		parts = append(parts, "bootable: "+strings.Join(i.BootMethods, "+"))
	} else {
		parts = append(parts, "not bootable")
	}
	return strings.Join(parts, "  ")
}

// Inspect reads an image file's headers. It fails only when the file cannot be
// read at all; an unrecognised format is described as such and left writable.
func Inspect(path string) (*Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not an image file", path)
	}
	if st.Size() == 0 {
		return nil, fmt.Errorf("%s is empty", path)
	}

	img := &Image{Path: path, Size: st.Size(), Format: "unrecognised image"}
	img.HybridUSB = hasPartitionTable(f)

	readISO(f, img)
	if img.Format == "unrecognised image" && img.HybridUSB {
		img.Format = "raw disk image"
	}
	img.addWarnings()
	return img, nil
}

// hasPartitionTable reports whether the first sectors hold an MBR or a GPT.
func hasPartitionTable(r io.ReaderAt) bool {
	mbr := make([]byte, 512)
	if _, err := r.ReadAt(mbr, 0); err != nil {
		return false
	}
	if mbr[510] != 0x55 || mbr[511] != 0xAA {
		return false
	}
	// The signature alone is not enough: an ISO's system area can end in those
	// two bytes by accident. A real table has at least one non-empty entry.
	for e := 0; e < 4; e++ {
		entry := mbr[446+e*16 : 446+(e+1)*16]
		if entry[4] != 0 && binary.LittleEndian.Uint32(entry[12:]) != 0 {
			return true
		}
	}
	// A GPT may be present with only a protective MBR entry we rejected above.
	sig := make([]byte, 8)
	if _, err := r.ReadAt(sig, 512); err == nil && string(sig) == "EFI PART" {
		return true
	}
	return false
}

// readISO fills in what the ISO 9660 and El Torito descriptors say.
func readISO(r io.ReaderAt, img *Image) {
	var pvd []byte
	platforms := map[byte]bool{}

	// Volume descriptors run from sector 16 until a terminator. A handful is
	// all any real image uses; the bound stops a corrupt file spinning here.
	for sector := int64(16); sector < 64; sector++ {
		buf := make([]byte, isoSectorSize)
		if _, err := r.ReadAt(buf, sector*isoSectorSize); err != nil {
			break
		}
		id := string(buf[1:6])
		switch id {
		case "CD001":
		case "BEA01", "NSR02", "NSR03", "TEA01":
			// The UDF bridge format an ISO uses for files above 4 GiB.
			if img.Format == "unrecognised image" {
				img.Format = "UDF image"
			}
			continue
		default:
			continue
		}

		switch buf[0] {
		case 1: // primary volume descriptor
			if pvd == nil {
				pvd = buf
				img.Format = "ISO 9660 image"
				img.Label = strings.TrimSpace(string(buf[40:72]))
			}
		case 0: // boot record
			if strings.HasPrefix(string(buf[7:39]), "EL TORITO SPECIFICATION") {
				catalog := int64(binary.LittleEndian.Uint32(buf[71:75]))
				for p := range bootPlatforms(r, catalog) {
					platforms[p] = true
				}
			}
		case 255: // terminator
			sector = 64
		}
	}

	if pvd != nil {
		img.WindowsInstaller = looksLikeWindows(r, pvd)
	}
	for _, p := range sortedKeys(platforms) {
		switch p {
		case 0x00:
			img.BootMethods = append(img.BootMethods, "BIOS")
		case 0xEF:
			img.BootMethods = append(img.BootMethods, "UEFI")
		}
	}
}

// bootPlatforms reads the El Torito boot catalogue and returns the firmware
// platforms it holds bootable entries for.
func bootPlatforms(r io.ReaderAt, catalogSector int64) map[byte]bool {
	out := map[byte]bool{}
	if catalogSector <= 0 {
		return out
	}
	buf := make([]byte, isoSectorSize)
	if _, err := r.ReadAt(buf, catalogSector*isoSectorSize); err != nil {
		return out
	}
	// The validation entry names the platform of the default entry that
	// follows it; later section headers name theirs.
	if buf[0] == 1 && buf[30] == 0x55 && buf[31] == 0xAA {
		if buf[32] == 0x88 { // default entry, bootable
			out[buf[1]] = true
		}
	}
	for off := 32; off+32 <= len(buf); off += 32 {
		e := buf[off : off+32]
		if e[0] != 0x90 && e[0] != 0x91 {
			continue
		}
		platform := e[1]
		count := int(binary.LittleEndian.Uint16(e[2:4]))
		for n := 1; n <= count && off+32*(n+1) <= len(buf); n++ {
			if buf[off+32*n] == 0x88 {
				out[platform] = true
			}
		}
		if e[0] == 0x91 { // final header
			break
		}
	}
	return out
}

// looksLikeWindows reports whether the image's root directory is the layout of
// a Microsoft installation medium.
func looksLikeWindows(r io.ReaderAt, pvd []byte) bool {
	names := rootEntries(r, pvd)
	var sources, boot bool
	for _, n := range names {
		switch strings.ToLower(n) {
		case "sources":
			sources = true
		case "bootmgr", "setup.exe":
			boot = true
		}
	}
	return sources && boot
}

// rootEntries lists the names directly inside the image's root directory.
func rootEntries(r io.ReaderAt, pvd []byte) []string {
	// The root directory record is embedded in the descriptor at offset 156.
	root := pvd[156:190]
	extent := int64(binary.LittleEndian.Uint32(root[2:6]))
	length := int64(binary.LittleEndian.Uint32(root[10:14]))
	if extent <= 0 || length <= 0 || length > 1<<20 {
		return nil
	}
	buf := make([]byte, length)
	if _, err := r.ReadAt(buf, extent*isoSectorSize); err != nil {
		return nil
	}

	var out []string
	for off := 0; off < len(buf); {
		recLen := int(buf[off])
		if recLen == 0 {
			// Records never straddle a sector; a zero length means padding to
			// the end of this one.
			off = (off/isoSectorSize + 1) * isoSectorSize
			continue
		}
		if off+recLen > len(buf) || recLen < 34 {
			break
		}
		nameLen := int(buf[off+32])
		if nameLen > 0 && off+33+nameLen <= len(buf) {
			name := string(buf[off+33 : off+33+nameLen])
			// "." and ".." are stored as single NUL bytes.
			if name != "\x00" && name != "\x01" {
				out = append(out, strings.TrimSuffix(name, ";1"))
			}
		}
		off += recLen
	}
	return out
}

// addWarnings records the reasons a write may produce a drive that does not
// boot. They are worth saying before the write rather than after the reboot.
func (i *Image) addWarnings() {
	switch {
	case i.WindowsInstaller && !i.HybridUSB:
		i.Warnings = append(i.Warnings,
			"This is a Windows installation image, and Microsoft does not make them hybrid. "+
				"Copied to a drive as-is it produces a disc image on a stick: UEFI firmware that "+
				"reads only FAT will find nothing to boot. The usual answer is a FAT32 volume "+
				"(`oliver format`) with the image's files copied onto it.")
	case strings.HasPrefix(i.Format, "ISO 9660") && !i.HybridUSB:
		i.Warnings = append(i.Warnings,
			"This ISO carries no partition table, so it was built to be burned to a disc. "+
				"Written to a drive it will still hold every file, but the firmware may find no "+
				"partition to boot from.")
	}
	if strings.HasPrefix(i.Format, "ISO 9660") && !i.Bootable() {
		i.Warnings = append(i.Warnings,
			"This ISO declares no boot catalogue, so it is data rather than installation media.")
	}
	if i.Size%isoSectorSize != 0 && strings.HasPrefix(i.Format, "ISO 9660") {
		i.Warnings = append(i.Warnings,
			"The file does not end on a 2048-byte boundary, which usually means a truncated "+
				"download. Check it against the publisher's checksum before trusting it.")
	}
}

func sortedKeys(m map[byte]bool) []byte {
	out := make([]byte, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// HumanBytes renders a byte count the way a drive's label would.
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < 4 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTP"[exp])
}
