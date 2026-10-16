// Etros Installer puts Etros on a microSD card: Batocera for the Raspberry
// Pi 5 and the R36S on one card, with the Etros look and Ollie OS.
//
// Run it with no arguments to get the installer window (a page in your
// browser). For the command line:
//
//	etros-installer -list
//	etros-installer -target disk:ID -yes
//	etros-installer -target file:etros.img
package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed ui/index.html
var indexHTML []byte

//go:embed assets/ui/PressStart2P-Regular.ttf
var uiFont []byte

//go:embed assets/ui/splash.png
var uiSplash []byte

var version = "dev"

func main() {
	var (
		list      = flag.Bool("list", false, "list the cards that can be written to, and exit")
		target    = flag.String("target", "", "install without the window: disk:ID (from -list) or file:PATH")
		yes       = flag.Bool("yes", false, "with -target disk:..., don't ask before erasing the card")
		noOllie   = flag.Bool("no-ollie", false, "leave out Ollie OS")
		games     = flag.String("games", "", "folder of your games (one folder per console) to put on the card")
		port      = flag.Int("port", 0, "port for the installer window (default: any free one)")
		noBrowser = flag.Bool("no-browser", false, "don't open the installer window in a browser")
		elevated  = flag.Bool("elevated", false, "internal: already asked for administrator rights")
		token     = flag.String("token", "", "internal: the installer window's access token")
		s         Sources
	)
	flag.StringVar(&s.Pi5Boot, "pi5-boot", "", "Batocera boot.tar.xz for the Pi 5 (file or URL)")
	flag.StringVar(&s.R36SBoot, "r36s-boot", "", "Batocera boot.tar.xz for the R36S (file or URL)")
	flag.StringVar(&s.R36SImage, "r36s-image", "", "Batocera .img.gz for the R36S, for its bootloader (file or URL)")
	flag.StringVar(&s.RPiOS, "rpios", "", "Raspberry Pi OS 64-bit desktop .img.xz, for Ollie OS (file or URL)")
	flag.StringVar(&s.Channel, "channel", "", "Batocera release channel (default stable)")
	cache := flag.String("cache", defaultCache(), "where to keep downloads")
	flag.Parse()

	opts := Options{Sources: s, Ollie: !*noOllie, GamesDir: *games, CacheDir: *cache}

	switch {
	case *list:
		disks, err := listDisks()
		if err != nil {
			fail(err)
		}
		for _, d := range disks {
			fmt.Printf("disk:%s\t%s\n", d.ID, d.Label)
		}
		if ok, why := canWriteDisks(); !ok {
			fmt.Fprintln(os.Stderr, why)
		}
	case *target != "":
		if err := runCLI(opts, *target, *yes); err != nil {
			fail(err)
		}
	default:
		if !*elevated && relaunchElevated(*port) {
			return
		}
		if err := serveUI(opts, *port, *token, !*noBrowser); err != nil {
			fail(err)
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "etros-installer:", err)
	if runtime.GOOS == "windows" {
		// Keep the window open long enough to read when double-clicked.
		time.Sleep(30 * time.Second)
	}
	os.Exit(1)
}

func defaultCache() string {
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "etros-installer")
	}
	return filepath.Join(os.TempDir(), "etros-installer")
}

func defaultImageFile() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Downloads")
	if _, err := os.Stat(dir); err != nil {
		dir = home
	}
	return filepath.Join(dir, "etros.img")
}

// target is something the card can be written to.
type writeTarget interface {
	WriteAt(p []byte, off int64) (int, error)
	Finish() error
}

type fileTarget struct{ *os.File }

func (f fileTarget) Finish() error {
	if err := f.Sync(); err != nil {
		return err
	}
	return f.Close()
}

// install plans and writes a card to target ("disk:ID" or "file:PATH").
func install(o Options, target string, rep Reporter) error {
	kind, where, _ := strings.Cut(target, ":")
	if kind != "disk" && kind != "file" || where == "" {
		return fmt.Errorf("unknown target %q (want disk:ID or file:PATH)", target)
	}
	if kind == "disk" {
		if ok, why := canWriteDisks(); !ok {
			return errors.New(why)
		}
	}
	if o.GamesDir != "" {
		if st, err := os.Stat(o.GamesDir); err != nil || !st.IsDir() {
			return fmt.Errorf("games folder %q not found", o.GamesDir)
		}
	}
	card, err := Plan(o, rep)
	if err != nil {
		return err
	}

	var out writeTarget
	if kind == "disk" {
		d, err := openDisk(where)
		if err != nil {
			return err
		}
		if d.size < card.Size {
			d.Close()
			return fmt.Errorf("the card is too small: Etros needs %.1f GB, the card has %.1f GB", float64(card.Size)/1e9, float64(d.size)/1e9)
		}
		out = d
	} else {
		f, err := os.Create(where)
		if err != nil {
			return err
		}
		if err := f.Truncate(card.Size); err != nil {
			f.Close()
			return err
		}
		out = fileTarget{f}
	}
	if err := card.Write(out, rep); err != nil {
		out.Finish()
		return err
	}
	rep.Stage("Finishing")
	return out.Finish()
}

