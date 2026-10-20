package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveR36SBoot(t *testing.T) {
	have := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !have[r.URL.Path] {
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := batoceraBase
	batoceraBase = srv.URL
	defer func() { batoceraBase = old }()

	have["/rk3326/stable/last/boot.tar.xz"] = true
	if url, err := resolveR36SBoot("stable"); err != nil || !strings.HasSuffix(url, "/rk3326/stable/last/boot.tar.xz") {
		t.Errorf("only rk3326 there: got %q, %v", url, err)
	}
	have["/r36s/stable/last/boot.tar.xz"] = true
	if url, err := resolveR36SBoot("stable"); err != nil || !strings.HasSuffix(url, "/r36s/stable/last/boot.tar.xz") {
		t.Errorf("both there: got %q, %v", url, err)
	}
	have = map[string]bool{}
	if _, err := resolveR36SBoot("stable"); err == nil || !strings.Contains(err.Error(), "batocera.org") {
		t.Errorf("neither there: want advice to download it, got %v", err)
	}
}

func TestR36SImageURLFromVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rk3326/stable/last/batocera.version":
			w.Write([]byte("43.1 2026/05/29 10:12\n"))
		case "/rk3326/stable/last/batocera-rk3326-43.1-20260530.img.gz": // a day after
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	url, err := r36sImageURL(srv.URL + "/rk3326/stable/last/boot.tar.xz")
	if err != nil || !strings.HasSuffix(url, "/rk3326/stable/last/batocera-rk3326-43.1-20260530.img.gz") {
		t.Errorf("rk3326: got %q, %v", url, err)
	}
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/r36s/stable/last/batocera.version":
			w.Write([]byte("44 2026/11/02 08:00\n"))
		case "/r36s/stable/last/batocera-rk3326-r36s-44-20261102.img.gz":
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv2.Close()
	url, err = r36sImageURL(srv2.URL + "/r36s/stable/last/boot.tar.xz")
	if err != nil || !strings.HasSuffix(url, "batocera-rk3326-r36s-44-20261102.img.gz") {
		t.Errorf("r36s: got %q, %v", url, err)
	}
}
