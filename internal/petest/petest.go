// Package petest builds minimal but structurally valid PE images for tests.
//
// Constructing images by hand keeps tests hermetic and fast, and lets each
// case isolate exactly one property of the header.
package petest

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"os"
	"sort"
	"unicode/utf16"
)

// Machine identifiers, re-exported so callers need not import debug/pe.
const (
	MachineI386  = pe.IMAGE_FILE_MACHINE_I386
	MachineAMD64 = pe.IMAGE_FILE_MACHINE_AMD64
	MachineARM64 = pe.IMAGE_FILE_MACHINE_ARM64
)

// Subsystem values from the PE specification.
const (
	SubsystemGUI     = 2
	SubsystemConsole = 3
)

// Options describes the image to build.
type Options struct {
	Machine   uint16
	Subsystem uint16
	// PE32Plus selects the 64-bit optional header.
	PE32Plus bool
	// CLRHeader marks the image as a managed .NET assembly.
	CLRHeader bool
	// SubsystemMajor and SubsystemMinor set the targeted Windows version.
	SubsystemMajor uint16
	SubsystemMinor uint16
	// Trailing bytes are appended after the headers, for signature scanning.
	Trailing []byte
	// Imports are DLL names to place in a real import table.
	Imports []string
	// FirstSectionName overrides the name of the first section, which is how
	// executable packers are recognised (e.g. "UPX0").
	FirstSectionName string
	// Manifest is embedded so that manifest parsing can be exercised.
	Manifest string
	// VersionStrings are VS_VERSIONINFO entries (e.g. "ProductName": "Rufus"),
	// encoded as UTF-16 key/value pairs in a .rsrc section.
	VersionStrings map[string]string
}

// Build returns the bytes of a PE image matching opts.
func Build(o Options) []byte {
	if o.SubsystemMajor == 0 {
		o.SubsystemMajor, o.SubsystemMinor = 6, 1
	}
	if o.Subsystem == 0 {
		o.Subsystem = SubsystemGUI
	}
	if o.Machine == 0 {
		o.Machine = MachineAMD64
	}

	const (
		importRVA = 0x2000 // where the synthetic .idata section is mapped
		importRaw = 0x600  // and where its bytes sit in the file
	)
	idata := buildImportTable(o.Imports, importRVA)

	const (
		rsrcRVA = 0x3000
		rsrcRaw = 0x800
	)
	rsrc := buildVersionResource(o.VersionStrings)

	sections := uint16(1)
	if len(idata) > 0 {
		sections++
	}
	if len(rsrc) > 0 {
		sections++
	}

	var buf bytes.Buffer
	// DOS stub: "MZ", then e_lfanew at 0x3C pointing past the stub.
	dos := make([]byte, 0x40)
	copy(dos, "MZ")
	binary.LittleEndian.PutUint32(dos[0x3C:], 0x40)
	buf.Write(dos)
	buf.WriteString("PE\x00\x00")

	var optSize uint16 = 224
	if o.PE32Plus {
		optSize = 240
	}
	_ = binary.Write(&buf, binary.LittleEndian, pe.FileHeader{
		Machine:              o.Machine,
		NumberOfSections:     sections,
		SizeOfOptionalHeader: optSize,
		Characteristics:      0x0002, // IMAGE_FILE_EXECUTABLE_IMAGE
	})

	var dirs [16]pe.DataDirectory
	if o.CLRHeader {
		// Index 14 is IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR.
		dirs[14] = pe.DataDirectory{VirtualAddress: 0x2000, Size: 0x48}
	}
	if len(idata) > 0 {
		// Index 1 is IMAGE_DIRECTORY_ENTRY_IMPORT.
		dirs[1] = pe.DataDirectory{VirtualAddress: importRVA, Size: uint32(len(idata))}
	}
	if o.PE32Plus {
		_ = binary.Write(&buf, binary.LittleEndian, pe.OptionalHeader64{
			Magic:                 0x20b,
			SectionAlignment:      0x1000,
			FileAlignment:         0x200,
			MajorSubsystemVersion: o.SubsystemMajor,
			MinorSubsystemVersion: o.SubsystemMinor,
			SizeOfImage:           0x3000,
			SizeOfHeaders:         0x200,
			Subsystem:             o.Subsystem,
			NumberOfRvaAndSizes:   16,
			DataDirectory:         dirs,
		})
	} else {
		_ = binary.Write(&buf, binary.LittleEndian, pe.OptionalHeader32{
			Magic:                 0x10b,
			SectionAlignment:      0x1000,
			FileAlignment:         0x200,
			MajorSubsystemVersion: o.SubsystemMajor,
			MinorSubsystemVersion: o.SubsystemMinor,
			SizeOfImage:           0x3000,
			SizeOfHeaders:         0x200,
			Subsystem:             o.Subsystem,
			NumberOfRvaAndSizes:   16,
			DataDirectory:         dirs,
		})
	}

	firstName := ".text"
	if o.FirstSectionName != "" {
		firstName = o.FirstSectionName
	}
	_ = binary.Write(&buf, binary.LittleEndian, pe.SectionHeader32{
		Name:             sectionName(firstName),
		VirtualSize:      0x1000,
		VirtualAddress:   0x1000,
		SizeOfRawData:    0x200,
		PointerToRawData: 0x400,
		Characteristics:  0x60000020,
	})
	if len(idata) > 0 {
		_ = binary.Write(&buf, binary.LittleEndian, pe.SectionHeader32{
			Name:             sectionName(".idata"),
			VirtualSize:      0x1000,
			VirtualAddress:   importRVA,
			SizeOfRawData:    0x200,
			PointerToRawData: importRaw,
			Characteristics:  0x40000040,
		})
	}

	if len(rsrc) > 0 {
		_ = binary.Write(&buf, binary.LittleEndian, pe.SectionHeader32{
			Name:             sectionName(".rsrc"),
			VirtualSize:      0x1000,
			VirtualAddress:   rsrcRVA,
			SizeOfRawData:    0x400,
			PointerToRawData: rsrcRaw,
			Characteristics:  0x40000040,
		})
	}

	for buf.Len() < 0x400 {
		buf.WriteByte(0)
	}
	buf.Write(make([]byte, 0x200)) // .text payload

	if len(idata) > 0 {
		for buf.Len() < importRaw {
			buf.WriteByte(0)
		}
		payload := make([]byte, 0x200)
		copy(payload, idata)
		buf.Write(payload)
	}

	if len(rsrc) > 0 {
		for buf.Len() < rsrcRaw {
			buf.WriteByte(0)
		}
		payload := make([]byte, 0x400)
		copy(payload, rsrc)
		buf.Write(payload)
	}

	if o.Manifest != "" {
		buf.WriteString(o.Manifest)
	}
	buf.Write(o.Trailing)
	return buf.Bytes()
}

