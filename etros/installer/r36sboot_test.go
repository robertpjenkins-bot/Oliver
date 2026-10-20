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
