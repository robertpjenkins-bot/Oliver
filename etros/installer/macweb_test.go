package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeMac stands in for a 12-inch MacBook with an ExFAT partition for
// Etros, and a USB stick.
type fakeMac struct {
	mu       sync.Mutex
	got      macTarget
	gotEtros int64
	ssid     string
	slow     time.Duration
}

func fakeDisks() []dDisk {
	return []dDisk{
		{id: "disk0", name: "APPLE SSD SM0256G", size: 251 << 30, internal: true, parts: []dPart{
			{"disk0s1", "EFI", "EFI", 200 << 20}, {"disk0s2", "Apple_APFS", "", 180 << 30}, {"disk0s3", "Microsoft Basic Data", "ETROS", 70 << 30},
		}},
		{id: "disk2", name: "SanDisk Ultra", size: 64 << 30, parts: []dPart{{"disk2s1", "Microsoft Basic Data", "STICK", 64 << 30}}},
	}
}

func (f *fakeMac) targets() ([]macTarget, error) { return macTargets(fakeDisks(), "disk0s2"), nil }
func (f *fakeMac) roomFree(string) int64         { return 90 << 30 }
func (f *fakeMac) wifiName() string              { return "Jenkins Wi-Fi" }
func (f *fakeMac) install(x macTarget, etros int64, ssid, key string, rep Reporter) error {
	f.mu.Lock()
	f.got, f.gotEtros, f.ssid = x, etros, ssid
	f.mu.Unlock()
	rep.Stage("Downloading Batocera for PCs (about 3 GB)")
	for i := int64(0); i <= 10; i++ {
		rep.Progress(i*300<<20, 3000<<20)
		time.Sleep(f.slow)
	}
	rep.Log("diskutil eraseVolume free none " + x.part.id)
	rep.Stage("Copying Batocera and Etros")
	time.Sleep(f.slow)
	rep.Stage("Done")
	return nil
}

func TestMacWeb(t *testing.T) {
	f := &fakeMac{}
	quit := make(chan struct{})
	srv := httptest.NewServer(macWebHandler(f, "s3cr3t", func() { close(quit) }))
	defer srv.Close()
	base := srv.URL + "/s3cr3t/"

	if r, _ := http.Get(srv.URL + "/drives"); r.StatusCode != 404 {
		t.Error("API without the secret")
	}
	r, err := http.Get(base)
	if err != nil || r.StatusCode != 200 {
		t.Fatal(err)
	}
	var drives struct {
		Targets []webTarget `json:"targets"`
		Wifi    string      `json:"wifi"`
	}
	r, _ = http.Get(base + "drives")
	json.NewDecoder(r.Body).Decode(&drives)
	if drives.Wifi != "Jenkins Wi-Fi" || len(drives.Targets) != 5 {
		t.Fatalf("drives: %+v", drives)
	}
	exfat := -1
	for _, x := range drives.Targets {
		if x.ID == "disk0s3" {
			exfat = x.N
		}
	}
	post := func(path string, v any) *http.Response {
		b, _ := json.Marshal(v)
		r, _ := http.Post(base+path, "application/json", bytes.NewReader(b))
		return r
	}
	if r := post("install", map[string]any{"n": exfat, "confirm": "nope"}); r.StatusCode != 400 {
		t.Error("installed without ERASE")
	}
	if r := post("install", map[string]any{"n": 0, "confirm": "ERASE"}); r.StatusCode != 400 {
		t.Error("installed on EFI")
	}
	if r := post("install", map[string]any{"n": exfat, "confirm": "erase", "ssid": "Jenkins Wi-Fi", "key": "pw"}); r.StatusCode != 200 {
		t.Fatalf("install: %d", r.StatusCode)
	}
	var st webState
	for i := 0; i < 100; i++ {
		r, _ := http.Get(base + "status")
		st = webState{}
		json.NewDecoder(r.Body).Decode(&st)
		if st.Finished {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !st.Finished || st.Error != "" || !st.Ollie || st.Kind != "part" || !strings.Contains(strings.Join(st.Lines, "\n"), "eraseVolume free none disk0s3") {
		t.Errorf("status: finished=%v error=%q ollie=%v kind=%q lines=%q", st.Finished, st.Error, st.Ollie, st.Kind, st.Lines)
	}
	if f.got.part.id != "disk0s3" || f.gotEtros != 70<<30 || f.ssid != "Jenkins Wi-Fi" {
		t.Errorf("installed %+v %d %q", f.got.part, f.gotEtros, f.ssid)
	}
	post("quit", map[string]any{})
	select {
	case <-quit:
	case <-time.After(2 * time.Second):
		t.Error("didn't quit")
	}
}

// For looking at the page in a browser: ETROS_WEB_DEMO=1 go test -run
// TestMacWebDemo -timeout 0, then open the address it prints.
func TestMacWebDemo(t *testing.T) {
	if os.Getenv("ETROS_WEB_DEMO") == "" {
		t.Skip("set ETROS_WEB_DEMO")
	}
	serveMacWeb(&fakeMac{slow: 400 * time.Millisecond}, func(url string) error {
		os.WriteFile(os.Getenv("ETROS_WEB_DEMO"), []byte(url), 0o644)
		return nil
	})
}
