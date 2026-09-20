// Package webui serves Oliver's drive tools as a page in a browser.
//
// A browser cannot open a raw block device, and no page served from the
// internet ever will. So the page here is only a front end: every drive is
// read, written, erased and verified by this program, which the page talks to
// over a loopback socket. The rules about which drives may be touched are the
// same ones the command line uses, enforced here rather than in the page,
// because a front end is not where a safety check belongs.
package webui

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/device"
	"github.com/robertpjenkins-bot/oliver/internal/imaging"
)

// Server answers the page's requests.
type Server struct {
	// Scanner builds the block device scanner. It is a field so tests can
	// describe a machine's drives without touching the real ones.
	Scanner func() *device.Scanner

	// Token must accompany every request. Without it the socket is open to
	// anything else running on this machine, and to any web page that guesses
	// the port.
	Token string

	// Home is where the file browser starts looking for images.
	Home string

	mu   sync.Mutex
	jobs map[string]*Job
}

// New returns a server with a fresh token.
func New() *Server {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	return &Server{
		Scanner: device.NewScanner,
		Token:   newToken(),
		Home:    home,
		jobs:    map[string]*Job{},
	}
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// A predictable token would leave the socket open to anything on the
		// machine, so there is nothing sensible to carry on with.
		panic("oliver: no source of randomness for the session token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// Handler returns the routes, with every one of them behind the token.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// The page itself takes the token from the query string, since a browser
	// being opened at a URL is the only thing it can carry.
	mux.HandleFunc("GET /{$}", s.page)

	api := map[string]http.HandlerFunc{
		"GET /api/status":            s.status,
		"GET /api/drives":            s.drives,
		"GET /api/image":             s.image,
		"GET /api/browse":            s.browse,
		"GET /api/jobs":              s.listJobs,
		"POST /api/jobs/{id}/cancel": s.cancelJob,
		"GET /api/diagnose":          s.diagnose,
		"POST /api/unprotect":        s.unprotect,
		"POST /api/write":            s.write,
		"POST /api/repair":           s.repair,
		"POST /api/test":             s.surfaceTest,
		"POST /api/erase":            s.erase,
		"POST /api/format":           s.format,
	}
	for pattern, handler := range api {
		mux.HandleFunc(pattern, s.check(handler, headerToken))
	}

	// The progress stream is the one exception: a browser's EventSource
	// cannot set a header, so its token travels in the query string. It is
	// read-only, and reports on a job the caller must already have started.
	mux.HandleFunc("GET /api/jobs/{id}/events", s.check(s.jobEvents, queryToken))
	return s.preflight(mux)
}

// preflight answers the permission question a browser asks before sending a
// request that carries a custom header.
//
// It cannot require the token, because the token is exactly what the browser
// is asking permission to send. That costs nothing: a preflight grants only
// the right to try, and the request that follows is checked like any other.
func (s *Server) preflight(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		origin, ok := s.allowedOrigin(r)
		if !ok || origin == "" || !loopbackHost(r.Host) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		allowFrom(w, origin)
		w.Header().Set("Access-Control-Allow-Headers", "X-Oliver-Token, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.WriteHeader(http.StatusNoContent)
	})
}

// allowedOrigin reports which origins may be answered, and what to echo back.
//
// Two are allowed. The page Oliver served itself, which is same-origin. And a
// page opened from a file rather than from a server — the standalone copy of
// this same page, downloaded or carried on a stick — which every browser
// labels "null".
//
// Allowing "null" gives away nothing. Authority here is the token: it is not
// a cookie the browser attaches on its own, it is never sent anywhere else,
// and a page that does not have it gets a refusal whatever its origin says.
func (s *Server) allowedOrigin(r *http.Request) (string, bool) {
	switch origin := r.Header.Get("Origin"); {
	case origin == "":
		return "", true // not a cross-origin request at all
	case origin == "null":
		return "null", true
	case sameOrigin(origin, r.Host):
		return origin, true
	default:
		return "", false
	}
}

