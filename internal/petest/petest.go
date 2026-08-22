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
		NumberOfSections:     1,
		SizeOfOptionalHeader: optSize,
		Characteristics:      0x0002, // IMAGE_FILE_EXECUTABLE_IMAGE
	})

	var dirs [16]pe.DataDirectory
	if o.CLRHeader {
		// Index 14 is IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR.
		dirs[14] = pe.DataDirectory{VirtualAddress: 0x2000, Size: 0x48}
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

	_ = binary.Write(&buf, binary.LittleEndian, pe.SectionHeader32{
		Name:             [8]byte{'.', 't', 'e', 'x', 't'},
		VirtualSize:      0x1000,
		VirtualAddress:   0x1000,
		SizeOfRawData:    0x200,
		PointerToRawData: 0x400,
		Characteristics:  0x60000020,
	})

	for buf.Len() < 0x400 {
		buf.WriteByte(0)
	}
	buf.Write(make([]byte, 0x200))
	buf.Write(o.Trailing)
	return buf.Bytes()
}

// WriteFile writes a PE image to path.
func WriteFile(path string, o Options) error {
	return os.WriteFile(path, Build(o), 0o755)
}
