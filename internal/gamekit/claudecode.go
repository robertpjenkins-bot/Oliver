package gamekit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Designing through the Claude Code that is already on the machine.
//
// The other way to reach Claude is an API key, and for somebody who just wants
// to make a game that is a chore: find the console, make a key, keep it
// somewhere. Anyone who has installed Claude Code has already logged in, and
// that login can design a game as well as a key can. If the `claude` command
// is on the machine, the studio offers it, and then there is no key at all.
//
// It runs as an ordinary child process: one prompt in, one answer out. No
// session is kept, nothing is left behind, and it runs in an empty directory
// so a design request cannot wander into somebody's files.

// ClaudeCode is the command-line Claude on this machine.
type ClaudeCode struct {
	Bin     string
	Model   string
	Timeout time.Duration
}

// FindClaudeCode looks for the command.
//
// Looking on PATH alone is not enough, and the reason is the whole point of
// this studio being an AppImage: a program opened from a desktop icon gets a
// minimal PATH — /usr/bin and little else — while Claude Code installs into a
// home directory. "Claude Code is not installed" would then be wrong on a
// machine that has it, which is a bad thing to tell somebody about their own
// computer. So the places it actually installs are looked in by name.
func FindClaudeCode() *ClaudeCode {
	for _, p := range claudeSearch() {
		if isRunnable(p) {
			return &ClaudeCode{Bin: p, Timeout: 6 * time.Minute}
		}
	}
	return nil
}

// claudeSearch is the list of places to look. It is a variable so a test can
// look somewhere other than the machine it is running on.
var claudeSearch = claudeCandidates

// claudeCandidates lists every path worth trying, most explicit first.
func claudeCandidates() []string {
	var paths []string
	add := func(p string) {
		if p != "" {
			paths = append(paths, p)
		}
	}

	// An explicit path is an instruction, not a hint: if one is set, that is
	// the only place looked at. Guessing elsewhere would quietly ignore
	// somebody who told the studio exactly what to use — and it is what lets
	// anyone who would rather the studio left Claude Code alone point it at
	// nothing and mean it.
	// A path somebody put into the studio beats one in the environment: it is
	// the more deliberate of the two, and it is the one they just typed.
	if saved := LoadPreferences().ClaudeBin; saved != "" {
		return []string{saved}
	}
	if explicit := os.Getenv("OLIVER_CLAUDE_BIN"); explicit != "" {
		return []string{explicit}
	}
	if found, err := exec.LookPath("claude"); err == nil {
		add(found)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		for _, rel := range []string{
			".local/bin/claude",
			".claude/local/claude",
			".claude/local/node_modules/.bin/claude",
			".npm-global/bin/claude",
			".npm/bin/claude",
			".bun/bin/claude",
			".yarn/bin/claude",
			"node_modules/.bin/claude",
			"bin/claude",
		} {
			add(filepath.Join(home, rel))
		}
		// Node installed through nvm or fnm puts its binaries under a
		// version directory, and which version is anybody's guess.
		for _, pattern := range []string{
			".nvm/versions/node/*/bin/claude",
			".local/share/fnm/node-versions/*/installation/bin/claude",
			".volta/bin/claude",
		} {
			matches, _ := filepath.Glob(filepath.Join(home, pattern))
			for _, m := range matches {
				add(m)
			}
		}
	}

	for _, p := range []string{
		"/usr/local/bin/claude",
		"/usr/bin/claude",
		"/snap/bin/claude",
		"/opt/homebrew/bin/claude",
		"/opt/claude-code/bin/claude",
	} {
		add(p)
	}
	return paths
}

// isRunnable reports whether a path is something this process can execute.
func isRunnable(path string) bool {
	info, err := os.Stat(path) // follows symlinks, which is what an installer leaves
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

// SearchedFor lists where the studio looked, for when it found nothing. A
// person who knows their own machine can then see what was missed.
func SearchedFor() []string { return claudeSearch() }

// Available reports whether this is a usable provider.
func (c *ClaudeCode) Available() bool { return c != nil && c.Bin != "" }

// Version returns what the command reports, as proof it is really there.
func (c *ClaudeCode) Version(ctx context.Context) (string, error) {
	if !c.Available() {
		return "", fmt.Errorf("Claude Code is not installed on this machine")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, c.Bin, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("%s --version failed: %w", c.Bin, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// cliResult is the shape Claude Code prints with --output-format json.
type cliResult struct {
	Type     string  `json:"type"`
	Subtype  string  `json:"subtype"`
	IsError  bool    `json:"is_error"`
	Result   string  `json:"result"`
	Error    string  `json:"error"`
	APIError string  `json:"api_error_status"`
	Cost     float64 `json:"total_cost_usd"`
}

// Ask sends one prompt and returns the JSON object in the answer.
//
// There is no tool-use plumbing here the way there is with the API, so the
// prompt asks for the document on its own and the object is taken out of
// whatever comes back. A model that wrapped it in a sentence or a code fence
// is still understood.
func (c *ClaudeCode) Ask(ctx context.Context, prompt string) ([]byte, error) {
	if !c.Available() {
		return nil, fmt.Errorf("Claude Code is not installed on this machine")
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 6 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// An empty working directory: this is a request for a document, not an
	// invitation to read the machine it runs on.
	work, err := os.MkdirTemp("", "oliver-design-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)

	args := []string{"--print", prompt, "--output-format", "json"}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}
	out, err := c.run(ctx, work, args)
	if err != nil && c.Model != "" {
		// A model this installation does not offer is not worth failing over:
		// its own default designs games perfectly well.
		out, err = c.run(ctx, work, []string{"--print", prompt, "--output-format", "json"})
	}
	if err != nil {
		return nil, err
	}

	var res cliResult
	if jsonErr := json.Unmarshal(out, &res); jsonErr != nil {
		// Older or newer versions may print something else; the answer is
		// still in there.
		if obj := jsonObjectIn(string(out)); obj != "" {
			return []byte(obj), nil
		}
		return nil, fmt.Errorf("could not read what Claude Code printed: %w", jsonErr)
	}
	if res.IsError || res.Subtype != "success" {
		detail := firstNonEmpty(res.Error, res.APIError, res.Result, res.Subtype)
		return nil, fmt.Errorf("Claude Code could not answer: %s", truncate(strings.TrimSpace(detail), 200))
	}
	game := jsonObjectIn(res.Result)
	if game == "" {
		return nil, fmt.Errorf("Claude Code replied without a game in it")
	}
	return []byte(game), nil
}

func (c *ClaudeCode) run(ctx context.Context, dir string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.Bin, args...)
	cmd.Dir = dir
	// Claude Code reads the terminal for anything interactive; there is no
	// terminal here, and a prompt nobody can answer would hang the studio.
	cmd.Stdin = strings.NewReader("")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Claude Code was still thinking when the studio gave up waiting")
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("Claude Code failed: %s", truncate(detail, 300))
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "no detail given"
}