func allowFrom(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Add("Vary", "Origin")
}

// tokenIn says where a route accepts its token.
type tokenIn int

const (
	headerToken tokenIn = iota
	queryToken
)

// check rejects anything that is not this page talking to this server.
//
// Three checks, each closing a different door. The token proves the caller was
// told the URL. The custom header it arrives in cannot be set by a form on
// another site, and the preflight that would let one try is answered for two
// origins only, so a page you happen to be visiting cannot drive this one. The
// Host check closes DNS rebinding, where a name an attacker controls is
// pointed at 127.0.0.1.
func (s *Server) check(next http.HandlerFunc, where tokenIn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !loopbackHost(r.Host) {
			fail(w, http.StatusForbidden, fmt.Errorf("this server answers only on localhost"))
			return
		}
		origin, ok := s.allowedOrigin(r)
		if !ok {
			fail(w, http.StatusForbidden, fmt.Errorf("refusing a request from %s", r.Header.Get("Origin")))
			return
		}
		if origin != "" {
			// Without this the browser will not let the page read the answer.
			allowFrom(w, origin)
		}
		token := r.Header.Get("X-Oliver-Token")
		if where == queryToken && token == "" {
			token = r.URL.Query().Get("t")
		}
		if !s.validToken(token) {
			fail(w, http.StatusForbidden, fmt.Errorf("this request carries no valid session token"))
			return
		}
		next(w, r)
	}
}

func (s *Server) validToken(got string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.Token)) == 1
}

// loopbackHost reports whether a request's Host header names this machine.
func loopbackHost(host string) bool {
	name, _, err := net.SplitHostPort(host)
	if err != nil {
		name = host
	}
	if name == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(name, "[]"))
	return ip != nil && ip.IsLoopback()
}

func sameOrigin(origin, host string) bool {
	rest, ok := strings.CutPrefix(origin, "http://")
	if !ok {
		return false
	}
	return rest == host && loopbackHost(rest)
}

// driveView is a drive as the page needs it: the facts, plus the answers to
// the questions the page would otherwise have to work out for itself.
type driveView struct {
	Path       string          `json:"path"`
	Name       string          `json:"name"`
	Model      string          `json:"model,omitempty"`
	Size       int64           `json:"size"`
	SizeHuman  string          `json:"size_human"`
	Removable  bool            `json:"removable"`
	Writable   bool            `json:"writable"`
	Contents   string          `json:"contents"`
	SystemDisk bool            `json:"system_disk"`
	Busy       bool            `json:"busy"`
	Mounts     []device.Mount  `json:"mounts,omitempty"`
	Partitions []partitionView `json:"partitions,omitempty"`
	// Protections are the reasons the drive will not take writes, if any.
	Protections []device.Protection `json:"protections,omitempty"`

	// Faults are what is wrong with how the drive describes itself: the
	// reason a stick turns up at a fraction of its size, or read-only.
	Faults []imaging.Fault `json:"faults,omitempty"`
	// Described is how much of the drive its partition table accounts for.
	Described string `json:"described,omitempty"`

	// Refusal is why this drive cannot be used, empty when it can be.
	Refusal string `json:"refusal,omitempty"`
	// Reason names which refusal it is, so the page can offer the way past
	// the ones that have one.
	Reason string `json:"reason,omitempty"`
}

type partitionView struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	SizeHuman string `json:"size_human"`
	Contents  string `json:"contents"`
	MountedAt string `json:"mounted_at,omitempty"`
}

// status reports what this instance of Oliver can and cannot do, so the page
// can say so before somebody presses a button and finds out.
func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	elevated := device.Elevated()
	out := struct {
		Elevated bool   `json:"elevated"`
		Warning  string `json:"warning,omitempty"`
		Command  string `json:"command,omitempty"`
	}{Elevated: elevated}

	if !elevated {
		out.Warning = "Oliver is not running as root, so it cannot write to a drive or clear a " +
			"read-only flag. Close this and start it again with sudo."
		self, err := device.Self()
		if err != nil {
			self = "oliver"
		}
		out.Command = "sudo " + self + " ui"
	}
	send(w, out)
}

