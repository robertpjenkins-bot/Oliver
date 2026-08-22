package pe

import (
	"bytes"
	"debug/pe"
	"strings"
	"unicode/utf16"
)

// VersionInfo is the identity a Windows program declares about itself in its
// VS_VERSIONINFO resource.
//
// This is far more reliable than the file name: it survives renaming, the
// "(1)" a browser appends to a second download, and any prefix a download
// manager adds. It also survives executable packing, because packers leave the
// resource section intact so that Explorer can still show the icon and
// properties.
type VersionInfo struct {
	ProductName      string `json:"product_name,omitempty"`
	OriginalFilename string `json:"original_filename,omitempty"`
	CompanyName      string `json:"company_name,omitempty"`
	FileDescription  string `json:"file_description,omitempty"`
	FileVersion      string `json:"file_version,omitempty"`
	ProductVersion   string `json:"product_version,omitempty"`
}

// Empty reports whether nothing could be read.
func (v VersionInfo) Empty() bool { return v == VersionInfo{} }

// Describe renders the program's self-declared identity for display.
func (v VersionInfo) Describe() string {
	name := v.ProductName
	if name == "" {
		name = v.FileDescription
	}
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(name)
	if ver := v.ProductVersion; ver != "" {
		b.WriteString(" " + ver)
	} else if v.FileVersion != "" {
		b.WriteString(" " + v.FileVersion)
	}
	if v.CompanyName != "" {
		b.WriteString(" (" + v.CompanyName + ")")
	}
	return b.String()
}

// versionKeys are the StringFileInfo entries worth reading.
var versionKeys = []struct {
	key   string
	field func(*VersionInfo) *string
}{
	{"ProductName", func(v *VersionInfo) *string { return &v.ProductName }},
	{"OriginalFilename", func(v *VersionInfo) *string { return &v.OriginalFilename }},
	{"CompanyName", func(v *VersionInfo) *string { return &v.CompanyName }},
	{"FileDescription", func(v *VersionInfo) *string { return &v.FileDescription }},
	{"FileVersion", func(v *VersionInfo) *string { return &v.FileVersion }},
	{"ProductVersion", func(v *VersionInfo) *string { return &v.ProductVersion }},
}

// readVersionInfo extracts the version resource's string table.
//
// A VS_VERSIONINFO resource is a tree of length-prefixed blocks, but every
// value we want has the same shape: a UTF-16 key, padding to a 4-byte
// boundary, then a UTF-16 value. Locating each key directly is far shorter
// than walking the tree, and tolerates the malformed resources that real
// software ships.
func readVersionInfo(pf *pe.File) VersionInfo {
	var out VersionInfo
	rsrc := pf.Section(".rsrc")
	if rsrc == nil {
		return out
	}
	data, err := rsrc.Data()
	if err != nil {
		return out
	}

	for _, k := range versionKeys {
		if v := findVersionValue(data, k.key); v != "" {
			*k.field(&out) = v
		}
	}
	return out
}

// findVersionValue locates a UTF-16 key and returns the string that follows it.
func findVersionValue(data []byte, key string) string {
	needle := append(utf16Bytes(key), 0, 0) // keys are NUL-terminated
	from := 0
	for {
		idx := bytes.Index(data[from:], needle)
		if idx < 0 {
			return ""
		}
		pos := from + idx + len(needle)
		if v := readUTF16Value(data, pos); v != "" {
			return v
		}
		// A key with an empty value: keep looking for another occurrence.
		from += idx + len(needle)
		if from >= len(data) {
			return ""
		}
	}
}

// readUTF16Value skips the padding after a key and reads the NUL-terminated
// UTF-16 string that follows.
func readUTF16Value(data []byte, pos int) string {
	// Values are aligned to a 4-byte boundary, so at most three padding bytes
	// separate the key from its value.
	for pad := 0; pad < 4 && pos < len(data) && data[pos] == 0; pad++ {
		pos++
	}
	if pos >= len(data) {
		return ""
	}
	// Alignment works in 16-bit units, so step back onto an even offset.
	if pos%2 == 1 {
		pos--
	}

	const maxChars = 256
	var units []uint16
	for i := pos; i+1 < len(data) && len(units) < maxChars; i += 2 {
		u := uint16(data[i]) | uint16(data[i+1])<<8
		if u == 0 {
			break
		}
		units = append(units, u)
	}
	return strings.TrimSpace(string(utf16.Decode(units)))
}

// utf16Bytes encodes a string as little-endian UTF-16, as stored in resources.
func utf16Bytes(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, len(units)*2)
	for _, u := range units {
		out = append(out, byte(u), byte(u>>8))
	}
	return out
}