// buildVersionResource encodes key/value pairs the way a VS_VERSIONINFO string
// table stores them: a UTF-16 key, NUL, padding to a 4-byte boundary, then a
// UTF-16 value.
func buildVersionResource(strs map[string]string) []byte {
	if len(strs) == 0 {
		return nil
	}
	// Sorted so the fixture is byte-for-byte reproducible.
	keys := make([]string, 0, len(strs))
	for k := range strs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []byte
	for _, k := range keys {
		out = append(out, utf16LE(k)...)
		out = append(out, 0, 0) // key terminator
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
		out = append(out, utf16LE(strs[k])...)
		out = append(out, 0, 0) // value terminator
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	return out
}

func utf16LE(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, len(units)*2)
	for _, u := range units {
		out = append(out, byte(u), byte(u>>8))
	}
	return out
}

func sectionName(s string) [8]byte {
	var out [8]byte
	copy(out[:], s)
	return out
}

// buildImportTable lays out an IMAGE_IMPORT_DESCRIPTOR array followed by the
// DLL name strings it points at, as a real linker would.
func buildImportTable(names []string, baseRVA uint32) []byte {
	if len(names) == 0 {
		return nil
	}
	const descriptorSize = 20
	// One descriptor per DLL, plus the all-zero terminator.
	stringsStart := uint32((len(names) + 1) * descriptorSize)

	table := make([]byte, stringsStart)
	var strs []byte
	for i, name := range names {
		nameRVA := baseRVA + stringsStart + uint32(len(strs))
		off := i * descriptorSize
		binary.LittleEndian.PutUint32(table[off:], baseRVA+0x800)    // OriginalFirstThunk
		binary.LittleEndian.PutUint32(table[off+12:], nameRVA)       // Name
		binary.LittleEndian.PutUint32(table[off+16:], baseRVA+0x900) // FirstThunk
		strs = append(strs, []byte(name)...)
		strs = append(strs, 0)
	}
	return append(table, strs...)
}

// WriteFile writes a PE image to path.
func WriteFile(path string, o Options) error {
	return os.WriteFile(path, Build(o), 0o755)
}