func (s *Server) drives(w http.ResponseWriter, r *http.Request) {
	scanner := s.Scanner()
	devices, err := scanner.List()
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}

	out := make([]driveView, 0, len(devices))
	for _, d := range devices {
		v := driveView{
			Path:       d.Path,
			Name:       d.Name,
			Model:      d.Model,
			Size:       d.Size,
			SizeHuman:  imaging.HumanBytes(d.Size),
			Removable:  d.Removable,
			Writable:   d.Writable,
			SystemDisk: d.SystemCritical(),
			Busy:       d.Busy(),
			Mounts:     d.Mounts,
		}
		if v.SystemDisk {
			v.Contents = "the running system"
		} else if len(d.Partitions) == 0 {
			v.Contents = imaging.Describe(d.Path)
		} else {
			v.Contents = plural(len(d.Partitions), "partition")
		}
		for _, p := range d.Partitions {
			pv := partitionView{
				Path:      p.Path,
				Name:      p.Name,
				SizeHuman: imaging.HumanBytes(p.Size),
				Contents:  imaging.Describe(p.Path),
			}
			if len(p.Mounts) > 0 {
				pv.MountedAt = p.Mounts[0].Target
			}
			v.Partitions = append(v.Partitions, pv)
		}
		// "The disk is write-protected" is the other complaint people arrive
		// with, and the drive should say which kind it is on its own card.
		if !v.SystemDisk {
			v.Protections = scanner.WriteProtection(d)
		}

		// Reading a few sectors is cheap, and a drive that is lying about its
		// size should say so on the card rather than only when asked.
		if !v.SystemDisk {
			if diagnosis, err := imaging.Diagnose(d.Path); err == nil && !diagnosis.Healthy() {
				v.Faults = diagnosis.Faults
				if diagnosis.Described > 0 {
					v.Described = imaging.HumanBytes(diagnosis.Described)
				}
			}
		}

		// The page shows the reason a drive is unavailable rather than hiding
		// it: seeing the system disk listed and locked is how somebody
		// confirms they are looking at the right list.
		if err := device.CheckWritable(d, "write to", false); err != nil {
			v.Refusal = err.Error()
			if reason, ok := device.RefusedFor(err); ok {
				v.Reason = reasonName(reason)
			}
		}
		out = append(out, v)
	}
	send(w, out)
}

// plural renders a count with its noun, since "1 partitions" in a tool this
// careful about wording is a distraction.
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func reasonName(r device.Reason) string {
	switch r {
	case device.SystemDisk:
		return "system_disk"
	case device.FixedDisk:
		return "fixed_disk"
	case device.Mounted:
		return "mounted"
	case device.NotWritable:
		return "not_writable"
	}
	return "unknown"
}

func (s *Server) image(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		fail(w, http.StatusBadRequest, errors.New("no image path given"))
		return
	}
	img, err := imaging.Inspect(path)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	send(w, struct {
		*imaging.Image
		SizeHuman string `json:"size_human"`
		Summary   string `json:"summary"`
	}{img, imaging.HumanBytes(img.Size), img.Describe()})
}

type browseEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Dir       bool   `json:"dir"`
	SizeHuman string `json:"size_human,omitempty"`
}

// browse lists directories and image files, so an image can be chosen by
// pointing at it. A browser's own file picker cannot be used for this: it
// hands over the file's contents, not its path, and these files are measured
// in gigabytes.
func (s *Server) browse(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir = s.Home
	}
	dir = filepath.Clean(dir)
	if !filepath.IsAbs(dir) {
		fail(w, http.StatusBadRequest, fmt.Errorf("%q is not an absolute path", dir))
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}

	out := []browseEntry{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			out = append(out, browseEntry{Name: e.Name(), Path: full, Dir: true})
			continue
		}
		if !imageExtension(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, browseEntry{Name: e.Name(), Path: full, SizeHuman: imaging.HumanBytes(info.Size())})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})

	parent := filepath.Dir(dir)
	if parent == dir {
		parent = ""
	}
	send(w, struct {
		Dir     string        `json:"dir"`
		Parent  string        `json:"parent"`
		Entries []browseEntry `json:"entries"`
	}{dir, parent, out})
}

