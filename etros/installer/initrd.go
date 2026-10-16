package main

// Patching Batocera's initramfs: unpack the lz4 cpio archive, add the Etros
// hook to /init, and pack it again, all in memory.

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"strconv"
	"strings"

	"github.com/pierrec/lz4/v4"
)

//go:embed assets/etros-hook.sh
var etrosHook []byte

// initEdits are anchored on lines of Batocera's /init. If an anchor is
// missing or repeated, Batocera has changed its init and we stop rather
// than guess.
var initEdits = []struct{ before, anchor, replace string }{
	{"    . /etros-hook.sh && etros_start || return 1\n\n", "    # update the squashfs\n", ""},
	{"    etros_mount_layer || return 1\n\n", "    # mount the future root in read write\n", ""},
	{"", "lowerdir=/overlay_root/base$EXTRALAYERS,", "lowerdir=${ETROS_LAYER}/overlay_root/base$EXTRALAYERS,"},
}

func patchInit(init string) (string, error) {
	if strings.Contains(init, "etros-hook.sh") {
		return "", fmt.Errorf("init is already patched for Etros")
	}
	for _, e := range initEdits {
		if n := strings.Count(init, e.anchor); n != 1 {
			return "", fmt.Errorf("Batocera's boot script has changed: expected one %q, found %d", strings.TrimSpace(e.anchor), n)
		}
		if e.replace != "" {
			init = strings.Replace(init, e.anchor, e.replace, 1)
		} else {
			init = strings.Replace(init, e.anchor, e.before+e.anchor, 1)
		}
	}
	return init, nil
}

type cpioEntry struct {
	name string
	hdr  [13]uint32 // ino mode uid gid nlink mtime filesize devmajor devminor rdevmajor rdevminor namesize check
	data []byte
}

func readCPIO(b []byte) ([]cpioEntry, error) {
	var out []cpioEntry
	p := 0
	for {
		if p+110 > len(b) {
			return nil, fmt.Errorf("cpio: truncated header")
		}
		if string(b[p:p+6]) != "070701" && string(b[p:p+6]) != "070702" {
			return nil, fmt.Errorf("cpio: not a newc archive")
		}
		var e cpioEntry
		for i := 0; i < 13; i++ {
			v, err := strconv.ParseUint(string(b[p+6+i*8:p+14+i*8]), 16, 32)
			if err != nil {
				return nil, fmt.Errorf("cpio: bad header")
			}
			e.hdr[i] = uint32(v)
		}
		nameSize := int(e.hdr[11])
		p += 110
		e.name = strings.TrimRight(string(b[p:p+nameSize]), "\x00")
		p = align4(p + nameSize)
		size := int(e.hdr[6])
		if p+size > len(b) {
			return nil, fmt.Errorf("cpio: truncated %s", e.name)
		}
		e.data = b[p : p+size]
		p = align4(p + size)
		if e.name == "TRAILER!!!" {
			return out, nil
		}
		out = append(out, e)
	}
}

func align4(n int) int { return (n + 3) &^ 3 }

func writeCPIO(entries []cpioEntry) []byte {
	var buf bytes.Buffer
	write := func(e cpioEntry) {
		e.hdr[6] = uint32(len(e.data))
		e.hdr[11] = uint32(len(e.name) + 1)
		fmt.Fprintf(&buf, "070701")
		for _, v := range e.hdr {
			fmt.Fprintf(&buf, "%08X", v)
		}
		buf.WriteString(e.name)
		buf.WriteByte(0)
		for buf.Len()%4 != 0 {
			buf.WriteByte(0)
		}
		buf.Write(e.data)
		for buf.Len()%4 != 0 {
			buf.WriteByte(0)
		}
	}
	for _, e := range entries {
		write(e)
	}
	var trailer cpioEntry
	trailer.name = "TRAILER!!!"
	trailer.hdr[4] = 1
	write(trailer)
	return buf.Bytes()
}

const ubootMagic = 0x27051956

// PatchInitrd returns a copy of a Batocera initramfs (bare lz4, or wrapped
// in a U-Boot header as the R36S uses) with the Etros hook added.
func PatchInitrd(in []byte) ([]byte, error) {
	uboot := len(in) > 64 && binary.BigEndian.Uint32(in) == ubootMagic
	payload := in
	if uboot {
		payload = in[64:]
	}
	raw, err := io.ReadAll(lz4.NewReader(bytes.NewReader(payload)))
	if err != nil {
		return nil, fmt.Errorf("initramfs: %w", err)
	}
	entries, err := readCPIO(raw)
	if err != nil {
		return nil, err
	}
	found := false
	var mode uint32 = 0o100755
	for i, e := range entries {
		if e.name == "init" || e.name == "./init" {
			patched, err := patchInit(string(e.data))
			if err != nil {
				return nil, err
			}
			entries[i].data = []byte(patched)
			mode = e.hdr[1]
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("initramfs has no /init")
	}
	hook := cpioEntry{name: "etros-hook.sh", data: etrosHook}
	hook.hdr[0] = 0xE7505 // inode, unused by the kernel beyond uniqueness
	hook.hdr[1] = mode
	hook.hdr[4] = 1
	entries = append(entries, hook)

	var out bytes.Buffer
	w := lz4.NewWriter(&out)
	if err := w.Apply(lz4.LegacyOption(true), lz4.CompressionLevelOption(lz4.Level9)); err != nil {
		return nil, err
	}
	if _, err := w.Write(writeCPIO(entries)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	if !uboot {
		return out.Bytes(), nil
	}
	return ubootWrap(in[:64], out.Bytes()), nil
}

// ubootWrap rebuilds a legacy U-Boot image header around new data, keeping
// the original's type, architecture and name.
func ubootWrap(orig []byte, data []byte) []byte {
	h := make([]byte, 64)
	copy(h, orig)
	binary.BigEndian.PutUint32(h[4:], 0)                         // header CRC, computed below
	binary.BigEndian.PutUint32(h[12:], uint32(len(data)))        // data size
	binary.BigEndian.PutUint32(h[24:], crc32.ChecksumIEEE(data)) // data CRC
	binary.BigEndian.PutUint32(h[4:], crc32.ChecksumIEEE(h))
	return append(h, data...)
}
