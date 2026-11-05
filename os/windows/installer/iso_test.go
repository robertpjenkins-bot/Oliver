package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The ISO in two parts on the release: both links, in order, the whole
// size, its name, and the checksum from its .sha256.
func TestNewestISOParts(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + repo + "/releases/tags/" + tag:
			fmt.Fprintf(w, `{"assets":[
				{"name":"ollie-os-2026.11.03-x86_64.iso.part1","size":200,"browser_download_url":"%[1]s/p1"},
				{"name":"Ollie-OS-Drive-Writer-x86_64.AppImage","size":5,"browser_download_url":"%[1]s/dw"},
				{"name":"ollie-os-2026.11.03-x86_64.iso.part0","size":1000,"browser_download_url":"%[1]s/p0"},
				{"name":"ollie-os-2026.11.03-x86_64.iso.sha256","size":97,"browser_download_url":"%[1]s/sum"}]}`, srv.URL)
		case "/sum":
			fmt.Fprint(w, "ABCDEF0123  ollie-os-2026.11.03-x86_64.iso\n")
		}
	}))
	defer srv.Close()
	t.Setenv("OLLIE_API", srv.URL)
	parts, sum, size, name, err := newestISO()
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 2 || parts[0] != srv.URL+"/p0" || parts[1] != srv.URL+"/p1" {
		t.Errorf("parts = %v", parts)
	}
	if sum != "abcdef0123" || size != 1200 || name != "ollie-os-2026.11.03-x86_64.iso" {
		t.Errorf("sum %q size %d name %q", sum, size, name)
	}
}
