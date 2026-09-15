package gamekit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// FindClaudeCode looks for the command. A machine without it simply has no
// second option; nothing about the studio changes.
func FindClaudeCode() *ClaudeCode {
	bin := os.Getenv("OLIVER_CLAUDE_BIN")
	if bin == "" {
		found, err := exec.LookPath("claude")
		if err != nil {
			return nil
		}
		bin = found
	}
	return &ClaudeCode{Bin: bin, Timeout: 6 * time.Minute}
}

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
