package main

// Etros on an Intel Mac: what's the same whichever computer puts it there
// (mac_linux.go from a Steam Machine, mac_darwin.go on the Mac itself).

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

const (
	macBootSize  = 8 << 30  // BATOCERA: Batocera for PCs is about 3.5 GB
	macMinShare  = 8 << 30  // SHARE without Ollie OS
	macOllieRoom = 24 << 30 // what Ollie OS takes of SHARE, at most

	archBootstrapURL = "https://geo.mirror.pkgbuild.com/iso/latest/archlinux-bootstrap-x86_64.tar.zst"
)

// partNum is the number at the end of a partition's name (sdb3, disk0s3: 3).
func partNum(id string) int {
	i := len(id)
	for i > 0 && id[i-1] >= '0' && id[i-1] <= '9' {
		i--
	}
	n, _ := strconv.Atoi(id[i:])
	return n
}

func pcBatoceraURL() string { return batoceraBase + "/x86_64/stable/last/boot.tar.xz" }

// getPCBatocera finds Batocera for PCs (its boot.tar.xz) in Downloads, or
// downloads it.
func (t *tui) getPCBatocera() (string, error) {
	for _, pat := range []string{"*x86_64*boot*.tar.xz", "*x86_64*.boot.tar.xz"} {
		if m, _ := filepath.Glob(filepath.Join(downloadsDir(), pat)); len(m) > 0 {
			note("Using " + filepath.Base(m[0]) + " from Downloads.")
			return m[0], nil
		}
	}
	rep := newTermReporter()
	defer rep.end()
	rep.Stage("Downloading Batocera for PCs (about 3 GB)")
	return fetch(pcBatoceraURL(), defaultCache(), "batocera-x86_64-boot", rep.Progress, rep.Log)
}

// Where Batocera for PCs and Arch Linux's bootstrap are downloaded from:
// the first that works.
func pcBatoceraURLs() []string {
	return []string{pcBatoceraURL(), "https://mirrors.o2switch.fr/batocera/x86_64/stable/last/boot.tar.xz"}
}

var archBootstrapURLs = []string{
	archBootstrapURL,
	"https://mirrors.kernel.org/archlinux/iso/latest/archlinux-bootstrap-x86_64.tar.zst",
	"https://mirrors.mit.edu/archlinux/iso/latest/archlinux-bootstrap-x86_64.tar.zst",
}

// fetchFirst downloads from the first of urls that works, and says in
// words why none did.
func fetchFirst(what, name string, urls []string, rep Reporter) (string, error) {
	var why []string
	for _, u := range urls {
		got, err := fetch(u, defaultCache(), name, rep.Progress, rep.Log)
		if err == nil {
			return got, nil
		}
		logf("downloading %s: %v", u, err)
		rep.Log("couldn't download from " + hostOf(u) + ": " + err.Error())
		why = append(why, hostOf(u)+": "+err.Error())
	}
	return "", fmt.Errorf("couldn't download %s (%s). Check this Mac is on the internet and try again", what, strings.Join(why, "; "))
}

func hostOf(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	h, _, _ := strings.Cut(u, "/")
	return h
}

// roomForDownloads says why there isn't room to download need bytes, or "".
func roomForDownloads(need int64) string {
	dir := defaultCache()
	os.MkdirAll(dir, 0o755)
	if free := freeSpace(dir); free >= 0 && free < need && os.Getenv("ETROS_TEST_NO_ROOM_CHECK") == "" { // for tests
		return fmt.Sprintf("this Mac needs %s free for the downloads, and has %s: empty the Trash or delete some big files, then try again", gbs(need), gbs(free))
	}
	return ""
}

// fetchPCBatocera finds Batocera for PCs in Downloads or downloads it,
// saying so through rep (for the Safari installer).
func fetchPCBatocera(rep Reporter) (string, error) {
	for _, pat := range []string{"*x86_64*boot*.tar.xz", "*x86_64*.boot.tar.xz"} {
		if m, _ := filepath.Glob(filepath.Join(downloadsDir(), pat)); len(m) > 0 {
			rep.Log("using " + filepath.Base(m[0]) + " from Downloads")
			return m[0], nil
		}
	}
	if why := roomForDownloads(5 << 30); why != "" {
		return "", errors.New(why)
	}
	rep.Stage("Downloading Batocera for PCs (about 3 GB)")
	return fetchFirst("Batocera for PCs", "batocera-x86_64-boot", pcBatoceraURLs(), rep)
}

