// Ollie OS Installer for Windows: downloads the newest Ollie OS from its
// GitHub releases and writes it onto a USB drive, which then starts any PC,
// the Steam Machine or an Intel Mac into Ollie OS. The drive is erased.
//
// Windows asks once to allow it to make changes (it writes to a drive).
// It lists only USB drives, never the drive Windows runs from.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	repo = "robertpjenkins-bot/Ollie-OS"
	tag  = "ollie-os-newest"
)

var in = bufio.NewReader(os.Stdin)

func main() {
	fmt.Println()
	fmt.Println("  Ollie OS Installer  (puts Ollie OS on a USB drive)")
	fmt.Println()
	if err := run(); err != nil {
		fmt.Println()
		fmt.Println("Error:", err)
	}
	fmt.Println()
	fmt.Print("Press Enter to close.")
	in.ReadString('\n')
}

func run() error {
	if !isAdmin() {
		// Writing to a drive needs Windows' "allow changes": start again with it.
		self, _ := os.Executable()
		if err := exec.Command("powershell", "-NoProfile", "-Command",
			"Start-Process -FilePath '"+strings.ReplaceAll(self, "'", "''")+"' -Verb RunAs").Run(); err != nil {
			return errors.New("Windows did not allow it to make changes, which writing to a drive needs. Run it again and click Yes")
		}
		os.Exit(0) // the new window, with the changes allowed, carries on
	}

	fmt.Println("==> Finding the newest Ollie OS")
	parts, sum, size, name, err := newestISO()
	if err != nil {
		return err
	}
	fmt.Printf("  %s (%d MB)\n", name, size/1000000)

	disks, err := usbDisks()
	if err != nil {
		return err
	}
	if len(disks) == 0 {
		return errors.New("no USB drive was found. Plug in a USB drive (8 GB or more), then run this again")
	}
	fmt.Println()
	fmt.Println("==> USB drives (everything on the one you pick is erased)")
	for i, d := range disks {
		fmt.Printf("  %d) %-40s %6.1f GB\n", i+1, d.FriendlyName, float64(d.Size)/1e9)
	}
	fmt.Println()
	pick := ask(fmt.Sprintf("Put Ollie OS on which drive? [1-%d] ", len(disks)))
	n, err := strconv.Atoi(pick)
	if err != nil || n < 1 || n > len(disks) {
		return errors.New("no such drive; nothing was changed")
	}
	d := disks[n-1]
	if d.Size < size {
		return fmt.Errorf("%s is too small for Ollie OS (it needs %d MB)", d.FriendlyName, size/1000000)
	}
	fmt.Printf("\nEVERYTHING on %s (%.1f GB) will be erased.\n", d.FriendlyName, float64(d.Size)/1e9)
	if ask("Type ERASE to put Ollie OS on it: ") != "ERASE" {
		return errors.New("cancelled; nothing was changed")
	}

	fmt.Println()
	fmt.Println("==> Erasing the drive")
	// No parts left that Windows has open, so the whole drive can be written.
	exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("Clear-Disk -Number %d -RemoveData -RemoveOEM -Confirm:$false", d.Number)).Run()
	dev, err := os.OpenFile(fmt.Sprintf(`\\.\PhysicalDrive%d`, d.Number), os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("could not open the drive (%v). Unplug it, plug it in again, and run this again", err)
	}
	defer dev.Close()

	fmt.Println("==> Downloading Ollie OS onto the drive (10 to 30 minutes)")
	h := sha256.New()
	w := &progress{total: size, start: time.Now()}
	buf := make([]byte, 0, 4<<20)
	block := make([]byte, 4<<20)
	flush := func(final bool) error {
		if len(buf) == 0 {
			return nil
		}
		if final && len(buf)%4096 != 0 { // a drive takes whole sectors
			buf = append(buf, make([]byte, 4096-len(buf)%4096)...)
		}
		_, err := dev.Write(buf)
		buf = buf[:0]
		return err
	}
	for _, url := range parts {
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("the download stopped (%v). Is the internet connected? Run this again", err)
		}
		for {
			k, rerr := resp.Body.Read(block)
			if k > 0 {
				h.Write(block[:k])
				w.add(k)
				buf = append(buf, block[:k]...)
				if len(buf) >= 4<<20 {
					if err := flush(false); err != nil {
						resp.Body.Close()
						return fmt.Errorf("could not write to the drive: %v", err)
					}
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				resp.Body.Close()
				return fmt.Errorf("the download stopped (%v). Run this again", rerr)
			}
		}
		resp.Body.Close()
	}
	if err := flush(true); err != nil {
		return fmt.Errorf("could not write to the drive: %v", err)
	}
	fmt.Println()
	if sum != "" && hex.EncodeToString(h.Sum(nil)) != sum {
		return errors.New("the download was damaged on the way. Run this again")
	}
	fmt.Println()
	fmt.Println("  Done: Ollie OS is on the USB drive, and checked.")
	fmt.Println()
	fmt.Println("  Start the computer from it: Dell and most PCs, tap F12 (or F11, Esc)")
	fmt.Println("  while it starts; Steam Machine, hold Volume Down; Mac, hold Option.")
	fmt.Println("  (Windows may say the drive needs formatting: say Cancel. That is Ollie OS.)")
	return nil
}

