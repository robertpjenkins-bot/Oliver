package main

// Where the pieces of an Etros card come from, and fetching them.

import (
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	batoceraBase = "https://updates.batocera.org"
	rpiOSLatest  = "https://downloads.raspberrypi.com/raspios_arm64_latest"
	// A PC (x86-64) build of Wine that needs no 32-bit libraries (WoW64),
	// for Etros's Windows folder: the Pi 5 runs it through Box64.
	wineLatest = "https://github.com/Kron4ek/Wine-Builds/releases/download/10.0/wine-10.0-amd64-wow64.tar.xz"
	userAgent  = "etros-installer"
)

// Sources names each input as a local file or a URL. Empty means the
// default: the latest stable Batocera, and the latest Raspberry Pi OS.
type Sources struct {
	Pi5Boot   string // Batocera boot.tar.xz for the Pi 5 (bcm2712)
	R36SBoot  string // Batocera boot.tar.xz for the R36S
	R36SImage string // Batocera .img.gz for the R36S, for its bootloader
	RPiOS     string // Raspberry Pi OS (64-bit, desktop) .img.xz, for Ollie OS
	Wine      string // Wine for PCs (wine-*-amd64*.tar.xz), for the Windows folder; "none" for none
	Channel   string // Batocera release channel, "stable" by default
}

func (s *Sources) defaults() {
	if s.Channel == "" {
		s.Channel = "stable"
	}
	if s.Pi5Boot == "" {
		s.Pi5Boot = fmt.Sprintf("%s/bcm2712/%s/last/boot.tar.xz", batoceraBase, s.Channel)
	}
	if s.R36SBoot == "" {
		s.R36SBoot = fmt.Sprintf("%s/r36s/%s/last/boot.tar.xz", batoceraBase, s.Channel)
	}
	if s.RPiOS == "" {
		s.RPiOS = rpiOSLatest
	}
	if s.Wine == "" {
		s.Wine = wineLatest
	}
}

func isURL(s string) bool { return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") }

var client = &http.Client{Timeout: 0}

func get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return resp, nil
}

func getText(url string) (string, error) {
	resp, err := get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return string(b), err
}

// Progress reports how a long step is going.
type Progress func(done, total int64)

type countingReader struct {
	r     io.Reader
	n     int64
	total int64
	every time.Time
	p     Progress
}

func (c *countingReader) Read(b []byte) (int, error) {
	n, err := c.r.Read(b)
	c.n += int64(n)
	if c.p != nil && (time.Since(c.every) > 250*time.Millisecond || err == io.EOF) {
		c.every = time.Now()
		c.p(c.n, c.total)
	}
	return n, err
}

// fetch makes src available as a local file, downloading it into cacheDir
// if it is a URL. A finished download is reused on the next run.
func fetch(src, cacheDir, name string, p Progress, log func(string)) (string, error) {
	if !isURL(src) {
		if _, err := os.Stat(src); err != nil {
			return "", err
		}
		return src, nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	resp, err := get(src)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Name the cache file after where a redirect landed, so a new release
	// is a new file.
	final := resp.Request.URL.String()
	sum := md5.Sum([]byte(final))
	dest := filepath.Join(cacheDir, name+"-"+hex.EncodeToString(sum[:4])+filepath.Ext(final))
	if st, err := os.Stat(dest); err == nil && resp.ContentLength > 0 && st.Size() == resp.ContentLength {
		log("using the copy downloaded earlier: " + filepath.Base(dest))
		return dest, nil
	}
	log("downloading " + final)
	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, &countingReader{r: resp.Body, total: resp.ContentLength, p: p})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("downloading %s: %w", final, err)
	}
	// Batocera publishes an MD5 next to each file.
	if strings.Contains(final, "batocera") {
		if want, err := getText(final + ".md5"); err == nil {
			want = strings.Fields(want + " x")[0]
			got, err := md5File(tmp)
			if err != nil {
				return "", err
			}
			if !strings.EqualFold(got, want) {
				os.Remove(tmp)
				return "", fmt.Errorf("%s is damaged (checksum mismatch); try again", filepath.Base(final))
			}
		}
	}
	return dest, os.Rename(tmp, dest)
}

func md5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var imgGzLink = regexp.MustCompile(`href="([^"]+\.img\.gz)"`)

// r36sImageURL finds the R36S image next to its boot.tar.xz.
func r36sImageURL(bootURL string) (string, error) {
	dir := bootURL[:strings.LastIndex(bootURL, "/")+1]
	listing, err := getText(dir)
	if err != nil {
		return "", err
	}
	for _, m := range imgGzLink.FindAllStringSubmatch(listing, -1) {
		name := m[1][strings.LastIndex(m[1], "/")+1:]
		return dir + name, nil
	}
	return "", fmt.Errorf("no R36S image (.img.gz) listed at %s", dir)
}

// The R36S's bootloader lives in the first 16 MiB of its image, ahead of
// the boot partition: idbloader at 32 KiB, U-Boot at 8 MiB, trust at 12 MiB.
const loaderEnd = 16 << 20

// r36sLoader reads the start of the R36S image: just enough to take its
// bootloader, so only that much is downloaded.
func r36sLoader(src string) ([]byte, error) {
	var r io.ReadCloser
	if isURL(src) {
		resp, err := get(src)
		if err != nil {
			return nil, err
		}
		r = resp.Body
	} else {
		f, err := os.Open(src)
		if err != nil {
			return nil, err
		}
		r = f
	}
	defer r.Close()
	gz, err := gzip.NewReader(bufio.NewReaderSize(r, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("R36S image: %w", err)
	}
	head := make([]byte, loaderEnd)
	if _, err := io.ReadFull(gz, head); err != nil {
		return nil, fmt.Errorf("R36S image: %w", err)
	}
	parts := readMBR(head[:512])
	if len(parts) == 0 || parts[0].start*512 < loaderEnd {
		return nil, fmt.Errorf("R36S image: unexpected layout (boot partition not at 16 MiB or later)")
	}
	return head, nil
}

type mbrPart struct {
	kind         byte
	start, count uint64 // sectors
}

func readMBR(b []byte) []mbrPart {
	if len(b) < 512 || b[510] != 0x55 || b[511] != 0xAA {
		return nil
	}
	var out []mbrPart
	for i := 0; i < 4; i++ {
		e := b[446+16*i:]
		if e[4] == 0 {
			continue
		}
		out = append(out, mbrPart{e[4], uint64(binary.LittleEndian.Uint32(e[8:])), uint64(binary.LittleEndian.Uint32(e[12:]))})
	}
	return out
}

// wineName is the name Wine has on the card (etros/wine/<name>), which
// etros-launch-windows looks for as wine-*.tar.xz.
func wineName(src string) string {
	name := path.Base(src)
	if strings.HasPrefix(name, "wine-") && strings.HasSuffix(name, ".tar.xz") {
		return name
	}
	return "wine-etros.tar.xz"
}

// getWine gets Wine (from Downloads or the web) and returns where it is. A
// card without Wine still works; its Windows folder just says so.
func getWine(src, cacheDir string, rep Reporter) (string, bool) {
	if src == "none" {
		return "", false
	}
	p, err := fetch(src, cacheDir, "wine", rep.Progress, rep.Log)
	if err != nil {
		rep.Log("note: couldn't get Wine for the Windows folder (" + err.Error() + "); put wine-10.0-amd64-wow64.tar.xz in Downloads and run Etros Updater to add it")
		logf("wine: %v", err)
		return "", false
	}
	return p, true
}