func imageExtension(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".iso", ".img", ".raw", ".bin", ".dd", ".usb":
		return true
	}
	return false
}

// request is what every destructive call carries.
type request struct {
	Device     string `json:"device"`
	Image      string `json:"image,omitempty"`
	Confirm    string `json:"confirm"`
	AllowFixed bool   `json:"allow_fixed"`
	Unmount    bool   `json:"unmount"`
	Verify     bool   `json:"verify"`
	Full       bool   `json:"full"`
	Raw        bool   `json:"raw"`
	// Quick samples across a drive instead of covering all of it.
	Quick bool   `json:"quick"`
	Name  string `json:"name"`
	// Force permits the steps that have a cost of their own, such as marking
	// a dirty filesystem clean without checking it.
	Force bool `json:"force"`
}

// prepare resolves and checks a request's drive. Every rule the command line
// applies is applied here too, because the page is only a front end and a
// front end's checks can be skipped by anything that can reach the socket.
func (s *Server) prepare(req *request, action string) (*device.Device, error) {
	scanner := s.Scanner()
	dev, err := scanner.Find(req.Device)
	if err != nil {
		return nil, err
	}
	if err := device.CheckWritable(dev, action, req.AllowFixed); err != nil {
		return nil, err
	}
	// The drive's own name, typed back. The page asks for it, but it is
	// checked here: a confirmation a front end can skip is not one.
	if !strings.EqualFold(strings.TrimSpace(req.Confirm), dev.Name) {
		return nil, fmt.Errorf("this would destroy everything on %s; confirm by naming the drive", dev.Path)
	}
	if err := device.CheckIdle(dev, action); err != nil {
		if !req.Unmount {
			return nil, err
		}
		if err := device.Unmount(dev); err != nil {
			return nil, err
		}
	}

	// Knowing why a drive is read-only and then failing on it is no use to
	// anybody. The drive is about to be overwritten; a flag saying "do not
	// write" is much the smaller thing to undo.
	if _, err := device.MakeWritable(scanner, dev); err != nil {
		return nil, err
	}
	return dev, nil
}

func (s *Server) write(w http.ResponseWriter, r *http.Request) {
	req, ok := decode(w, r)
	if !ok {
		return
	}
	img, err := imaging.Inspect(req.Image)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	dev, err := s.prepare(req, "write to")
	if err != nil {
		failRefusal(w, err)
		return
	}
	if dev.Size > 0 && img.Size > dev.Size {
		fail(w, http.StatusBadRequest, fmt.Errorf("%s holds %s, which is smaller than the %s image",
			dev.Path, imaging.HumanBytes(dev.Size), imaging.HumanBytes(img.Size)))
		return
	}

	job, err := s.start("write", dev, req.Image, func(ctx context.Context, j *Job) error {
		res, err := imaging.Write(ctx, req.Image, dev.Path, imaging.Options{
			Verify:   req.Verify,
			Progress: j.progress,
		})
		if res != nil {
			j.finishWith(res, writeSummary(res, req.Verify))
		}
		return err
	})
	if err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	send(w, job.Snapshot())
}

func writeSummary(res *imaging.Result, verified bool) string {
	summary := fmt.Sprintf("wrote %s in %s (%s/s)",
		imaging.HumanBytes(res.Bytes), shortDuration(res.Duration), imaging.HumanBytes(int64(res.Rate())))
	if res.Verified {
		return summary + " — read back and compared: the drive holds the image exactly"
	}
	if !verified {
		return summary + " — not verified"
	}
	return summary
}

