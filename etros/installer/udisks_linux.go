//go:build linux

package main

// Putting an Etros image on a drive with no password, through UDisks.
//
// UDisks only lets you open a whole drive for writing with the
// administrator password, even a USB stick on your own desk. What the
// person at the computer may do on a removable drive without one is make
// partitions, format them, mount them and copy files on. So that's how the
// card is put together here:
//
//   - a fresh partition table, the boot partition (FAT32, BATOCERA) at the
//     place the image has it, and the storage partition (ext4, SHARE) in all
//     the rest of the card;
//   - the boot partition's files are copied out of the image;
//   - Ollie OS, which the image has as a partition of its own, becomes a
//     file on the storage partition (etros/ollie-rootfs.img), which Etros
//     mounts when it starts Ollie OS.
//
// The R36S bootloader lives outside any partition, so an R36S card can't be
// made this way; that still needs the whole-drive write.

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const ollieFileOnShare = "etros/ollie-rootfs.img"

type udisksDrive struct {
	conn *dbus.Conn
	dev  string // /dev/sdb
}

// call never lets UDisks ask for a password: if something would need one,
// it fails and says so instead.
func (u *udisksDrive) call(path dbus.ObjectPath, method string, args ...any) *dbus.Call {
	return u.conn.Object(udisks, path).Call(udisks+"."+method, 0, args...)
}

func noOpts() map[string]dbus.Variant { return map[string]dbus.Variant{} }

func explainUDisks(what string, err error) error {
	if strings.Contains(err.Error(), "NotAuthorized") || strings.Contains(err.Error(), "not authorized") {
		return fmt.Errorf("%s: this computer wants the administrator password for that, which Etros won't ask for. "+
			"Choose 2 to finish, and put the image on the card with Raspberry Pi Imager instead", what)
	}
	return fmt.Errorf("%s: %w", what, err)
}

// filesInstallSupported tells whether a card can be made without a password
// here: on Linux, as yourself, with UDisks running.
func filesInstallSupported() bool {
	if os.Getenv("ETROS_FORCE_FILES") == "1" { // for tests, which run as root
		return true
	}
	if os.Geteuid() == 0 || os.Getenv("ETROS_TEST_DISK") != "" {
		return false
	}
	conn, err := dbus.SystemBus()
	if err != nil {
		return false
	}
	var v dbus.Variant
	err = conn.Object(udisks, "/org/freedesktop/UDisks2/Manager").Call(
		"org.freedesktop.DBus.Properties.Get", 0, udisks+".Manager", "Version").Store(&v)
	return err == nil
}

