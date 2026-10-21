//go:build linux

package main

import (
	"net/http"
	"os"
	"testing"
)

func TestLatestProtonURLExists(t *testing.T) {
	if os.Getenv("ETROS_NET_TESTS") == "" {
		t.Skip("set ETROS_NET_TESTS=1")
	}
	url := latestProtonURL()
	t.Log(url)
	resp, err := http.Head(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: %s", url, resp.Status)
	}
	t.Logf("%d MB", resp.ContentLength>>20)
}
