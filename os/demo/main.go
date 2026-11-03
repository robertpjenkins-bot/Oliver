// Ollie OS Demo: opens a pretend Ollie OS desktop (web/index.html, carried
// inside this program) in the computer's web browser, served only to this
// computer. It stops by itself about a minute after the demo's tab closes.
// The same program is the Windows .exe, the Mac app (in the .dmg) and the
// Linux / SteamOS AppImage.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync/atomic"
	"time"
)

//go:embed web
var web embed.FS

func main() {
	site, _ := fs.Sub(web, "web")
	var last atomic.Int64
	last.Store(time.Now().Unix())
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(site)))
	mux.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		last.Store(time.Now().Unix())
		w.WriteHeader(http.StatusNoContent)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("Could not start the demo:", err)
		os.Exit(1)
	}
	url := fmt.Sprintf("http://%s/", ln.Addr())
	fmt.Println("Ollie OS Demo is open in your web browser:", url)
	fmt.Println("It stops by itself when you close the demo's tab.")
	go func() {
		// The page checks in every 10 seconds; a minute without it, the demo is closed.
		for range time.Tick(15 * time.Second) {
			if time.Now().Unix()-last.Load() > 60 {
				os.Exit(0)
			}
		}
	}()
	if os.Getenv("OLLIE_DEMO_NO_BROWSER") == "" {
		openBrowser(url)
	}
	http.Serve(ln, mux)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("Open this in your web browser:", url)
	}
}