// fetchArchBootstrap finds Arch Linux's bootstrap in Downloads or
// downloads it.
func fetchArchBootstrap(rep Reporter) (string, error) {
	if m, _ := filepath.Glob(filepath.Join(downloadsDir(), "archlinux-bootstrap*x86_64*.tar.zst")); len(m) > 0 {
		rep.Log("using " + filepath.Base(m[0]) + " from Downloads")
		return m[0], nil
	}
	if why := roomForDownloads(1 << 30); why != "" {
		return "", errors.New(why)
	}
	rep.Stage("Downloading Arch Linux for Ollie OS (about 150 MB)")
	return fetchFirst("Arch Linux", "archlinux-bootstrap", archBootstrapURLs, rep)
}

// getArchBootstrap finds Arch Linux's bootstrap tarball in Downloads, or
// asks for it, or downloads it.
func (t *tui) getArchBootstrap() (string, error) {
	if m, _ := filepath.Glob(filepath.Join(downloadsDir(), "archlinux-bootstrap*x86_64*.tar.zst")); len(m) > 0 {
		note("Using " + filepath.Base(m[0]) + " from Downloads.")
		return m[0], nil
	}
	for {
		p := t.askForFile("ARCH LINUX FOR OLLIE OS",
			"Drag archlinux-bootstrap-x86_64.tar.zst (from archlinux.org) onto this window, then press Enter.",
			"Or just press Enter to let Etros download it (about 150 MB).")
		if p == "" {
			rep := newTermReporter()
			rep.Stage("Getting Arch Linux")
			got, err := fetch(archBootstrapURL, defaultCache(), "archlinux-bootstrap", rep.Progress, rep.Log)
			rep.end()
			return got, err
		}
		n := strings.ToLower(filepath.Base(p))
		if strings.Contains(n, "bootstrap") && strings.Contains(n, "x86_64") {
			return p, nil
		}
		note(cRed + filepath.Base(p) + " isn't Arch Linux's bootstrap (archlinux-bootstrap-x86_64.tar.zst)." + cReset)
	}
}

