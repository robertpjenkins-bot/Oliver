package builder

import (
	"bufio"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// The bootstrap method downloads Arch Linux's official bootstrap image (a
// minimal Arch system, about 150 MB) and builds the ISO inside it. That is
// all it takes on any distribution: no Arch install, no Docker or Podman,
// just the password once.

//go:embed assets/arch-build.sh
var archBuildScript []byte

// BootstrapFile is the name Arch publishes the latest bootstrap image under.
const BootstrapFile = "archlinux-bootstrap-x86_64.tar.zst"

// Mirrors are tried in order for the bootstrap image and its checksum.
var Mirrors = []string{
	"https://geo.mirror.pkgbuild.com",
	"https://mirrors.kernel.org/archlinux",
	"https://mirror.rackspace.com/archlinux",
}

// CacheDir is where the downloaded and unpacked Arch Linux is kept between
// builds.
func CacheDir() string {
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "osmaker", "arch")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "osmaker", "arch")
}

// EnsureBootstrap makes sure cache holds an uncompressed copy of the latest
// bootstrap image, downloading and checking it when there is a newer one,
// and returns its path.
func EnsureBootstrap(ctx context.Context, cache string, log io.Writer) (string, error) {
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return "", err
	}
	tarball := filepath.Join(cache, "bootstrap.tar")
	marker := filepath.Join(cache, "bootstrap.sha256")

	var lastErr error
	for _, mirror := range Mirrors {
		base := mirror + "/iso/latest/"
		want, err := fetchChecksum(ctx, base+"sha256sums.txt")
		if err != nil {
			lastErr = err
			fmt.Fprintf(log, "  %s: %v\n", mirror, err)
			continue
		}
		if have, _ := os.ReadFile(marker); strings.TrimSpace(string(have)) == want {
			if _, err := os.Stat(tarball); err == nil {
				fmt.Fprintln(log, "Arch Linux is already downloaded and up to date.")
				return tarball, nil
			}
		}
		fmt.Fprintf(log, "Downloading Arch Linux from %s …\n", mirror)
		if err := download(ctx, base+BootstrapFile, want, tarball, log); err != nil {
			lastErr = err
			fmt.Fprintf(log, "  %v\n", err)
			continue
		}
		if err := os.WriteFile(marker, []byte(want+"\n"), 0o644); err != nil {
			return "", err
		}
		return tarball, nil
	}
	return "", fmt.Errorf("could not download Arch Linux (is the internet connected?): %w", lastErr)
}

// fetchChecksum reads the bootstrap image's SHA-256 from Arch's checksum list.
func fetchChecksum(ctx context.Context, url string) (string, error) {
	body, err := get(ctx, url)
	if err != nil {
		return "", err
	}
	defer body.Close()
	sc := bufio.NewScanner(io.LimitReader(body, 1<<20))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && f[1] == BootstrapFile && len(f[0]) == 64 {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", errors.New("the checksum list does not mention the bootstrap image")
}

// download fetches url, checks its SHA-256 against want, and stores it
// decompressed at dest.
func download(ctx context.Context, url, want, dest string, log io.Writer) error {
	body, err := get(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()

	// Keep the compressed file only until it is checked: a corrupt or
	// tampered download must never be unpacked.
	part := dest + ".zst.part"
	defer os.Remove(part)
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	sum := sha256.New()
	pw := &progress{w: log, total: body.size}
	if _, err := io.Copy(io.MultiWriter(f, sum, pw), body); err != nil {
		f.Close()
		return fmt.Errorf("download interrupted: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); got != want {
		return fmt.Errorf("the download is damaged (checksum %s, expected %s)", got[:12], want[:12])
	}
	fmt.Fprintln(log, "Download checked. Unpacking …")

	in, err := os.Open(part)
	if err != nil {
		return err
	}
	defer in.Close()
	dec, err := zstd.NewReader(in)
	if err != nil {
		return err
	}
	defer dec.Close()
	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, dec); err != nil {
		out.Close()
		os.Remove(tmp)
		return fmt.Errorf("unpacking: %w", err)
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

type sizedBody struct {
	io.ReadCloser
	size int64
}

func get(ctx context.Context, url string) (*sizedBody, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("%s: %s", url, res.Status)
	}
	return &sizedBody{res.Body, res.ContentLength}, nil
}

// progress reports the download every 10%.
type progress struct {
	w     io.Writer
	total int64
	done  int64
	shown int64
}

func (p *progress) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if p.total > 0 {
		pct := p.done * 100 / p.total
		if pct/10 > p.shown/10 {
			p.shown = pct
			fmt.Fprintf(p.w, "  %d%% of %d MB\n", pct, p.total>>20)
		}
	}
	return len(b), nil
}

// writeArchBuildScript puts the script that runs as root next to the cache.
func writeArchBuildScript(cache string) (string, error) {
	path := filepath.Join(cache, "osmaker-arch-build.sh")
	if err := os.WriteFile(path, archBuildScript, 0o755); err != nil {
		return "", err
	}
	return path, os.Chmod(path, 0o755)
}