// filesInstall puts the Etros image at imagePath on the drive id.
func filesInstall(imagePath, id string, rep Reporter) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("can't reach UDisks: %w", err)
	}
	u := &udisksDrive{conn: conn, dev: id}

	// The image's layout: where its boot partition is, and its Ollie OS.
	r, _, _, closer, err := openEtrosImage(imagePath)
	if err != nil {
		return err
	}
	mbr := make([]byte, 512)
	_, err = io.ReadFull(r, mbr)
	closer()
	if err != nil {
		return err
	}
	parts := readMBR(mbr)
	if len(parts) < 2 || parts[0].kind != 0x0C {
		return fmt.Errorf("%s isn't an Etros image", baseName(imagePath))
	}
	fatStart, fatSize := int64(parts[0].start)*512, int64(parts[0].count)*512
	var ollieStart, ollieSize int64
	if len(parts) >= 3 {
		ollieStart, ollieSize = int64(parts[2].start)*512, int64(parts[2].count)*512
	}
	img := &imageSource{path: imagePath}
	if err := img.scan(); err != nil {
		return err
	}
	for _, e := range img.files {
		if strings.EqualFold(e.path, "uInitrd") {
			return fmt.Errorf("this image is for the R36S as well, whose start-up code can't be copied on without the administrator password. Make a Raspberry Pi 5 only image (answer n to the R36S question), or use Raspberry Pi Imager")
		}
	}

	rep.Stage("Getting the drive ready")
	devs, _ := lsblk()
	for _, d := range devs {
		if d.Path != id {
			continue
		}
		for _, c := range d.Children {
			if len(mounts(c)) > 0 {
				if err := u.call(udisksPath(c.Path), "Filesystem.Unmount", noOpts()).Err; err != nil {
					return explainUDisks("could not unmount "+c.Path, err)
				}
			}
		}
	}
	disk := udisksPath(id)
	if err := u.call(disk, "Block.Format", "dos", noOpts()).Err; err != nil {
		return explainUDisks("could not clear the drive", err)
	}
	rep.Log("made a new partition table")

	var bootPart dbus.ObjectPath
	err = u.call(disk, "PartitionTable.CreatePartitionAndFormat",
		uint64(fatStart), uint64(fatSize), "0x0c", "", noOpts(),
		"vfat", map[string]dbus.Variant{"label": dbus.MakeVariant("BATOCERA")}).Store(&bootPart)
	if err != nil {
		return explainUDisks("could not make the boot partition", err)
	}
	u.call(bootPart, "Partition.SetFlags", uint64(0x80), noOpts()) // bootable; not essential
	rep.Log("made the boot partition (BATOCERA)")

	var sharePart dbus.ObjectPath
	err = u.call(disk, "PartitionTable.CreatePartitionAndFormat",
		uint64(fatStart+fatSize), uint64(0), "0x83", "", noOpts(),
		"ext4", map[string]dbus.Variant{
			"label":          dbus.MakeVariant("SHARE"),
			"take-ownership": dbus.MakeVariant(true),
		}).Store(&sharePart)
	if err != nil {
		return explainUDisks("could not make the storage partition", err)
	}
	rep.Log("made the storage partition (SHARE)")

	mount := func(p dbus.ObjectPath) (string, error) {
		var dir string
		if err := u.call(p, "Filesystem.Mount", noOpts()).Store(&dir); err != nil {
			// Some desktops mount new partitions by themselves first.
			var mp [][]byte
			if v, gerr := u.conn.Object(udisks, p).GetProperty(udisks + ".Filesystem.MountPoints"); gerr == nil {
				if v.Store(&mp) == nil && len(mp) > 0 {
					return string(bytes.TrimRight(mp[0], "\x00")), nil
				}
			}
			return "", explainUDisks("could not open a partition", err)
		}
		return dir, nil
	}
	unmount := func(p dbus.ObjectPath) {
		for i := 0; i < 10; i++ {
			if u.call(p, "Filesystem.Unmount", noOpts()).Err == nil {
				return
			}
			time.Sleep(time.Second)
		}
	}

	bootDir, err := mount(bootPart)
	if err != nil {
		return err
	}
	rep.Stage("Copying Batocera and Etros onto the boot partition")
	err = copyImageFiles(img, bootDir, rep)
	syncDir(bootDir)
	unmount(bootPart)
	if err != nil {
		return fmt.Errorf("copying onto the boot partition: %w", err)
	}

	if ollieSize > 0 {
		shareDir, err := mount(sharePart)
		if err != nil {
			return err
		}
		rep.Stage("Copying Ollie OS onto the storage partition")
		err = copyOllie(imagePath, ollieStart, ollieSize, filepath.Join(shareDir, ollieFileOnShare), rep)
		syncDir(shareDir)
		unmount(sharePart)
		if err != nil {
			return fmt.Errorf("copying Ollie OS: %w", err)
		}
	}
	return nil
}

// copyImageFiles copies every file and folder of the image's boot
// partition into dir.
func copyImageFiles(img *imageSource, dir string, rep Reporter) error {
	var total, done int64
	for _, e := range img.files {
		if e.isDir {
			if err := os.MkdirAll(filepath.Join(dir, e.path), 0o755); err != nil {
				return err
			}
		} else {
			total += e.size
		}
	}
	files, err := img.Files()
	if err != nil {
		return err
	}
	big := map[string]bool{}
	write := func(path string, r io.Reader) error {
		dest := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		n, err := io.Copy(f, &countingReader{r: r, p: func(d, _ int64) { rep.Progress(done+d, total) }})
		done += n
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return err
	}
	for _, f := range files {
		if f.data == nil {
			big[f.path] = true
			continue
		}
		if err := write(f.path, bytes.NewReader(f.data)); err != nil {
			return err
		}
	}
	if len(big) > 0 {
		if err := img.Stream(big, write); err != nil {
			return err
		}
	}
	rep.Progress(total, total)
	return nil
}

// copyOllie writes the image's Ollie OS partition into a file, skipping
// the empty stretches so the file takes only the room it needs.
func copyOllie(imagePath string, start, size int64, dest string, rep Reporter) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	r, _, _, closer, err := openEtrosImage(imagePath)
	if err != nil {
		return err
	}
	defer closer()
	// The image is compressed, so getting to Ollie OS means unpacking all
	// that comes before it. Count that in the progress too: it takes a while.
	rep.Log("finding Ollie OS in the image, then copying it (takes a while; don't stop it)")
	skip := &countingReader{r: io.LimitReader(r, start), p: func(d, _ int64) { rep.Progress(d, start+size) }}
	if _, err := io.Copy(io.Discard, skip); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	buf := make([]byte, mib)
	zero := make([]byte, mib)
	var off int64
	for off < size {
		n := min(int64(len(buf)), size-off)
		if _, err := io.ReadFull(r, buf[:n]); err != nil {
			f.Close()
			return err
		}
		if !bytes.Equal(buf[:n], zero[:n]) {
			if _, err := f.WriteAt(buf[:n], off); err != nil {
				f.Close()
				return err
			}
		}
		off += n
		rep.Progress(start+off, start+size)
	}
	if err := f.Truncate(size); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func syncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		d.Sync()
		d.Close()
	}
}
