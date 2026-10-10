package builder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func fakeMirror(t *testing.T, payload []byte, sum string) (*httptest.Server, *int) {
	t.Helper()
	var buf bytes.Buffer
	enc, _ := zstd.NewWriter(&buf)
	enc.Write(payload)
	enc.Close()
	compressed := buf.Bytes()
	if sum == "" {
		h := sha256.Sum256(compressed)
		sum = hex.EncodeToString(h[:])
	}
	downloads := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/iso/latest/sha256sums.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("0000  archlinux-2026.10.01-x86_64.iso\n" + sum + "  " + BootstrapFile + "\n"))
	})
	mux.HandleFunc("/iso/latest/"+BootstrapFile, func(w http.ResponseWriter, r *http.Request) {
		downloads++
		w.Write(compressed)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &downloads
}

func withMirrors(t *testing.T, m ...string) {
	old := Mirrors
	Mirrors = m
	t.Cleanup(func() { Mirrors = old })
}

func TestEnsureBootstrapDownloadsChecksAndCaches(t *testing.T) {
	payload := []byte("pretend this is a tar of Arch Linux")
	srv, downloads := fakeMirror(t, payload, "")
	// The first mirror is down; it must fall through to the next.
	withMirrors(t, "http://127.0.0.1:1", srv.URL)
	cache := t.TempDir()
	var log bytes.Buffer

	tarball, err := EnsureBootstrap(context.Background(), cache, &log)
	if err != nil {
		t.Fatalf("%v\n%s", err, log.String())
	}
	if got, _ := os.ReadFile(tarball); !bytes.Equal(got, payload) {
		t.Fatalf("unpacked %q", got)
	}
	if _, err := EnsureBootstrap(context.Background(), cache, &log); err != nil || *downloads != 1 {
		t.Fatalf("second call should reuse the download (downloads=%d, err=%v)", *downloads, err)
	}
	if !strings.Contains(log.String(), "already downloaded") {
		t.Errorf("log: %s", log.String())
	}
}

func TestEnsureBootstrapRejectsDamagedDownload(t *testing.T) {
	srv, _ := fakeMirror(t, []byte("data"), strings.Repeat("ab", 32))
	withMirrors(t, srv.URL)
	cache := t.TempDir()
	var log bytes.Buffer
	if _, err := EnsureBootstrap(context.Background(), cache, &log); err == nil {
		t.Fatal("a download with the wrong checksum was accepted")
	}
	if !strings.Contains(log.String(), "damaged") {
		t.Errorf("log should say the download is damaged: %s", log.String())
	}
	if m, _ := filepath.Glob(filepath.Join(cache, "*")); len(m) != 0 {
		t.Errorf("damaged download left files behind: %v", m)
	}
}

func TestArchBuildScriptParses(t *testing.T) {
	path, err := writeArchBuildScript(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("bash", "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}