func (s *Server) erase(w http.ResponseWriter, r *http.Request) {
	req, ok := decode(w, r)
	if !ok {
		return
	}
	// A name that cannot be written must stop the request now, not after the
	// drive has been emptied.
	label, err := imaging.NormaliseLabel(req.Name)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	dev, err := s.prepare(req, "erase")
	if err != nil {
		failRefusal(w, err)
		return
	}

	job, err := s.start("erase", dev, "", func(ctx context.Context, j *Job) error {
		res, err := imaging.Wipe(ctx, dev.Path, imaging.WipeOptions{Full: req.Full, Progress: j.progress})
		if err != nil {
			return err
		}
		summary := fmt.Sprintf("erased %s in %s", imaging.HumanBytes(res.Bytes), shortDuration(res.Duration))
		if req.Raw {
			j.finishWith(res, summary+" — the drive now has no partition table")
			return nil
		}
		j.setPhase("formatting")
		formatted, err := imaging.FormatFAT32(dev.Path, imaging.FormatOptions{Label: label})
		if err != nil {
			return err
		}
		j.finishWith(res, summary+", "+formatSummary(formatted))
		return nil
	})
	if err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	send(w, job.Snapshot())
}

// diagnose reports what is wrong with a drive, without touching it.
func (s *Server) diagnose(w http.ResponseWriter, r *http.Request) {
	dev, err := s.Scanner().Find(r.URL.Query().Get("device"))
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	d, err := imaging.Diagnose(dev.Path)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	send(w, struct {
		*imaging.Diagnosis
		CapacityHuman  string `json:"capacity_human"`
		DescribedHuman string `json:"described_human,omitempty"`
		Healthy        bool   `json:"healthy"`
	}{d, imaging.HumanBytes(d.Capacity), imaging.HumanBytes(d.Described), d.Healthy()})
}

// unprotect clears the flags keeping a drive read-only. It changes no data,
// so it asks for none of the confirmation a destructive call does.
func (s *Server) unprotect(w http.ResponseWriter, r *http.Request) {
	req, ok := decode(w, r)
	if !ok {
		return
	}
	scanner := s.Scanner()
	dev, err := scanner.Find(req.Device)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	if dev.SystemCritical() {
		failRefusal(w, device.CheckWritable(dev, "unprotect", true))
		return
	}

	protections := scanner.WriteProtection(dev)
	// Marking a dirty filesystem clean says it is safe without checking, so
	// the page asks for that one separately rather than folding it in.
	done, err := device.Unprotect(dev, protections, req.Force)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}

	// The drive has the last word on whether it worked.
	after := s.Scanner()
	remaining := protections
	if fresh, err := after.Find(dev.Path); err == nil {
		remaining = after.WriteProtection(fresh)
		// Flags reading clear proves only that the flags changed. A write
		// proves the drive takes writes, which is the actual question.
		if len(remaining) == 0 {
			if err := device.TestWrite(dev.Path); err != nil {
				remaining = []device.Protection{device.HardwareProtection(fresh)}
			}
		}
	}
	send(w, struct {
		Cleared   []string            `json:"cleared"`
		Remaining []device.Protection `json:"remaining,omitempty"`
		Free      bool                `json:"free"`
	}{done, remaining, len(remaining) == 0})
}

// surfaceTest writes to every part of a drive and reads it back, which is the
// only way to find a drive that takes writes and loses them.
func (s *Server) surfaceTest(w http.ResponseWriter, r *http.Request) {
	req, ok := decode(w, r)
	if !ok {
		return
	}
	dev, err := s.prepare(req, "test")
	if err != nil {
		failRefusal(w, err)
		return
	}

	job, err := s.start("test", dev, "", func(ctx context.Context, j *Job) error {
		res, err := imaging.SurfaceTest(ctx, dev.Path, imaging.SurfaceOptions{
			Quick:    req.Quick,
			Progress: j.progress,
		})
		if err != nil {
			return err
		}
		switch {
		case res.Counterfeit():
			j.finishWith(nil, fmt.Sprintf(
				"this drive is not the size it claims: it says %s and has %s. "+
					"Everything written past the real size wraps round and replaces what was written "+
					"first, which is why files on a drive like this go missing. There is no fixing it.",
				imaging.HumanBytes(res.Claimed), imaging.HumanBytes(res.RealSize)))
		case !res.Healthy():
			j.finishWith(nil, fmt.Sprintf(
				"%d region(s) did not hold what was written to them. Worn flash does this, and it "+
					"will keep losing what is put there.", len(res.Bad)))
		default:
			j.finishWith(nil, fmt.Sprintf("every part of the drive held what was written to it — %s checked",
				imaging.HumanBytes(res.Checked)))
		}
		return nil
	})
	if err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	send(w, job.Snapshot())
}

