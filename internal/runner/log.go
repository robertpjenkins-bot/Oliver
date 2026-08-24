package runner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogDir is where run logs are kept.
//
// A program launched from a file manager has no terminal, so anything it or
// Wine prints is lost. When such a launch fails all the user sees is a brief
// spinner, which is impossible to act on. Recording every run means there is
// always something to look at afterwards.
func LogDir() string {
	if d := os.Getenv("OLIVER_LOG_DIR"); d != "" {
		return d
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "oliver", "logs")
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "oliver", "logs")
}

// LogPathFor returns the log file for a named prefix.
func LogPathFor(name string) string {
	if name == "" {
		name = "run"
	}
	return filepath.Join(LogDir(), name+".log")
}

// openLog creates the log file and writes a header describing the launch, so
// a log read days later still says what was run and how.
func openLog(path string, plan *Plan, program string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(f, "oliver run %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(f, "program: %s\n", program)
	if plan.NewPrefix {
		fmt.Fprintf(f, "prefix:  created for this run\n")
	}
	for _, s := range plan.Setup {
		fmt.Fprintf(f, "\n--- %s ---\n%s\n", s.Desc, s.Pretty())
	}
	fmt.Fprintf(f, "\n--- %s ---\n%s\n\n", plan.Launch.Desc, plan.Launch.Pretty())
	fmt.Fprintf(f, "--- output ---\n")
	return f, nil
}

// Tail returns the last n non-empty lines of a log file.
func Tail(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scan.Scan() {
		line := strings.TrimRight(scan.Text(), " \t\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) > n*4 {
			lines = lines[len(lines)-n:]
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// LatestLog returns the most recently written log file, if any.
func LatestLog() (string, error) {
	entries, err := os.ReadDir(LogDir())
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no runs have been logged yet")
		}
		return "", err
	}
	type candidate struct {
		path string
		mod  time.Time
	}
	var found []candidate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		found = append(found, candidate{filepath.Join(LogDir(), e.Name()), info.ModTime()})
	}
	if len(found) == 0 {
		return "", fmt.Errorf("no runs have been logged yet")
	}
	sort.Slice(found, func(i, j int) bool { return found[i].mod.After(found[j].mod) })
	return found[0].path, nil
}

// Logs lists every log file, newest first.
func Logs() ([]string, error) {
	entries, err := os.ReadDir(LogDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			out = append(out, filepath.Join(LogDir(), e.Name()))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := os.Stat(out[i])
		b, _ := os.Stat(out[j])
		if a == nil || b == nil {
			return out[i] < out[j]
		}
		return a.ModTime().After(b.ModTime())
	})
	return out, nil
}

// looksLikeFailure picks out the lines most worth showing a user from a log.
// Wine is verbose and most of it is noise; the lines that name a real problem
// follow a small number of recognisable shapes.
func looksLikeFailure(line string) bool {
	l := strings.ToLower(line)
	for _, marker := range []string{
		"err:", "fatal", "error:", "unhandled exception", "no such file",
		"cannot find", "not found", "failed to", "permission denied",
		"is not a valid win32 application", "wine cannot find",
		"segmentation fault", "aborted", "traceback",
	} {
		if strings.Contains(l, marker) {
			return true
		}
	}
	return false
}

// Diagnose reads a log and returns the lines that explain a failure, falling
// back to the tail when nothing stands out.
func Diagnose(path string, max int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var hits []string
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line != "" && looksLikeFailure(line) {
			hits = append(hits, line)
			if len(hits) >= max {
				break
			}
		}
	}
	if len(hits) > 0 {
		return hits
	}
	return Tail(path, max)
}