// pcInitrd is Batocera for PCs' start-up file with Etros's hook added.
func pcInitrd(bootTarball string) ([]byte, error) {
	var initrd []byte
	err := eachTarFile(bootTarball, func(h *tarHeader, r io.Reader) error {
		if h.Name == "boot/initrd.gz" {
			b, err := io.ReadAll(r)
			initrd = b
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if initrd == nil {
		return nil, fmt.Errorf("%s isn't Batocera for PCs (no boot/initrd.gz)", filepath.Base(bootTarball))
	}
	out, err := PatchInitrd(initrd)
	if err != nil {
		return nil, fmt.Errorf("adding Etros to Batocera's start-up: %w", err)
	}
	return out, nil
}

// copyPCBatocera copies Batocera for PCs' boot files to boot, with Etros.
func copyPCBatocera(bootTarball string, initrd []byte, boot string, rep Reporter) error {
	st, err := os.Stat(bootTarball)
	if err != nil {
		return err
	}
	var done int64
	err = eachTarFile(bootTarball, func(h *tarHeader, r io.Reader) error {
		dest := filepath.Join(boot, filepath.FromSlash(h.Name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if h.Name == "boot/initrd.gz" {
			return os.WriteFile(dest, initrd, 0o644)
		}
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		// Progress by the compressed size, roughly: the tarball is xz.
		n, err := io.Copy(f, &countingReader{r: r, p: func(d, _ int64) {
			rep.Progress(min(done+d/3, st.Size()), st.Size())
		}})
		done += n / 3
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return err
	})
	if err != nil {
		return err
	}
	rep.Progress(st.Size(), st.Size())
	if err := os.MkdirAll(filepath.Join(boot, "etros"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(boot, "etros", "etros.squashfs"), etrosLayer, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(boot, "etros", "intel-mac.txt"), []byte("Etros on an Intel Mac, in place of Boot Camp\n"), 0o644); err != nil {
		return err
	}
	// SHARE is made to fill the rest already.
	return noAutoresize(boot)
}

// gzBootstrap rewrites Arch Linux's bootstrap tarball (zstd, all in
// root.x86_64/) as a plain .tar.gz of the file system itself, which Etros
// on the Mac can unpack with the tools Batocera has.
func gzBootstrap(src, dest string, rep Reporter) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, _ := in.Stat()
	zr, err := zstd.NewReader(&countingReader{r: in, p: func(d, _ int64) { rep.Progress(d, st.Size()) }})
	if err != nil {
		return err
	}
	defer zr.Close()
	out, err := os.Create(dest + ".new")
	if err != nil {
		return err
	}
	gw, _ := gzip.NewWriterLevel(out, gzip.BestSpeed)
	tw := tar.NewWriter(gw)
	tr := tar.NewReader(zr)
	err = func() error {
		for {
			h, err := tr.Next()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(strings.TrimPrefix(h.Name, "./"), "root.x86_64")
			name = strings.TrimPrefix(name, "/")
			if name == "" {
				continue
			}
			h.Name = name
			if h.Typeflag == tar.TypeLink {
				h.Linkname = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(h.Linkname, "./"), "root.x86_64"), "/")
			}
			if err := tw.WriteHeader(h); err != nil {
				return err
			}
			if _, err := io.Copy(tw, tr); err != nil {
				return err
			}
		}
	}()
	for _, c := range []io.Closer{tw, gw, out} {
		if cerr := c.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		os.Remove(dest + ".new")
		return fmt.Errorf("%s: %w", filepath.Base(src), err)
	}
	return os.Rename(dest+".new", dest)
}

// The drives and partitions diskutil sees (mac_darwin.go), and where Etros
// can go on them, worked out here so it can be checked anywhere.

type dPart struct {
	id, content, name string
	size              int64
}

func (p dPart) num() int { return partNum(p.id) }

type dDisk struct {
	id, name string
	size     int64
	internal bool
	parts    []dPart
}

// A macTarget is a place Etros can go.
type macTarget struct {
	kind   string // "part": a partition (and extra), erased; "room": free room in macOS's APFS; "whole": a whole drive, erased
	disk   dDisk
	part   dPart
	extra  []dPart // erased along with part (an earlier Etros's SHARE)
	after  string  // part: the partition before it, which the new ones follow
	what   string  // what it is, for the list
	action string  // what happens, for the list
	why    string  // why it can't be used ("" if it can)
}

// readDisks reads diskutil list -plist physical; info gives diskutil
// info -plist for a disk.
func readDisks(v any, info func(id string) any) []dDisk {
	var disks []dDisk
	all, _ := plistGet(v, "AllDisksAndPartitions").([]any)
	for i := range all {
		d := dDisk{id: plistString(all[i], "DeviceIdentifier"), size: plistInt(all[i], "Size")}
		if in := info(d.id); in != nil {
			d.internal, _ = plistGet(in, "Internal").(bool)
			d.name = strings.TrimSpace(plistString(in, "MediaName"))
		}
		parts, _ := plistGet(all[i], "Partitions").([]any)
		for j := range parts {
			d.parts = append(d.parts, dPart{
				id:      plistString(parts[j], "DeviceIdentifier"),
				content: plistString(parts[j], "Content"),
				name:    plistString(parts[j], "VolumeName"),
				size:    plistInt(parts[j], "Size"),
			})
		}
		disks = append(disks, d)
	}
	return disks
}

func partWhat(p dPart) string {
	kind := map[string]string{
		"EFI": "EFI", "Apple_APFS": "macOS (APFS)", "Apple_HFS": "Mac (HFS+)", "Apple_CoreStorage": "macOS",
		"Apple_Boot": "macOS Recovery", "Apple_APFS_Recovery": "macOS Recovery", "Apple_APFS_ISC": "Mac start-up",
		"Microsoft Basic Data": "FAT/ExFAT/NTFS", "Windows_NTFS": "NTFS", "Windows Recovery": "Windows Recovery",
		"Microsoft Reserved": "Windows (reserved)", "Linux Filesystem": "Linux",
	}[p.content]
	if kind == "" {
		kind = p.content
	}
	if p.name != "" {
		kind += " \"" + p.name + "\""
	}
	return kind
}

// macTargets lists every drive and partition and what Etros can do with
// it. bootStore is the partition the running macOS is on (its APFS
// container's), which is never erased: Etros can only take free room
// from it.
func macTargets(disks []dDisk, bootStore string) []macTarget {
	var out []macTarget
	min := int64(macBootSize + macMinShare)
	for _, d := range disks {
		if !d.internal {
			t := macTarget{kind: "whole", disk: d, what: "the whole drive", action: "erase it all: Etros on this drive"}
			if d.size < min {
				t.why = "too small (Etros needs " + gbs(min) + ")"
			}
			out = append(out, t)
		}
		for i, p := range d.parts {
			t := macTarget{kind: "part", disk: d, part: p, what: partWhat(p), action: "erase it: Etros goes here"}
			if i > 0 {
				t.after = d.parts[i-1].id
			}
			switch {
			case p.content == "EFI" || p.content == "Apple_Boot" || p.content == "Apple_APFS_Recovery" ||
				p.content == "Apple_APFS_ISC" || p.content == "Microsoft Reserved" || p.content == "Apple_KernelCoreDump":
				t.why = "the computer's start-up files"
			case p.id == bootStore || (bootStore == "" && d.internal && (p.content == "Apple_APFS" || p.content == "Apple_HFS" || p.content == "Apple_CoreStorage")):
				if p.content != "Apple_APFS" {
					t.why = "the macOS you're using"
				} else {
					t.kind, t.action = "room", "use its free room (macOS and its files stay)"
				}
			case p.name == "SHARE" && i > 0 && d.parts[i-1].name == "BATOCERA":
				t.why = "Etros's storage (choose its BATOCERA)"
			case p.content == "Apple_APFS" || p.content == "Apple_HFS":
				t.action = "erase it (another Mac system or drive!): Etros goes here"
			}
			if p.name == "BATOCERA" {
				t.action = "Etros already: put it on again (its games and saves are erased)"
				if i+1 < len(d.parts) && (d.parts[i+1].name == "SHARE" || d.parts[i+1].content == "Linux Filesystem") {
					t.extra = []dPart{d.parts[i+1]}
				}
			}
			size := p.size
			for _, e := range t.extra {
				size += e.size
			}
			if t.why == "" && t.kind == "part" && t.after == "" {
				t.why = "first on its drive (choose the whole drive)"
			}
			if t.why == "" && t.kind == "part" && size < min {
				t.why = "too small (Etros needs " + gbs(min) + ")"
			}
			out = append(out, t)
		}
	}
	return out
}

// A macCheck is something looked at before Etros goes on a Mac.
type macCheck struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Why      string `json:"why"`
	Blocking bool   `json:"blocking"` // Etros can't go on until it's sorted
}

// reachable tells whether the first byte of a download can be fetched,
// and if not, why.
func reachable(urls []string) (bool, string) {
	c := &http.Client{Timeout: 15 * time.Second}
	var why []string
	for _, u := range urls {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Range", "bytes=0-0")
		resp, err := c.Do(req)
		if err != nil {
			why = append(why, hostOf(u)+": "+err.Error())
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 400 {
			return true, ""
		}
		why = append(why, hostOf(u)+": "+resp.Status)
	}
	return false, strings.Join(why, "; ")
}

// downloadChecks are the checks for what Etros downloads: Batocera for
// PCs (needed) and Arch Linux (for Ollie OS), unless they're in Downloads.
func downloadChecks() []macCheck {
	inDownloads := func(pats ...string) bool {
		for _, p := range pats {
			if m, _ := filepath.Glob(filepath.Join(downloadsDir(), p)); len(m) > 0 {
				return true
			}
		}
		return false
	}
	var out []macCheck
	bato := macCheck{Name: "Batocera for PCs can be downloaded (about 3 GB)", Blocking: true}
	if inDownloads("*x86_64*boot*.tar.xz") {
		bato.Name, bato.OK = "Batocera for PCs is in Downloads", true
	} else if bato.OK, bato.Why = reachable(pcBatoceraURLs()); !bato.OK {
		bato.Why = "this Mac can't reach batocera.org (" + bato.Why + "). Is it on the internet?"
	}
	arch := macCheck{Name: "Arch Linux can be downloaded, for Ollie OS (about 150 MB)"}
	if inDownloads("archlinux-bootstrap*x86_64*.tar.zst") {
		arch.Name, arch.OK = "Arch Linux is in Downloads, for Ollie OS", true
	} else if arch.OK, arch.Why = reachable(archBootstrapURLs); !arch.OK {
		arch.Why = "can't reach Arch Linux's download sites (" + arch.Why + "); Etros works without Ollie OS"
	}
	room := macCheck{Name: "Room on this Mac for the downloads", Blocking: true}
	need := int64(5 << 30)
	if bato.OK && strings.HasSuffix(bato.Name, "Downloads") {
		need = 1 << 30
	}
	if room.Why = roomForDownloads(need); room.Why == "" {
		room.OK = true
	}
	return append(out, bato, arch, room)
}