// cliReporter prints progress to the terminal.
type cliReporter struct{ last time.Time }

func (c *cliReporter) Stage(s string) { fmt.Printf("\n== %s\n", s) }
func (c *cliReporter) Log(s string)   { fmt.Println("  " + s) }
func (c *cliReporter) Progress(done, total int64) {
	if time.Since(c.last) < 2*time.Second {
		return
	}
	c.last = time.Now()
	if total > 0 {
		fmt.Printf("  %.0f%% (%.2f of %.2f GB)\n", 100*float64(done)/float64(total), float64(done)/1e9, float64(total)/1e9)
	} else {
		fmt.Printf("  %.2f GB\n", float64(done)/1e9)
	}
}

func runCLI(o Options, target string, yes bool) error {
	if strings.HasPrefix(target, "disk:") && !yes {
		fmt.Printf("Everything on %s will be erased. Type ERASE to go on: ", target)
		var answer string
		fmt.Scanln(&answer)
		if answer != "ERASE" {
			return errors.New("stopped; nothing was written")
		}
	}
	if err := install(o, target, &cliReporter{}); err != nil {
		return err
	}
	fmt.Println("\nDone. Put the card in your R36S or Raspberry Pi 5 and switch it on.")
	return nil
}

// job is the install the window started.
type job struct {
	mu       sync.Mutex
	running  bool
	stage    string
	log      []string
	done     int64
	total    int64
	finished bool
	err      string
	toFile   string
}

func (j *job) Stage(s string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.stage, j.done, j.total = s, 0, 0
	j.log = append(j.log, "== "+s)
}

func (j *job) Log(s string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.log = append(j.log, s)
}

func (j *job) Progress(done, total int64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.done, j.total = done, total
}

func serveUI(o Options, port int, token string, openBrowser bool) error {
	if token == "" {
		tok := make([]byte, 16)
		rand.Read(tok)
		token = hex.EncodeToString(tok)
	}
	j := &job{}

	mux := http.NewServeMux()
	page := func(ctype string, b []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", ctype)
			w.Write(b)
		}
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		page("text/html; charset=utf-8", indexHTML)(w, r)
	})
	mux.HandleFunc("/font.ttf", page("font/ttf", uiFont))
	mux.HandleFunc("/splash.png", page("image/png", uiSplash))

	// The API only answers requests carrying the token from the address we
	// opened. A custom header can't be sent cross-site without the browser
	// asking first, which this server never allows, so no web page can
	// start an install.
	guard := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Etros-Token") != token {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			h(w, r)
		}
	}
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("/api/disks", guard(func(w http.ResponseWriter, r *http.Request) {
		disks, err := listDisks()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if disks == nil {
			disks = []Disk{}
		}
		ok, why := canWriteDisks()
		writeJSON(w, map[string]any{"disks": disks, "canWrite": ok, "whyNot": why, "defaultFile": defaultImageFile()})
	}))
	mux.HandleFunc("/api/install", guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Target string `json:"target"`
			Ollie  bool   `json:"ollie"`
			Games  string `json:"games"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		j.mu.Lock()
		if j.running {
			j.mu.Unlock()
			http.Error(w, "an install is already running", http.StatusConflict)
			return
		}
		*j = job{running: true, stage: "Starting"}
		if f, ok := strings.CutPrefix(req.Target, "file:"); ok {
			j.toFile = f
		}
		j.mu.Unlock()
		run := o
		run.Ollie = req.Ollie
		run.GamesDir = strings.TrimSpace(req.Games)
		go func() {
			err := install(run, req.Target, j)
			j.mu.Lock()
			defer j.mu.Unlock()
			j.running = false
			if err != nil {
				j.err = err.Error()
				j.log = append(j.log, "ERROR: "+err.Error())
			} else {
				j.finished = true
				j.stage = "Done"
			}
		}()
		writeJSON(w, map[string]bool{"ok": true})
	}))
	mux.HandleFunc("/api/status", guard(func(w http.ResponseWriter, r *http.Request) {
		j.mu.Lock()
		defer j.mu.Unlock()
		writeJSON(w, map[string]any{"stage": j.stage, "log": j.log, "done": j.done, "total": j.total,
			"finished": j.finished, "error": j.err, "toFile": j.toFile})
	}))

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s/?t=%s", ln.Addr(), token)
	fmt.Println("Etros Installer is open in your browser. If it isn't, go to:")
	fmt.Println("  " + url)
	fmt.Println("Keep this window open until the install is finished.")
	if openBrowser {
		go openURL(url)
	}
	return http.Serve(ln, mux)
}

func openURL(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		// Under sudo, open the browser as the person who ran it.
		if u := os.Getenv("SUDO_USER"); u != "" && os.Geteuid() == 0 {
			exec.Command("sudo", "-u", u, "xdg-open", url).Start()
			return
		}
		exec.Command("xdg-open", url).Start()
	}
}