// repair clears what describes the drive and writes a full-size volume, which
// is what gives a drive its size back.
func (s *Server) repair(w http.ResponseWriter, r *http.Request) {
	req, ok := decode(w, r)
	if !ok {
		return
	}
	label, err := imaging.NormaliseLabel(req.Name)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	dev, err := s.prepare(req, "repair")
	if err != nil {
		failRefusal(w, err)
		return
	}
	before, err := imaging.Diagnose(dev.Path)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}

	job, err := s.start("repair", dev, "", func(ctx context.Context, j *Job) error {
		wiped, err := imaging.Wipe(ctx, dev.Path, imaging.WipeOptions{Progress: j.progress})
		if err != nil {
			return err
		}
		j.setPhase("formatting")
		formatted, err := imaging.FormatFAT32(dev.Path, imaging.FormatOptions{Label: label})
		if err != nil {
			return err
		}
		result := &imaging.RepairResult{Before: before, Format: formatted, Cleared: wiped.Bytes}
		summary := formatSummary(formatted)
		if recovered := result.Recovered(); recovered > 0 {
			summary += ", and " + imaging.HumanBytes(recovered) +
				" the drive was not admitting to is usable again"
		}
		j.finishWith(nil, summary)
		return nil
	})
	if err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	send(w, job.Snapshot())
}

func (s *Server) format(w http.ResponseWriter, r *http.Request) {
	req, ok := decode(w, r)
	if !ok {
		return
	}
	label, err := imaging.NormaliseLabel(req.Name)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	dev, err := s.prepare(req, "format")
	if err != nil {
		failRefusal(w, err)
		return
	}
	if n := dev.LogicalBlockSize; n != 0 && n != 512 {
		fail(w, http.StatusBadRequest, fmt.Errorf(
			"%s addresses %d-byte sectors, and Oliver writes FAT32 volumes with 512-byte ones; "+
				"formatting it here would produce a drive nothing can read", dev.Path, n))
		return
	}

	job, err := s.start("format", dev, "", func(ctx context.Context, j *Job) error {
		j.setPhase("formatting")
		res, err := imaging.FormatFAT32(dev.Path, imaging.FormatOptions{Label: label})
		if err != nil {
			return err
		}
		j.finishWith(nil, formatSummary(res))
		return nil
	})
	if err != nil {
		fail(w, http.StatusConflict, err)
		return
	}
	send(w, job.Snapshot())
}

func formatSummary(res *imaging.FormatResult) string {
	name := res.Label
	if name == "" {
		name = "unnamed"
	}
	return fmt.Sprintf("formatted as FAT32 %q, %s usable in %s clusters",
		name, imaging.HumanBytes(res.Usable), imaging.HumanBytes(int64(res.ClusterBytes)))
}

func decode(w http.ResponseWriter, r *http.Request) (*request, bool) {
	var req request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, fmt.Errorf("could not read the request: %w", err))
		return nil, false
	}
	return &req, true
}

// fail writes an error the page can show.
func fail(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// failRefusal reports a refused drive with the reason attached, so the page can
// offer the way past the ones that have one.
func failRefusal(w http.ResponseWriter, err error) {
	reason, ok := device.RefusedFor(err)
	if !ok {
		fail(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":       err.Error(),
		"reason":      reasonName(reason),
		"overridable": reason.Overridable(),
	})
}

func send(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func shortDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}