type disk struct {
	Number       int
	FriendlyName string
	Size         int64
}

// The USB drives, as Windows sees them.
func usbDisks() ([]disk, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"@(Get-Disk | Where-Object { $_.BusType -eq 'USB' -and -not $_.IsBoot -and -not $_.IsSystem } | Select-Object Number,FriendlyName,Size) | ConvertTo-Json -Compress").Output()
	if err != nil {
		return nil, fmt.Errorf("could not list the drives: %v", err)
	}
	var disks []disk
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	if strings.HasPrefix(s, "{") {
		s = "[" + s + "]"
	}
	if err := json.Unmarshal([]byte(s), &disks); err != nil {
		return nil, fmt.Errorf("could not read the list of drives: %v", err)
	}
	return disks, nil
}

// The ISO's download links (one file, or its parts in order), its SHA-256,
// size and name, from the newest release.
func newestISO() (parts []string, sum string, size int64, name string, err error) {
	api := "https://api.github.com"
	if a := os.Getenv("OLLIE_API"); a != "" { // for tests
		api = a
	}
	resp, err := http.Get(api + "/repos/" + repo + "/releases/tags/" + tag)
	if err != nil {
		return nil, "", 0, "", fmt.Errorf("could not reach GitHub (%v). Is the internet connected?", err)
	}
	defer resp.Body.Close()
	var rel struct {
		Assets []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if resp.StatusCode != 200 || json.NewDecoder(resp.Body).Decode(&rel) != nil {
		return nil, "", 0, "", fmt.Errorf("GitHub has no Ollie OS release right now (%s). Try again later", resp.Status)
	}
	sumURL := ""
	for _, a := range rel.Assets {
		switch {
		case strings.HasSuffix(a.Name, ".iso"):
			name, size, parts = a.Name, a.Size, []string{a.URL}
		case strings.HasSuffix(a.Name, ".iso.sha256"):
			sumURL = a.URL
		}
	}
	if parts == nil { // the ISO in parts (.iso.part0, .part1 ...), in order
		for i := 0; ; i++ {
			found := false
			for _, a := range rel.Assets {
				if strings.HasSuffix(a.Name, fmt.Sprintf(".iso.part%d", i)) {
					parts = append(parts, a.URL)
					size += a.Size
					name = strings.TrimSuffix(a.Name, fmt.Sprintf(".part%d", i))
					found = true
				}
			}
			if !found {
				break
			}
		}
	}
	if parts == nil {
		return nil, "", 0, "", errors.New("the newest Ollie OS release has no ISO in it yet. Try again later")
	}
	if sumURL != "" {
		if r, err := http.Get(sumURL); err == nil {
			b, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if f := strings.Fields(string(b)); len(f) > 0 {
				sum = strings.ToLower(f[0])
			}
		}
	}
	return parts, sum, size, name, nil
}

func isAdmin() bool {
	return exec.Command("net", "session").Run() == nil
}

func ask(q string) string {
	fmt.Print(q)
	s, _ := in.ReadString('\n')
	return strings.TrimSpace(s)
}

type progress struct {
	total, done int64
	last        int
	start       time.Time
}

func (p *progress) add(n int) {
	p.done += int64(n)
	pct := int(p.done * 100 / p.total)
	if pct != p.last {
		p.last = pct
		fmt.Printf("\r  %3d%%  %d of %d MB  (%d min)", pct, p.done/1000000, p.total/1000000, int(time.Since(p.start).Minutes()))
	}
}
